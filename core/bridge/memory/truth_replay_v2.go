package memory

import "strings"

func truthApplyEvent(objects map[string]MemoryObject, claims map[string]MemoryClaim, event truthEvent) {
	normalized := normalizeTruthEvent(event)
	switch normalized.EventType {
	case truthEventTypeEvidenceMessageObserved, truthEventTypeEvidenceMarkdownNoteObserved, truthEventTypeEvidenceToolResultObserved:
		truthApplyEvidenceEvent(objects, normalized)
	case truthEventTypeClaimAsserted:
		if normalized.Payload.Claim != nil {
			truthArbitrateClaims(claims, []MemoryClaim{*normalized.Payload.Claim})
		}
	case truthEventTypeClaimStatusChanged:
		truthApplyClaimStatusEvent(claims, normalized)
	case truthEventTypeClaimValidityChanged:
		truthApplyClaimValidityEvent(claims, normalized)
	case truthEventTypeClaimRetracted:
		truthApplyClaimRetractionEvent(claims, normalized)
	case truthEventTypeObjectProjected:
		truthApplyProjectedObject(objects, normalized)
	case truthEventTypeArchiveMessage, truthEventTypeMarkdownNode, truthEventTypeDecisionMemo:
		truthApplyLegacyObjectEvent(objects, claims, normalized)
	default:
		if normalized.Payload.Claim != nil {
			truthArbitrateClaims(claims, []MemoryClaim{*normalized.Payload.Claim})
		}
		if normalized.Payload.Evidence != nil {
			truthApplyEvidenceEvent(objects, normalized)
		}
		if normalized.Payload.Object != nil || normalized.Object.ObjectID != "" {
			truthApplyProjectedObject(objects, normalized)
		}
	}
}

func truthApplyLegacyObjectEvent(objects map[string]MemoryObject, claims map[string]MemoryClaim, event truthEvent) {
	truthApplyProjectedObject(objects, event)
	object, ok := truthEventObject(event)
	if !ok {
		return
	}
	if len(object.Claims) > 0 {
		truthArbitrateClaims(claims, object.Claims)
	}
}

func truthApplyProjectedObject(objects map[string]MemoryObject, event truthEvent) {
	object, ok := truthEventObject(event)
	if !ok {
		return
	}
	normalized := normalizeMemoryObject(object)
	current, exists := objects[normalized.ObjectID]
	if exists {
		mergedEvidence := append(current.RawEvidence, normalized.RawEvidence...)
		normalized.RawEvidence = mergedEvidence
		normalized.EmbeddingRefs = append(current.EmbeddingRefs, normalized.EmbeddingRefs...)
		normalized.SourceRefs = append(current.SourceRefs, normalized.SourceRefs...)
		normalized.Claims = append(current.Claims, normalized.Claims...)
		if strings.TrimSpace(normalized.Summary) == "" {
			normalized.Summary = current.Summary
		}
		if normalized.CreatedAt.IsZero() {
			normalized.CreatedAt = current.CreatedAt
		}
	}
	objects[normalized.ObjectID] = normalizeMemoryObject(normalized)
}

func truthApplyEvidenceEvent(objects map[string]MemoryObject, event truthEvent) {
	if event.Payload.Evidence == nil {
		return
	}
	objectID := strings.TrimSpace(event.ObjectID)
	if objectID == "" && event.Payload.Object != nil {
		objectID = event.Payload.Object.ObjectID
	}
	if objectID == "" {
		return
	}
	object, ok := objects[objectID]
	if !ok {
		object = MemoryObject{ObjectID: objectID, ObjectType: event.ObjectType, CreatedAt: event.OccurredAt, UpdatedAt: event.OccurredAt}
	}
	if event.Payload.Object != nil {
		object.ObjectType = firstNonEmpty(object.ObjectType, event.Payload.Object.ObjectType)
		object.SourceRefs = append(object.SourceRefs, event.Payload.Object.SourceRefs...)
	}
	object.RawEvidence = append(object.RawEvidence, *event.Payload.Evidence)
	object.UpdatedAt = effectiveDecisionTimestamp(event.OccurredAt, object.UpdatedAt, object.CreatedAt)
	objects[objectID] = normalizeMemoryObject(object)
}

func truthApplyClaimStatusEvent(claims map[string]MemoryClaim, event truthEvent) {
	claim, ok := truthEventClaim(event)
	if !ok {
		return
	}
	existing, exists := claims[claim.ClaimID]
	if exists {
		existing.Status = normalizeTruthClaimStatus(firstNonEmpty(event.Payload.Status, claim.Status))
		existing.Supersedes = uniqueStrings(append(existing.Supersedes, event.Payload.Supersedes...))
		existing.ValidFrom = firstNonZeroTime(event.Payload.ValidFrom, claim.ValidFrom, existing.ValidFrom)
		existing.ValidTo = firstNonZeroTime(event.Payload.ValidTo, claim.ValidTo, existing.ValidTo)
		claims[existing.ClaimID] = normalizeMemoryClaim(existing, existing.ObjectID)
		return
	}
	claim.Status = normalizeTruthClaimStatus(firstNonEmpty(event.Payload.Status, claim.Status))
	claims[claim.ClaimID] = claim
}

func truthApplyClaimValidityEvent(claims map[string]MemoryClaim, event truthEvent) {
	claim, ok := truthEventClaim(event)
	if !ok {
		return
	}
	existing, exists := claims[claim.ClaimID]
	if !exists {
		existing = claim
	}
	existing.ValidFrom = firstNonZeroTime(event.Payload.ValidFrom, claim.ValidFrom, existing.ValidFrom)
	existing.ValidTo = firstNonZeroTime(event.Payload.ValidTo, claim.ValidTo, existing.ValidTo)
	claims[existing.ClaimID] = normalizeMemoryClaim(existing, existing.ObjectID)
}

func truthApplyClaimRetractionEvent(claims map[string]MemoryClaim, event truthEvent) {
	claim, ok := truthEventClaim(event)
	if !ok {
		return
	}
	existing, exists := claims[claim.ClaimID]
	if !exists {
		existing = claim
	}
	existing.Status = truthClaimStatusSuperseded
	claims[existing.ClaimID] = normalizeMemoryClaim(existing, existing.ObjectID)
}

func truthEventObject(event truthEvent) (MemoryObject, bool) {
	if event.Payload.Object != nil {
		return normalizeMemoryObject(*event.Payload.Object), true
	}
	if event.Object.ObjectID != "" {
		return normalizeMemoryObject(event.Object), true
	}
	return MemoryObject{}, false
}

func truthEventClaim(event truthEvent) (MemoryClaim, bool) {
	if event.Payload.Claim != nil {
		return normalizeMemoryClaim(*event.Payload.Claim, event.Payload.Claim.ObjectID), true
	}
	return MemoryClaim{}, false
}
