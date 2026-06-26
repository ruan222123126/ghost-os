package api

import "ghost-os/bridge/taskdefs"

type AssistantSessionEndSignalPayload struct {
	Signal  string `json:"signal"`
	Message string `json:"message"`
}

type AgentRequest struct {
	Mode             string                         `json:"mode,omitempty"`
	Message          string                         `json:"message,omitempty"`
	Images           []SessionImageContent          `json:"images,omitempty"`
	SessionID        string                         `json:"session_id,omitempty"`
	ProjectRoot      string                         `json:"project_root,omitempty"`
	RuntimeOverrides *taskdefs.TaskRuntimeOverrides `json:"runtime_overrides,omitempty"`
	TraceID          string                         `json:"trace_id,omitempty"`
}

type AgentParams struct {
	Mode             string                         `json:"mode,omitempty"`
	Message          string                         `json:"message,omitempty"`
	Images           []SessionImageContent          `json:"images,omitempty"`
	SessionID        string                         `json:"session_id,omitempty"`
	ProjectRoot      string                         `json:"project_root,omitempty"`
	RuntimeOverrides *taskdefs.TaskRuntimeOverrides `json:"runtime_overrides,omitempty"`
}

type AskHumanOption struct {
	Label       string `json:"label"`
	AllowCustom bool   `json:"allow_custom,omitempty"`
}

type AgentResponse struct {
	Message      string                            `json:"message"`
	SessionID    string                            `json:"session_id"`
	SessionEnded bool                              `json:"session_ended"`
	Mode         string                            `json:"mode,omitempty"`
	SessionEnd   *AssistantSessionEndSignalPayload `json:"session_end,omitempty"`
}

type AskHumanAwaitingResponse struct {
	Status        string           `json:"status"`
	SessionID     string           `json:"session_id"`
	QuestionID    string           `json:"question_id"`
	Prompt        string           `json:"prompt"`
	SelectionMode string           `json:"selection_mode,omitempty"`
	Options       []AskHumanOption `json:"options,omitempty"`
}

type AgentStopParams struct {
	SessionID string `json:"session_id,omitempty"`
	TraceID   string `json:"trace_id,omitempty"`
}

type AgentStopResponse struct {
	Status    string `json:"status"`
	Message   string `json:"message"`
	SessionID string `json:"session_id,omitempty"`
}

type HumanResponseParams struct {
	SessionID  string `json:"session_id"`
	QuestionID string `json:"question_id"`
	Answer     string `json:"answer"`
	Cancelled  bool   `json:"cancelled,omitempty"`
}

type HumanResponseAck struct {
	SessionID  string `json:"session_id"`
	QuestionID string `json:"question_id"`
	Accepted   bool   `json:"accepted"`
}
