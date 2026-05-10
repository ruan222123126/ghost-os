package api

type AgentStreamEventContract struct {
	ID        string         `json:"id"`
	StepID    string         `json:"step_id"`
	TraceID   string         `json:"trace_id"`
	SessionID string         `json:"session_id,omitempty"`
	Turn      int            `json:"turn"`
	Type      string         `json:"type"`
	Payload   map[string]any `json:"payload"`
	At        string         `json:"at,omitempty"`
}

type SessionPushEventContract struct {
	ID        string         `json:"id"`
	Type      string         `json:"type"`
	TraceID   string         `json:"trace_id,omitempty"`
	SessionID string         `json:"session_id"`
	Payload   map[string]any `json:"payload"`
	At        string         `json:"at,omitempty"`
}

type AgentRunStartedPayload struct {
	SessionID string `json:"session_id,omitempty"`
}

type AgentCompletionDeltaPayload struct {
	Kind              string `json:"kind"`
	Text              string `json:"text,omitempty"`
	Thinking          string `json:"thinking,omitempty"`
	ToolCallIndex     int    `json:"tool_call_index,omitempty"`
	ToolCallID        string `json:"tool_call_id,omitempty"`
	ToolName          string `json:"tool_name,omitempty"`
	ArgumentsFragment string `json:"arguments_fragment,omitempty"`
}

type AgentToolCallStartedPayload struct {
	Tool          string `json:"tool,omitempty"`
	ToolCallID    string `json:"tool_call_id,omitempty"`
	ArgumentsJson string `json:"arguments_json,omitempty"`
}

type AgentToolCallFinishedPayload struct {
	Tool       string `json:"tool,omitempty"`
	ToolCallID string `json:"tool_call_id,omitempty"`
	Status     string `json:"status,omitempty"`
	Error      string `json:"error,omitempty"`
	Output     string `json:"output,omitempty"`
}

type AgentStreamMessagePayload struct {
	Text      string `json:"text"`
	SessionID string `json:"session_id,omitempty"`
}

type AgentDonePayload struct {
	SessionID    string `json:"session_id,omitempty"`
	SessionEnded bool   `json:"session_ended,omitempty"`
}

type AgentErrorPayload struct {
	Message   string `json:"message"`
	SessionID string `json:"session_id,omitempty"`
	Code      int    `json:"code,omitempty"`
}

type AssistantMessagePushPayload struct {
	Message      string                            `json:"message"`
	SessionEnded bool                              `json:"session_ended"`
	SessionEnd   *AssistantSessionEndSignalPayload `json:"session_end,omitempty"`
}

type AwaitingHumanPushPayload struct {
	QuestionID    string           `json:"question_id"`
	Prompt        string           `json:"prompt"`
	SelectionMode string           `json:"selection_mode,omitempty"`
	Options       []AskHumanOption `json:"options,omitempty"`
}
