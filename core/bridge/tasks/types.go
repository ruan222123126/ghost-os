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

	DefaultRunLogRetention   = 100
	RunStatusRunning         = "running"
	RunStatusSuccess         = "success"
	RunStatusIncomplete      = "incomplete"
	RunStatusCancelled       = "cancelled"
	RunStatusError           = "error"
	RunStatusSkipped         = "skipped"
	RunStatusAwaitingHuman   = "awaiting_human"
	MaxResponsePreviewRunes  = 240
	KindAgentMessage         = "agent_message"
	KindSystemAction         = "system_action"
	KindWorkflow             = "workflow"
	KindOrchestration        = "orchestration"
	LoadIssueInvalidFilename = "invalid_filename"
	LoadIssueReadError       = "read_error"
	LoadIssueDecodeError     = "decode_error"
	LoadIssueInvalidConfig   = "invalid_config"
	LoadIssueIDMismatch      = "id_mismatch"
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

type TaskRuntimeOverrides struct {
	ProviderName      string   `json:"provider_name,omitempty"`
	Model             string   `json:"model,omitempty"`
	SystemPrompt      string   `json:"system_prompt,omitempty"`
	PresetID          string   `json:"preset_id,omitempty"`
	ToolAllowlist     []string `json:"tool_allowlist,omitempty"`
	ToolAllowlistOnly *bool    `json:"tool_allowlist_only,omitempty"`
	MaxTurns          *int     `json:"max_turns,omitempty"`
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

type RunLog struct {
	TaskID          string          `json:"task_id"`
	RunID           string          `json:"run_id"`
	TraceID         string          `json:"trace_id"`
	TaskKind        string          `json:"task_kind,omitempty"`
	Action          string          `json:"action,omitempty"`
	ScheduledAt     time.Time       `json:"scheduled_at"`
	StartedAt       time.Time       `json:"started_at,omitempty"`
	FinishedAt      time.Time       `json:"finished_at,omitempty"`
	Status          string          `json:"status"`
	SessionIDInput  string          `json:"session_id_input,omitempty"`
	SessionIDOutput string          `json:"session_id_output,omitempty"`
	ResponsePreview string          `json:"response_preview,omitempty"`
	NodeResults     []RunNodeResult `json:"node_results,omitempty"`
	Error           string          `json:"error,omitempty"`
}

type LoadIssue struct {
	Kind   string `json:"kind,omitempty"`
	TaskID string `json:"task_id,omitempty"`
	Path   string `json:"path,omitempty"`
	Error  string `json:"error"`
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

func CloneTaskRuntimeOverrides(input *TaskRuntimeOverrides) *TaskRuntimeOverrides {
	if input == nil {
		return nil
	}
	return &TaskRuntimeOverrides{
		ProviderName:      strings.TrimSpace(input.ProviderName),
		Model:             strings.TrimSpace(input.Model),
		SystemPrompt:      strings.TrimSpace(input.SystemPrompt),
		PresetID:          strings.TrimSpace(input.PresetID),
		ToolAllowlist:     append([]string(nil), input.ToolAllowlist...),
		ToolAllowlistOnly: cloneOptionalBoolPointer(input.ToolAllowlistOnly),
		MaxTurns:          cloneOptionalIntPointer(input.MaxTurns),
	}
}

func cloneOptionalBoolPointer(input *bool) *bool {
	if input == nil {
		return nil
	}
	value := *input
	return &value
}

func cloneOptionalIntPointer(input *int) *int {
	if input == nil {
		return nil
	}
	value := *input
	return &value
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
	run.NodeResults = CloneRunNodeResults(run.NodeResults)
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

func CloneRunNodeResults(input []RunNodeResult) []RunNodeResult {
	if len(input) == 0 {
		return nil
	}
	out := make([]RunNodeResult, len(input))
	for index, item := range input {
		out[index] = normalizeRunNodeResult(item)
	}
	return out
}

func normalizeRunNodeResult(input RunNodeResult) RunNodeResult {
	out := RunNodeResult{
		NodeID:       strings.TrimSpace(input.NodeID),
		NodeType:     strings.TrimSpace(input.NodeType),
		Status:       strings.TrimSpace(input.Status),
		StartedAt:    input.StartedAt,
		FinishedAt:   input.FinishedAt,
		CompletedSeq: input.CompletedSeq,
		BranchID:     strings.TrimSpace(input.BranchID),
		Input:        cloneJSONValue(input.Input),
		Output:       cloneJSONValue(input.Output),
		Preview:      strings.TrimSpace(input.Preview),
		Error:        strings.TrimSpace(input.Error),
	}
	if !out.StartedAt.IsZero() {
		out.StartedAt = out.StartedAt.UTC()
	}
	if !out.FinishedAt.IsZero() {
		out.FinishedAt = out.FinishedAt.UTC()
	}
	return out
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
