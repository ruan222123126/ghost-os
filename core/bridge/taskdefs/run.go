package taskdefs

import "time"

const (
	RunStatusRunning        = "running"
	RunStatusSuccess        = "success"
	RunStatusIncomplete     = "incomplete"
	RunStatusCancelled      = "cancelled"
	RunStatusError          = "error"
	RunStatusSkipped        = "skipped"
	RunStatusAwaitingHuman  = "awaiting_human"
	MaxResponsePreviewRunes = 240

	RunCardKindAgentTask           = "agent_task"
	RunCardKindWorkflowAgent       = "workflow_agent"
	RunCardKindWorkflowLLM         = "workflow_llm"
	RunCardKindOrchestrationOwner  = "orchestration_owner"
	RunCardKindOrchestrationMember = "orchestration_member"
	RunCardKindRelayRound          = "relay_round"
)

type ExecutionResult struct {
	Status          string
	SessionIDOutput string
	ResponsePreview string
	NodeResults     []RunNodeResult
	RunCards        []RunCard
	Error           string
}

type RunNodeResult struct {
	NodeID       string    `json:"node_id"`
	NodeType     string    `json:"node_type"`
	Status       string    `json:"status"`
	StartedAt    time.Time `json:"started_at,omitempty"`
	FinishedAt   time.Time `json:"finished_at,omitempty"`
	CompletedSeq int       `json:"completed_seq,omitempty"`
	BranchID     string    `json:"branch_id,omitempty"`
	Input        any       `json:"input,omitempty"`
	Output       any       `json:"output,omitempty"`
	Preview      string    `json:"preview,omitempty"`
	Error        string    `json:"error,omitempty"`
}

type RunCard struct {
	CardID          string               `json:"card_id"`
	RunID           string               `json:"run_id,omitempty"`
	Kind            string               `json:"kind"`
	Title           string               `json:"title,omitempty"`
	NodeID          string               `json:"node_id,omitempty"`
	NodeType        string               `json:"node_type,omitempty"`
	Round           int                  `json:"round,omitempty"`
	Iteration       int                  `json:"iteration,omitempty"`
	BranchID        string               `json:"branch_id,omitempty"`
	SourceSessionID string               `json:"source_session_id,omitempty"`
	StartedAt       time.Time            `json:"started_at,omitempty"`
	Status          string               `json:"status,omitempty"`
	FinishedAt      time.Time            `json:"finished_at,omitempty"`
	Preview         string               `json:"preview,omitempty"`
	Error           string               `json:"error,omitempty"`
	FinalText       string               `json:"final_text,omitempty"`
	SourceEvents    []RunCardSourceEvent `json:"source_events,omitempty"`
}

type RunCardSourceEvent struct {
	ID        string         `json:"id"`
	StepID    string         `json:"step_id"`
	TraceID   string         `json:"trace_id"`
	SessionID string         `json:"session_id,omitempty"`
	Turn      int            `json:"turn"`
	Type      string         `json:"type"`
	Payload   map[string]any `json:"payload"`
	At        time.Time      `json:"at,omitempty"`
}
