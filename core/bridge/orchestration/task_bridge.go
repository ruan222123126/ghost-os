package orchestration

import (
	"context"
	"fmt"
	"strings"
	"time"

	bridgeTasks "ghost-os/bridge/tasks"
)

const (
	taskScheduleTypeInterval     = bridgeTasks.ScheduleTypeInterval
	taskScheduleTypeCron         = bridgeTasks.ScheduleTypeCron
	defaultTaskRunLogRetention   = bridgeTasks.DefaultRunLogRetention
	taskRunStatusSuccess         = bridgeTasks.RunStatusSuccess
	taskRunStatusCancelled       = bridgeTasks.RunStatusCancelled
	taskRunStatusError           = bridgeTasks.RunStatusError
	taskRunStatusSkipped         = bridgeTasks.RunStatusSkipped
	taskRunStatusAwaitingHuman   = bridgeTasks.RunStatusAwaitingHuman
	maxTaskResponsePreviewRunes  = bridgeTasks.MaxResponsePreviewRunes
	taskKindAgentMessage         = bridgeTasks.KindAgentMessage
	taskKindSystemAction         = bridgeTasks.KindSystemAction
	taskKindWorkflow             = bridgeTasks.KindWorkflow
	taskKindOrchestration        = bridgeTasks.KindOrchestration
	orchestrationNodeTypeStart   = bridgeTasks.OrchestrationNodeTypeStart
	orchestrationNodeTypeGroup   = bridgeTasks.OrchestrationNodeTypeGroup
	orchestrationNodeTypeAgent   = bridgeTasks.OrchestrationNodeTypeAgent
	orchestrationNodeTypeEnd     = bridgeTasks.OrchestrationNodeTypeEnd
	orchestrationEdgeKindControl = bridgeTasks.OrchestrationEdgeKindControl
	orchestrationEdgeKindMember  = bridgeTasks.OrchestrationEdgeKindMember
	orchestrationModeSequential  = bridgeTasks.OrchestrationSpeakingModeSequential
	orchestrationModeParallel    = bridgeTasks.OrchestrationSpeakingModeParallel
	orchestrationModeOwner       = bridgeTasks.OrchestrationSpeakingModeOwner
	taskLoadIssueInvalidFilename = bridgeTasks.LoadIssueInvalidFilename
	taskLoadIssueReadError       = bridgeTasks.LoadIssueReadError
	taskLoadIssueDecodeError     = bridgeTasks.LoadIssueDecodeError
	taskLoadIssueInvalidConfig   = bridgeTasks.LoadIssueInvalidConfig
	taskLoadIssueIDMismatch      = bridgeTasks.LoadIssueIDMismatch
)

var (
	ErrTaskNotFound               = bridgeTasks.ErrTaskNotFound
	ErrTaskCorrupted              = bridgeTasks.ErrTaskCorrupted
	ErrInvalidTaskID              = bridgeTasks.ErrInvalidTaskID
	ErrInvalidTaskConfig          = bridgeTasks.ErrInvalidTaskConfig
	ErrTaskSchedulerStopped       = bridgeTasks.ErrTaskSchedulerStopped
	ErrTaskSchedulerNotConfigured = bridgeTasks.ErrTaskSchedulerNotConfigured
)

type ScheduledTask = bridgeTasks.ScheduledTask
type TaskRuntimeOverrides = bridgeTasks.TaskRuntimeOverrides
type WorkflowDefinition = bridgeTasks.WorkflowDefinition
type WorkflowNode = bridgeTasks.WorkflowNode
type WorkflowStartNode = bridgeTasks.WorkflowStartNode
type WorkflowInputVariable = bridgeTasks.WorkflowInputVariable
type WorkflowToolNode = bridgeTasks.WorkflowToolNode
type WorkflowLLMNode = bridgeTasks.WorkflowLLMNode
type WorkflowAgentNode = bridgeTasks.WorkflowAgentNode
type WorkflowIfNode = bridgeTasks.WorkflowIfNode
type WorkflowLoopNode = bridgeTasks.WorkflowLoopNode
type WorkflowEdge = bridgeTasks.WorkflowEdge
type OrchestrationDefinition = bridgeTasks.OrchestrationDefinition
type OrchestrationNode = bridgeTasks.OrchestrationNode
type OrchestrationGroupNode = bridgeTasks.OrchestrationGroupNode
type OrchestrationAgentNode = bridgeTasks.OrchestrationAgentNode
type OrchestrationEdge = bridgeTasks.OrchestrationEdge
type RunNodeResult = bridgeTasks.RunNodeResult
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

func NewTaskSchedulerWithTimeout(
	store *TaskStore,
	service *bridgeService,
	executionTimeout time.Duration,
) *TaskScheduler {
	return bridgeTasks.NewTaskSchedulerWithTimeout(
		store,
		taskExecutorAdapter{service: service},
		executionTimeout,
	)
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

func cloneTaskOrchestration(input *OrchestrationDefinition) *OrchestrationDefinition {
	return bridgeTasks.CloneOrchestrationDefinition(input)
}

func cloneTaskRuntimeOverrides(input *TaskRuntimeOverrides) *TaskRuntimeOverrides {
	return bridgeTasks.CloneTaskRuntimeOverrides(input)
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
	case taskKindOrchestration:
		return a.executeOrchestrationTask(ctx, task, traceID)
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
	startedAt := time.Now().UTC()
	result := a.runAgentAction(ctx, agentParams{
		Message:   task.Message,
		SessionID: task.SessionID,
	}, cloneTaskRuntimeOverrides(task.RuntimeOverrides), traceID)
	result.NodeResults = []bridgeTasks.RunNodeResult{
		buildAgentMessageNodeResult(task, result, startedAt, time.Now().UTC()),
	}
	return result
}

func (a taskExecutorAdapter) runAgentAction(
	ctx context.Context,
	params agentParams,
	runtimeOverrides *TaskRuntimeOverrides,
	traceID string,
) bridgeTasks.ExecutionResult {
	if a.service == nil {
		return bridgeTasks.ExecutionResult{Status: taskRunStatusError, Error: "task executor service is not configured"}
	}
	result, err := a.service.executeAgentActionWithRuntimeOverrides(ctx, params, runtimeOverrides, traceID)
	if err != nil {
		return bridgeTasks.ExecutionResult{Status: taskRunStatusError, Error: err.Error()}
	}
	return taskExecutionResultFromAgentPayload(result.Payload)
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

func buildAgentMessageNodeResult(
	task ScheduledTask,
	result bridgeTasks.ExecutionResult,
	startedAt time.Time,
	finishedAt time.Time,
) bridgeTasks.RunNodeResult {
	status := strings.TrimSpace(result.Status)
	if status == "" {
		status = taskRunStatusError
	}
	input := map[string]any{
		"message":    task.Message,
		"session_id": strings.TrimSpace(task.SessionID),
	}
	runtimePayload := map[string]any{}
	if runtimeOverrides := cloneTaskRuntimeOverrides(task.RuntimeOverrides); runtimeOverrides != nil {
		runtimePayload = taskRuntimeOverrideSnapshot(runtimeOverrides)
	}
	input["runtime_overrides"] = runtimePayload
	output := map[string]any{
		"session_id_output": strings.TrimSpace(result.SessionIDOutput),
		"response_preview":  strings.TrimSpace(result.ResponsePreview),
	}
	if strings.TrimSpace(result.Error) != "" {
		output["error"] = strings.TrimSpace(result.Error)
	}
	return bridgeTasks.RunNodeResult{
		NodeID:       taskKindAgentMessage,
		NodeType:     taskKindAgentMessage,
		Status:       status,
		StartedAt:    startedAt,
		FinishedAt:   finishedAt,
		CompletedSeq: 1,
		Input:        input,
		Output:       output,
		Preview:      strings.TrimSpace(result.ResponsePreview),
		Error:        strings.TrimSpace(result.Error),
	}
}

func taskRuntimeOverrideSnapshot(runtimeOverrides *TaskRuntimeOverrides) map[string]any {
	if runtimeOverrides == nil {
		return map[string]any{}
	}
	snapshot := map[string]any{
		"provider_name": strings.TrimSpace(runtimeOverrides.ProviderName),
		"model":         strings.TrimSpace(runtimeOverrides.Model),
		"system_prompt": strings.TrimSpace(runtimeOverrides.SystemPrompt),
		"preset_id":     strings.TrimSpace(runtimeOverrides.PresetID),
	}
	if runtimeOverrides.ToolAllowlistOnly != nil {
		snapshot["tool_allowlist_only"] = *runtimeOverrides.ToolAllowlistOnly
	}
	if runtimeOverrides.MaxTurns != nil {
		snapshot["max_turns"] = *runtimeOverrides.MaxTurns
	}
	if runtimeOverrides.ToolAllowlist != nil {
		snapshot["tool_allowlist"] = append([]string(nil), runtimeOverrides.ToolAllowlist...)
	}
	return snapshot
}

func (a taskExecutorAdapter) executeSystemTask(_ context.Context, task ScheduledTask, _ string) bridgeTasks.ExecutionResult {
	if a.service == nil {
		return bridgeTasks.ExecutionResult{Status: taskRunStatusError, Error: "task executor service is not configured"}
	}
	return bridgeTasks.ExecutionResult{
		Status: taskRunStatusError,
		Error:  "unsupported system action: " + strings.TrimSpace(task.Action),
	}
}
