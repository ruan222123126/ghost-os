package taskdefs

import (
	"strings"
	"time"
)

const (
	ScheduleTypeInterval = "interval"
	ScheduleTypeCron     = "cron"

	KindAgentMessage  = "agent_message"
	KindSystemAction  = "system_action"
	KindWorkflow      = "workflow"
	KindOrchestration = "orchestration"
)

type ScheduledTask struct {
	ID               string                   `json:"id"`
	Name             string                   `json:"name,omitempty"`
	Message          string                   `json:"message,omitempty"`
	SessionID        string                   `json:"session_id,omitempty"`
	RuntimeOverrides *TaskRuntimeOverrides    `json:"runtime_overrides,omitempty"`
	AgentMode        string                   `json:"agent_mode,omitempty"`
	Relay            *TaskRelayConfig         `json:"relay,omitempty"`
	TaskKind         string                   `json:"task_kind,omitempty"`
	Action           string                   `json:"action,omitempty"`
	ActionParams     map[string]any           `json:"action_params,omitempty"`
	Workflow         *WorkflowDefinition      `json:"workflow,omitempty"`
	Orchestration    *OrchestrationDefinition `json:"orchestration,omitempty"`
	ScheduleType     string                   `json:"schedule_type"`
	IntervalSeconds  int                      `json:"interval_seconds,omitempty"`
	CronExpr         string                   `json:"cron_expr,omitempty"`
	Enabled          bool                     `json:"enabled"`
	CreatedAt        time.Time                `json:"created_at"`
	UpdatedAt        time.Time                `json:"updated_at"`
	LastRunAt        time.Time                `json:"last_run_at,omitempty"`
	NextRunAt        time.Time                `json:"next_run_at,omitempty"`
	LastError        string                   `json:"last_error,omitempty"`
}

func NormalizeTaskKind(kind string) string {
	normalized := strings.TrimSpace(kind)
	switch normalized {
	case "", KindAgentMessage:
		return KindAgentMessage
	case KindSystemAction:
		return KindSystemAction
	case KindWorkflow:
		return KindWorkflow
	case KindOrchestration:
		return KindOrchestration
	default:
		return normalized
	}
}

func IsSupportedTaskKind(kind string) bool {
	switch NormalizeTaskKind(kind) {
	case KindAgentMessage, KindSystemAction, KindWorkflow, KindOrchestration:
		return true
	default:
		return false
	}
}
