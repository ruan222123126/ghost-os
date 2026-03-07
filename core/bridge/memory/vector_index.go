package memory

import (
	"fmt"
	"strings"
	"time"
)

func buildVectorDocument(object MemoryObject) (VectorDocument, bool) {
	normalized := normalizeMemoryObject(object)
	if !vectorIndexableObject(normalized) {
		return VectorDocument{}, false
	}
	text := buildVectorDocumentText(normalized)
	weights, norm := buildVectorWeights(text)
	if len(weights) == 0 || norm == 0 {
		return VectorDocument{}, false
	}
	ref := vectorEmbeddingRef(normalized, vectorEmbeddingStatusIndexed, "")
	return normalizeVectorDocument(VectorDocument{
		ObjectID:      normalized.ObjectID,
		ObjectType:    normalized.ObjectType,
		Summary:       firstNonEmpty(normalized.Summary, summarizeLine(text, 220)),
		Text:          text,
		Terms:         vectorTokenize(text),
		Weights:       weights,
		Norm:          norm,
		SourceRefs:    vectorSourceRefs(normalized),
		EvidenceCount: len(normalized.RawEvidence),
		Confidence:    clamp01(normalized.Confidence),
		CreatedAt:     normalized.CreatedAt,
		UpdatedAt:     normalized.UpdatedAt,
		EmbeddingRef:  ref,
	}), true
}

func buildVectorDocuments(objects map[string]MemoryObject) []VectorDocument {
	if len(objects) == 0 {
		return nil
	}
	documents := make([]VectorDocument, 0, len(objects))
	for _, object := range objects {
		doc, ok := buildVectorDocument(object)
		if !ok {
			continue
		}
		documents = append(documents, doc)
	}
	return documents
}

func vectorIndexableObject(object MemoryObject) bool {
	switch strings.TrimSpace(object.ObjectType) {
	case truthObjectTypeSemanticNote, truthObjectTypeProcedureMemo:
		return true
	case truthObjectTypeEvidenceMessage:
		if len(object.Claims) == 0 {
			return false
		}
		confidence := clamp01(object.Confidence)
		for _, claim := range object.Claims {
			confidence = maxFloat(confidence, clamp01(claim.Confidence))
		}
		return confidence >= vectorEvidenceMinConfidence
	default:
		return false
	}
}

func buildVectorDocumentText(object MemoryObject) string {
	parts := make([]string, 0, 1+len(object.Claims)+len(object.RawEvidence))
	if summary := strings.TrimSpace(object.Summary); summary != "" {
		parts = append(parts, summary)
	}
	for _, claim := range object.Claims {
		value := firstNonEmpty(
			strings.TrimSpace(claim.Value),
			strings.TrimSpace(claim.IntentKey),
			strings.TrimSpace(claim.AnchorKey),
			strings.TrimSpace(claim.EntityID),
		)
		if value == "" {
			continue
		}
		parts = append(parts, value)
	}
	for _, evidence := range object.RawEvidence {
		summary := strings.TrimSpace(firstNonEmpty(evidence.Summary, summarizeLine(evidence.Text, 220)))
		if summary == "" {
			continue
		}
		parts = append(parts, summary)
	}
	return strings.Join(uniqueStrings(parts), "\n")
}

func vectorSourceRefs(object MemoryObject) []SourceRef {
	refs := append([]SourceRef(nil), object.SourceRefs...)
	for _, claim := range object.Claims {
		refs = append(refs, claim.SourceRefs...)
	}
	for _, evidence := range object.RawEvidence {
		refs = append(refs, evidence.SourceRefs...)
	}
	return normalizeSourceRefs(refs)
}

func vectorEmbeddingRef(object MemoryObject, status string, errorText string) EmbeddingRef {
	ref := EmbeddingRef{
		EmbeddingID: fmt.Sprintf("vector:%s", strings.TrimSpace(object.ObjectID)),
		Source:      vectorEmbeddingSource,
		Status:      strings.TrimSpace(status),
		SourceRefs:  vectorSourceRefs(object),
		Metadata: map[string]any{
			"object_type": strings.TrimSpace(object.ObjectType),
		},
	}
	if strings.TrimSpace(errorText) != "" {
		ref.Metadata["error"] = summarizeLine(errorText, 180)
		ref.Metadata["error_at"] = time.Now().UTC().Format(time.RFC3339Nano)
	}
	return normalizeEmbeddingRef(ref, object.ObjectID)
}
