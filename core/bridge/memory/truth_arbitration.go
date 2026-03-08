package memory

import (
	"math"
	"sort"
	"strings"
	"time"
)

func legacyTruthPredicate(claim MemoryClaim) string {
	switch {
	case strings.TrimSpace(claim.Predicate) != "":
		return strings.TrimSpace(claim.Predicate)
	case strings.TrimSpace(claim.IntentKey) != "":
		return "intent.key"
	case strings.TrimSpace(claim.AnchorKey) != "":
		key := truthIndexKey(claim.AnchorKey)
		if strings.Contains(key, "language") {
			return "preference.language"
		}
		if strings.TrimSpace(claim.ConstraintType) != "" {
			return "anchor.constraint"
		}
		if strings.TrimSpace(claim.RiskType) != "" {
			return "anchor.avoidance"
		}
		return "anchor.preference"
	case strings.TrimSpace(claim.EntityID) != "":
		return "entity.ref"
	case strings.TrimSpace(claim.ConstraintType) != "":
		return "constraint.has"
	case strings.TrimSpace(claim.RiskType) != "":
		if truthIndexKey(claim.RiskType) == "avoid_pattern" {
			return "avoids_pattern"
		}
		return "failure_reason"
	case strings.TrimSpace(claim.Type) == truthClaimTypeTool:
		return "uses_tool"
	case strings.TrimSpace(claim.Type) == truthClaimTypeOutcome:
		return "outcome.summary"
	default:
		return truthIndexKey(firstNonEmpty(claim.Type, "claim"))
	}
}

func truthLegacyClaimSubject(claim MemoryClaim) ClaimTerm {
	if claim.Subject.ID != "" {
		return normalizeClaimTerm(claim.Subject)
	}
	if claim.ObjectID != "" {
		return ClaimTerm{Kind: "object", ID: claim.ObjectID}
	}
	return ClaimTerm{Kind: "memory", ID: firstNonEmpty(claim.IntentKey, claim.AnchorKey, claim.EntityID, claim.ConstraintType, claim.RiskType, claim.Type, claim.Value)}
}

func truthInitialClaimStatus(claim MemoryClaim) string {
	if strings.TrimSpace(claim.Status) != "" {
		return normalizeTruthClaimStatus(claim.Status)
	}
	if truthClaimSupportCount(claim) >= 2 || claim.Confidence >= 0.85 {
		return truthClaimStatusActive
	}
	return truthClaimStatusUnverified
}

func truthClaimDomainKey(claim MemoryClaim) string {
	subject := normalizeClaimTerm(claim.Subject)
	predicate := truthIndexKey(firstNonEmpty(claim.Predicate, legacyTruthPredicate(claim)))
	if subject.Kind == "" || subject.ID == "" || predicate == "" {
		return truthConflictDimension(claim)
	}
	return strings.Join([]string{subject.Kind, subject.ID, predicate}, "|")
}

func truthClaimObjectKey(claim MemoryClaim) string {
	objectTerm := normalizeClaimTerm(claim.Object)
	return strings.Join([]string{objectTerm.Kind, objectTerm.ID, truthIndexKey(objectTerm.Label)}, "|")
}

func truthClaimValueKey(claim MemoryClaim) string {
	value := firstNonEmpty(claim.Value, claim.IntentKey, claim.AnchorKey, claim.EntityID, claim.ConstraintType, claim.RiskType)
	return strings.Join([]string{truthClaimObjectKey(claim), truthIndexKey(value), truthIndexKey(claim.Datatype)}, "|")
}

func truthClaimSupportCount(claim MemoryClaim) int {
	count := len(claim.EvidenceRefs)
	if count == 0 {
		count = len(claim.SourceRefs)
	}
	return count
}

func truthClaimSourceDiversity(claim MemoryClaim) int {
	if len(claim.SourceRefs) == 0 {
		return 0
	}
	seen := make(map[string]struct{}, len(claim.SourceRefs))
	for _, ref := range claim.SourceRefs {
		key := truthIndexKey(ref.SourceKind) + "|" + truthIndexKey(ref.SourceID)
		if key == "|" {
			key = truthIndexKey(ref.SessionID) + "|" + truthIndexKey(ref.TraceID)
		}
		seen[key] = struct{}{}
	}
	return len(seen)
}

func truthClaimStrength(claim MemoryClaim) float64 {
	confidence := clamp01(claim.Confidence)
	support := float64(min(truthClaimSupportCount(claim), 4)) * 0.08
	diversity := float64(min(truthClaimSourceDiversity(claim), 3)) * 0.05
	freshness := 0.0
	assertedAt := effectiveDecisionTimestamp(claim.AssertedAt, claim.CreatedAt)
	if !assertedAt.IsZero() {
		ageHours := time.Since(assertedAt).Hours()
		if ageHours < 0 {
			ageHours = -ageHours
		}
		freshness = math.Max(0, 0.12-math.Min(ageHours/(24*30), 0.12))
	}
	return confidence + support + diversity + freshness
}

func truthClaimValidityWindowsOverlap(left MemoryClaim, right MemoryClaim) bool {
	leftStart := left.ValidFrom
	leftEnd := left.ValidTo
	rightStart := right.ValidFrom
	rightEnd := right.ValidTo
	if leftStart.IsZero() && leftEnd.IsZero() {
		return true
	}
	if rightStart.IsZero() && rightEnd.IsZero() {
		return true
	}
	if leftEnd.IsZero() {
		leftEnd = time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC)
	}
	if rightEnd.IsZero() {
		rightEnd = time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC)
	}
	return !leftEnd.Before(rightStart) && !rightEnd.Before(leftStart)
}

func truthClaimNewerOrStronger(existing MemoryClaim, incoming MemoryClaim) bool {
	incomingTime := effectiveDecisionTimestamp(incoming.AssertedAt, incoming.CreatedAt, incoming.ValidFrom)
	existingTime := effectiveDecisionTimestamp(existing.AssertedAt, existing.CreatedAt, existing.ValidFrom)
	if !incomingTime.IsZero() && !existingTime.IsZero() && incomingTime.After(existingTime.Add(time.Minute)) {
		return true
	}
	return truthClaimStrength(incoming) >= truthClaimStrength(existing)+0.05
}

func truthSingleValueClaimsConflict(existing MemoryClaim, incoming MemoryClaim) bool {
	confDelta := math.Abs(existing.Confidence - incoming.Confidence)
	timeDelta := effectiveDecisionTimestamp(existing.AssertedAt, existing.CreatedAt).Sub(effectiveDecisionTimestamp(incoming.AssertedAt, incoming.CreatedAt))
	if timeDelta < 0 {
		timeDelta = -timeDelta
	}
	return confDelta <= 0.08 && timeDelta <= 24*time.Hour
}

func truthClaimCompetitors(snapshot map[string]MemoryClaim, claim MemoryClaim) []MemoryClaim {
	if len(snapshot) == 0 {
		return nil
	}
	domain := truthClaimDomainKey(claim)
	if domain == "" {
		return nil
	}
	policy := truthPredicatePolicyFor(claim.Predicate)
	if normalizeTruthPredicateCardinality(policy.Cardinality) != truthPredicateCardinalitySingleValue {
		return nil
	}
	out := make([]MemoryClaim, 0, 2)
	for _, existing := range snapshot {
		if existing.ClaimID == claim.ClaimID {
			continue
		}
		if !truthClaimStatusIsLive(existing.Status) {
			continue
		}
		if truthClaimDomainKey(existing) != domain {
			continue
		}
		out = append(out, normalizeMemoryClaim(existing, existing.ObjectID))
	}
	return out
}

func truthRecordStatusChange(result *ClaimArbitrationResult, claim MemoryClaim, explanation string) {
	if result == nil {
		return
	}
	for i := range result.StatusChanges {
		if result.StatusChanges[i].ClaimID == claim.ClaimID {
			result.StatusChanges[i] = claim
			if strings.TrimSpace(explanation) != "" {
				result.Explanations = append(result.Explanations, explanation)
			}
			return
		}
	}
	result.StatusChanges = append(result.StatusChanges, claim)
	if strings.TrimSpace(explanation) != "" {
		result.Explanations = append(result.Explanations, explanation)
	}
}

func truthRecordUpsert(result *ClaimArbitrationResult, claim MemoryClaim) {
	if result == nil {
		return
	}
	for i := range result.UpsertedClaims {
		if result.UpsertedClaims[i].ClaimID == claim.ClaimID {
			result.UpsertedClaims[i] = claim
			return
		}
	}
	result.UpsertedClaims = append(result.UpsertedClaims, claim)
}

func truthArbitrateClaims(snapshot map[string]MemoryClaim, incomingClaims []MemoryClaim) ClaimArbitrationResult {
	if snapshot == nil {
		snapshot = make(map[string]MemoryClaim)
	}
	result := ClaimArbitrationResult{}
	for _, raw := range incomingClaims {
		incoming := normalizeMemoryClaim(raw, raw.ObjectID)
		if incoming.ClaimID == "" {
			continue
		}
		if existing, ok := snapshot[incoming.ClaimID]; ok {
			merged := normalizeMemoryClaim(incoming, incoming.ObjectID)
			merged.SourceRefs = normalizeSourceRefs(append(existing.SourceRefs, merged.SourceRefs...))
			merged.EvidenceRefs = normalizeEvidenceRefs(append(existing.EvidenceRefs, merged.EvidenceRefs...))
			merged.Confidence = maxFloat(existing.Confidence, merged.Confidence)
			merged.Status = truthInitialClaimStatus(merged)
			snapshot[merged.ClaimID] = merged
			truthRecordUpsert(&result, merged)
			continue
		}
		if normalizeTruthPredicateCardinality(truthPredicatePolicyFor(incoming.Predicate).Cardinality) == truthPredicateCardinalityMultiValue {
			incoming.Status = truthInitialClaimStatus(incoming)
			snapshot[incoming.ClaimID] = incoming
			truthRecordUpsert(&result, incoming)
			if incoming.Status == truthClaimStatusActive {
				result.ActiveIDs = append(result.ActiveIDs, incoming.ClaimID)
			}
			continue
		}
		incoming.Status = truthInitialClaimStatus(incoming)
		competitors := truthClaimCompetitors(snapshot, incoming)
		if len(competitors) == 0 {
			snapshot[incoming.ClaimID] = incoming
			truthRecordUpsert(&result, incoming)
			if incoming.Status == truthClaimStatusActive {
				result.ActiveIDs = append(result.ActiveIDs, incoming.ClaimID)
			}
			continue
		}
		for _, existing := range competitors {
			if !truthClaimValidityWindowsOverlap(existing, incoming) {
				if truthClaimNewerOrStronger(existing, incoming) {
					existing.Status = truthClaimStatusSuperseded
					snapshot[existing.ClaimID] = existing
					truthRecordStatusChange(&result, existing, "single-value non-overlap picked newer/stronger incoming claim")
					result.SupersededIDs = append(result.SupersededIDs, existing.ClaimID)
					continue
				}
				incoming.Status = truthClaimStatusSuperseded
				incoming.Supersedes = uniqueStrings(append(incoming.Supersedes, existing.ClaimID))
				continue
			}
			if truthSingleValueClaimsConflict(existing, incoming) {
				existing.Status = truthClaimStatusConflicted
				incoming.Status = truthClaimStatusConflicted
				snapshot[existing.ClaimID] = existing
				truthRecordStatusChange(&result, existing, "single-value overlap entered conflicted state")
				result.Conflicts = append(result.Conflicts, existing.ClaimID, incoming.ClaimID)
				continue
			}
			if truthClaimNewerOrStronger(existing, incoming) {
				existing.Status = truthClaimStatusSuperseded
				snapshot[existing.ClaimID] = existing
				truthRecordStatusChange(&result, existing, "single-value incoming claim superseded previous active claim")
				result.SupersededIDs = append(result.SupersededIDs, existing.ClaimID)
				continue
			}
			incoming.Status = truthClaimStatusSuperseded
			incoming.Supersedes = uniqueStrings(append(incoming.Supersedes, existing.ClaimID))
		}
		snapshot[incoming.ClaimID] = incoming
		truthRecordUpsert(&result, incoming)
		if incoming.Status == truthClaimStatusActive {
			result.ActiveIDs = append(result.ActiveIDs, incoming.ClaimID)
		}
		if incoming.Status == truthClaimStatusConflicted {
			truthRecordStatusChange(&result, incoming, "incoming claim remains conflicted after arbitration")
		}
		if incoming.Status == truthClaimStatusSuperseded {
			result.SupersededIDs = append(result.SupersededIDs, incoming.ClaimID)
		}
	}
	result.ActiveIDs = uniqueStrings(result.ActiveIDs)
	result.Conflicts = uniqueStrings(result.Conflicts)
	result.SupersededIDs = uniqueStrings(result.SupersededIDs)
	sort.SliceStable(result.UpsertedClaims, func(i, j int) bool { return result.UpsertedClaims[i].ClaimID < result.UpsertedClaims[j].ClaimID })
	sort.SliceStable(result.StatusChanges, func(i, j int) bool { return result.StatusChanges[i].ClaimID < result.StatusChanges[j].ClaimID })
	sort.Strings(result.ActiveIDs)
	sort.Strings(result.Conflicts)
	sort.Strings(result.SupersededIDs)
	return result
}

func truthProjectClaimsOntoObjects(objects map[string]MemoryObject, claims map[string]MemoryClaim, includeSuperseded bool) {
	if len(objects) == 0 {
		return
	}
	grouped := make(map[string][]MemoryClaim)
	for _, claim := range claims {
		normalized := normalizeMemoryClaim(claim, claim.ObjectID)
		if normalized.ObjectID == "" {
			continue
		}
		if !includeSuperseded && normalized.Status == truthClaimStatusSuperseded {
			continue
		}
		grouped[normalized.ObjectID] = append(grouped[normalized.ObjectID], normalized)
	}
	for objectID, object := range objects {
		object.Claims = grouped[objectID]
		objects[objectID] = normalizeMemoryObject(object)
	}
}

func truthClaimStatusTotals(claims map[string]MemoryClaim) (active int, conflicted int, superseded int, unverified int) {
	for _, claim := range claims {
		switch normalizeTruthClaimStatus(claim.Status) {
		case truthClaimStatusConflicted:
			conflicted++
		case truthClaimStatusSuperseded:
			superseded++
		case truthClaimStatusUnverified:
			unverified++
		default:
			active++
		}
	}
	return
}
