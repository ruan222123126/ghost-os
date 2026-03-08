package memory

import (
	"encoding/json"
	"strings"
	"time"
)

type ClaimTerm struct {
	Kind  string `json:"kind,omitempty"`
	ID    string `json:"id,omitempty"`
	Label string `json:"label,omitempty"`
}

type ClaimRef struct {
	ClaimID string `json:"claim_id,omitempty"`
	Role    string `json:"role,omitempty"`
}

type EvidenceRef struct {
	EvidenceID string `json:"evidence_id,omitempty"`
	Role       string `json:"role,omitempty"`
}

type TruthEventActor struct {
	Kind  string `json:"kind,omitempty"`
	ID    string `json:"id,omitempty"`
	Label string `json:"label,omitempty"`
}

type truthEventPayload struct {
	Object     *MemoryObject   `json:"object,omitempty"`
	Evidence   *MemoryEvidence `json:"evidence,omitempty"`
	Claim      *MemoryClaim    `json:"claim,omitempty"`
	ClaimRef   *ClaimRef       `json:"claim_ref,omitempty"`
	Status     string          `json:"status,omitempty"`
	ValidFrom  time.Time       `json:"valid_from,omitempty"`
	ValidTo    time.Time       `json:"valid_to,omitempty"`
	Supersedes []string        `json:"supersedes,omitempty"`
	Metadata   map[string]any  `json:"metadata,omitempty"`
}

type ClaimArbitrationResult struct {
	UpsertedClaims []MemoryClaim `json:"upserted_claims,omitempty"`
	StatusChanges  []MemoryClaim `json:"status_changes,omitempty"`
	Conflicts      []string      `json:"conflicts,omitempty"`
	SupersededIDs  []string      `json:"superseded_ids,omitempty"`
	ActiveIDs      []string      `json:"active_ids,omitempty"`
	Explanations   []string      `json:"explanations,omitempty"`
}

type truthEvent struct {
	SchemaVersion int               `json:"schema_version"`
	EventID       string            `json:"event_id"`
	EventType     string            `json:"event_type"`
	Namespace     string            `json:"namespace,omitempty"`
	WorkspaceID   string            `json:"workspace_id,omitempty"`
	SessionID     string            `json:"session_id,omitempty"`
	TurnID        string            `json:"turn_id,omitempty"`
	TraceID       string            `json:"trace_id,omitempty"`
	OccurredAt    time.Time         `json:"occurred_at,omitempty"`
	WrittenAt     time.Time         `json:"written_at,omitempty"`
	Actor         TruthEventActor   `json:"actor,omitempty"`
	ObjectID      string            `json:"object_id,omitempty"`
	ObjectType    string            `json:"object_type,omitempty"`
	Payload       truthEventPayload `json:"payload,omitempty"`
	Object        MemoryObject      `json:"object,omitempty"`
}

func normalizeClaimTerm(term ClaimTerm) ClaimTerm {
	out := term
	out.Kind = strings.TrimSpace(out.Kind)
	out.ID = strings.TrimSpace(out.ID)
	out.Label = strings.TrimSpace(out.Label)
	if out.Kind == "" && out.ID != "" {
		out.Kind = "entity"
	}
	return out
}

func normalizeEvidenceRefs(refs []EvidenceRef) []EvidenceRef {
	if len(refs) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(refs))
	out := make([]EvidenceRef, 0, len(refs))
	for _, ref := range refs {
		normalized := EvidenceRef{EvidenceID: strings.TrimSpace(ref.EvidenceID), Role: strings.TrimSpace(ref.Role)}
		if normalized.EvidenceID == "" {
			continue
		}
		key := normalized.EvidenceID + "|" + normalized.Role
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, normalized)
	}
	return out
}

func normalizeClaimRef(ref ClaimRef) ClaimRef {
	return ClaimRef{ClaimID: strings.TrimSpace(ref.ClaimID), Role: strings.TrimSpace(ref.Role)}
}

func normalizeTruthActor(actor TruthEventActor) TruthEventActor {
	actor.Kind = strings.TrimSpace(actor.Kind)
	actor.ID = strings.TrimSpace(actor.ID)
	actor.Label = strings.TrimSpace(actor.Label)
	return actor
}

func normalizeTruthEventPayload(payload truthEventPayload) truthEventPayload {
	out := payload
	if out.Object != nil {
		normalized := normalizeMemoryObject(*out.Object)
		out.Object = &normalized
	}
	if out.Evidence != nil {
		objectID := ""
		if out.Object != nil {
			objectID = out.Object.ObjectID
		}
		normalized := normalizeMemoryEvidence(*out.Evidence, objectID)
		out.Evidence = &normalized
	}
	if out.Claim != nil {
		objectID := ""
		if out.Object != nil {
			objectID = out.Object.ObjectID
		}
		normalized := normalizeMemoryClaim(*out.Claim, objectID)
		out.Claim = &normalized
	}
	if out.ClaimRef != nil {
		normalized := normalizeClaimRef(*out.ClaimRef)
		out.ClaimRef = &normalized
	}
	out.Status = normalizeTruthClaimStatus(out.Status)
	out.ValidFrom = normalizeLedgerTime(out.ValidFrom)
	out.ValidTo = normalizeLedgerTime(out.ValidTo)
	out.Supersedes = uniqueStrings(out.Supersedes)
	return out
}

func normalizeTruthEvent(event truthEvent) truthEvent {
	out := event
	out.SchemaVersion = truthSchemaVersion
	out.EventType = strings.TrimSpace(out.EventType)
	out.Namespace = strings.TrimSpace(out.Namespace)
	out.WorkspaceID = strings.TrimSpace(out.WorkspaceID)
	out.SessionID = strings.TrimSpace(out.SessionID)
	out.TurnID = strings.TrimSpace(out.TurnID)
	out.TraceID = strings.TrimSpace(out.TraceID)
	out.Actor = normalizeTruthActor(out.Actor)
	out.ObjectID = strings.TrimSpace(out.ObjectID)
	out.ObjectType = strings.TrimSpace(out.ObjectType)
	out.OccurredAt = normalizeLedgerTime(out.OccurredAt)
	out.WrittenAt = normalizeLedgerTime(out.WrittenAt)
	out.Payload = normalizeTruthEventPayload(out.Payload)
	if out.Payload.Object != nil {
		out.ObjectID = firstNonEmpty(out.ObjectID, out.Payload.Object.ObjectID)
		out.ObjectType = firstNonEmpty(out.ObjectType, out.Payload.Object.ObjectType)
	}
	if out.ObjectID == "" && out.Object.ObjectID != "" {
		normalized := normalizeMemoryObject(out.Object)
		out.Object = normalized
		out.ObjectID = normalized.ObjectID
		out.ObjectType = normalized.ObjectType
	}
	if out.EventID == "" {
		out.EventID = buildTruthGenericEventID(out.EventType, out)
	}
	return out
}

func buildTruthGenericEventID(eventType string, payload any) string {
	data, err := json.Marshal(payload)
	if err != nil {
		return truthHashID("evt", strings.TrimSpace(eventType))
	}
	return truthHashID("evt", strings.TrimSpace(eventType), string(data))
}
