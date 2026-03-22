package memorystore

import "time"

const (
	EventStatusActive   = "active"
	EventStatusArchived = "archived"

	EventEdgeParentOf  = "parent_of"
	EventEdgeBlocks    = "blocks"
	EventEdgeRelatedTo = "related_to"
	EventEdgeSameGoal  = "same_goal"
)

type EventNode struct {
	ID              string    `json:"id"`
	SessionID       string    `json:"session_id"`
	Title           string    `json:"title"`
	Summary         string    `json:"summary"`
	Status          string    `json:"status"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	LastActivatedAt time.Time `json:"last_activated_at,omitempty"`
}

type EventNodeInput struct {
	SessionID string
	Title     string
	Summary   string
	Status    string
}

type EventNodeFilter struct {
	SessionID string
	Statuses  []string
	Query     string
	Limit     int
}

type EventEdge struct {
	SessionID   string    `json:"session_id"`
	FromEventID string    `json:"from_event_id"`
	ToEventID   string    `json:"to_event_id"`
	EdgeType    string    `json:"edge_type"`
	Confidence  float64   `json:"confidence"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type EventEdgeInput struct {
	SessionID   string
	FromEventID string
	ToEventID   string
	EdgeType    string
	Confidence  float64
}

type EventMemory struct {
	ID         string         `json:"id"`
	EventID    string         `json:"event_id"`
	SessionID  string         `json:"session_id"`
	MemoryType string         `json:"memory_type"`
	MemoryKey  string         `json:"memory_key,omitempty"`
	Content    string         `json:"content"`
	Summary    string         `json:"summary"`
	Metadata   map[string]any `json:"metadata_json,omitempty"`
	Confidence float64        `json:"confidence"`
	Status     string         `json:"status"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	LastUsedAt time.Time      `json:"last_used_at,omitempty"`
}

type EventMemoryInput struct {
	EventID    string
	MemoryType string
	MemoryKey  string
	Content    string
	Summary    string
	Metadata   map[string]any
	Confidence float64
}

type EventMemoryListFilter struct {
	EventID    string
	SessionID  string
	MemoryType string
	Statuses   []string
	Query      string
	Limit      int
	Offset     int
}

type SessionEventState struct {
	SessionID       string         `json:"session_id"`
	PrimaryEventID  string         `json:"primary_event_id,omitempty"`
	ActiveEventIDs  []string       `json:"active_event_ids,omitempty"`
	PlannerSnapshot map[string]any `json:"planner_snapshot,omitempty"`
	UpdatedAt       time.Time      `json:"updated_at"`
}
