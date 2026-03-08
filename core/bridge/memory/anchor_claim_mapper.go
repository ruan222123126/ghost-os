package memory

import (
	"strings"
	"time"
)

func anchorClaimType(anchorType string) string {
	switch strings.ToLower(strings.TrimSpace(anchorType)) {
	case MemoryAnchorConstraint:
		return truthClaimTypeConstraint
	case MemoryAnchorAvoidance:
		return truthClaimTypeRisk
	default:
		return truthClaimTypeAnchor
	}
}

func anchorClaimPredicate(anchor MemoryAnchor) string {
	normalized := normalizeAnchor(anchor)
	switch normalized.Type {
	case MemoryAnchorPreference:
		key := truthIndexKey(normalized.Key)
		if key == "" {
			return "preference.has"
		}
		return "preference." + strings.ReplaceAll(key, " ", "_")
	case MemoryAnchorAvoidance:
		return "avoidance.has"
	case MemoryAnchorConstraint:
		return "constraint.has"
	case MemoryAnchorIdentity:
		return "identity.self"
	case MemoryAnchorEmotion:
		return "emotion.state"
	default:
		return firstNonEmpty(strings.TrimSpace(normalized.Type), "anchor") + ".has"
	}
}

func anchorClaimAnchorKey(anchor MemoryAnchor) string {
	normalized := normalizeAnchor(anchor)
	parts := []string{normalized.Type}
	if normalized.Key != "" {
		parts = append(parts, truthIndexKey(normalized.Key))
	}
	if normalized.Value != "" {
		parts = append(parts, truthIndexKey(normalized.Value))
	}
	return strings.Join(parts, ".")
}

func mapAnchorToClaim(anchor MemoryAnchor, objectID string, subject ClaimTerm, evidenceRefs []EvidenceRef, sourceRefs []SourceRef, fallbackConfidence float64, fallbackTime time.Time, relatedTo []string, sourceIDs []string) (MemoryClaim, bool) {
	normalized := normalizeAnchor(anchor)
	if normalized.Type == "" || normalized.Value == "" {
		return MemoryClaim{}, false
	}
	assertedAt := effectiveDecisionTimestamp(normalized.DetectedAt, fallbackTime)
	claim := MemoryClaim{
		ObjectID:       strings.TrimSpace(objectID),
		Subject:        subject,
		Predicate:      anchorClaimPredicate(normalized),
		Type:           anchorClaimType(normalized.Type),
		Value:          normalized.Value,
		Datatype:       "string",
		EvidenceRefs:   normalizeEvidenceRefs(evidenceRefs),
		AnchorKey:      anchorClaimAnchorKey(normalized),
		ConstraintType: firstNonEmpty(normalized.Type, ""),
		Confidence:     maxFloat(normalized.Weight, fallbackConfidence),
		ValidFrom:      normalizeLedgerTime(normalized.DetectedAt),
		ValidTo:        normalizeLedgerTime(normalized.ExpiresAt),
		AssertedAt:     assertedAt,
		CreatedAt:      assertedAt,
		SourceRefs:     normalizeSourceRefs(sourceRefs),
		Metadata: map[string]any{
			"anchor_type": normalized.Type,
			"anchor_key":  normalized.Key,
			"reason":      normalized.Reason,
			"related_to":  append([]string(nil), relatedTo...),
			"source_ids":  append([]string(nil), sourceIDs...),
		},
	}
	if claim.Predicate == "preference.language" {
		claim.Datatype = "language"
	}
	if normalized.Type == MemoryAnchorAvoidance {
		claim.RiskType = "avoid_pattern"
	}
	if normalized.Type != MemoryAnchorConstraint {
		claim.ConstraintType = ""
	}
	return normalizeMemoryClaim(claim, objectID), true
}

func claimReason(claim MemoryClaim) string {
	if len(claim.Metadata) == 0 {
		return ""
	}
	if value, ok := claim.Metadata["reason"].(string); ok {
		return strings.TrimSpace(value)
	}
	return ""
}

func claimAnchorType(claim MemoryClaim) string {
	predicate := truthIndexKey(claim.Predicate)
	switch {
	case strings.HasPrefix(predicate, "preference."):
		return MemoryAnchorPreference
	case strings.HasPrefix(predicate, "avoidance."):
		return MemoryAnchorAvoidance
	case strings.HasPrefix(predicate, "constraint."):
		return MemoryAnchorConstraint
	case strings.HasPrefix(predicate, "identity."):
		return MemoryAnchorIdentity
	case strings.HasPrefix(predicate, "emotion."):
		return MemoryAnchorEmotion
	}
	switch truthIndexKey(claim.Type) {
	case truthIndexKey(truthClaimTypeConstraint):
		return MemoryAnchorConstraint
	case truthIndexKey(truthClaimTypeRisk):
		if truthIndexKey(claim.RiskType) == "avoid_pattern" {
			return MemoryAnchorAvoidance
		}
	}
	return ""
}

func claimAnchorKey(claim MemoryClaim, anchorType string) string {
	if strings.TrimSpace(claim.AnchorKey) != "" {
		parts := strings.Split(strings.TrimSpace(claim.AnchorKey), ".")
		if len(parts) >= 2 {
			return parts[len(parts)-2]
		}
	}
	predicate := truthIndexKey(claim.Predicate)
	if strings.HasPrefix(predicate, "preference.") {
		return strings.TrimPrefix(predicate, "preference.")
	}
	if strings.HasPrefix(predicate, "identity.") || strings.HasPrefix(predicate, "emotion.") {
		return anchorType
	}
	return strings.TrimSpace(claim.ConstraintType)
}

func claimToAnchor(claim MemoryClaim, now time.Time) (MemoryAnchor, bool) {
	normalized := normalizeMemoryClaim(claim, claim.ObjectID)
	if normalized.ClaimID == "" || strings.TrimSpace(normalized.Value) == "" {
		return MemoryAnchor{}, false
	}
	if normalizeTruthClaimStatus(normalized.Status) == truthClaimStatusSuperseded {
		return MemoryAnchor{}, false
	}
	if !normalized.ValidTo.IsZero() && now.UTC().After(normalized.ValidTo.UTC()) {
		return MemoryAnchor{}, false
	}
	anchorType := claimAnchorType(normalized)
	if anchorType == "" {
		return MemoryAnchor{}, false
	}
	anchor := MemoryAnchor{
		Type:       anchorType,
		Key:        claimAnchorKey(normalized, anchorType),
		Value:      normalized.Value,
		Weight:     clamp01(normalized.Confidence),
		Reason:     claimReason(normalized),
		DetectedAt: normalizeLedgerTime(normalized.ValidFrom),
		ExpiresAt:  normalizeLedgerTime(normalized.ValidTo),
		SessionID:  truthPrimarySessionID(nil, normalized.SourceRefs, &normalized),
	}
	if anchor.DetectedAt.IsZero() {
		anchor.DetectedAt = effectiveDecisionTimestamp(normalized.AssertedAt, normalized.CreatedAt)
	}
	return normalizeAnchor(anchor), true
}

func projectAnchorsFromClaims(claims []MemoryClaim, now time.Time) []MemoryAnchor {
	if len(claims) == 0 {
		return nil
	}
	anchors := make([]MemoryAnchor, 0, len(claims))
	for _, claim := range claims {
		anchor, ok := claimToAnchor(claim, now)
		if !ok {
			continue
		}
		anchors = append(anchors, anchor)
	}
	return normalizeAnchors(anchors)
}

func projectAnchorsFromClaimIDs(reader *TruthReader, claimIDs []string, now time.Time) []MemoryAnchor {
	if reader == nil || len(claimIDs) == 0 {
		return nil
	}
	claims := reader.ResolveClaims(claimIDs)
	return projectAnchorsFromClaims(claims, now)
}
