package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

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
