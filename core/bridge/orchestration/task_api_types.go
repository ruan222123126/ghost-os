package orchestration

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

type taskPayload = ScheduledTask

type taskDeleteResponse struct {
	ID      string `json:"id"`
	Deleted bool   `json:"deleted"`
}

type taskRunPayload struct {
	Task taskPayload       `json:"task"`
	Run  taskRunLogPayload `json:"run"`
}

type taskRunLogPayload = TaskRunLog
