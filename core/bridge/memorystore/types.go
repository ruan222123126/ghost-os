package memorystore

import "time"

const (
	ScopeTypeUser    = "user"
	ScopeTypeSession = "session"

	SourceKindExplicit = "explicit"
	SourceKindLearned  = "learned"

	MemoryTypeProfile    = "profile"
	MemoryTypePreference = "preference"
	MemoryTypeWorkflow   = "workflow"
	MemoryTypeFact       = "fact"

	MemoryStatusActive     = "active"
	MemoryStatusSuperseded = "superseded"
	MemoryStatusDeleted    = "deleted"

	DefaultUserScopeID = "local-user"
)

type Record struct {
	URI        string         `json:"uri"`
	Content    string         `json:"content"`
	Metadata   map[string]any `json:"metadata,omitempty"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	LastUsedAt time.Time      `json:"last_used_at,omitempty"`
}

type MemoryEntry struct {
	ID         string         `json:"id"`
	ScopeType  string         `json:"scope_type"`
	ScopeID    string         `json:"scope_id"`
	SourceKind string         `json:"source_kind"`
	MemoryType string         `json:"memory_type"`
	Content    string         `json:"content"`
	Summary    string         `json:"summary"`
	Metadata   map[string]any `json:"metadata_json,omitempty"`
	Confidence float64        `json:"confidence"`
	Status     string         `json:"status"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	LastUsedAt time.Time      `json:"last_used_at,omitempty"`
}

type LearnedMemoryInput struct {
	ScopeType  string
	ScopeID    string
	MemoryType string
	Content    string
	Summary    string
	Metadata   map[string]any
	Confidence float64
}

type LearnedListFilter struct {
	ScopeType  string
	ScopeID    string
	MemoryType string
	Statuses   []string
	Query      string
	Limit      int
	Offset     int
}

type LearningEvent struct {
	ID             string    `json:"id"`
	SessionID      string    `json:"session_id"`
	TraceID        string    `json:"trace_id,omitempty"`
	Status         string    `json:"status"`
	InputJSON      string    `json:"input_json,omitempty"`
	FilteredJSON   string    `json:"filtered_json,omitempty"`
	CandidatesJSON string    `json:"candidates_json,omitempty"`
	ResultJSON     string    `json:"result_json,omitempty"`
	ErrorText      string    `json:"error_text,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

type LearningEventInput struct {
	SessionID      string
	TraceID        string
	Status         string
	InputJSON      string
	FilteredJSON   string
	CandidatesJSON string
	ResultJSON     string
	ErrorText      string
}
