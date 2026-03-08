package decision

import (
	"strings"
	"time"

	"ghost-os/bridge/llm"
)

type Counter interface{ Add(uint64) }

type DecisionMetrics struct {
	recipeSelections      Counter
	recipeApplied         Counter
	recipeSuccess         Counter
	recipePartial         Counter
	recipeFailure         Counter
	recipeHumanBlocked    Counter
	recipeDeviations      Counter
	recipeFallbacks       Counter
	recipeBackfillScanned Counter
	recipeBackfillCreated Counter
	recipeBackfillUpdated Counter
	recipeDefaultGrayHits Counter
}

func NewDecisionMetrics(recipeSelections Counter, recipeApplied Counter, recipeSuccess Counter, recipePartial Counter, recipeFailure Counter, recipeHumanBlocked Counter, recipeDeviations Counter, recipeFallbacks Counter, recipeBackfillScanned Counter, recipeBackfillCreated Counter, recipeBackfillUpdated Counter, recipeDefaultGrayHits Counter) *DecisionMetrics {
	return &DecisionMetrics{
		recipeSelections:      recipeSelections,
		recipeApplied:         recipeApplied,
		recipeSuccess:         recipeSuccess,
		recipePartial:         recipePartial,
		recipeFailure:         recipeFailure,
		recipeHumanBlocked:    recipeHumanBlocked,
		recipeDeviations:      recipeDeviations,
		recipeFallbacks:       recipeFallbacks,
		recipeBackfillScanned: recipeBackfillScanned,
		recipeBackfillCreated: recipeBackfillCreated,
		recipeBackfillUpdated: recipeBackfillUpdated,
		recipeDefaultGrayHits: recipeDefaultGrayHits,
	}
}

type MemoryType string

const (
	MemoryTypeMessage   MemoryType = "message"
	MemoryTypeSummary   MemoryType = "summary"
	MemoryTypeKnowledge MemoryType = "knowledge"
)

type MemoryAnchor struct {
	Type       string    `json:"type,omitempty" yaml:"type,omitempty"`
	Key        string    `json:"key,omitempty" yaml:"key,omitempty"`
	Value      string    `json:"value,omitempty" yaml:"value,omitempty"`
	Weight     float64   `json:"weight,omitempty" yaml:"weight,omitempty"`
	Reason     string    `json:"reason,omitempty" yaml:"reason,omitempty"`
	DetectedAt time.Time `json:"detected_at,omitempty" yaml:"detected_at,omitempty"`
	ExpiresAt  time.Time `json:"expires_at,omitempty" yaml:"expires_at,omitempty"`
	SessionID  string    `json:"session_id,omitempty" yaml:"session_id,omitempty"`
}

type MemoryEntry struct {
	ID             string         `json:"id"`
	Content        string         `json:"content"`
	Type           MemoryType     `json:"type"`
	Timestamp      time.Time      `json:"timestamp"`
	AccessCount    int            `json:"access_count"`
	LastAccessedAt time.Time      `json:"last_accessed_at,omitempty"`
	Metadata       map[string]any `json:"metadata,omitempty"`
	Importance     float64        `json:"importance,omitempty"`
	ExpiresAt      time.Time      `json:"expires_at,omitempty"`
	RelatedTo      []string       `json:"related_to,omitempty"`
	Source         string         `json:"source,omitempty"`
	Summary        string         `json:"summary,omitempty"`
	Confidence     float64        `json:"confidence,omitempty"`
	WhyMatched     string         `json:"why_matched,omitempty"`
}

type TimeRange struct {
	Start time.Time `json:"start,omitempty"`
	End   time.Time `json:"end,omitempty"`
}

func (r *TimeRange) Contains(ts time.Time) bool {
	if r == nil {
		return true
	}
	t := ts.UTC()
	if !r.Start.IsZero() && t.Before(r.Start.UTC()) {
		return false
	}
	if !r.End.IsZero() && t.After(r.End.UTC()) {
		return false
	}
	return true
}

type MemoryQuery struct {
	TimeRange         *TimeRange              `json:"time_range,omitempty"`
	Limit             int                     `json:"limit,omitempty"`
	Keywords          []string                `json:"keywords,omitempty"`
	Metadata          map[string]any          `json:"metadata,omitempty"`
	SemanticQuery     string                  `json:"semantic_query,omitempty"`
	MinConfidence     float64                 `json:"min_confidence,omitempty"`
	IncludeDecision   bool                    `json:"include_decision,omitempty"`
	DecisionDebug     bool                    `json:"decision_debug,omitempty"`
	DecisionReuseOnly bool                    `json:"decision_reuse_only,omitempty"`
	DecisionTypes     []string                `json:"decision_types,omitempty"`
	EnvironmentStrict bool                    `json:"environment_strict,omitempty"`
	MinReuseScore     float64                 `json:"min_reuse_score,omitempty"`
	Environment       *DecisionEnvFingerprint `json:"-"`
}

type SessionScope struct {
	SessionID   string
	Environment *DecisionEnvFingerprint
}

type QueryIntentPlan struct {
	Constraints []string `json:"constraints,omitempty"`
	Environment []string `json:"environment,omitempty"`
	Risks       []string `json:"risks,omitempty"`
}

type HybridRerankItem struct {
	EntryID     string  `json:"entry_id,omitempty"`
	HybridScore float64 `json:"hybrid_score,omitempty"`
}

type HybridRerankReport struct {
	Candidates []HybridRerankItem `json:"candidates,omitempty"`
}

type MarkdownNode struct {
	ID          string         `yaml:"id"`
	Importance  float64        `yaml:"importance"`
	CreatedAt   time.Time      `yaml:"created_at"`
	RelatedTo   []string       `yaml:"related_to,omitempty"`
	Tags        []string       `yaml:"tags,omitempty"`
	SessionID   string         `yaml:"session_id,omitempty"`
	EmbeddingID string         `yaml:"embedding_id,omitempty"`
	Summary     string         `yaml:"summary,omitempty"`
	Anchors     []MemoryAnchor `yaml:"anchors,omitempty"`
	SourceIDs   []string       `yaml:"source_ids,omitempty"`
	Confidence  float64        `yaml:"confidence,omitempty"`
	LastSeenAt  time.Time      `yaml:"last_seen_at,omitempty"`
	Content     string         `yaml:"-"`
}

type ColdArchive struct {
	SessionID  string        `json:"session_id"`
	ArchivedAt time.Time     `json:"archived_at"`
	Messages   []llm.Message `json:"messages"`
}

type ColdStore interface {
	ListArchives(timeRange *TimeRange) ([]ColdArchive, error)
	ListMarkdownNodes() ([]string, error)
	LoadMarkdownNode(id string) (MarkdownNode, error)
}

type DecisionMemoExtractor interface {
	ExtractDecisionMemo(input DecisionCaptureInput) (DecisionMemo, error)
}

type TruthShadow interface {
	WriteDecisionMemo(memo DecisionMemo, input DecisionCaptureInput) error
}

type MemoryConfig struct {
	DecisionEnabled                bool
	DecisionCaptureOnTurn          bool
	DecisionCaptureOnTurnSet       bool
	DecisionPath                   string
	DecisionMaxHits                int
	DecisionMinConfidence          float64
	DecisionMinReuseScore          float64
	DecisionRecipeEnabled          bool
	DecisionRecipeInterval         time.Duration
	DecisionRecipeMinSupport       int
	DecisionDebugEnabled           bool
	RecipeReuseEnabled             bool
	RecipeExecutionTrackingEnabled bool
	RecipeBackfillEnabled          bool
	RecipeDefaultEnabled           bool
	RecipeDefaultGrayPercent       int
	RecipeMinSelectionConfidence   float64
	RecipeMinSuccessRate           float64
	RecipeBackfillBatchSize        int
	RecipeBackfillInterval         time.Duration
	WorkerExtractor                DecisionMemoExtractor
}

func normalizeEntry(entry MemoryEntry) MemoryEntry {
	out := entry
	out.ID = strings.TrimSpace(out.ID)
	out.Content = strings.TrimSpace(out.Content)
	if out.Type == "" {
		out.Type = MemoryTypeMessage
	}
	if out.Timestamp.IsZero() {
		out.Timestamp = time.Now().UTC()
	} else {
		out.Timestamp = out.Timestamp.UTC()
	}
	if out.Metadata == nil {
		out.Metadata = make(map[string]any, 2)
	}
	if out.Importance < 0 {
		out.Importance = 0
	}
	if out.Importance > 1 {
		out.Importance = 1
	}
	out.RelatedTo = uniqueStrings(out.RelatedTo)
	out.Source = strings.TrimSpace(out.Source)
	out.Summary = strings.TrimSpace(out.Summary)
	out.Confidence = clamp01(out.Confidence)
	out.WhyMatched = strings.TrimSpace(out.WhyMatched)
	return out
}

func markdownNodeTimestamp(node MarkdownNode) time.Time {
	if !node.LastSeenAt.IsZero() {
		return node.LastSeenAt.UTC()
	}
	if !node.CreatedAt.IsZero() {
		return node.CreatedAt.UTC()
	}
	return time.Now().UTC()
}
