package orchestration

import (
	"time"

	"ghost-os/bridge/orchestration/internal/adapters/toolregistry"
	"ghost-os/bridge/taskdefs"
	bridgeTasks "ghost-os/bridge/tasks"
)

const (
	taskScheduleTypeInterval      = bridgeTasks.ScheduleTypeInterval
	taskScheduleTypeCron          = bridgeTasks.ScheduleTypeCron
	defaultTaskRunLogRetention    = bridgeTasks.DefaultRunLogRetention
	taskRunStatusRunning          = bridgeTasks.RunStatusRunning
	taskRunStatusSuccess          = bridgeTasks.RunStatusSuccess
	taskRunStatusIncomplete       = bridgeTasks.RunStatusIncomplete
	taskRunStatusCancelled        = bridgeTasks.RunStatusCancelled
	taskRunStatusError            = bridgeTasks.RunStatusError
	taskRunStatusSkipped          = bridgeTasks.RunStatusSkipped
	taskRunStatusAwaitingHuman    = bridgeTasks.RunStatusAwaitingHuman
	maxTaskResponsePreviewRunes   = bridgeTasks.MaxResponsePreviewRunes
	taskKindAgentMessage          = bridgeTasks.KindAgentMessage
	taskKindSystemAction          = bridgeTasks.KindSystemAction
	taskKindWorkflow              = bridgeTasks.KindWorkflow
	taskKindOrchestration         = bridgeTasks.KindOrchestration
	taskAgentModeSingle           = taskdefs.AgentModeSingle
	taskAgentModeRelay            = taskdefs.AgentModeRelay
	taskRelayStopPolicyAIDecides  = taskdefs.RelayStopPolicyAIDecides
	taskRelayStopPolicyMaxRounds  = taskdefs.RelayStopPolicyMaxRounds
	orchestrationNodeTypeStart    = "start"
	orchestrationNodeTypeGroup    = taskdefs.OrchestrationNodeTypeGroup
	orchestrationNodeTypeAgent    = taskdefs.OrchestrationNodeTypeAgent
	orchestrationNodeTypeEnd      = "end"
	orchestrationEdgeKindControl  = taskdefs.OrchestrationEdgeKindControl
	orchestrationEdgeKindMember   = taskdefs.OrchestrationEdgeKindMember
	orchestrationModeSequential   = taskdefs.OrchestrationSpeakingModeSequential
	orchestrationModeParallel     = taskdefs.OrchestrationSpeakingModeParallel
	orchestrationModeOwner        = taskdefs.OrchestrationSpeakingModeOwner
	taskLoadIssueInvalidFilename  = bridgeTasks.LoadIssueInvalidFilename
	taskLoadIssueReadError        = bridgeTasks.LoadIssueReadError
	taskLoadIssueDecodeError      = bridgeTasks.LoadIssueDecodeError
	taskLoadIssueInvalidConfig    = bridgeTasks.LoadIssueInvalidConfig
	taskLoadIssueIDMismatch       = bridgeTasks.LoadIssueIDMismatch
	orchestrationDispatchToolName = toolregistry.DispatchToolName
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
type TaskRuntimeOverrides = taskdefs.TaskRuntimeOverrides
type TaskRelayConfig = taskdefs.TaskRelayConfig
type WorkflowDefinition = taskdefs.WorkflowDefinition
type WorkflowNode = taskdefs.WorkflowNode
type WorkflowStartNode = taskdefs.WorkflowStartNode
type WorkflowInputVariable = taskdefs.WorkflowInputVariable
type WorkflowToolNode = taskdefs.WorkflowToolNode
type WorkflowLLMNode = taskdefs.WorkflowLLMNode
type WorkflowAgentNode = taskdefs.WorkflowAgentNode
type WorkflowIfNode = taskdefs.WorkflowIfNode
type WorkflowLoopNode = taskdefs.WorkflowLoopNode
type WorkflowEdge = taskdefs.WorkflowEdge
type OrchestrationDefinition = taskdefs.OrchestrationDefinition
type OrchestrationNode = taskdefs.OrchestrationNode
type OrchestrationGroupNode = taskdefs.OrchestrationGroupNode
type OrchestrationAgentNode = taskdefs.OrchestrationAgentNode
type OrchestrationEdge = taskdefs.OrchestrationEdge
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
	return taskdefs.CloneActionParams(input)
}

func cloneTaskWorkflow(input *WorkflowDefinition) *WorkflowDefinition {
	return taskdefs.CloneWorkflowDefinition(input)
}

func cloneTaskOrchestration(input *OrchestrationDefinition) *OrchestrationDefinition {
	return taskdefs.CloneOrchestrationDefinition(input)
}

func cloneTaskRuntimeOverrides(input *TaskRuntimeOverrides) *TaskRuntimeOverrides {
	return taskdefs.CloneTaskRuntimeOverrides(input)
}

func decodeActionParamsMap[T any](input map[string]any) (T, error) {
	return taskdefs.DecodeParamsMap[T](input)
}

func nextTaskRunAt(task ScheduledTask, now time.Time) (time.Time, error) {
	return bridgeTasks.NextTaskRunAt(task, now)
}

// Implementation moved to task_executor_adapter.go.
