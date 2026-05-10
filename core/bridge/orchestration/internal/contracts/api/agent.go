package api

type AssistantSessionEndSignalPayload struct {
	Signal  string `json:"signal"`
	Message string `json:"message"`
}

type AgentRequest struct {
	Mode        string                `json:"mode,omitempty"`
	Message     string                `json:"message,omitempty"`
	Images      []SessionImageContent `json:"images,omitempty"`
	SessionID   string                `json:"session_id,omitempty"`
	ProjectRoot string                `json:"project_root,omitempty"`
	TraceID     string                `json:"trace_id,omitempty"`
}

type AgentParams struct {
	Mode        string                `json:"mode,omitempty"`
	Message     string                `json:"message,omitempty"`
	Images      []SessionImageContent `json:"images,omitempty"`
	SessionID   string                `json:"session_id,omitempty"`
	ProjectRoot string                `json:"project_root,omitempty"`
}

type AgentIterationSummaryItem struct {
	Iteration      int    `json:"iteration"`
	Did            string `json:"did"`
	Remaining      string `json:"remaining"`
	Completed      bool   `json:"completed,omitempty"`
	TraceID        string `json:"trace_id,omitempty"`
	RecordedAt     string `json:"recorded_at,omitempty"`
	FinalChangeLog string `json:"final_change_log,omitempty"`
}

type AskHumanOption struct {
	Label       string `json:"label"`
	AllowCustom bool   `json:"allow_custom,omitempty"`
}

type AgentResponse struct {
	Message          string                            `json:"message"`
	SessionID        string                            `json:"session_id"`
	SessionEnded     bool                              `json:"session_ended"`
	Mode             string                            `json:"mode,omitempty"`
	IterationCount   int                               `json:"iteration_count,omitempty"`
	StoppedBy        string                            `json:"stopped_by,omitempty"`
	FinalChangeLog   string                            `json:"final_change_log,omitempty"`
	IterationSummary []AgentIterationSummaryItem       `json:"iteration_summary,omitempty"`
	SessionEnd       *AssistantSessionEndSignalPayload `json:"session_end,omitempty"`
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
