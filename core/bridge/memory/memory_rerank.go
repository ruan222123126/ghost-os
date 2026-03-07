package memory

import "sort"

type hybridRerankConfig struct {
	conflictPenalty float64
}

func hybridRerankWeights() HybridRerankWeights {
	return HybridRerankWeights{
		Semantic:    0.22,
		Lexical:     0.18,
		Truth:       0.18,
		Decision:    0.14,
		Graph:       0.10,
		Freshness:   0.08,
		Importance:  0.06,
		Environment: 0.04,
	}
}

func rerankRecallCandidates(candidates []RecallCandidate, config hybridRerankConfig) ([]RecallCandidate, *HybridRerankReport) {
	if len(candidates) == 0 {
		return nil, &HybridRerankReport{Enabled: true, Weights: hybridRerankWeights()}
	}
	weights := hybridRerankWeights()
	out := make([]RecallCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		item := normalizeRecallCandidate(candidate)
		importance := item.ImportanceScore
		if importance <= 0 {
			importance = clamp01(item.Entry.Importance)
		}
		score := item.SemanticScore*weights.Semantic +
			item.LexicalScore*weights.Lexical +
			item.TruthScore*weights.Truth +
			item.DecisionScore*weights.Decision +
			item.GraphScore*weights.Graph +
			item.Freshness*weights.Freshness +
			importance*weights.Importance +
			item.EnvironmentScore*weights.Environment
		if item.ConflictCount > 0 || item.TruthStatus == truthStatusConflicted {
			score -= maxFloat(config.conflictPenalty, 0.12) * float64(max(item.ConflictCount, 1))
		}
		item.RerankScore = clamp01(score)
		item.Entry.RerankScore = item.RerankScore
		out = append(out, normalizeRecallCandidate(item))
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].RerankScore != out[j].RerankScore {
			return out[i].RerankScore > out[j].RerankScore
		}
		if out[i].Entry.Confidence != out[j].Entry.Confidence {
			return out[i].Entry.Confidence > out[j].Entry.Confidence
		}
		if out[i].Freshness != out[j].Freshness {
			return out[i].Freshness > out[j].Freshness
		}
		if out[i].Entry.Timestamp.Equal(out[j].Entry.Timestamp) {
			return out[i].Entry.ID < out[j].Entry.ID
		}
		return out[i].Entry.Timestamp.After(out[j].Entry.Timestamp)
	})
	report := &HybridRerankReport{Enabled: true, Weights: weights, Candidates: make([]HybridRerankItem, 0, len(out))}
	for _, item := range out {
		report.Candidates = append(report.Candidates, HybridRerankItem{
			EntryID:       item.Entry.ID,
			Layer:         item.Layer,
			ObjectID:      item.ObjectID,
			MatchedBy:     append([]string(nil), item.MatchedBy...),
			WhyMatched:    item.WhyMatched,
			HybridScore:   item.RerankScore,
			Confidence:    item.Entry.Confidence,
			TruthStatus:   item.TruthStatus,
			SupportCount:  item.SupportCount,
			ConflictCount: item.ConflictCount,
		})
	}
	return out, report
}
