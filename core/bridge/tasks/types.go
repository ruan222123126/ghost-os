package tasks

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync/atomic"
	"time"
)

const (
	ScheduleTypeInterval = "interval"
	ScheduleTypeCron     = "cron"

	DefaultRunLogRetention  = 100
	RunStatusSuccess        = "success"
	RunStatusCancelled      = "cancelled"
	RunStatusError          = "error"
	RunStatusSkipped        = "skipped"
	RunStatusAwaitingHuman  = "awaiting_human"
	MaxResponsePreviewRunes = 240
	KindAgentMessage        = "agent_message"
	KindSystemAction        = "system_action"
	KindWorkflow            = "workflow"
	LoadIssueReadError      = "read_error"
	LoadIssueDecodeError    = "decode_error"
	LoadIssueInvalidConfig  = "invalid_config"
	LoadIssueIDMismatch     = "id_mismatch"
)

var (
	ErrTaskNotFound      = errors.New("task not found")
	ErrTaskCorrupted     = errors.New("task corrupted")
	ErrInvalidTaskID     = errors.New("invalid task id")
	ErrInvalidTaskConfig = errors.New("invalid task config")

	taskIDPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,127}$`)
	taskIDCounter uint64
	runIDCounter  uint64
)

type ScheduledTask struct {
	ID              string              `json:"id"`
	Message         string              `json:"message,omitempty"`
	SessionID       string              `json:"session_id,omitempty"`
	TaskKind        string              `json:"task_kind,omitempty"`
	Action          string              `json:"action,omitempty"`
	ActionParams    map[string]any      `json:"action_params,omitempty"`
	Workflow        *WorkflowDefinition `json:"workflow,omitempty"`
	ScheduleType    string              `json:"schedule_type"`
	IntervalSeconds int                 `json:"interval_seconds,omitempty"`
	CronExpr        string              `json:"cron_expr,omitempty"`
	Enabled         bool                `json:"enabled"`
	CreatedAt       time.Time           `json:"created_at"`
	UpdatedAt       time.Time           `json:"updated_at"`
	LastRunAt       time.Time           `json:"last_run_at,omitempty"`
	NextRunAt       time.Time           `json:"next_run_at,omitempty"`
	LastError       string              `json:"last_error,omitempty"`
}

type RunLog struct {
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

type LoadIssue struct {
	Kind   string `json:"kind,omitempty"`
	TaskID string `json:"task_id,omitempty"`
	Path   string `json:"path,omitempty"`
	Error  string `json:"error"`
}

type DefinitionValidator func(*ScheduledTask) error

func NormalizeKind(kind string) string {
	switch strings.TrimSpace(kind) {
	case KindSystemAction:
		return KindSystemAction
	case KindWorkflow:
		return KindWorkflow
	default:
		return KindAgentMessage
	}
}

func CloneActionParams(input map[string]any) map[string]any {
	if len(input) == 0 {
		return nil
	}
	out := make(map[string]any, len(input))
	for key, value := range input {
		out[key] = cloneJSONValue(value)
	}
	return out
}

func DecodeParamsMap[T any](input map[string]any) (T, error) {
	var out T
	if len(input) == 0 {
		return out, nil
	}
	data, err := json.Marshal(input)
	if err != nil {
		return out, err
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return out, err
	}
	return out, nil
}

func NormalizeScheduledTask(task *ScheduledTask, validator DefinitionValidator) error {
	if task == nil {
		return errors.New("task is nil")
	}
	task.ID = strings.TrimSpace(task.ID)
	task.Message = strings.TrimSpace(task.Message)
	task.SessionID = strings.TrimSpace(task.SessionID)
	task.TaskKind = NormalizeKind(task.TaskKind)
	task.Action = strings.TrimSpace(task.Action)
	task.ActionParams = CloneActionParams(task.ActionParams)
	task.Workflow = CloneWorkflowDefinition(task.Workflow)
	task.ScheduleType = strings.TrimSpace(task.ScheduleType)
	task.CronExpr = strings.TrimSpace(task.CronExpr)
	task.LastError = strings.TrimSpace(task.LastError)
	if task.ID == "" {
		task.ID = newTaskID()
	}
	if !IsValidTaskID(task.ID) {
		return fmt.Errorf("%w: %q", ErrInvalidTaskID, task.ID)
	}
	if validator != nil {
		if err := validator(task); err != nil {
			return err
		}
	}
	if task.CreatedAt.IsZero() {
		task.CreatedAt = time.Now().UTC()
	} else {
		task.CreatedAt = task.CreatedAt.UTC()
	}
	if !task.LastRunAt.IsZero() {
		task.LastRunAt = task.LastRunAt.UTC()
	}
	if !task.NextRunAt.IsZero() {
		task.NextRunAt = task.NextRunAt.UTC()
	}
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

func NormalizeRunLog(run *RunLog) {
	if run == nil {
		return
	}
	run.TaskID = strings.TrimSpace(run.TaskID)
	run.RunID = strings.TrimSpace(run.RunID)
	run.TraceID = strings.TrimSpace(run.TraceID)
	run.TaskKind = NormalizeKind(run.TaskKind)
	run.Action = strings.TrimSpace(run.Action)
	run.Status = strings.TrimSpace(run.Status)
	run.SessionIDInput = strings.TrimSpace(run.SessionIDInput)
	run.SessionIDOutput = strings.TrimSpace(run.SessionIDOutput)
	run.ResponsePreview = truncateRunes(strings.TrimSpace(run.ResponsePreview), MaxResponsePreviewRunes)
	run.Error = strings.TrimSpace(run.Error)
	if !run.ScheduledAt.IsZero() {
		run.ScheduledAt = run.ScheduledAt.UTC()
	}
	if !run.StartedAt.IsZero() {
		run.StartedAt = run.StartedAt.UTC()
	}
	if !run.FinishedAt.IsZero() {
		run.FinishedAt = run.FinishedAt.UTC()
	}
}

func RunLogSortTime(run RunLog) time.Time {
	if !run.StartedAt.IsZero() {
		return run.StartedAt
	}
	return run.ScheduledAt
}

func IsValidTaskID(taskID string) bool {
	return taskIDPattern.MatchString(strings.TrimSpace(taskID))
}

func NewRunID() string {
	return fmt.Sprintf("run-%d-%d", time.Now().UnixMilli(), atomic.AddUint64(&runIDCounter, 1))
}

func newTaskID() string {
	return fmt.Sprintf("task-%d-%d", time.Now().UnixMilli(), atomic.AddUint64(&taskIDCounter, 1))
}

func cloneJSONValue(input any) any {
	switch typed := input.(type) {
	case map[string]any:
		return CloneActionParams(typed)
	case []any:
		out := make([]any, len(typed))
		for i := range typed {
			out[i] = cloneJSONValue(typed[i])
		}
		return out
	default:
		return typed
	}
}

func truncateRunes(value string, limit int) string {
	if limit <= 0 {
		return ""
	}
	runes := []rune(strings.TrimSpace(value))
	if len(runes) <= limit {
		return string(runes)
	}
	return string(runes[:limit])
}
