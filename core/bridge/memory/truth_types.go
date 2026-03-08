package memory

import (
	"sort"
	"strings"
	"time"
)

const truthSchemaVersion = 1

const (
	truthObjectTypeEvidenceMessage = "evidence.message"
	truthObjectTypeSemanticNote    = "semantic.note"
	truthObjectTypeProcedureMemo   = "procedure.memo"
)

const (
	truthEventTypeArchiveMessage = "archive.message.upsert"
	truthEventTypeMarkdownNode   = "markdown.node.upsert"
	truthEventTypeDecisionMemo   = "decision.memo.upsert"
)

const (
	truthSourceKindArchiveMessage = "archive_message"
	truthSourceKindMarkdownNode   = "markdown_node"
	truthSourceKindMarkdownSource = "markdown_source"
	truthSourceKindDecisionMemo   = "decision_memo"
	truthSourceKindDecisionInput  = "decision_input"
)

const (
	truthClaimTypeIntent     = "intent"
	truthClaimTypeAnchor     = "anchor"
	truthClaimTypeEntity     = "entity"
	truthClaimTypeConstraint = "constraint"
	truthClaimTypeRisk       = "risk"
	truthClaimTypeTool       = "tool"
	truthClaimTypeOutcome    = "outcome"
)

// SourceRef 统一描述对象/证据/claim 的来源引用，保证后续可追溯。
type SourceRef struct {
	Namespace   string `json:"namespace,omitempty"`
	WorkspaceID string `json:"workspace_id,omitempty"`
	BucketKey   string `json:"bucket_key,omitempty"`
	BucketMonth string `json:"bucket_month,omitempty"`
	SessionID  string `json:"session_id,omitempty"`
	TurnID     string `json:"turn_id,omitempty"`
	TraceID    string `json:"trace_id,omitempty"`
	SourceKind string `json:"source_kind,omitempty"`
	SourceID   string `json:"source_id,omitempty"`
	OccurredAt time.Time `json:"occurred_at,omitempty"`
}

// MemoryEvidence 保存对象的原始证据视图。
type MemoryEvidence struct {
	EvidenceID string         `json:"evidence_id"`
	Kind       string         `json:"kind,omitempty"`
	Text       string         `json:"text,omitempty"`
	Summary    string         `json:"summary,omitempty"`
	Timestamp  time.Time      `json:"timestamp,omitempty"`
	Confidence float64        `json:"confidence,omitempty"`
	SourceRefs []SourceRef    `json:"source_refs,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}

// MemoryClaim 保存对象上的稳定 claim 视图。
type MemoryClaim struct {
	SchemaVersion  int            `json:"schema_version"`
	ClaimID        string         `json:"claim_id"`
	ObjectID       string         `json:"object_id,omitempty"`
	Type           string         `json:"type,omitempty"`
	Value          string         `json:"value,omitempty"`
	IntentKey      string         `json:"intent_key,omitempty"`
	AnchorKey      string         `json:"anchor_key,omitempty"`
	EntityID       string         `json:"entity_id,omitempty"`
	ConstraintType string         `json:"constraint_type,omitempty"`
	RiskType       string         `json:"risk_type,omitempty"`
	Confidence     float64        `json:"confidence,omitempty"`
	CreatedAt      time.Time      `json:"created_at,omitempty"`
	SourceRefs     []SourceRef    `json:"source_refs,omitempty"`
	Metadata       map[string]any `json:"metadata,omitempty"`
}

// EmbeddingRef 对齐现有 EmbeddingID，先只承载占位引用。
type EmbeddingRef struct {
	RefID       string         `json:"ref_id"`
	EmbeddingID string         `json:"embedding_id,omitempty"`
	Source      string         `json:"source,omitempty"`
	Status      string         `json:"status,omitempty"`
	SourceRefs  []SourceRef    `json:"source_refs,omitempty"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

// MemoryObject 是 schema v1 的统一真相对象。
type MemoryObject struct {
	SchemaVersion int              `json:"schema_version"`
	ObjectID      string           `json:"object_id"`
	ObjectType    string           `json:"object_type,omitempty"`
	Summary       string           `json:"summary,omitempty"`
	RawEvidence   []MemoryEvidence `json:"raw_evidence,omitempty"`
	Claims        []MemoryClaim    `json:"claims,omitempty"`
	EmbeddingRefs []EmbeddingRef   `json:"embedding_refs,omitempty"`
	SourceRefs    []SourceRef      `json:"source_refs,omitempty"`
	CreatedAt     time.Time        `json:"created_at,omitempty"`
	UpdatedAt     time.Time        `json:"updated_at,omitempty"`
	Confidence    float64          `json:"confidence,omitempty"`
	Metadata      map[string]any   `json:"metadata,omitempty"`
}

// TruthWriteResult 描述一次 shadow write / replay 的计数结果。
type TruthWriteResult struct {
	SchemaVersion  int       `json:"schema_version"`
	EventID        string    `json:"event_id,omitempty"`
	ObjectID       string    `json:"object_id,omitempty"`
	ObjectType     string    `json:"object_type,omitempty"`
	ObjectCount    int       `json:"object_count,omitempty"`
	ClaimCount     int       `json:"claim_count,omitempty"`
	SourceRefCount int       `json:"source_ref_count,omitempty"`
	OccurredAt     time.Time `json:"occurred_at,omitempty"`
}

type truthEvent struct {
	SchemaVersion int          `json:"schema_version"`
	EventID       string       `json:"event_id"`
	EventType     string       `json:"event_type"`
	ObjectID      string       `json:"object_id,omitempty"`
	ObjectType    string       `json:"object_type,omitempty"`
	OccurredAt    time.Time    `json:"occurred_at,omitempty"`
	WrittenAt     time.Time    `json:"written_at,omitempty"`
	TraceID       string       `json:"trace_id,omitempty"`
	Object        MemoryObject `json:"object"`
}

func normalizeSourceRef(ref SourceRef) SourceRef {
	out := ref
	out.SessionID = strings.TrimSpace(out.SessionID)
	out.Namespace = strings.TrimSpace(out.Namespace)
	out.WorkspaceID = strings.TrimSpace(out.WorkspaceID)
	out.BucketKey = strings.TrimSpace(out.BucketKey)
	out.BucketMonth = strings.TrimSpace(out.BucketMonth)
	out.TurnID = strings.TrimSpace(out.TurnID)
	out.TraceID = strings.TrimSpace(out.TraceID)
	out.SourceKind = strings.TrimSpace(out.SourceKind)
	out.SourceID = strings.TrimSpace(out.SourceID)
	out.OccurredAt = normalizeLedgerTime(out.OccurredAt)
	return out
}

func normalizeSourceRefs(refs []SourceRef) []SourceRef {
	if len(refs) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(refs))
	out := make([]SourceRef, 0, len(refs))
	for _, ref := range refs {
		normalized := normalizeSourceRef(ref)
		if normalized.SourceKind == "" && normalized.SourceID == "" && normalized.SessionID == "" && normalized.TurnID == "" && normalized.TraceID == "" {
			continue
		}
		fingerprint := truthSourceRefFingerprint(normalized)
		if _, ok := seen[fingerprint]; ok {
			continue
		}
		seen[fingerprint] = struct{}{}
		out = append(out, normalized)
	}
	if len(out) == 0 {
		return nil
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].SourceKind != out[j].SourceKind {
			return out[i].SourceKind < out[j].SourceKind
		}
		if out[i].SourceID != out[j].SourceID {
			return out[i].SourceID < out[j].SourceID
		}
		if out[i].SessionID != out[j].SessionID {
			return out[i].SessionID < out[j].SessionID
		}
		if out[i].TurnID != out[j].TurnID {
			return out[i].TurnID < out[j].TurnID
		}
		return out[i].TraceID < out[j].TraceID
	})
	return out
}

func normalizeMemoryEvidence(evidence MemoryEvidence, objectID string) MemoryEvidence {
	out := evidence
	out.Kind = strings.TrimSpace(out.Kind)
	out.Text = strings.TrimSpace(out.Text)
	out.Summary = strings.TrimSpace(out.Summary)
	if out.Summary == "" {
		out.Summary = summarizeLine(out.Text, 220)
	}
	out.Confidence = clamp01(out.Confidence)
	if out.Timestamp.IsZero() {
		out.Timestamp = time.Now().UTC()
	} else {
		out.Timestamp = out.Timestamp.UTC()
	}
	out.SourceRefs = normalizeSourceRefs(out.SourceRefs)
	out.EvidenceID = strings.TrimSpace(out.EvidenceID)
	if out.EvidenceID == "" {
		out.EvidenceID = buildTruthEvidenceID(objectID, out)
	}
	return out
}

func normalizeMemoryClaim(claim MemoryClaim, objectID string) MemoryClaim {
	out := claim
	out.SchemaVersion = truthSchemaVersion
	out.ObjectID = strings.TrimSpace(firstNonEmpty(out.ObjectID, objectID))
	out.Type = strings.TrimSpace(out.Type)
	out.Value = strings.TrimSpace(out.Value)
	out.IntentKey = strings.TrimSpace(out.IntentKey)
	out.AnchorKey = strings.TrimSpace(out.AnchorKey)
	out.EntityID = strings.TrimSpace(out.EntityID)
	out.ConstraintType = strings.TrimSpace(out.ConstraintType)
	out.RiskType = strings.TrimSpace(out.RiskType)
	out.Confidence = clamp01(out.Confidence)
	if out.CreatedAt.IsZero() {
		out.CreatedAt = time.Now().UTC()
	} else {
		out.CreatedAt = out.CreatedAt.UTC()
	}
	out.SourceRefs = normalizeSourceRefs(out.SourceRefs)
	out.ClaimID = strings.TrimSpace(out.ClaimID)
	if out.ClaimID == "" {
		out.ClaimID = buildTruthClaimID(out.ObjectID, out)
	}
	return out
}

func normalizeEmbeddingRef(ref EmbeddingRef, objectID string) EmbeddingRef {
	out := ref
	out.RefID = strings.TrimSpace(out.RefID)
	out.EmbeddingID = strings.TrimSpace(out.EmbeddingID)
	out.Source = strings.TrimSpace(out.Source)
	out.Status = strings.TrimSpace(out.Status)
	out.SourceRefs = normalizeSourceRefs(out.SourceRefs)
	if out.EmbeddingID == "" {
		return EmbeddingRef{}
	}
	if out.RefID == "" {
		out.RefID = buildTruthEmbeddingRefID(objectID, out)
	}
	return out
}

func normalizeMemoryObject(object MemoryObject) MemoryObject {
	out := object
	out.SchemaVersion = truthSchemaVersion
	out.ObjectID = strings.TrimSpace(out.ObjectID)
	out.ObjectType = strings.TrimSpace(out.ObjectType)
	out.Summary = strings.TrimSpace(out.Summary)
	out.Confidence = clamp01(out.Confidence)
	out.SourceRefs = normalizeSourceRefs(out.SourceRefs)

	evidence := make([]MemoryEvidence, 0, len(out.RawEvidence))
	for _, item := range out.RawEvidence {
		normalized := normalizeMemoryEvidence(item, out.ObjectID)
		if normalized.EvidenceID == "" {
			continue
		}
		evidence = append(evidence, normalized)
	}
	sort.SliceStable(evidence, func(i, j int) bool {
		if evidence[i].Timestamp.Equal(evidence[j].Timestamp) {
			return evidence[i].EvidenceID < evidence[j].EvidenceID
		}
		return evidence[i].Timestamp.Before(evidence[j].Timestamp)
	})
	out.RawEvidence = evidence

	claims := make([]MemoryClaim, 0, len(out.Claims))
	seenClaims := make(map[string]struct{}, len(out.Claims))
	for _, item := range out.Claims {
		normalized := normalizeMemoryClaim(item, out.ObjectID)
		if normalized.ClaimID == "" {
			continue
		}
		if _, ok := seenClaims[normalized.ClaimID]; ok {
			continue
		}
		seenClaims[normalized.ClaimID] = struct{}{}
		claims = append(claims, normalized)
	}
	sort.SliceStable(claims, func(i, j int) bool {
		return claims[i].ClaimID < claims[j].ClaimID
	})
	out.Claims = claims

	embeddings := make([]EmbeddingRef, 0, len(out.EmbeddingRefs))
	seenEmbeddings := make(map[string]struct{}, len(out.EmbeddingRefs))
	for _, item := range out.EmbeddingRefs {
		normalized := normalizeEmbeddingRef(item, out.ObjectID)
		if normalized.RefID == "" {
			continue
		}
		if _, ok := seenEmbeddings[normalized.RefID]; ok {
			continue
		}
		seenEmbeddings[normalized.RefID] = struct{}{}
		embeddings = append(embeddings, normalized)
	}
	sort.SliceStable(embeddings, func(i, j int) bool {
		return embeddings[i].RefID < embeddings[j].RefID
	})
	out.EmbeddingRefs = embeddings

	if out.CreatedAt.IsZero() {
		if len(out.RawEvidence) > 0 {
			out.CreatedAt = out.RawEvidence[0].Timestamp.UTC()
		} else {
			out.CreatedAt = time.Now().UTC()
		}
	} else {
		out.CreatedAt = out.CreatedAt.UTC()
	}
	if out.UpdatedAt.IsZero() {
		out.UpdatedAt = out.CreatedAt
	} else {
		out.UpdatedAt = out.UpdatedAt.UTC()
	}
	if out.Summary == "" {
		for _, item := range out.RawEvidence {
			if summary := firstNonEmpty(item.Summary, item.Text); summary != "" {
				out.Summary = summarizeLine(summary, 220)
				break
			}
		}
	}
	if out.ObjectID == "" {
		out.ObjectID = buildTruthObjectID(out.ObjectType, out.Summary, out.CreatedAt.Format(time.RFC3339Nano))
	}
	return out
}

func truthSourceRefFingerprint(ref SourceRef) string {
	normalized := normalizeSourceRef(ref)
	return strings.Join([]string{
		normalized.Namespace,
		normalized.WorkspaceID,
		normalized.BucketKey,
		normalized.BucketMonth,
		normalized.SessionID,
		normalized.TurnID,
		normalized.TraceID,
		normalized.SourceKind,
		normalized.SourceID,
		normalized.OccurredAt.Format(time.RFC3339Nano),
	}, "|")
}
