package orchestration

import (
	"context"
	"errors"

	bridgeconfig "ghost-os/bridge/config"
	runtimeadapter "ghost-os/bridge/orchestration/internal/adapters/runtime"
	"ghost-os/bridge/orchestration/internal/adapters/toolregistry"
	apprelay "ghost-os/bridge/orchestration/internal/app/agentturn/relay"
	appexternal "ghost-os/bridge/orchestration/internal/app/externalagent"
	apptasks "ghost-os/bridge/orchestration/internal/app/tasks"
	"ghost-os/bridge/orchestration/internal/contracts/bus"
	"ghost-os/bridge/orchestration/internal/dispatch"
	workflowdomain "ghost-os/bridge/orchestration/internal/domain/workflow"
	internaltrace "ghost-os/bridge/orchestration/internal/trace"
	"ghost-os/bridge/session"
	"ghost-os/bridge/streaming"
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

func (s *bridgeService) executeTaskListDispatchAction(
	_ context.Context,
	params taskListParams,
	traceID string,
) (ServiceResult, error) {
	scope, err := dispatch.NormalizeTaskListScope(params.Scope)
	if err != nil {
		return ServiceResult{}, bus.WrapError(ServiceErrorInvalidInput, err)
	}
	return s.executeTaskListActionResult(scope, traceID)
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

func validateWorkflowTaskRuntime(definition *WorkflowDefinition, cfg bridgeconfig.TaskConfig) error {
	return apptasks.ValidateWorkflowRuntime(definition, cfg)
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

func cloneTaskActionParams(input map[string]any) map[string]any {
	return taskdefs.CloneActionParams(input)
}

func cloneScheduledTask(task ScheduledTask) ScheduledTask {
	return apptasks.CloneScheduledTask(task)
}

func cloneTaskRuntimeOverrides(input *TaskRuntimeOverrides) *TaskRuntimeOverrides {
	return taskdefs.CloneTaskRuntimeOverrides(input)
}

func (s *bridgeService) externalAgentManager() *appexternal.Manager {
	if s == nil {
		return nil
	}
	if s.externalAgents == nil {
		s.externalAgents = appexternal.NewManager(s.configStore, s.sessionStore)
	}
	return s.externalAgents
}

func (s *bridgeService) executeExternalAgentStartAction(ctx context.Context, params externalAgentRequest, traceID string) (ServiceResult, error) {
	return s.executeExternalAgentStreamless(ctx, params, traceID, true)
}

func (s *bridgeService) executeExternalAgentSendAction(ctx context.Context, params externalAgentRequest, traceID string) (ServiceResult, error) {
	return s.executeExternalAgentStreamless(ctx, params, traceID, false)
}

func (s *bridgeService) executeExternalAgentModelsGetAction(ctx context.Context, traceID string) (ServiceResult, error) {
	manager := s.externalAgentManager()
	if manager == nil {
		return ServiceResult{}, bus.WrapError(ServiceErrorInternal, errors.New("external agent manager is not configured"))
	}
	logAction(traceID, BusActionExternalAgentModelsGet, "running", nil)
	catalog, err := manager.ListModels(ctx)
	if err != nil {
		logAction(traceID, BusActionExternalAgentModelsGet, "error", err)
		return ServiceResult{}, bus.WrapError(mapExternalAgentError(err), err)
	}
	logAction(traceID, BusActionExternalAgentModelsGet, "success", nil)
	return bus.ResultSuccess(codexModelCatalog{
		Models:       catalog.Models,
		DefaultModel: catalog.DefaultModel,
	}), nil
}

func (s *bridgeService) executeExternalAgentStreamless(ctx context.Context, params externalAgentRequest, traceID string, forceStart bool) (ServiceResult, error) {
	manager := s.externalAgentManager()
	if manager == nil {
		return ServiceResult{}, bus.WrapError(ServiceErrorInternal, errors.New("external agent manager is not configured"))
	}
	action := BusActionExternalAgentSend
	if forceStart {
		action = BusActionExternalAgentStart
	}
	logAction(traceID, action, "running", nil)
	sink := internaltrace.NewSessionStreamBroadcastSink(streaming.NopSink{}, s.sessionPushHub())
	message, sessionID, err := manager.ExecuteStream(ctx, params, traceID, sink, forceStart)
	if err != nil {
		logAction(traceID, action, "error", err)
		return ServiceResult{}, bus.WrapError(mapExternalAgentError(err), err)
	}
	logAction(traceID, action, "success", nil)
	return bus.ResultSuccess(externalAgentResponse{
		Status:    appexternal.StatusIdle,
		Provider:  appexternal.ProviderCodex,
		SessionID: sessionID,
		ThreadID:  externalThreadID(s.sessionStore, sessionID),
		TurnID:    "",
	}), validateExternalStreamlessMessage(message)
}

func validateExternalStreamlessMessage(_ string) error { return nil }

func externalThreadID(store *session.Store, sessionID string) string {
	if store == nil || sessionID == "" {
		return ""
	}
	sess, err := store.Load(sessionID)
	if err != nil || sess.ExternalRuntime == nil {
		return ""
	}
	return sess.ExternalRuntime.ThreadID
}

// Implementation moved to task_executor_adapter.go.
