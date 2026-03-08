package decision

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"

	"ghost-os/bridge/llm"
)

func decisionMemoDerivedClaimIDs(memo DecisionMemo) []string {
	memo = normalizeDecisionMemoShallow(memo)
	objectID := decisionTruthObjectID("procedure.memo", memo.ID)
	subjectKind, subjectID, subjectLabel := decisionTruthSubject(memo.SessionID, objectID)
	claimIDs := make([]string, 0, 1+len(memo.AnchorKeys)+len(memo.GraphNodeRefs)+len(memo.Constraints)+len(memo.ValidationChecks)+len(memo.AvoidPatterns)+len(memo.FailureReasons)+len(memo.ToolsUsed)+1)
	appendClaim := func(claimType string, predicate string, objectKind string, objectIDValue string, objectLabel string, intentKey string, anchorKey string, entityID string, constraintType string, riskType string, value string, datatype string) {
		claimIDs = append(claimIDs, decisionTruthHashID("clm",
			objectID,
			claimType,
			predicate,
			subjectKind,
			subjectID,
			subjectLabel,
			objectKind,
			objectIDValue,
			objectLabel,
			intentKey,
			anchorKey,
			entityID,
			constraintType,
			riskType,
			value,
			datatype,
		))
	}
	if memo.IntentKey != "" {
		appendClaim("intent", "intent.key", "", "", "", memo.IntentKey, "", "", "", "", memo.IntentKey, "keyword")
	}
	for _, anchorKey := range memo.AnchorKeys {
		appendClaim("anchor", "anchor.preference", "", "", "", "", anchorKey, "", "", "", anchorKey, "keyword")
	}
	for _, entityID := range memo.GraphNodeRefs {
		appendClaim("entity", "entity.ref", "entity", entityID, entityID, "", "", entityID, "", "", entityID, "entity_ref")
	}
	for _, item := range memo.Constraints {
		appendClaim("constraint", "constraint.has", "", "", "", "", "", "", "constraint", "", item, "string")
	}
	for _, item := range memo.ValidationChecks {
		appendClaim("constraint", "validation.check", "", "", "", "", "", "", "validation_check", "", item, "string")
	}
	for _, item := range memo.AvoidPatterns {
		appendClaim("risk", "avoids_pattern", "", "", "", "", "", "", "", "avoid_pattern", item, "string")
	}
	for _, item := range memo.FailureReasons {
		appendClaim("risk", "failure_reason", "", "", "", "", "", "", "", "failure_reason", item, "string")
	}
	for _, tool := range memo.ToolsUsed {
		appendClaim("tool", "uses_tool", "", "", "", "", "", "", "", "", tool.Name, "tool_name")
	}
	if memo.Outcome != "" || memo.OutcomeSummary != "" {
		appendClaim("outcome", "outcome.summary", "", "", "", "", "", "", "", "", firstNonEmpty(memo.OutcomeSummary, memo.Outcome), "string")
	}
	return uniqueStrings(claimIDs)
}

func normalizeDecisionMemoShallow(memo DecisionMemo) DecisionMemo {
	out := memo
	out.ID = strings.TrimSpace(out.ID)
	out.SessionID = strings.TrimSpace(out.SessionID)
	out.IntentKey = strings.TrimSpace(out.IntentKey)
	out.Outcome = normalizeDecisionOutcome(out.Outcome)
	out.OutcomeSummary = strings.TrimSpace(out.OutcomeSummary)
	out.AnchorKeys = uniqueStrings(out.AnchorKeys)
	out.GraphNodeRefs = uniqueStrings(out.GraphNodeRefs)
	out.Constraints = uniqueStrings(out.Constraints)
	out.ValidationChecks = uniqueStrings(out.ValidationChecks)
	out.AvoidPatterns = uniqueStrings(out.AvoidPatterns)
	out.FailureReasons = uniqueStrings(out.FailureReasons)
	out.ToolsUsed = normalizeDecisionToolUses(out.ToolsUsed)
	return out
}

func decisionTruthSubject(sessionID string, fallbackID string) (kind string, id string, label string) {
	if trimmed := strings.TrimSpace(sessionID); trimmed != "" {
		return "session", trimmed, trimmed
	}
	trimmed := strings.TrimSpace(fallbackID)
	return "object", trimmed, trimmed
}

func decisionTruthObjectID(kind string, parts ...string) string {
	values := append([]string{strings.TrimSpace(kind)}, parts...)
	return decisionTruthHashID("obj", values...)
}

func decisionTruthHashID(prefix string, parts ...string) string {
	normalized := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		normalized = append(normalized, trimmed)
	}
	hash := sha1.Sum([]byte(strings.Join(normalized, "|")))
	return prefix + "-" + hex.EncodeToString(hash[:10])
}

func decisionCaptureLineage(input DecisionCaptureInput) DecisionLineage {
	events := make([]string, 0, len(input.NewMessages)+len(input.AnsweredQuestions)+1)
	evidence := make([]string, 0, len(input.NewMessages)+len(input.AnsweredQuestions)+1)
	if turnID := strings.TrimSpace(firstNonEmpty(input.TurnID, input.TraceID, input.SessionID)); turnID != "" {
		events = append(events, "turn:"+turnID)
	}
	for index, msg := range input.NewMessages {
		text := strings.TrimSpace(messageToContent(msg))
		role := strings.TrimSpace(string(msg.Role))
		eventID := decisionTruthHashID("evt", firstNonEmpty(input.SessionID, "session"), firstNonEmpty(input.TurnID, input.TraceID), role, fmt.Sprintf("%d", index), strings.TrimSpace(msg.ToolCallID), text)
		events = append(events, eventID)
		if text != "" {
			evidence = append(evidence, decisionTruthHashID("evd", role, strings.TrimSpace(msg.ToolCallID), text, eventID))
		}
		for _, call := range msg.ToolCalls {
			callID := strings.TrimSpace(call.ID)
			name := strings.TrimSpace(call.Name)
			if callID == "" && name == "" {
				continue
			}
			events = append(events, decisionTruthHashID("evt", "tool_call", callID, name, string(call.Arguments)))
			evidence = append(evidence, decisionTruthHashID("evd", "tool_call", callID, name, string(call.Arguments)))
		}
	}
	for _, item := range normalizeDecisionAnsweredQuestions(input.AnsweredQuestions) {
		eventID := decisionTruthHashID("evt", "ask_human", item.QuestionID, item.TraceID, item.Prompt, item.Answer)
		events = append(events, eventID)
		evidence = append(evidence,
			decisionTruthHashID("evd", "ask_human.prompt", item.QuestionID, item.Prompt),
			decisionTruthHashID("evd", "ask_human.answer", item.QuestionID, item.Answer),
		)
	}
	return normalizeDecisionLineage(DecisionLineage{
		SourceEventIDs:    events,
		SourceEvidenceIDs: evidence,
		LineageVersion:    decisionLineageVersion,
		LineagePartial:    len(evidence) == 0,
	})
}

func mergeDecisionLineage(base DecisionLineage, overlays ...DecisionLineage) DecisionLineage {
	out := cloneDecisionLineage(base)
	for _, overlay := range overlays {
		normalized := normalizeDecisionLineage(overlay)
		out.SourceEventIDs = uniqueStrings(append(out.SourceEventIDs, normalized.SourceEventIDs...))
		out.SourceEvidenceIDs = uniqueStrings(append(out.SourceEvidenceIDs, normalized.SourceEvidenceIDs...))
		out.SourceClaimIDs = uniqueStrings(append(out.SourceClaimIDs, normalized.SourceClaimIDs...))
		out.DerivedClaimIDs = uniqueStrings(append(out.DerivedClaimIDs, normalized.DerivedClaimIDs...))
		out.ContradictedClaimIDs = uniqueStrings(append(out.ContradictedClaimIDs, normalized.ContradictedClaimIDs...))
		out.SelectionClaimIDs = uniqueStrings(append(out.SelectionClaimIDs, normalized.SelectionClaimIDs...))
		out.SelectionEvidenceIDs = uniqueStrings(append(out.SelectionEvidenceIDs, normalized.SelectionEvidenceIDs...))
		out.ExecutionEvidenceIDs = uniqueStrings(append(out.ExecutionEvidenceIDs, normalized.ExecutionEvidenceIDs...))
		out.EmittedClaimIDs = uniqueStrings(append(out.EmittedClaimIDs, normalized.EmittedClaimIDs...))
		out.InvalidatedClaimIDs = uniqueStrings(append(out.InvalidatedClaimIDs, normalized.InvalidatedClaimIDs...))
		out.MatchedClaimIDs = uniqueStrings(append(out.MatchedClaimIDs, normalized.MatchedClaimIDs...))
		out.MatchedEvidenceIDs = uniqueStrings(append(out.MatchedEvidenceIDs, normalized.MatchedEvidenceIDs...))
		out.MissingRequiredClaimIDs = uniqueStrings(append(out.MissingRequiredClaimIDs, normalized.MissingRequiredClaimIDs...))
		out.ConflictedClaimIDs = uniqueStrings(append(out.ConflictedClaimIDs, normalized.ConflictedClaimIDs...))
		if out.LineageSummary == "" {
			out.LineageSummary = normalized.LineageSummary
		}
		if out.DistillerVersion == "" {
			out.DistillerVersion = normalized.DistillerVersion
		}
		out.FallbackDueToClaimConflict = out.FallbackDueToClaimConflict || normalized.FallbackDueToClaimConflict
		out.LineagePartial = out.LineagePartial || normalized.LineagePartial
	}
	return normalizeDecisionLineage(out)
}

func decisionLineageFromMemo(memo DecisionMemo) DecisionLineage {
	return normalizeDecisionLineage(DecisionLineage{
		SourceEventIDs:    memo.SourceEventIDs,
		SourceEvidenceIDs: memo.SourceEvidenceIDs,
		SourceClaimIDs:    append(cloneStrings(memo.SourceClaimIDs), memo.DerivedClaimIDs...),
		DerivedClaimIDs:   memo.DerivedClaimIDs,
		LineageVersion:    firstNonEmpty(memo.LineageVersion, decisionLineageVersion),
		LineagePartial:    memo.LineagePartial,
	})
}

func decisionLineageSummary(prefix string, claimIDs []string, evidenceIDs []string, memoIDs []string) string {
	parts := make([]string, 0, 4)
	if trimmed := strings.TrimSpace(prefix); trimmed != "" {
		parts = append(parts, trimmed)
	}
	if len(memoIDs) > 0 {
		parts = append(parts, fmt.Sprintf("memos=%d", len(uniqueStrings(memoIDs))))
	}
	if len(claimIDs) > 0 {
		parts = append(parts, fmt.Sprintf("claims=%d", len(uniqueStrings(claimIDs))))
	}
	if len(evidenceIDs) > 0 {
		parts = append(parts, fmt.Sprintf("evidence=%d", len(uniqueStrings(evidenceIDs))))
	}
	return strings.Join(parts, " ")
}

func decisionCountStableIDs(values []string, sampleCount int, minRatio float64) []string {
	if len(values) == 0 {
		return nil
	}
	counts := make(map[string]int, len(values))
	for _, value := range values {
		counts[strings.TrimSpace(value)]++
	}
	threshold := 1
	if sampleCount > 1 {
		threshold = max(1, int(float64(sampleCount)*minRatio+0.999))
	}
	out := make([]string, 0, len(counts))
	for value, count := range counts {
		if value == "" || count < threshold {
			continue
		}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func intersectStrings(left []string, right []string) []string {
	if len(left) == 0 || len(right) == 0 {
		return nil
	}
	set := make(map[string]struct{}, len(right))
	for _, item := range right {
		trimmed := strings.TrimSpace(item)
		if trimmed != "" {
			set[trimmed] = struct{}{}
		}
	}
	out := make([]string, 0, len(left))
	for _, item := range left {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		if _, ok := set[trimmed]; ok {
			out = append(out, trimmed)
		}
	}
	return uniqueStrings(out)
}

func diffStrings(left []string, right []string) []string {
	if len(left) == 0 {
		return nil
	}
	set := make(map[string]struct{}, len(right))
	for _, item := range right {
		trimmed := strings.TrimSpace(item)
		if trimmed != "" {
			set[trimmed] = struct{}{}
		}
	}
	out := make([]string, 0, len(left))
	for _, item := range left {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		if _, ok := set[trimmed]; ok {
			continue
		}
		out = append(out, trimmed)
	}
	return uniqueStrings(out)
}

func lineageIDsFromMessages(messages []llm.Message) []string {
	if len(messages) == 0 {
		return nil
	}
	out := make([]string, 0, len(messages))
	for index, msg := range messages {
		text := strings.TrimSpace(messageToContent(msg))
		if text == "" && len(msg.ToolCalls) == 0 {
			continue
		}
		out = append(out, decisionTruthHashID("evd", string(msg.Role), fmt.Sprintf("%d", index), strings.TrimSpace(msg.ToolCallID), text))
	}
	return uniqueStrings(out)
}

func latestLineageTime(values ...time.Time) time.Time {
	var out time.Time
	for _, value := range values {
		if value.IsZero() {
			continue
		}
		value = value.UTC()
		if out.IsZero() || value.After(out) {
			out = value
		}
	}
	return out
}
