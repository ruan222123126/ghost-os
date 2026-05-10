package api

import bridgeTasks "ghost-os/bridge/tasks"

type TaskCreateParams struct {
	Name             string                               `json:"name,omitempty"`
	Message          string                               `json:"message"`
	SessionID        string                               `json:"session_id,omitempty"`
	RuntimeOverrides *bridgeTasks.TaskRuntimeOverrides    `json:"runtime_overrides,omitempty"`
	AgentMode        string                               `json:"agent_mode,omitempty"`
	Relay            *bridgeTasks.TaskRelayConfig         `json:"relay,omitempty"`
	TaskKind         string                               `json:"task_kind,omitempty"`
	Action           string                               `json:"action,omitempty"`
	ActionParams     map[string]any                       `json:"action_params,omitempty"`
	Workflow         *bridgeTasks.WorkflowDefinition      `json:"workflow,omitempty"`
	Orchestration    *bridgeTasks.OrchestrationDefinition `json:"orchestration,omitempty"`
	IntervalSeconds  int                                  `json:"interval_seconds,omitempty"`
	CronExpr         string                               `json:"cron_expr,omitempty"`
	TraceID          string                               `json:"trace_id,omitempty"`
	Scope            string                               `json:"scope,omitempty"`
}

type TaskUpdateParams struct {
	ID               string                               `json:"id,omitempty"`
	Name             *string                              `json:"name,omitempty"`
	Message          *string                              `json:"message,omitempty"`
	SessionID        *string                              `json:"session_id,omitempty"`
	RuntimeOverrides *bridgeTasks.TaskRuntimeOverrides    `json:"runtime_overrides,omitempty"`
	AgentMode        *string                              `json:"agent_mode,omitempty"`
	Relay            *bridgeTasks.TaskRelayConfig         `json:"relay,omitempty"`
	TaskKind         *string                              `json:"task_kind,omitempty"`
	Action           *string                              `json:"action,omitempty"`
	ActionParams     *map[string]any                      `json:"action_params,omitempty"`
	Workflow         *bridgeTasks.WorkflowDefinition      `json:"workflow,omitempty"`
	Orchestration    *bridgeTasks.OrchestrationDefinition `json:"orchestration,omitempty"`
	IntervalSeconds  *int                                 `json:"interval_seconds,omitempty"`
	CronExpr         *string                              `json:"cron_expr,omitempty"`
	Enabled          *bool                                `json:"enabled,omitempty"`
	TraceID          string                               `json:"trace_id,omitempty"`
	Scope            string                               `json:"scope,omitempty"`
}

type TaskIDParams struct {
	ID    string `json:"id"`
	Scope string `json:"scope,omitempty"`
}

type TaskLogsParams struct {
	ID    string `json:"id"`
	Limit int    `json:"limit,omitempty"`
	Scope string `json:"scope,omitempty"`
}

type TaskPayload = bridgeTasks.ScheduledTask

type TaskDeleteResponse struct {
	ID      string `json:"id"`
	Deleted bool   `json:"deleted"`
}

type TaskRunPayload struct {
	Task TaskPayload       `json:"task"`
	Run  TaskRunLogPayload `json:"run"`
}

type TaskRunLogPayload = bridgeTasks.RunLog
