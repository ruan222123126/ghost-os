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

type PlannerInput struct {
	SessionID      string        `json:"session_id"`
	UserScope      string        `json:"user_scope_id"`
	UserMessage    string        `json:"user_message"`
	RecentMessages []TurnMessage `json:"recent_messages,omitempty"`
	ProjectRoot    string        `json:"project_root,omitempty"`
}

type PlannerEventRef struct {
	EventID string `json:"event_id,omitempty"`
	Reason  string `json:"reason,omitempty"`
}

type PlannerEdgeUpdate struct {
	FromEventID string  `json:"from_event_id,omitempty"`
	ToEventID   string  `json:"to_event_id,omitempty"`
	EdgeType    string  `json:"edge_type,omitempty"`
	Confidence  float64 `json:"confidence,omitempty"`
}

type RecallPlan struct {
	EventIDs           []string `json:"event_ids,omitempty"`
	IncludeNodeSummary bool     `json:"include_node_summary"`
	IncludeWorkflow    bool     `json:"include_workflow"`
	IncludePreference  bool     `json:"include_preference"`
	IncludeFact        bool     `json:"include_fact"`
	IncludeProfile     bool     `json:"include_profile"`
	AllowLearning      bool     `json:"allow_learning"`
}

type PlannerDecision struct {
	PrimaryEvent    PlannerEventRef     `json:"primary_event"`
	AdjacentEvents  []PlannerEventRef   `json:"adjacent_events,omitempty"`
	ReuseExisting   bool                `json:"reuse_existing"`
	CreateNewEvent  bool                `json:"create_new_event"`
	NewEventTitle   string              `json:"new_event_title,omitempty"`
	NewEventSummary string              `json:"new_event_summary,omitempty"`
	EdgeUpdates     []PlannerEdgeUpdate `json:"edge_updates,omitempty"`
	RecallPlan      RecallPlan          `json:"recall_plan"`
	RawJSON         string              `json:"-"`
}

type RecallInput struct {
	SessionID      string     `json:"session_id"`
	PrimaryEventID string     `json:"primary_event_id"`
	ActiveEventIDs []string   `json:"active_event_ids"`
	FocusText      string     `json:"focus_text,omitempty"`
	RecallPlan     RecallPlan `json:"recall_plan"`
}

type RecallEventHit struct {
	Event  memorystore.EventNode `json:"event"`
	Role   string                `json:"role"`
	Reason string                `json:"reason"`
}

type RecallMemoryHit struct {
	Event  memorystore.EventNode   `json:"event"`
	Entry  memorystore.EventMemory `json:"entry"`
	Reason string                  `json:"reason"`
}

type RecallOutput struct {
	PrimaryEvent      *RecallEventHit           `json:"primary_event,omitempty"`
	AdjacentEvents    []RecallEventHit          `json:"adjacent_events,omitempty"`
	PrimaryMemories   []RecallMemoryHit         `json:"primary_memories,omitempty"`
	AdjacentMemories  []RecallMemoryHit         `json:"adjacent_memories,omitempty"`
	GlobalPreferences []memorystore.MemoryEntry `json:"global_preferences,omitempty"`
	PromptBlock       string                    `json:"prompt_block,omitempty"`
}

type LearnFromTurnInput struct {
	SessionID      string        `json:"session_id"`
	UserScope      string        `json:"user_scope_id"`
	TraceID        string        `json:"trace_id,omitempty"`
	PrimaryEventID string        `json:"primary_event_id,omitempty"`
	ActiveEventIDs []string      `json:"active_event_ids,omitempty"`
	AllowWrite     bool          `json:"allow_write"`
	Messages       []TurnMessage `json:"messages"`
}

type IntentPlanner interface {
	Plan(ctx context.Context, input PlannerInput) (PlannerDecision, error)
}

type RecallService interface {
	Recall(ctx context.Context, input RecallInput) (RecallOutput, error)
}

type LearningService interface {
	LearnFromTurn(ctx context.Context, input LearnFromTurnInput) error
}

type plannerStore interface {
	CreateEventNode(ctx context.Context, input memorystore.EventNodeInput) (memorystore.EventNode, error)
	GetEventNode(ctx context.Context, id string) (memorystore.EventNode, error)
	ListEventNodes(ctx context.Context, filter memorystore.EventNodeFilter) ([]memorystore.EventNode, error)
	ListEventEdges(ctx context.Context, sessionID string, eventIDs []string) ([]memorystore.EventEdge, error)
	SaveSessionEventState(ctx context.Context, state memorystore.SessionEventState) error
	LoadSessionEventState(ctx context.Context, sessionID string) (memorystore.SessionEventState, error)
	TouchEventNodes(ctx context.Context, ids []string) error
	UpsertEventEdge(ctx context.Context, input memorystore.EventEdgeInput) (memorystore.EventEdge, error)
}

type recallStore interface {
	GetEventNode(ctx context.Context, id string) (memorystore.EventNode, error)
	ListEventMemories(ctx context.Context, filter memorystore.EventMemoryListFilter) ([]memorystore.EventMemory, int, error)
	ListGlobalPreferences(ctx context.Context, memoryKeys []string) ([]memorystore.MemoryEntry, error)
	TouchEventMemories(ctx context.Context, ids []string) error
	TouchExplicitRecords(ctx context.Context, uris []string) error
	TouchLearned(ctx context.Context, ids []string) error
}

type learningStore interface {
	CreateLearningEvent(ctx context.Context, input memorystore.LearningEventInput) (memorystore.LearningEvent, error)
	ListGlobalPreferences(ctx context.Context, memoryKeys []string) ([]memorystore.MemoryEntry, error)
	FindActiveLearnedByMemoryKey(ctx context.Context, scopeType string, scopeID string, memoryKey string) (memorystore.MemoryEntry, error)
	CreateLearned(ctx context.Context, input memorystore.LearnedMemoryInput, supersedesIDs []string) (memorystore.MemoryEntry, error)
	RefreshLearned(ctx context.Context, id string, confidence float64) error
	ListEventMemories(ctx context.Context, filter memorystore.EventMemoryListFilter) ([]memorystore.EventMemory, int, error)
	FindActiveEventMemoryByKey(ctx context.Context, eventID string, memoryKey string) (memorystore.EventMemory, error)
	CreateEventMemory(ctx context.Context, input memorystore.EventMemoryInput, supersedesIDs []string) (memorystore.EventMemory, error)
	RefreshEventMemory(ctx context.Context, id string, confidence float64) error
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
	MemoryKey    string   `json:"memory_key,omitempty"`
	Value        string   `json:"value,omitempty"`
	Summary      string   `json:"summary"`
	Content      string   `json:"content"`
	ScopeType    string   `json:"scope_type"`
	ScopeID      string   `json:"scope_id"`
	Confidence   float64  `json:"confidence"`
	SupersedesID []string `json:"supersedes_ids"`
	Reason       string   `json:"reason"`
}

type EventMemoryExtractor interface {
	Extract(ctx context.Context, input EventExtractInput) (EventExtractOutput, error)
}

type EventExtractInput struct {
	SessionID      string                    `json:"session_id"`
	PrimaryEventID string                    `json:"primary_event_id"`
	ActiveEventIDs []string                  `json:"active_event_ids,omitempty"`
	Transcript     []TurnMessage             `json:"transcript"`
	Existing       []memorystore.EventMemory `json:"existing,omitempty"`
}

type EventExtractOutput struct {
	RawJSON string                 `json:"raw_json"`
	Items   []EventMemoryCandidate `json:"items"`
}

type EventMemoryCandidate struct {
	MemoryType   string   `json:"memory_type"`
	MemoryKey    string   `json:"memory_key,omitempty"`
	Summary      string   `json:"summary"`
	Content      string   `json:"content"`
	Confidence   float64  `json:"confidence"`
	SupersedesID []string `json:"supersedes_ids"`
	Reason       string   `json:"reason"`
}

type applyOutcome struct {
	Created        []memorystore.MemoryEntry `json:"created_global_preferences,omitempty"`
	Refreshed      []string                  `json:"refreshed_global_preferences,omitempty"`
	EventCreated   []memorystore.EventMemory `json:"created_event_memories,omitempty"`
	EventRefreshed []string                  `json:"refreshed_event_memories,omitempty"`
	Skipped        []string                  `json:"skipped,omitempty"`
	Superseded     []string                  `json:"superseded,omitempty"`
}
