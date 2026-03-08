package memory

import (
	"fmt"
	"strings"
	"time"

	"ghost-os/bridge/llm"
)

// TruthMapper 负责把现有 sidecar 结构投影到 truth v2，并保留兼容 object projection。
type TruthMapper struct{}

func NewTruthMapper() *TruthMapper {
	return &TruthMapper{}
}

func (m *TruthMapper) MapArchiveMessages(sessionID string, archivedAt time.Time, messages []llm.Message) []MemoryObject {
	sid := strings.TrimSpace(sessionID)
	if sid == "" || len(messages) == 0 {
		return nil
	}
	base := archivedAt.UTC()
	if base.IsZero() {
		base = time.Now().UTC()
	}
	objects := make([]MemoryObject, 0, len(messages))
	for index, msg := range messages {
		content := strings.TrimSpace(messageToContent(msg))
		if content == "" {
			continue
		}
		sourceID := fmt.Sprintf("%s:%06d", sid, index)
		timestamp := base.Add(time.Duration(index) * time.Millisecond)
		primarySource := SourceRef{
			Namespace:   defaultLedgerNamespace,
			SessionID:   sid,
			BucketMonth: bucketMonthFromTime(timestamp),
			OccurredAt:  timestamp,
			SourceKind:  truthSourceKindArchiveMessage,
			SourceID:    sourceID,
		}
		objectID := buildTruthObjectID(truthObjectTypeEvidenceMessage, sid, fmt.Sprintf("%06d", index))
		evidence := normalizeMemoryEvidence(MemoryEvidence{
			Kind:       "chat.message",
			Text:       content,
			Summary:    summarizeLine(content, 220),
			Timestamp:  timestamp,
			Confidence: 1,
			ToolCallID: strings.TrimSpace(msg.ToolCallID),
			SourceRefs: []SourceRef{primarySource},
			Metadata: map[string]any{
				"role":          string(msg.Role),
				"tool_call_id":  strings.TrimSpace(msg.ToolCallID),
				"message_index": index,
			},
		}, objectID)
		object := MemoryObject{
			ObjectID:    objectID,
			ObjectType:  truthObjectTypeEvidenceMessage,
			Summary:     evidence.Summary,
			RawEvidence: []MemoryEvidence{evidence},
			SourceRefs:  []SourceRef{primarySource},
			CreatedAt:   timestamp,
			UpdatedAt:   timestamp,
			Confidence:  1,
			Metadata: map[string]any{
				"session_id":    sid,
				"message_index": index,
				"role":          string(msg.Role),
			},
		}
		objects = append(objects, normalizeMemoryObject(object))
	}
	return objects
}

func (m *TruthMapper) MapMarkdownNode(node MarkdownNode) MemoryObject {
	node = normalizeMarkdownNode(node)
	primarySource := SourceRef{
		Namespace:   firstNonEmpty(node.Namespace, defaultLedgerNamespace),
		WorkspaceID: node.WorkspaceID,
		BucketKey:   node.BucketKey,
		BucketMonth: bucketMonthFromTime(firstNonZeroTime(node.LastSeenAt, node.CreatedAt)),
		SessionID:   node.SessionID,
		OccurredAt:  firstNonZeroTime(node.LastSeenAt, node.CreatedAt),
		SourceKind:  truthSourceKindMarkdownNode,
		SourceID:    node.ID,
	}
	sourceRefs := []SourceRef{primarySource}
	for _, sourceID := range node.SourceIDs {
		sourceRefs = append(sourceRefs, SourceRef{
			Namespace:   primarySource.Namespace,
			WorkspaceID: primarySource.WorkspaceID,
			BucketKey:   primarySource.BucketKey,
			BucketMonth: primarySource.BucketMonth,
			SessionID:   node.SessionID,
			OccurredAt:  primarySource.OccurredAt,
			SourceKind:  truthSourceKindMarkdownSource,
			SourceID:    sourceID,
		})
	}
	objectID := buildTruthObjectID(truthObjectTypeSemanticNote, node.ID)
	evidenceText := firstNonEmpty(node.Content, node.Summary)
	evidence := []MemoryEvidence{}
	if evidenceText != "" {
		evidence = append(evidence, normalizeMemoryEvidence(MemoryEvidence{
			Kind:       "markdown.note",
			Text:       evidenceText,
			Summary:    firstNonEmpty(node.Summary, summarizeLine(evidenceText, 220)),
			Timestamp:  effectiveDecisionTimestamp(node.LastSeenAt, node.CreatedAt),
			Confidence: maxFloat(node.Confidence, strongestAnchorWeight(node.Anchors)),
			SourceRefs: []SourceRef{primarySource},
		}, objectID))
	}
	evidenceRefs := truthEvidenceRefsForObject(objectID, evidence)
	subject := truthSubjectFromSource(primarySource, objectID)
	claims := make([]MemoryClaim, 0, len(node.Anchors))
	for _, anchor := range node.Anchors {
		normalized := normalizeAnchor(anchor)
		if normalized.Type == "" || normalized.Value == "" {
			continue
		}
		claim := MemoryClaim{
			ObjectID:     objectID,
			Subject:      subject,
			Predicate:    truthAnchorPredicate(normalized),
			Type:         truthClaimTypeAnchor,
			Value:        normalized.Value,
			Datatype:     "string",
			EvidenceRefs: evidenceRefs,
			AnchorKey:    firstNonEmpty(normalized.Key, truthHashID("anchor", anchorFingerprint(normalized))),
			Confidence:   maxFloat(normalized.Weight, node.Confidence),
			AssertedAt:   effectiveDecisionTimestamp(normalized.DetectedAt, node.CreatedAt),
			CreatedAt:    effectiveDecisionTimestamp(normalized.DetectedAt, node.CreatedAt),
			SourceRefs:   sourceRefs,
			Metadata: map[string]any{
				"anchor_type": normalized.Type,
				"reason":      normalized.Reason,
				"related_to":  append([]string(nil), node.RelatedTo...),
				"source_ids":  append([]string(nil), node.SourceIDs...),
			},
		}
		if normalized.Type == MemoryAnchorConstraint {
			claim.ConstraintType = normalized.Type
		}
		if normalized.Type == MemoryAnchorAvoidance {
			claim.RiskType = "avoid_pattern"
		}
		if claim.Predicate == "preference.language" {
			claim.Datatype = "language"
		}
		claims = append(claims, claim)
	}
	embeddingRefs := make([]EmbeddingRef, 0, 1)
	if node.EmbeddingID != "" {
		embeddingRefs = append(embeddingRefs, EmbeddingRef{
			EmbeddingID: node.EmbeddingID,
			Source:      "markdown_node",
			Status:      "linked",
			SourceRefs:  []SourceRef{primarySource},
		})
	}
	object := MemoryObject{
		ObjectID:      objectID,
		ObjectType:    truthObjectTypeSemanticNote,
		Summary:       firstNonEmpty(node.Summary, summarizeLine(evidenceText, 220)),
		Claims:        claims,
		RawEvidence:   evidence,
		EmbeddingRefs: embeddingRefs,
		SourceRefs:    sourceRefs,
		CreatedAt:     node.CreatedAt,
		UpdatedAt:     effectiveDecisionTimestamp(node.LastSeenAt, node.CreatedAt),
		Confidence:    maxFloat(node.Confidence, strongestAnchorWeight(node.Anchors)),
		Metadata: map[string]any{
			"session_id": node.SessionID,
			"tags":       append([]string(nil), node.Tags...),
			"source_ids": append([]string(nil), node.SourceIDs...),
			"related_to": append([]string(nil), node.RelatedTo...),
		},
	}
	return normalizeMemoryObject(object)
}

func (m *TruthMapper) MapDecisionMemo(memo DecisionMemo, input DecisionCaptureInput) MemoryObject {
	memo = normalizeDecisionMemo(memo)
	decisionAt := effectiveDecisionTimestamp(memo.LastUsedAt, memo.CreatedAt, input.TurnFinishedAt, input.TurnStartedAt)
	primarySource := SourceRef{
		Namespace:   normalizeDecisionNamespace(firstNonEmpty(memo.Namespace, input.Namespace)),
		BucketMonth: bucketMonthFromTime(decisionAt),
		SessionID:   memo.SessionID,
		TurnID:      memo.TurnID,
		TraceID:     memo.TraceID,
		OccurredAt:  decisionAt,
		SourceKind:  truthSourceKindDecisionMemo,
		SourceID:    memo.ID,
	}
	sourceRefs := []SourceRef{primarySource}
	inputSourceID := firstNonEmpty(strings.TrimSpace(input.TurnID), strings.TrimSpace(input.TraceID), strings.TrimSpace(input.SessionID))
	if inputSourceID != "" {
		sourceRefs = append(sourceRefs, SourceRef{
			Namespace:   primarySource.Namespace,
			BucketMonth: primarySource.BucketMonth,
			SessionID:   strings.TrimSpace(input.SessionID),
			TurnID:      strings.TrimSpace(input.TurnID),
			TraceID:     strings.TrimSpace(input.TraceID),
			OccurredAt:  effectiveDecisionTimestamp(input.TurnFinishedAt, input.TurnStartedAt),
			SourceKind:  truthSourceKindDecisionInput,
			SourceID:    inputSourceID,
		})
	}
	objectID := buildTruthObjectID(truthObjectTypeProcedureMemo, memo.ID)
	evidence := buildDecisionTruthEvidence(memo, input, primarySource, decisionAt)
	evidenceRefs := truthEvidenceRefsForObject(objectID, evidence)
	claims := buildDecisionTruthClaims(objectID, memo, sourceRefs, decisionAt, evidenceRefs)
	object := MemoryObject{
		ObjectID:    objectID,
		ObjectType:  truthObjectTypeProcedureMemo,
		Summary:     firstNonEmpty(memo.StrategySummary, memo.OutcomeSummary, memo.IntentSummary, memo.ProblemSummary),
		RawEvidence: evidence,
		Claims:      claims,
		SourceRefs:  sourceRefs,
		CreatedAt:   effectiveDecisionTimestamp(memo.CreatedAt, input.TurnFinishedAt, input.TurnStartedAt),
		UpdatedAt:   decisionAt,
		Confidence:  maxFloat(memo.Confidence, memo.ReuseScore),
		Metadata: map[string]any{
			"namespace":          memo.Namespace,
			"outcome":            memo.Outcome,
			"human_blocked":      memo.HumanBlocked,
			"graph_node_refs":    append([]string(nil), memo.GraphNodeRefs...),
			"graph_edge_refs":    append([]string(nil), memo.GraphEdgeRefs...),
			"anchor_keys":        append([]string(nil), memo.AnchorKeys...),
			"environment_domain": memo.Environment.Domain,
		},
	}
	return normalizeMemoryObject(object)
}

func buildDecisionTruthClaims(objectID string, memo DecisionMemo, sourceRefs []SourceRef, createdAt time.Time, evidenceRefs []EvidenceRef) []MemoryClaim {
	subject := truthSubjectFromSource(sourceRefs[0], objectID)
	claims := make([]MemoryClaim, 0, 1+len(memo.AnchorKeys)+len(memo.GraphNodeRefs)+len(memo.Constraints)+len(memo.ValidationChecks)+len(memo.AvoidPatterns)+len(memo.FailureReasons)+len(memo.ToolsUsed))
	if memo.IntentKey != "" {
		claims = append(claims, MemoryClaim{
			ObjectID:     objectID,
			Subject:      subject,
			Predicate:    "intent.key",
			Type:         truthClaimTypeIntent,
			IntentKey:    memo.IntentKey,
			Value:        memo.IntentKey,
			Datatype:     "keyword",
			EvidenceRefs: evidenceRefs,
			Confidence:   memo.Confidence,
			AssertedAt:   createdAt,
			CreatedAt:    createdAt,
			SourceRefs:   sourceRefs,
			Metadata: map[string]any{
				"summary": firstNonEmpty(memo.IntentSummary, memo.ProblemSummary, memo.StrategySummary),
			},
		})
	}
	for _, anchorKey := range memo.AnchorKeys {
		claims = append(claims, MemoryClaim{
			ObjectID:     objectID,
			Subject:      subject,
			Predicate:    "anchor.preference",
			Type:         truthClaimTypeAnchor,
			AnchorKey:    anchorKey,
			Value:        anchorKey,
			Datatype:     "keyword",
			EvidenceRefs: evidenceRefs,
			Confidence:   memo.Confidence,
			AssertedAt:   createdAt,
			CreatedAt:    createdAt,
			SourceRefs:   sourceRefs,
		})
	}
	for _, entityID := range memo.GraphNodeRefs {
		claims = append(claims, MemoryClaim{
			ObjectID:     objectID,
			Subject:      subject,
			Predicate:    "entity.ref",
			Object:       ClaimTerm{Kind: "entity", ID: entityID, Label: entityID},
			Type:         truthClaimTypeEntity,
			EntityID:     entityID,
			Value:        entityID,
			Datatype:     "entity_ref",
			EvidenceRefs: evidenceRefs,
			Confidence:   memo.Confidence,
			AssertedAt:   createdAt,
			CreatedAt:    createdAt,
			SourceRefs:   sourceRefs,
		})
	}
	for _, item := range memo.Constraints {
		claims = append(claims, MemoryClaim{
			ObjectID:       objectID,
			Subject:        subject,
			Predicate:      "constraint.has",
			Type:           truthClaimTypeConstraint,
			ConstraintType: "constraint",
			Value:          item,
			Datatype:       "string",
			EvidenceRefs:   evidenceRefs,
			Confidence:     memo.Confidence,
			AssertedAt:     createdAt,
			CreatedAt:      createdAt,
			SourceRefs:     sourceRefs,
		})
	}
	for _, item := range memo.ValidationChecks {
		claims = append(claims, MemoryClaim{
			ObjectID:       objectID,
			Subject:        subject,
			Predicate:      "validation.check",
			Type:           truthClaimTypeConstraint,
			ConstraintType: "validation_check",
			Value:          item,
			Datatype:       "string",
			EvidenceRefs:   evidenceRefs,
			Confidence:     memo.Confidence,
			AssertedAt:     createdAt,
			CreatedAt:      createdAt,
			SourceRefs:     sourceRefs,
		})
	}
	for _, item := range memo.AvoidPatterns {
		claims = append(claims, MemoryClaim{
			ObjectID:     objectID,
			Subject:      subject,
			Predicate:    "avoids_pattern",
			Type:         truthClaimTypeRisk,
			RiskType:     "avoid_pattern",
			Value:        item,
			Datatype:     "string",
			EvidenceRefs: evidenceRefs,
			Confidence:   memo.Confidence,
			AssertedAt:   createdAt,
			CreatedAt:    createdAt,
			SourceRefs:   sourceRefs,
		})
	}
	for _, item := range memo.FailureReasons {
		claims = append(claims, MemoryClaim{
			ObjectID:     objectID,
			Subject:      subject,
			Predicate:    "failure_reason",
			Type:         truthClaimTypeRisk,
			RiskType:     "failure_reason",
			Value:        item,
			Datatype:     "string",
			EvidenceRefs: evidenceRefs,
			Confidence:   memo.Confidence,
			AssertedAt:   createdAt,
			CreatedAt:    createdAt,
			SourceRefs:   sourceRefs,
		})
	}
	for _, tool := range memo.ToolsUsed {
		claims = append(claims, MemoryClaim{
			ObjectID:     objectID,
			Subject:      subject,
			Predicate:    "uses_tool",
			Type:         truthClaimTypeTool,
			Value:        tool.Name,
			Datatype:     "tool_name",
			EvidenceRefs: evidenceRefs,
			Confidence:   memo.Confidence,
			AssertedAt:   createdAt,
			CreatedAt:    createdAt,
			SourceRefs:   sourceRefs,
			Metadata: map[string]any{
				"purpose":        tool.Purpose,
				"input_summary":  tool.InputSummary,
				"output_summary": tool.OutputSummary,
			},
		})
	}
	if memo.Outcome != "" || memo.OutcomeSummary != "" {
		claims = append(claims, MemoryClaim{
			ObjectID:     objectID,
			Subject:      subject,
			Predicate:    "outcome.summary",
			Type:         truthClaimTypeOutcome,
			Value:        firstNonEmpty(memo.OutcomeSummary, memo.Outcome),
			Datatype:     "string",
			EvidenceRefs: evidenceRefs,
			Confidence:   memo.Confidence,
			AssertedAt:   createdAt,
			CreatedAt:    createdAt,
			SourceRefs:   sourceRefs,
			Metadata: map[string]any{
				"outcome": memo.Outcome,
			},
		})
	}
	return claims
}

func buildDecisionTruthEvidence(memo DecisionMemo, input DecisionCaptureInput, source SourceRef, decisionAt time.Time) []MemoryEvidence {
	evidence := make([]MemoryEvidence, 0, 2)
	objectID := buildTruthObjectID(truthObjectTypeProcedureMemo, memo.ID)
	memoText := renderDecisionTruthMemoText(memo)
	if memoText != "" {
		evidence = append(evidence, normalizeMemoryEvidence(MemoryEvidence{
			Kind:       "decision.memo",
			Text:       memoText,
			Summary:    firstNonEmpty(memo.StrategySummary, memo.OutcomeSummary, memo.IntentSummary, memo.ProblemSummary),
			Timestamp:  decisionAt,
			Confidence: maxFloat(memo.Confidence, memo.ReuseScore),
			SourceRefs: []SourceRef{source},
		}, objectID))
	}
	userMessage := strings.TrimSpace(input.UserMessage)
	if userMessage != "" {
		evidence = append(evidence, normalizeMemoryEvidence(MemoryEvidence{
			Kind:       "decision.input",
			Text:       userMessage,
			Summary:    summarizeLine(userMessage, 180),
			Timestamp:  effectiveDecisionTimestamp(input.TurnFinishedAt, input.TurnStartedAt, memo.CreatedAt),
			Confidence: 1,
			SourceRefs: []SourceRef{{
				SessionID:  strings.TrimSpace(input.SessionID),
				TurnID:     strings.TrimSpace(input.TurnID),
				TraceID:    strings.TrimSpace(input.TraceID),
				SourceKind: truthSourceKindDecisionInput,
				SourceID:   firstNonEmpty(strings.TrimSpace(input.TurnID), strings.TrimSpace(input.TraceID), strings.TrimSpace(input.SessionID)),
			}},
		}, objectID))
	}
	return evidence
}

func renderDecisionTruthMemoText(memo DecisionMemo) string {
	sections := make([]string, 0, 6)
	appendSection := func(title string, values ...string) {
		parts := make([]string, 0, len(values))
		for _, value := range values {
			trimmed := strings.TrimSpace(value)
			if trimmed == "" {
				continue
			}
			parts = append(parts, trimmed)
		}
		if len(parts) == 0 {
			return
		}
		sections = append(sections, title+": "+strings.Join(parts, " | "))
	}
	appendSection("intent", memo.IntentSummary, memo.IntentKey)
	appendSection("problem", memo.ProblemSummary)
	appendSection("context", memo.ContextSummary)
	appendSection("strategy", memo.StrategySummary)
	appendSection("outcome", memo.OutcomeSummary, memo.Outcome)
	appendSection("constraints", strings.Join(memo.Constraints, "; "))
	return strings.Join(sections, "\n")
}

func truthSubjectFromSource(primarySource SourceRef, fallbackID string) ClaimTerm {
	if strings.TrimSpace(primarySource.WorkspaceID) != "" {
		return ClaimTerm{Kind: "workspace", ID: strings.TrimSpace(primarySource.WorkspaceID), Label: strings.TrimSpace(primarySource.WorkspaceID)}
	}
	if strings.TrimSpace(primarySource.SessionID) != "" {
		return ClaimTerm{Kind: "session", ID: strings.TrimSpace(primarySource.SessionID), Label: strings.TrimSpace(primarySource.SessionID)}
	}
	return ClaimTerm{Kind: "object", ID: strings.TrimSpace(fallbackID), Label: strings.TrimSpace(fallbackID)}
}

func truthEvidenceRefsForObject(objectID string, evidence []MemoryEvidence) []EvidenceRef {
	if len(evidence) == 0 {
		return nil
	}
	refs := make([]EvidenceRef, 0, len(evidence))
	for _, item := range evidence {
		normalized := normalizeMemoryEvidence(item, objectID)
		if normalized.EvidenceID == "" {
			continue
		}
		refs = append(refs, EvidenceRef{EvidenceID: normalized.EvidenceID, Role: "support"})
	}
	return normalizeEvidenceRefs(refs)
}

func truthAnchorPredicate(anchor MemoryAnchor) string {
	switch anchor.Type {
	case MemoryAnchorPreference:
		if strings.Contains(truthIndexKey(anchor.Key), "language") {
			return "preference.language"
		}
		return "anchor.preference"
	case MemoryAnchorConstraint:
		return "constraint.has"
	case MemoryAnchorAvoidance:
		return "avoids_pattern"
	default:
		return "anchor." + truthIndexKey(anchor.Type)
	}
}
