package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
)

const (
	taskManageOperationCreate = "create"
	taskManageOperationUpdate = "update"
	taskManageOperationDelete = "delete"
	taskManageOperationList   = "list"
	taskManageOperationGet    = "get"

	taskManageKindAgentMessage = "agent_message"
)

// TaskPayload 是 task_manage 对外返回的稳定任务结构。
type TaskPayload struct {
	ID              string         `json:"id"`
	Message         string         `json:"message"`
	SessionID       string         `json:"session_id,omitempty"`
	TaskKind        string         `json:"task_kind,omitempty"`
	Action          string         `json:"action,omitempty"`
	ActionParams    map[string]any `json:"action_params,omitempty"`
	ScheduleType    string         `json:"schedule_type"`
	IntervalSeconds int            `json:"interval_seconds,omitempty"`
	CronExpr        string         `json:"cron_expr,omitempty"`
	Enabled         bool           `json:"enabled"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	LastRunAt       time.Time      `json:"last_run_at,omitempty"`
	NextRunAt       time.Time      `json:"next_run_at,omitempty"`
	LastError       string         `json:"last_error,omitempty"`
}

// TaskDeleteResult 对齐删除任务的最小返回结构。
type TaskDeleteResult struct {
	ID      string `json:"id"`
	Deleted bool   `json:"deleted"`
}

// TaskCreateRequest 描述创建普通消息任务所需的最小字段。
type TaskCreateRequest struct {
	Message         string `json:"message"`
	SessionID       string `json:"session_id,omitempty"`
	IntervalSeconds int    `json:"interval_seconds,omitempty"`
	CronExpr        string `json:"cron_expr,omitempty"`
}

// TaskUpdateRequest 描述更新普通消息任务的可变字段。
type TaskUpdateRequest struct {
	ID              string  `json:"id"`
	Message         *string `json:"message,omitempty"`
	SessionID       *string `json:"session_id,omitempty"`
	IntervalSeconds *int    `json:"interval_seconds,omitempty"`
	CronExpr        *string `json:"cron_expr,omitempty"`
	Enabled         *bool   `json:"enabled,omitempty"`
}

// TaskManager 定义 task_manage 工具依赖的最小 Bridge 能力。
type TaskManager interface {
	CreateAgentTask(context.Context, TaskCreateRequest, string) (TaskPayload, error)
	UpdateAgentTask(context.Context, TaskUpdateRequest, string) (TaskPayload, error)
	GetTask(context.Context, string, string) (TaskPayload, error)
	ListTasks(context.Context, string) ([]TaskPayload, error)
	DeleteTask(context.Context, string, string) (TaskDeleteResult, error)
}

type TaskManageTool struct {
	manager TaskManager
}

type taskManageArgs struct {
	Operation       string  `json:"operation"`
	ID              string  `json:"id,omitempty"`
	Message         *string `json:"message,omitempty"`
	SessionID       *string `json:"session_id,omitempty"`
	IntervalSeconds *int    `json:"interval_seconds,omitempty"`
	CronExpr        *string `json:"cron_expr,omitempty"`
	Enabled         *bool   `json:"enabled,omitempty"`
}

// NewTaskManageTool 创建统一的任务管理工具。
func NewTaskManageTool(manager TaskManager) Tool {
	return &TaskManageTool{manager: manager}
}

func (TaskManageTool) Name() string {
	return "task_manage"
}

func (TaskManageTool) Description() string {
	return "Manage scheduled agent_message tasks: create, update, list, get, or delete. session_id is optional; when omitted, runs start a new conversation."
}

func (TaskManageTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type":"object",
		"properties":{
			"operation":{"type":"string","enum":["create","update","delete","list","get"]},
			"id":{"type":"string","description":"Task ID for update, delete, or get."},
			"message":{"type":"string","description":"Agent message to send when the task runs."},
			"session_id":{"type":"string","description":"Optional target session ID. Omit to let each run open a new conversation."},
			"interval_seconds":{"type":"integer","minimum":1,"description":"Repeat every N seconds. Mutually exclusive with cron_expr."},
			"cron_expr":{"type":"string","description":"Cron expression. Mutually exclusive with interval_seconds."},
			"enabled":{"type":"boolean","description":"Whether the task remains scheduled after update."}
		},
		"required":["operation"],
		"additionalProperties":false
	}`)
}

func (t *TaskManageTool) Execute(ctx context.Context, argsJSON json.RawMessage, traceID string) (string, error) {
	if t == nil || t.manager == nil {
		return "", fmt.Errorf("task manager is not configured")
	}

	var args taskManageArgs
	if err := decodeTaskManageArgs(argsJSON, &args); err != nil {
		return "", err
	}

	switch strings.ToLower(strings.TrimSpace(args.Operation)) {
	case taskManageOperationCreate:
		return t.executeCreate(ctx, args, traceID)
	case taskManageOperationUpdate:
		return t.executeUpdate(ctx, args, traceID)
	case taskManageOperationDelete:
		return t.executeDelete(ctx, args, traceID)
	case taskManageOperationList:
		return t.executeList(ctx, traceID)
	case taskManageOperationGet:
		return t.executeGet(ctx, args, traceID)
	default:
		return "", fmt.Errorf("unsupported operation %q", strings.TrimSpace(args.Operation))
	}
}

func (t *TaskManageTool) executeCreate(ctx context.Context, args taskManageArgs, traceID string) (string, error) {
	message := trimmedOptionalString(args.Message)
	if message == nil || *message == "" {
		return "", fmt.Errorf("message is required for create")
	}
	if err := validateTaskManageSchedule(args.IntervalSeconds, args.CronExpr, true); err != nil {
		return "", err
	}

	payload, err := t.manager.CreateAgentTask(ctx, TaskCreateRequest{
		Message:         *message,
		SessionID:       optionalStringValue(args.SessionID),
		IntervalSeconds: optionalIntValue(args.IntervalSeconds),
		CronExpr:        optionalStringValue(args.CronExpr),
	}, traceID)
	if err != nil {
		return "", err
	}
	if err := ensureAgentTaskPayload(payload); err != nil {
		return "", err
	}
	return marshalTaskManageOutput(payload)
}

func (t *TaskManageTool) executeUpdate(ctx context.Context, args taskManageArgs, traceID string) (string, error) {
	id := strings.TrimSpace(args.ID)
	if id == "" {
		return "", fmt.Errorf("id is required for update")
	}
	if err := validateTaskManageSchedule(args.IntervalSeconds, args.CronExpr, false); err != nil {
		return "", err
	}
	current, err := t.manager.GetTask(ctx, id, traceID)
	if err != nil {
		return "", err
	}
	if err := ensureAgentTaskPayload(current); err != nil {
		return "", err
	}

	message := trimmedOptionalString(args.Message)
	sessionID := trimmedOptionalString(args.SessionID)
	cronExpr := trimmedOptionalString(args.CronExpr)
	payload, err := t.manager.UpdateAgentTask(ctx, TaskUpdateRequest{
		ID:              id,
		Message:         message,
		SessionID:       sessionID,
		IntervalSeconds: args.IntervalSeconds,
		CronExpr:        cronExpr,
		Enabled:         args.Enabled,
	}, traceID)
	if err != nil {
		return "", err
	}
	if err := ensureAgentTaskPayload(payload); err != nil {
		return "", err
	}
	return marshalTaskManageOutput(payload)
}

func (t *TaskManageTool) executeDelete(ctx context.Context, args taskManageArgs, traceID string) (string, error) {
	id := strings.TrimSpace(args.ID)
	if id == "" {
		return "", fmt.Errorf("id is required for delete")
	}
	current, err := t.manager.GetTask(ctx, id, traceID)
	if err != nil {
		return "", err
	}
	if err := ensureAgentTaskPayload(current); err != nil {
		return "", err
	}
	result, err := t.manager.DeleteTask(ctx, id, traceID)
	if err != nil {
		return "", err
	}
	return marshalTaskManageOutput(result)
}

func (t *TaskManageTool) executeList(ctx context.Context, traceID string) (string, error) {
	tasks, err := t.manager.ListTasks(ctx, traceID)
	if err != nil {
		return "", err
	}
	filtered := make([]TaskPayload, 0, len(tasks))
	for _, task := range tasks {
		if normalizeTaskManageKind(task.TaskKind) != taskManageKindAgentMessage {
			continue
		}
		filtered = append(filtered, task)
	}
	return marshalTaskManageOutput(filtered)
}

func (t *TaskManageTool) executeGet(ctx context.Context, args taskManageArgs, traceID string) (string, error) {
	id := strings.TrimSpace(args.ID)
	if id == "" {
		return "", fmt.Errorf("id is required for get")
	}
	payload, err := t.manager.GetTask(ctx, id, traceID)
	if err != nil {
		return "", err
	}
	if err := ensureAgentTaskPayload(payload); err != nil {
		return "", err
	}
	return marshalTaskManageOutput(payload)
}

func decodeTaskManageArgs(raw json.RawMessage, target *taskManageArgs) error {
	source := bytes.TrimSpace(raw)
	if len(source) == 0 || bytes.Equal(source, []byte("null")) {
		source = []byte("{}")
	}
	decoder := json.NewDecoder(bytes.NewReader(source))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("invalid params: %w", err)
	}
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err == nil {
		return errors.New("invalid params: multiple JSON values are not allowed")
	} else if !errors.Is(err, io.EOF) {
		return fmt.Errorf("invalid params: %w", err)
	}
	return nil
}

func validateTaskManageSchedule(intervalSeconds *int, cronExpr *string, requireOne bool) error {
	hasInterval := intervalSeconds != nil
	hasCron := cronExpr != nil
	if hasInterval && *intervalSeconds < 1 {
		return fmt.Errorf("interval_seconds must be >= 1")
	}
	if hasCron && strings.TrimSpace(*cronExpr) == "" {
		return fmt.Errorf("cron_expr must not be empty")
	}
	if hasInterval && hasCron {
		return fmt.Errorf("exactly one of interval_seconds or cron_expr is allowed")
	}
	if requireOne && !hasInterval && !hasCron {
		return fmt.Errorf("exactly one of interval_seconds or cron_expr is required")
	}
	return nil
}

func ensureAgentTaskPayload(task TaskPayload) error {
	if normalizeTaskManageKind(task.TaskKind) != taskManageKindAgentMessage {
		return fmt.Errorf("task_manage only supports agent_message tasks")
	}
	return nil
}

func normalizeTaskManageKind(kind string) string {
	if strings.TrimSpace(kind) == "" {
		return taskManageKindAgentMessage
	}
	return strings.TrimSpace(kind)
}

func marshalTaskManageOutput(payload any) (string, error) {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("encode task result: %w", err)
	}
	return string(encoded), nil
}

func trimmedOptionalString(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	return &trimmed
}

func optionalStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}

func optionalIntValue(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}
