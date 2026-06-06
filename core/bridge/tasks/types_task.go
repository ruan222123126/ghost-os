package tasks

import (
	"errors"
	"fmt"
	"ghost-os/bridge/taskdefs"
	"regexp"
	"strings"
	"sync/atomic"
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

var (
	ErrTaskNotFound      = errors.New("task not found")
	ErrTaskCorrupted     = errors.New("task corrupted")
	ErrInvalidTaskID     = errors.New("invalid task id")
	ErrInvalidTaskConfig = taskdefs.ErrInvalidTaskConfig

	taskIDPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,127}$`)
	taskIDCounter uint64
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

type DefinitionValidator func(*ScheduledTask) error

func NormalizeKind(kind string) string {
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

func IsSupportedKind(kind string) bool {
	switch NormalizeKind(kind) {
	case KindAgentMessage, KindSystemAction, KindWorkflow, KindOrchestration:
		return true
	default:
		return false
	}
}

func NormalizeScheduledTask(task *ScheduledTask, validator DefinitionValidator) error {
	if task == nil {
		return errors.New("task is nil")
	}
	normalizeScheduledTaskScalarFields(task)
	if err := ensureScheduledTaskID(task); err != nil {
		return err
	}
	if validator != nil {
		if err := validator(task); err != nil {
			return err
		}
	}
	normalizeScheduledTaskTimestamps(task, time.Now().UTC())
	return validateScheduledTaskSchedule(task)
}

func normalizeScheduledTaskScalarFields(task *ScheduledTask) {
	task.ID = strings.TrimSpace(task.ID)
	task.Name = strings.TrimSpace(task.Name)
	task.Message = strings.TrimSpace(task.Message)
	task.SessionID = strings.TrimSpace(task.SessionID)
	task.RuntimeOverrides = CloneTaskRuntimeOverrides(task.RuntimeOverrides)
	task.AgentMode = NormalizeAgentMode(task.AgentMode)
	task.Relay = CloneTaskRelayConfig(task.Relay)
	task.TaskKind = NormalizeKind(task.TaskKind)
	task.Action = strings.TrimSpace(task.Action)
	task.ActionParams = CloneActionParams(task.ActionParams)
	task.Workflow = CloneWorkflowDefinition(task.Workflow)
	task.Orchestration = CloneOrchestrationDefinition(task.Orchestration)
	task.ScheduleType = strings.TrimSpace(task.ScheduleType)
	task.CronExpr = strings.TrimSpace(task.CronExpr)
	task.LastError = strings.TrimSpace(task.LastError)
}

func ensureScheduledTaskID(task *ScheduledTask) error {
	if task.ID == "" {
		task.ID = newTaskID()
	}
	if !IsValidTaskID(task.ID) {
		return fmt.Errorf("%w: %q", ErrInvalidTaskID, task.ID)
	}
	return nil
}

func normalizeScheduledTaskTimestamps(task *ScheduledTask, now time.Time) {
	if task.CreatedAt.IsZero() {
		task.CreatedAt = now
	} else {
		task.CreatedAt = task.CreatedAt.UTC()
	}
	if !task.LastRunAt.IsZero() {
		task.LastRunAt = task.LastRunAt.UTC()
	}
	if !task.NextRunAt.IsZero() {
		task.NextRunAt = task.NextRunAt.UTC()
	}
}

func validateScheduledTaskSchedule(task *ScheduledTask) error {
	switch task.ScheduleType {
	case ScheduleTypeInterval:
		if task.IntervalSeconds <= 0 || task.CronExpr != "" {
			return fmt.Errorf("%w: interval task requires interval_seconds only", ErrInvalidTaskConfig)
		}
	case ScheduleTypeCron:
		if task.IntervalSeconds != 0 || task.CronExpr == "" {
			return fmt.Errorf("%w: cron task requires cron_expr only", ErrInvalidTaskConfig)
		}
	default:
		return fmt.Errorf("%w: schedule_type must be interval or cron", ErrInvalidTaskConfig)
	}
	return nil
}

func IsValidTaskID(taskID string) bool {
	return taskIDPattern.MatchString(strings.TrimSpace(taskID))
}

func newTaskID() string {
	return fmt.Sprintf("task-%d-%d", time.Now().UnixMilli(), atomic.AddUint64(&taskIDCounter, 1))
}
