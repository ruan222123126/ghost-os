package orchestration

import (
	"time"

	bridgeTasks "ghost-os/bridge/tasks"
)

const (
	taskScheduleTypeInterval     = bridgeTasks.ScheduleTypeInterval
	taskScheduleTypeCron         = bridgeTasks.ScheduleTypeCron
	defaultTaskRunLogRetention   = bridgeTasks.DefaultRunLogRetention
	taskRunStatusSuccess         = bridgeTasks.RunStatusSuccess
	taskRunStatusIncomplete      = bridgeTasks.RunStatusIncomplete
	taskRunStatusCancelled       = bridgeTasks.RunStatusCancelled
	taskRunStatusError           = bridgeTasks.RunStatusError
	taskRunStatusSkipped         = bridgeTasks.RunStatusSkipped
	taskRunStatusAwaitingHuman   = bridgeTasks.RunStatusAwaitingHuman
	maxTaskResponsePreviewRunes  = bridgeTasks.MaxResponsePreviewRunes
	taskKindAgentMessage         = bridgeTasks.KindAgentMessage
	taskKindSystemAction         = bridgeTasks.KindSystemAction
	taskKindWorkflow             = bridgeTasks.KindWorkflow
	taskKindOrchestration        = bridgeTasks.KindOrchestration
	taskAgentModeSingle          = bridgeTasks.AgentModeSingle
	taskAgentModeRelay           = bridgeTasks.AgentModeRelay
	taskRelayStopPolicyAIDecides = bridgeTasks.RelayStopPolicyAIDecides
	taskRelayStopPolicyMaxRounds = bridgeTasks.RelayStopPolicyMaxRounds
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
type TaskRelayConfig = bridgeTasks.TaskRelayConfig
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

// Implementation moved to task_executor_adapter.go.
