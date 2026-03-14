package memoryaug

import (
	"context"

	"ghost-os/bridge/memorystore"
)

type Settings struct {
	Enabled             bool
	LearningEnabled     bool
	RecallEnabled       bool
	MaxRecallItems      int
	MinConfidence       float64
	SessionScopeEnabled bool
	UserScopeEnabled    bool
	LLMModel            string
	UserScopeID         string
}

type TurnMessage struct {
	Role string `json:"role"`
	Text string `json:"text"`
}

type LearnFromTurnInput struct {
	SessionID string        `json:"session_id"`
	UserScope string        `json:"user_scope_id"`
	TraceID   string        `json:"trace_id,omitempty"`
	Messages  []TurnMessage `json:"messages"`
}

type RecallInput struct {
	SessionID string `json:"session_id"`
	UserScope string `json:"user_scope_id"`
	Query     string `json:"query"`
}

type RecallItem struct {
	Entry     memorystore.MemoryEntry `json:"entry"`
	TextScore float64                 `json:"text_score"`
	Reason    string                  `json:"reason"`
}

type LearningService interface {
	LearnFromTurn(ctx context.Context, input LearnFromTurnInput) error
}

type RecallService interface {
	Recall(ctx context.Context, input RecallInput) ([]RecallItem, error)
}

type learningStore interface {
	CreateLearningEvent(ctx context.Context, input memorystore.LearningEventInput) (memorystore.LearningEvent, error)
	SearchExplicitRecallRecords(ctx context.Context, query string, limit int) ([]memorystore.MemoryEntry, error)
	ListLearned(ctx context.Context, filter memorystore.LearnedListFilter) ([]memorystore.MemoryEntry, int, error)
	GetLearnedByIDs(ctx context.Context, ids []string) ([]memorystore.MemoryEntry, error)
	CreateLearned(ctx context.Context, input memorystore.LearnedMemoryInput, supersedesIDs []string) (memorystore.MemoryEntry, error)
	RefreshLearned(ctx context.Context, id string, confidence float64) error
}

type recallStore interface {
	SearchExplicitRecallRecords(ctx context.Context, query string, limit int) ([]memorystore.MemoryEntry, error)
	ListLearned(ctx context.Context, filter memorystore.LearnedListFilter) ([]memorystore.MemoryEntry, int, error)
	TouchExplicitRecords(ctx context.Context, uris []string) error
	TouchLearned(ctx context.Context, ids []string) error
}

type CandidateExtractor interface {
	Extract(ctx context.Context, input ExtractInput) (ExtractOutput, error)
}

type ExtractInput struct {
	SessionID        string                    `json:"session_id"`
	UserScopeID      string                    `json:"user_scope_id"`
	Transcript       []TurnMessage             `json:"transcript"`
	ExistingExplicit []memorystore.MemoryEntry `json:"existing_explicit,omitempty"`
	ExistingLearned  []memorystore.MemoryEntry `json:"existing_learned,omitempty"`
}

type ExtractOutput struct {
	RawJSON string      `json:"raw_json"`
	Items   []Candidate `json:"items"`
}

type Candidate struct {
	MemoryType   string   `json:"memory_type"`
	Summary      string   `json:"summary"`
	Content      string   `json:"content"`
	ScopeType    string   `json:"scope_type"`
	ScopeID      string   `json:"scope_id"`
	Confidence   float64  `json:"confidence"`
	SupersedesID []string `json:"supersedes_ids"`
	Reason       string   `json:"reason"`
}

type applyOutcome struct {
	Created    []memorystore.MemoryEntry `json:"created,omitempty"`
	Refreshed  []string                  `json:"refreshed,omitempty"`
	Skipped    []string                  `json:"skipped,omitempty"`
	Superseded []string                  `json:"superseded,omitempty"`
}
