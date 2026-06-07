package orchestration

import (
	"errors"
	"time"

	bridgeconfig "ghost-os/bridge/config"
	runtimeadapter "ghost-os/bridge/orchestration/internal/adapters/runtime"
	"ghost-os/bridge/orchestration/internal/adapters/toolregistry"
	apprelay "ghost-os/bridge/orchestration/internal/app/agentturn/relay"
	apptasks "ghost-os/bridge/orchestration/internal/app/tasks"
	workflowdomain "ghost-os/bridge/orchestration/internal/domain/workflow"
	"ghost-os/bridge/taskdefs"
	bridgeTasks "ghost-os/bridge/tasks"
)

const (
	taskListScopeUser             = apptasks.ScopeUser
	taskListScopeSystem           = apptasks.ScopeSystem
	taskListScopeOrchestration    = apptasks.ScopeOrchestration
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
	workflowNodeTypeStart         = workflowdomain.NodeTypeStart
	workflowNodeTypeTool          = workflowdomain.NodeTypeTool
	workflowNodeTypeLLM           = workflowdomain.NodeTypeLLM
	workflowNodeTypeAgent         = workflowdomain.NodeTypeAgent
	workflowNodeTypeIf            = workflowdomain.NodeTypeIf
	workflowNodeTypeLoop          = workflowdomain.NodeTypeLoop
	workflowNodeTypeEnd           = workflowdomain.NodeTypeEnd
	workflowInputTypeString       = workflowdomain.InputTypeString
	workflowInputTypeNumber       = workflowdomain.InputTypeNumber
	workflowInputTypeBoolean      = workflowdomain.InputTypeBoolean
	workflowInputTypeObject       = workflowdomain.InputTypeObject
	workflowInputTypeArray        = workflowdomain.InputTypeArray
	workflowIfOperatorEquals      = workflowdomain.IfOperatorEquals
	workflowIfOperatorNotEquals   = workflowdomain.IfOperatorNotEquals
	workflowIfOperatorContains    = workflowdomain.IfOperatorContains
	workflowIfOperatorNotContains = workflowdomain.IfOperatorNotContains
	workflowIfOperatorIsEmpty     = workflowdomain.IfOperatorIsEmpty
	workflowIfOperatorNotEmpty    = workflowdomain.IfOperatorNotEmpty
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
type workflowExecutionPlan = workflowdomain.Plan

type TaskScopeKind string

const (
	TaskScopeKindUser          TaskScopeKind = "user"
	TaskScopeKindOrchestration TaskScopeKind = "orchestration"
)

func (s *Service) ExecuteUserTaskListAction(traceID string) (ServiceResult, error) {
	return s.inner.executeTaskListActionResult(taskListScopeUser, traceID)
}

func (s *Service) ExecuteSystemTaskListAction(traceID string) (ServiceResult, error) {
	return s.inner.executeTaskListActionResult(taskListScopeSystem, traceID)
}

func (s *Service) ExecuteOrchestrationListAction(traceID string) (ServiceResult, error) {
	return s.inner.executeTaskListActionResult(taskListScopeOrchestration, traceID)
}

func (s *Service) ExecuteUserTaskCreateAction(req TaskCreateParams, traceID string) (ServiceResult, error) {
	return s.executeTaskCreateInScope(req, taskListScopeUser, traceID)
}

func (s *Service) ExecuteOrchestrationCreateAction(req TaskCreateParams, traceID string) (ServiceResult, error) {
	return s.executeTaskCreateInScope(req, taskListScopeOrchestration, traceID)
}

func (s *Service) executeTaskCreateInScope(req TaskCreateParams, scope string, traceID string) (ServiceResult, error) {
	req.Scope = scope
	return s.inner.executeTaskCreateActionResult(req, traceID)
}

type relayRuntimeBuilder struct {
	service *bridgeService
}

func NewTaskStore(baseDir string) (*TaskStore, error) {
	return bridgeTasks.NewStore(baseDir, validateTaskDefinition)
}

func validateTaskDefinition(task *ScheduledTask) error {
	return apptasks.ValidateDefinition(task)
}

func normalizeTaskDefinition(task *ScheduledTask) {
	apptasks.NormalizeDefinition(task)
}

func validateTaskRelayConfig(relay *TaskRelayConfig) error {
	return apptasks.ValidateRelayConfig(relay)
}

func ensureWorkflowAllowedForTaskKind(taskKind string, workflow *WorkflowDefinition) error {
	return apptasks.EnsureWorkflowAllowedForKind(taskKind, workflow)
}

func ensureOrchestrationAllowedForTaskKind(taskKind string, definition *OrchestrationDefinition) error {
	return apptasks.EnsureOrchestrationAllowedForKind(taskKind, definition)
}

func loadTaskRuntimeConfig(store bridgeconfig.Store) (bridgeconfig.TaskConfig, error) {
	return apptasks.LoadRuntimeConfig(store)
}

func validateWorkflowTaskRuntime(definition *WorkflowDefinition, cfg bridgeconfig.TaskConfig) error {
	return apptasks.ValidateWorkflowRuntime(definition, cfg)
}

func validateWorkflowAgentRuntime(definition *WorkflowDefinition, store bridgeconfig.Store) error {
	return apptasks.ValidateWorkflowAgentRuntime(definition, store)
}

func validateOrchestrationAgentRuntime(definition *OrchestrationDefinition, store bridgeconfig.Store) error {
	return apptasks.ValidateOrchestrationAgentRuntime(definition, store)
}

func newRelayTaskRunner(service *bridgeService) apprelay.Runner {
	if service == nil {
		return apprelay.Runner{}
	}
	return apprelay.Runner{
		RuntimeBuilder: relayRuntimeBuilder{service: service},
		SessionStore:   service.sessionStore,
		RunRegistry:    service.runRegistry,
	}
}

func (b relayRuntimeBuilder) Build(
	runtimeOverrides *TaskRuntimeOverrides,
) (apprelay.RuntimeDependencies, error) {
	if b.service == nil {
		return apprelay.RuntimeDependencies{}, errors.New("relay runtime service is not configured")
	}
	preparer := newSessionTurnPreparer(
		b.service.runtimeFactory,
		b.service.configStore,
		b.service.sessionStore,
		b.service.runRegistry,
		nil,
	)
	deps, _, _, err := preparer.BuildPrepareDependencies(runtimeOverrides)
	if err != nil {
		return apprelay.RuntimeDependencies{}, err
	}
	return runtimeadapter.ToRelayDependencies(deps), nil
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

func cloneScheduledTask(task ScheduledTask) ScheduledTask {
	return apptasks.CloneScheduledTask(task)
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

func buildWorkflowExecutionPlan(definition *WorkflowDefinition) (workflowExecutionPlan, error) {
	return workflowdomain.PlanBuilder{}.Build(definition)
}

// Implementation moved to task_executor_adapter.go.
