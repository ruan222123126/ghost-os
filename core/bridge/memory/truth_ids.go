package memory

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"
)

func buildTruthObjectID(kind string, parts ...string) string {
	values := append([]string{strings.TrimSpace(kind)}, parts...)
	return truthHashID("obj", values...)
}

func buildTruthEventID(eventType string, object MemoryObject) string {
	payload, err := json.Marshal(normalizeMemoryObject(object))
	if err != nil {
		payload = []byte(strings.Join([]string{
			strings.TrimSpace(object.ObjectID),
			strings.TrimSpace(object.ObjectType),
			strings.TrimSpace(object.Summary),
			object.UpdatedAt.UTC().Format(time.RFC3339Nano),
		}, "|"))
	}
	return truthHashID("evt", strings.TrimSpace(eventType), string(payload))
}

func buildTruthClaimID(objectID string, claim MemoryClaim) string {
	subject := normalizeClaimTerm(claim.Subject)
	objectTerm := normalizeClaimTerm(claim.Object)
	return truthHashID("clm",
		strings.TrimSpace(objectID),
		strings.TrimSpace(claim.Type),
		strings.TrimSpace(claim.Predicate),
		subject.Kind,
		subject.ID,
		subject.Label,
		objectTerm.Kind,
		objectTerm.ID,
		objectTerm.Label,
		strings.TrimSpace(claim.IntentKey),
		strings.TrimSpace(claim.AnchorKey),
		strings.TrimSpace(claim.EntityID),
		strings.TrimSpace(claim.ConstraintType),
		strings.TrimSpace(claim.RiskType),
		strings.TrimSpace(claim.Value),
		strings.TrimSpace(claim.Datatype),
	)
}

func buildTruthEvidenceID(objectID string, evidence MemoryEvidence) string {
	parts := []string{
		strings.TrimSpace(objectID),
		strings.TrimSpace(evidence.Kind),
		strings.TrimSpace(evidence.Summary),
		strings.TrimSpace(evidence.Text),
		evidence.Timestamp.UTC().Format(time.RFC3339Nano),
	}
	for _, ref := range normalizeSourceRefs(evidence.SourceRefs) {
		parts = append(parts, truthSourceRefFingerprint(ref))
	}
	return truthHashID("evd", parts...)
}

func buildTruthEmbeddingRefID(objectID string, ref EmbeddingRef) string {
	parts := []string{
		strings.TrimSpace(objectID),
		strings.TrimSpace(ref.EmbeddingID),
		strings.TrimSpace(ref.Source),
	}
	for _, item := range normalizeSourceRefs(ref.SourceRefs) {
		parts = append(parts, truthSourceRefFingerprint(item))
	}
	return truthHashID("emb", parts...)
}

func truthHashID(prefix string, parts ...string) string {
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
