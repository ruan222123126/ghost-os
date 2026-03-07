package memory

import "strings"

const (
	truthStatusVerified   = "verified"
	truthStatusSupported  = "supported"
	truthStatusCandidate  = "candidate"
	truthStatusConflicted = "conflicted"
)

// TruthHit 描述 truth live read 的调试命中视图。
type TruthHit struct {
	ObjectID      string      `json:"object_id,omitempty"`
	ObjectType    string      `json:"object_type,omitempty"`
	Score         float64     `json:"score,omitempty"`
	Confidence    float64     `json:"confidence,omitempty"`
	Freshness     float64     `json:"freshness,omitempty"`
	EvidenceCount int         `json:"evidence_count,omitempty"`
	SourceRefs    []SourceRef `json:"source_refs,omitempty"`
	MatchedBy     []string    `json:"matched_by,omitempty"`
	Status        string      `json:"status,omitempty"`
}

// HybridRerankWeights 保留当前线性 rerank 的解释性权重。
type HybridRerankWeights struct {
	Semantic    float64 `json:"semantic,omitempty"`
	Lexical     float64 `json:"lexical,omitempty"`
	Truth       float64 `json:"truth,omitempty"`
	Decision    float64 `json:"decision,omitempty"`
	Graph       float64 `json:"graph,omitempty"`
	Freshness   float64 `json:"freshness,omitempty"`
	Importance  float64 `json:"importance,omitempty"`
	Environment float64 `json:"environment,omitempty"`
}

// HybridRerankItem 描述单条候选的排序解释。
type HybridRerankItem struct {
	EntryID       string   `json:"entry_id,omitempty"`
	Layer         string   `json:"layer,omitempty"`
	ObjectID      string   `json:"object_id,omitempty"`
	MatchedBy     []string `json:"matched_by,omitempty"`
	WhyMatched    string   `json:"why_matched,omitempty"`
	HybridScore   float64  `json:"hybrid_score,omitempty"`
	Confidence    float64  `json:"confidence,omitempty"`
	TruthStatus   string   `json:"truth_status,omitempty"`
	SupportCount  int      `json:"support_count,omitempty"`
	ConflictCount int      `json:"conflict_count,omitempty"`
}

// HybridRerankReport 用于灰度观察候选归一与最终排序。
type HybridRerankReport struct {
	Enabled    bool                `json:"enabled,omitempty"`
	Weights    HybridRerankWeights `json:"weights,omitempty"`
	Candidates []HybridRerankItem  `json:"candidates,omitempty"`
}

func normalizeTruthStatus(status string) string {
	switch strings.TrimSpace(status) {
	case truthStatusVerified:
		return truthStatusVerified
	case truthStatusSupported:
		return truthStatusSupported
	case truthStatusConflicted:
		return truthStatusConflicted
	default:
		return truthStatusCandidate
	}
}
