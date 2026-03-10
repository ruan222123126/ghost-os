package app

import (
	"context"
	"fmt"

	"ghost-os/bridge/tools"
)

// CreateAgentTask 为 task_manage 工具暴露普通消息任务创建入口。
func (s *bridgeService) CreateAgentTask(_ context.Context, req tools.TaskCreateRequest, traceID string) (tools.TaskPayload, error) {
	payload, _, err := s.executeTaskCreateAction(taskCreateParams{
		Message:         req.Message,
		SessionID:       req.SessionID,
		TaskKind:        taskKindAgentMessage,
		IntervalSeconds: req.IntervalSeconds,
		CronExpr:        req.CronExpr,
	}, traceID)
	if err != nil {
		return tools.TaskPayload{}, err
	}
	task, ok := payload.(taskPayload)
	if !ok {
		return tools.TaskPayload{}, fmt.Errorf("unexpected task create payload type %T", payload)
	}
	return toToolTaskPayload(task), nil
}

// UpdateAgentTask 为 task_manage 工具暴露普通消息任务更新入口。
func (s *bridgeService) UpdateAgentTask(_ context.Context, req tools.TaskUpdateRequest, traceID string) (tools.TaskPayload, error) {
	payload, _, err := s.executeTaskUpdateAction(taskUpdateParams{
		ID:              req.ID,
		Message:         req.Message,
		SessionID:       req.SessionID,
		IntervalSeconds: req.IntervalSeconds,
		CronExpr:        req.CronExpr,
		Enabled:         req.Enabled,
	}, traceID)
	if err != nil {
		return tools.TaskPayload{}, err
	}
	task, ok := payload.(taskPayload)
	if !ok {
		return tools.TaskPayload{}, fmt.Errorf("unexpected task update payload type %T", payload)
	}
	return toToolTaskPayload(task), nil
}

// GetTask 为 task_manage 工具读取单个任务。
func (s *bridgeService) GetTask(_ context.Context, id string, traceID string) (tools.TaskPayload, error) {
	payload, _, err := s.executeTaskGetAction(taskIDParams{ID: id}, traceID)
	if err != nil {
		return tools.TaskPayload{}, err
	}
	task, ok := payload.(taskPayload)
	if !ok {
		return tools.TaskPayload{}, fmt.Errorf("unexpected task get payload type %T", payload)
	}
	return toToolTaskPayload(task), nil
}

// ListTasks 为 task_manage 工具列出任务。
func (s *bridgeService) ListTasks(_ context.Context, traceID string) ([]tools.TaskPayload, error) {
	payload, _, err := s.executeTaskListAction(taskListScopeUser, traceID)
	if err != nil {
		return nil, err
	}
	items, ok := payload.([]taskPayload)
	if !ok {
		return nil, fmt.Errorf("unexpected task list payload type %T", payload)
	}
	result := make([]tools.TaskPayload, 0, len(items))
	for _, item := range items {
		result = append(result, toToolTaskPayload(item))
	}
	return result, nil
}

// DeleteTask 为 task_manage 工具删除任务。
func (s *bridgeService) DeleteTask(_ context.Context, id string, traceID string) (tools.TaskDeleteResult, error) {
	payload, _, err := s.executeTaskDeleteAction(taskIDParams{ID: id}, traceID)
	if err != nil {
		return tools.TaskDeleteResult{}, err
	}
	result, ok := payload.(taskDeleteResponse)
	if !ok {
		return tools.TaskDeleteResult{}, fmt.Errorf("unexpected task delete payload type %T", payload)
	}
	return tools.TaskDeleteResult{ID: result.ID, Deleted: result.Deleted}, nil
}

func toToolTaskPayload(input taskPayload) tools.TaskPayload {
	return tools.TaskPayload{
		ID:              input.ID,
		Message:         input.Message,
		SessionID:       input.SessionID,
		TaskKind:        input.TaskKind,
		Action:          input.Action,
		ActionParams:    cloneTaskActionParams(input.ActionParams),
		ScheduleType:    input.ScheduleType,
		IntervalSeconds: input.IntervalSeconds,
		CronExpr:        input.CronExpr,
		Enabled:         input.Enabled,
		CreatedAt:       input.CreatedAt,
		UpdatedAt:       input.UpdatedAt,
		LastRunAt:       input.LastRunAt,
		NextRunAt:       input.NextRunAt,
		LastError:       input.LastError,
	}
}
