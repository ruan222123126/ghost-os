package orchestration

import (
	"context"
	"fmt"
	"strings"
	"time"

	bridgeTasks "ghost-os/bridge/tasks"
)

const (
	taskScheduleTypeInterval    = bridgeTasks.ScheduleTypeInterval
	taskScheduleTypeCron        = bridgeTasks.ScheduleTypeCron
	defaultTaskRunLogRetention  = bridgeTasks.DefaultRunLogRetention
	taskRunStatusSuccess        = bridgeTasks.RunStatusSuccess
	taskRunStatusCancelled      = bridgeTasks.RunStatusCancelled
	taskRunStatusError          = bridgeTasks.RunStatusError
	taskRunStatusSkipped        = bridgeTasks.RunStatusSkipped
	taskRunStatusAwaitingHuman  = bridgeTasks.RunStatusAwaitingHuman
	maxTaskResponsePreviewRunes = bridgeTasks.MaxResponsePreviewRunes
	taskKindAgentMessage        = bridgeTasks.KindAgentMessage
	taskKindSystemAction        = bridgeTasks.KindSystemAction
	taskKindWorkflow            = bridgeTasks.KindWorkflow
	taskLoadIssueReadError      = bridgeTasks.LoadIssueReadError
	taskLoadIssueDecodeError    = bridgeTasks.LoadIssueDecodeError
	taskLoadIssueInvalidConfig  = bridgeTasks.LoadIssueInvalidConfig
	taskLoadIssueIDMismatch     = bridgeTasks.LoadIssueIDMismatch
)

var (
	ErrTaskNotFound      = bridgeTasks.ErrTaskNotFound
	ErrTaskCorrupted     = bridgeTasks.ErrTaskCorrupted
	ErrInvalidTaskID     = bridgeTasks.ErrInvalidTaskID
	ErrInvalidTaskConfig = bridgeTasks.ErrInvalidTaskConfig
)

type ScheduledTask = bridgeTasks.ScheduledTask
type WorkflowDefinition = bridgeTasks.WorkflowDefinition
type WorkflowNode = bridgeTasks.WorkflowNode
type WorkflowToolNode = bridgeTasks.WorkflowToolNode
type WorkflowLLMNode = bridgeTasks.WorkflowLLMNode
type WorkflowAgentNode = bridgeTasks.WorkflowAgentNode
type WorkflowEdge = bridgeTasks.WorkflowEdge
type TaskRunLog = bridgeTasks.RunLog
type TaskLoadIssue = bridgeTasks.LoadIssue
type TaskStore = bridgeTasks.Store
type TaskScheduler = bridgeTasks.TaskScheduler

func NewTaskStore(baseDir string) (*TaskStore, error) {
	return bridgeTasks.NewStore(baseDir, validateTaskDefinition)
}

func NewTaskScheduler(store *TaskStore, service *bridgeService) *TaskScheduler {
	return bridgeTasks.NewTaskScheduler(store, taskExecutorAdapter{service: service})
}

func normalizeTaskKind(kind string) string {
	return bridgeTasks.NormalizeKind(kind)
}

func isSupportedTaskKind(kind string) bool {
	return bridgeTasks.IsSupportedKind(kind)
}

func cloneTaskActionParams(input map[string]any) map[string]any {
	return bridgeTasks.CloneActionParams(input)
}

func cloneTaskWorkflow(input *WorkflowDefinition) *WorkflowDefinition {
	return bridgeTasks.CloneWorkflowDefinition(input)
}

func decodeActionParamsMap[T any](input map[string]any) (T, error) {
	return bridgeTasks.DecodeParamsMap[T](input)
}

func nextTaskRunAt(task ScheduledTask, now time.Time) (time.Time, error) {
	return bridgeTasks.NextTaskRunAt(task, now)
}

type taskExecutorAdapter struct {
	service *bridgeService
}

func (a taskExecutorAdapter) Execute(ctx context.Context, task ScheduledTask, traceID string) bridgeTasks.ExecutionResult {
	switch kind := normalizeTaskKind(task.TaskKind); kind {
	case taskKindWorkflow:
		return a.executeWorkflowTask(ctx, task, traceID)
	case taskKindSystemAction:
		return a.executeSystemTask(ctx, task, traceID)
	case taskKindAgentMessage:
		return a.executeAgentTask(ctx, task, traceID)
	default:
		return bridgeTasks.ExecutionResult{
			Status: taskRunStatusError,
			Error:  fmt.Sprintf("unsupported task_kind %q", kind),
		}
	}
}

func (a taskExecutorAdapter) executeAgentTask(ctx context.Context, task ScheduledTask, traceID string) bridgeTasks.ExecutionResult {
	return a.runAgentAction(ctx, agentParams{
		Message:   task.Message,
		SessionID: task.SessionID,
	}, traceID)
}

func (a taskExecutorAdapter) runAgentAction(
	ctx context.Context,
	params agentParams,
	traceID string,
) bridgeTasks.ExecutionResult {
	if a.service == nil {
		return bridgeTasks.ExecutionResult{Status: taskRunStatusError, Error: "task executor service is not configured"}
	}
	payload, _, err := a.service.executeAgentAction(ctx, params, traceID)
	if err != nil {
		return bridgeTasks.ExecutionResult{Status: taskRunStatusError, Error: err.Error()}
	}
	return taskExecutionResultFromAgentPayload(payload)
}

func taskExecutionResultFromAgentPayload(payload any) bridgeTasks.ExecutionResult {
	switch typed := payload.(type) {
	case agentResponse:
		return bridgeTasks.ExecutionResult{
			Status:          taskRunStatusSuccess,
			SessionIDOutput: strings.TrimSpace(typed.SessionID),
			ResponsePreview: truncateRunes(strings.TrimSpace(typed.Message), maxTaskResponsePreviewRunes),
		}
	case askHumanAwaitingResponse:
		return bridgeTasks.ExecutionResult{
			Status:          taskRunStatusAwaitingHuman,
			SessionIDOutput: strings.TrimSpace(typed.SessionID),
			ResponsePreview: truncateRunes(strings.TrimSpace(typed.Prompt), maxTaskResponsePreviewRunes),
		}
	default:
		return bridgeTasks.ExecutionResult{Status: taskRunStatusSuccess}
	}
}

func (a taskExecutorAdapter) executeSystemTask(ctx context.Context, task ScheduledTask, traceID string) bridgeTasks.ExecutionResult {
	if a.service == nil {
		return bridgeTasks.ExecutionResult{Status: taskRunStatusError, Error: "task executor service is not configured"}
	}
	switch strings.TrimSpace(task.Action) {
	case busActionRSSInboxPoll:
		params, err := decodeRSSInboxPollParams(task.ActionParams)
		if err != nil {
			return bridgeTasks.ExecutionResult{Status: taskRunStatusError, Error: err.Error()}
		}
		payload, _, err := a.service.executeRSSInboxPollUsecase(ctx, params, task.ID, traceID)
		if err != nil {
			return bridgeTasks.ExecutionResult{Status: taskRunStatusError, Error: err.Error()}
		}
		return bridgeTasks.ExecutionResult{
			Status:          taskRunStatusSuccess,
			ResponsePreview: formatRSSInboxPollPreview(payload),
		}
	case busActionRSSBriefingBuild:
		params, err := decodeRSSBriefingParams(task.ActionParams)
		if err != nil {
			return bridgeTasks.ExecutionResult{Status: taskRunStatusError, Error: err.Error()}
		}
		params.TaskID = task.ID
		payload, _, err := a.service.executeRSSBriefingBuildAction(ctx, params, traceID)
		if err != nil {
			return bridgeTasks.ExecutionResult{Status: taskRunStatusError, Error: err.Error()}
		}
		typed, _ := payload.(RSSBriefingResult)
		return bridgeTasks.ExecutionResult{
			Status:          taskRunStatusSuccess,
			ResponsePreview: formatRSSBriefingPreview(typed),
		}
	default:
		return bridgeTasks.ExecutionResult{
			Status: taskRunStatusError,
			Error:  "unsupported system action: " + strings.TrimSpace(task.Action),
		}
	}
}
