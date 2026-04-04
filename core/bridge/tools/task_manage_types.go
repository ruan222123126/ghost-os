package tools

import (
	"context"
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
