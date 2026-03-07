package memory

import "strings"

// RecallCandidate 是跨 layer 的统一 live recall 候选。
type RecallCandidate struct {
	Entry            MemoryEntry `json:"entry"`
	Layer            string      `json:"layer,omitempty"`
	ObjectID         string      `json:"object_id,omitempty"`
	BaseScore        float64     `json:"base_score,omitempty"`
	LexicalScore     float64     `json:"lexical_score,omitempty"`
	SemanticScore    float64     `json:"semantic_score,omitempty"`
	GraphScore       float64     `json:"graph_score,omitempty"`
	DecisionScore    float64     `json:"decision_score,omitempty"`
	TruthScore       float64     `json:"truth_score,omitempty"`
	EnvironmentScore float64     `json:"environment_score,omitempty"`
	ImportanceScore  float64     `json:"importance_score,omitempty"`
	Freshness        float64     `json:"freshness,omitempty"`
	EvidenceCount    int         `json:"evidence_count,omitempty"`
	SourceRefs       []SourceRef `json:"source_refs,omitempty"`
	ConflictCount    int         `json:"conflict_count,omitempty"`
	SupportCount     int         `json:"support_count,omitempty"`
	WhyMatched       string      `json:"why_matched,omitempty"`
	TruthStatus      string      `json:"truth_status,omitempty"`
	RerankScore      float64     `json:"rerank_score,omitempty"`
	MatchedBy        []string    `json:"matched_by,omitempty"`
}

func normalizeRecallCandidate(candidate RecallCandidate) RecallCandidate {
	out := candidate
	out.Entry = normalizeEntry(out.Entry)
	out.Layer = strings.TrimSpace(out.Layer)
	out.ObjectID = strings.TrimSpace(out.ObjectID)
	out.BaseScore = clamp01(out.BaseScore)
	out.LexicalScore = clamp01(out.LexicalScore)
	out.SemanticScore = clamp01(out.SemanticScore)
	out.GraphScore = clamp01(out.GraphScore)
	out.DecisionScore = clamp01(out.DecisionScore)
	out.TruthScore = clamp01(out.TruthScore)
	out.EnvironmentScore = clamp01(out.EnvironmentScore)
	out.ImportanceScore = clamp01(out.ImportanceScore)
	out.Freshness = clamp01(out.Freshness)
	out.EvidenceCount = max(out.EvidenceCount, 0)
	out.SourceRefs = normalizeSourceRefs(out.SourceRefs)
	out.ConflictCount = max(out.ConflictCount, 0)
	out.SupportCount = max(out.SupportCount, 0)
	out.WhyMatched = strings.TrimSpace(out.WhyMatched)
	out.TruthStatus = normalizeTruthStatus(out.TruthStatus)
	out.RerankScore = clamp01(out.RerankScore)
	out.MatchedBy = uniqueStrings(out.MatchedBy)
	return out
}

func cloneRecallCandidate(candidate RecallCandidate) RecallCandidate {
	out := candidate
	out.Entry = cloneEntry(candidate.Entry)
	out.SourceRefs = append([]SourceRef(nil), candidate.SourceRefs...)
	out.MatchedBy = append([]string(nil), candidate.MatchedBy...)
	return out
}

func cloneRecallCandidates(candidates []RecallCandidate) []RecallCandidate {
	if len(candidates) == 0 {
		return nil
	}
	out := make([]RecallCandidate, len(candidates))
	for i := range candidates {
		out[i] = cloneRecallCandidate(candidates[i])
	}
	return out
}
