package memory

import (
	"math"
	"strings"
	"time"
)

type recallVerifyConfig struct {
	minSupportRefs  int
	conflictPenalty float64
}

func verifyRecallCandidates(candidates []RecallCandidate, truth *TruthReader, config recallVerifyConfig) []RecallCandidate {
	if len(candidates) == 0 {
		return nil
	}
	claims := map[string]MemoryClaim(nil)
	if truth != nil && truth.Enabled() {
		truth.mu.RLock()
		claims = cloneTruthClaims(truth.claims)
		truth.mu.RUnlock()
	}
	now := time.Now().UTC()
	out := make([]RecallCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		item := normalizeRecallCandidate(candidate)
		if item.ImportanceScore <= 0 {
			item.ImportanceScore = clamp01(item.Entry.Importance)
		}
		if item.Freshness <= 0 {
			item.Freshness = truthlessFreshness(item.Entry, now)
		}
		item.SourceRefs = normalizeSourceRefs(item.SourceRefs)
		item.SupportCount = max(item.SupportCount, truthlessSupportCount(item))
		if item.ObjectID != "" && truth != nil && truth.Enabled() {
			if object, ok := truth.LookupObject(item.ObjectID); ok {
				item.Entry = hydrateEntryWithTruthObject(item.Entry, object)
				item.EvidenceCount = max(item.EvidenceCount, len(object.RawEvidence))
				item.SourceRefs = normalizeSourceRefs(append(item.SourceRefs, truthObjectSourceRefs(object)...))
				item.Freshness = maxFloat(item.Freshness, truthObjectFreshness(object, now))
				item.SupportCount = max(item.SupportCount, truthObjectSupportCount(object)+max(0, len(uniqueStrings(item.MatchedBy))-1))
				item.ConflictCount = max(item.ConflictCount, truthConflictCount(object, claims))
				if item.TruthScore <= 0 {
					item.TruthScore = clamp01(object.Confidence)
				}
				status := truthObjectStatus(object, max(config.minSupportRefs, 2), item.ConflictCount)
				if item.DecisionScore > 0 && item.GraphScore > 0 && status == truthStatusSupported {
					status = truthStatusVerified
				}
				item.TruthStatus = status
				item.Entry.Confidence = truthObjectConfidence(object, status, max(config.minSupportRefs, 2), claims)
			} else {
				item.TruthStatus = truthStatusCandidate
				item.Entry.Confidence = candidateOnlyConfidence(item)
			}
		} else {
			if item.ConflictCount > 0 {
				item.TruthStatus = truthStatusConflicted
				item.Entry.Confidence = conflictedConfidence(item, config.conflictPenalty)
			} else {
				item.TruthStatus = truthStatusCandidate
				item.Entry.Confidence = candidateOnlyConfidence(item)
			}
		}
		item.Entry.Freshness = item.Freshness
		item.Entry.EvidenceCount = item.EvidenceCount
		item.Entry.SourceRefs = item.SourceRefs
		item.Entry.TruthStatus = item.TruthStatus
		item.Entry.WhyMatched = item.WhyMatched
		out = append(out, normalizeRecallCandidate(item))
	}
	return out
}

func truthlessSupportCount(candidate RecallCandidate) int {
	refs := len(normalizeSourceRefs(candidate.SourceRefs))
	layers := len(uniqueStrings(candidate.MatchedBy))
	return max(refs+max(0, layers-1), max(candidate.EvidenceCount, 0))
}

func truthlessFreshness(entry MemoryEntry, now time.Time) float64 {
	if entry.Timestamp.IsZero() {
		return 0
	}
	age := now.UTC().Sub(entry.Timestamp.UTC())
	if age < 0 {
		age = 0
	}
	return clamp01(maxFloat(entry.Freshness, math.Exp2(-age.Hours()/(21*24))))
}

func candidateOnlyConfidence(candidate RecallCandidate) float64 {
	base := maxFloat(candidate.LexicalScore, maxFloat(candidate.SemanticScore, candidate.BaseScore))
	confidence := 0.22 + base*0.16 + candidate.Freshness*0.05 + float64(minInt(candidate.SupportCount, 2))*0.04 + candidate.DecisionScore*0.03 + candidate.GraphScore*0.03
	return clamp01(min(0.55, maxFloat(0.18, confidence)))
}

func conflictedConfidence(candidate RecallCandidate, penalty float64) float64 {
	conflictPenalty := maxFloat(penalty, 0.1)
	confidence := 0.34 + candidate.TruthScore*0.05 + candidate.Freshness*0.04 - float64(max(candidate.ConflictCount, 1))*conflictPenalty
	return clamp01(min(0.35, maxFloat(0.12, confidence)))
}

func hydrateEntryWithTruthObject(entry MemoryEntry, object MemoryObject) MemoryEntry {
	out := cloneEntry(entry)
	if strings.TrimSpace(out.Content) == "" {
		out.Content = truthObjectSummary(object)
	}
	if strings.TrimSpace(out.Summary) == "" || out.Source == "truth" {
		out.Summary = truthObjectSummary(object)
	}
	if out.Timestamp.IsZero() {
		out.Timestamp = effectiveDecisionTimestamp(object.UpdatedAt, object.CreatedAt)
	}
	out.EvidenceCount = max(out.EvidenceCount, len(object.RawEvidence))
	out.SourceRefs = normalizeSourceRefs(append(out.SourceRefs, truthObjectSourceRefs(object)...))
	if out.Metadata == nil {
		out.Metadata = make(map[string]any, 2)
	}
	out.Metadata["object_id"] = object.ObjectID
	out.Metadata["object_type"] = object.ObjectType
	return normalizeEntry(out)
}
