package orchestration

import "time"

type taskCreateParams struct {
	Message          string                `json:"message"`
	SessionID        string                `json:"session_id,omitempty"`
	RuntimeOverrides *TaskRuntimeOverrides `json:"runtime_overrides,omitempty"`
	TaskKind         string                `json:"task_kind,omitempty"`
	Action           string                `json:"action,omitempty"`
	ActionParams     map[string]any        `json:"action_params,omitempty"`
	Workflow         *WorkflowDefinition   `json:"workflow,omitempty"`
	IntervalSeconds  int                   `json:"interval_seconds,omitempty"`
	CronExpr         string                `json:"cron_expr,omitempty"`
	TraceID          string                `json:"trace_id,omitempty"`
}

type taskUpdateParams struct {
	ID               string                `json:"id,omitempty"`
	Message          *string               `json:"message,omitempty"`
	SessionID        *string               `json:"session_id,omitempty"`
	RuntimeOverrides *TaskRuntimeOverrides `json:"runtime_overrides,omitempty"`
	TaskKind         *string               `json:"task_kind,omitempty"`
	Action           *string               `json:"action,omitempty"`
	ActionParams     *map[string]any       `json:"action_params,omitempty"`
	Workflow         *WorkflowDefinition   `json:"workflow,omitempty"`
	IntervalSeconds  *int                  `json:"interval_seconds,omitempty"`
	CronExpr         *string               `json:"cron_expr,omitempty"`
	Enabled          *bool                 `json:"enabled,omitempty"`
	TraceID          string                `json:"trace_id,omitempty"`
}

type taskIDParams struct {
	ID string `json:"id"`
}

type taskLogsParams struct {
	ID    string `json:"id"`
	Limit int    `json:"limit,omitempty"`
}

type taskPayload struct {
	ID               string                `json:"id"`
	Message          string                `json:"message,omitempty"`
	SessionID        string                `json:"session_id,omitempty"`
	RuntimeOverrides *TaskRuntimeOverrides `json:"runtime_overrides,omitempty"`
	TaskKind         string                `json:"task_kind"`
	Action           string                `json:"action,omitempty"`
	ActionParams     map[string]any        `json:"action_params,omitempty"`
	Workflow         *WorkflowDefinition   `json:"workflow,omitempty"`
	ScheduleType     string                `json:"schedule_type"`
	IntervalSeconds  int                   `json:"interval_seconds,omitempty"`
	CronExpr         string                `json:"cron_expr,omitempty"`
	Enabled          bool                  `json:"enabled"`
	CreatedAt        time.Time             `json:"created_at"`
	UpdatedAt        time.Time             `json:"updated_at"`
	LastRunAt        time.Time             `json:"last_run_at,omitempty"`
	NextRunAt        time.Time             `json:"next_run_at,omitempty"`
	LastError        string                `json:"last_error,omitempty"`
}

type taskDeleteResponse struct {
	ID      string `json:"id"`
	Deleted bool   `json:"deleted"`
}

type taskRunPayload struct {
	Task taskPayload       `json:"task"`
	Run  taskRunLogPayload `json:"run"`
}

type taskRunLogPayload struct {
	TaskID          string    `json:"task_id"`
	RunID           string    `json:"run_id"`
	TraceID         string    `json:"trace_id"`
	TaskKind        string    `json:"task_kind,omitempty"`
	Action          string    `json:"action,omitempty"`
	ScheduledAt     time.Time `json:"scheduled_at"`
	StartedAt       time.Time `json:"started_at,omitempty"`
	FinishedAt      time.Time `json:"finished_at,omitempty"`
	Status          string    `json:"status"`
	SessionIDInput  string    `json:"session_id_input,omitempty"`
	SessionIDOutput string    `json:"session_id_output,omitempty"`
	ResponsePreview string    `json:"response_preview,omitempty"`
	Error           string    `json:"error,omitempty"`
}
