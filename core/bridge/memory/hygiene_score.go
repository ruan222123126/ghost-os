package memory

import "time"

func hygieneScoreFromVotes(votes int) float64 {
	if votes <= 0 {
		return 0
	}
	return clamp01(float64(votes) * 0.2)
}

func hygieneQuarantineLevelFromVotes(votes int) string {
	switch {
	case votes >= 6:
		return QuarantineLevelHard
	case votes >= 4:
		return QuarantineLevelStrong
	case votes >= 2:
		return QuarantineLevelWeak
	default:
		return QuarantineLevelNone
	}
}

func hygieneScoredAtOrNow(scoredAt time.Time) time.Time {
	if scoredAt.IsZero() {
		return time.Now().UTC()
	}
	return scoredAt.UTC()
}

func mergeHygieneReasons(existing []HygieneReason, incoming []HygieneReason) []HygieneReason {
	if len(existing) == 0 && len(incoming) == 0 {
		return nil
	}
	out := make([]HygieneReason, 0, len(existing)+len(incoming))
	indexByCode := make(map[string]int, len(existing)+len(incoming))
	for _, reason := range existing {
		normalized := normalizeHygieneReason(reason)
		if hygieneReasonEmpty(normalized) {
			continue
		}
		if normalized.Code != "" {
			if index, ok := indexByCode[normalized.Code]; ok {
				out[index] = mergeHygieneReasonFields(out[index], normalized)
				continue
			}
			indexByCode[normalized.Code] = len(out)
		}
		out = append(out, normalized)
	}
	for _, reason := range incoming {
		normalized := normalizeHygieneReason(reason)
		if hygieneReasonEmpty(normalized) {
			continue
		}
		if normalized.Code != "" {
			if index, ok := indexByCode[normalized.Code]; ok {
				out[index] = mergeHygieneReasonFields(out[index], normalized)
				continue
			}
			indexByCode[normalized.Code] = len(out)
		}
		out = append(out, normalized)
	}
	return out
}

func mergeHygieneReasonFields(current HygieneReason, incoming HygieneReason) HygieneReason {
	merged := current
	if incoming.Label != "" {
		merged.Label = incoming.Label
	}
	if incoming.Evidence != "" {
		merged.Evidence = incoming.Evidence
	}
	if incoming.Weight > merged.Weight {
		merged.Weight = incoming.Weight
	}
	return merged
}
