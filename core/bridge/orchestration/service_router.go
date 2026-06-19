package orchestration

import (
	"context"
	"encoding/json"

	bridgeconfig "ghost-os/bridge/config"
	agentadapter "ghost-os/bridge/orchestration/internal/adapters/agent"
	serviceruntime "ghost-os/bridge/orchestration/internal/adapters/serviceruntime"
	sessionartifacts "ghost-os/bridge/orchestration/internal/adapters/sessionartifacts"
	appsessions "ghost-os/bridge/orchestration/internal/app/sessions"
	appskills "ghost-os/bridge/orchestration/internal/app/skills"
	"ghost-os/bridge/orchestration/internal/contracts/bus"
	"ghost-os/bridge/orchestration/internal/dispatch"
	internaltrace "ghost-os/bridge/orchestration/internal/trace"
	"ghost-os/bridge/session"
	bridgeskills "ghost-os/bridge/skills"
	"ghost-os/bridge/streaming"
)

var (
	ErrSessionInflight = internaltrace.ErrSessionInflight
	ErrRunNotFound     = internaltrace.ErrRunNotFound
	ErrRunCancelled    = internaltrace.ErrRunCancelled
	ErrRunRegistryNil  = internaltrace.ErrRunRegistryNil
)

const (
	sessionPushAssistantMessage    = internaltrace.SessionPushAssistantMessage
	sessionPushAwaitingHuman       = internaltrace.SessionPushAwaitingHuman
	sessionPushRunStarted          = internaltrace.SessionPushRunStarted
	sessionPushCompletionDelta     = internaltrace.SessionPushCompletionDelta
	sessionPushToolCallStarted     = internaltrace.SessionPushToolCallStarted
	sessionPushToolCallFinished    = internaltrace.SessionPushToolCallFinished
	sessionPushError               = internaltrace.SessionPushError
	sessionPushDone                = internaltrace.SessionPushDone
	sessionPushTaskRunCardStarted  = internaltrace.SessionPushTaskRunCardStarted
	sessionPushTaskRunCardEvent    = internaltrace.SessionPushTaskRunCardEvent
	sessionPushTaskRunCardFinished = internaltrace.SessionPushTaskRunCardFinished
)

type RunHandle = internaltrace.RunHandle
type RunRegistry = internaltrace.RunRegistry
type sessionPushEventType = internaltrace.SessionPushEventType
type sessionPushEvent = internaltrace.SessionPushEvent
type sessionPushHub = internaltrace.SessionPushHub

func newSessionPushHub() *sessionPushHub {
	return internaltrace.NewSessionPushHub()
}

type agentExecutorFunc func(
	ctx context.Context,
	message string,
	sessionID string,
	traceID string,
	store bridgeconfig.Store,
	sessionStore *session.Store,
) (string, string, error)
type agentStreamExecutorFunc func(
	ctx context.Context,
	message string,
	sessionID string,
	traceID string,
	store bridgeconfig.Store,
	sessionStore *session.Store,
	sink streaming.Sink,
) (string, string, error)

// bridgeService 负责 action 分发，不承载 transport 细节。
type bridgeService struct {
	configStore     bridgeconfig.Store
	sessionStore    *session.Store
	lifecycle       *serviceruntime.Lifecycle
	runtimeState    *serviceruntime.State
	actionRouter    *dispatch.Router
	skillHandler    *bridgeskills.ActionHandler
	agentRunner     SessionTurnRunner
	runRegistry     *RunRegistry
	runtimeFactory  AgentRuntimeFactory
	artifactStore   appsessions.ArtifactStore
	artifactInitErr error
}

// newBridgeService 组装 action -> handler 映射，并初始化会话与记忆依赖。
func newBridgeService(store bridgeconfig.Store, sessionStore *session.Store, executor agentExecutorFunc) *bridgeService {
	return newBridgeServiceWithStreamExecutor(store, sessionStore, executor, nil)
}

func newBridgeServiceWithStreamExecutor(
	store bridgeconfig.Store,
	sessionStore *session.Store,
	executor agentExecutorFunc,
	streamExecutor agentStreamExecutorFunc,
) *bridgeService {
	service := newBridgeServiceState(store, sessionStore)
	service.skillHandler = appskills.NewActionHandler(store, service.skillLogFunc())
	service.runtimeFactory = newAgentRuntimeFactory()
	service.agentRunner = newServiceAgentRunner(service, executor, streamExecutor)
	registerDefaultActions(service)
	return service
}

func newBridgeServiceState(store bridgeconfig.Store, sessionStore *session.Store) *bridgeService {
	runtimeState := serviceruntime.NewState()
	sessionPush := internaltrace.NewSessionPushHub()
	artifactStore, artifactInitErr := sessionartifacts.NewFromEnv()
	return &bridgeService{
		configStore:     store,
		sessionStore:    sessionStore,
		lifecycle:       serviceruntime.NewLifecycle(runtimeState, sessionPush),
		runtimeState:    runtimeState,
		actionRouter:    dispatch.NewRouter(21),
		runRegistry:     internaltrace.NewRunRegistry(),
		artifactStore:   artifactStore,
		artifactInitErr: artifactInitErr,
	}
}

func newServiceAgentRunner(service *bridgeService, executor agentExecutorFunc, streamExecutor agentStreamExecutorFunc) SessionTurnRunner {
	runner := agentadapter.NewExecutorRunner(
		service.configStore,
		service.sessionStore,
		agentadapter.ExecutorFunc(executor),
		agentadapter.StreamExecutorFunc(streamExecutor),
	)
	if runner != nil {
		return runner
	}
	return NewSessionAgentRunner(
		service.runtimeFactory,
		service.configStore,
		service.sessionStore,
		service.runRegistry,
	)
}

func (s *bridgeService) sessionPushHub() *internaltrace.SessionPushHub {
	if s == nil || s.lifecycle == nil {
		return nil
	}
	return s.lifecycle.SessionPushHub()
}

func (s *bridgeService) taskStore() *TaskStore {
	if s == nil || s.runtimeState == nil {
		return nil
	}
	return s.runtimeState.TaskStore()
}

func (s *bridgeService) taskScheduler() *TaskScheduler {
	if s == nil || s.runtimeState == nil {
		return nil
	}
	return s.runtimeState.TaskScheduler()
}

func (s *bridgeService) taskInitErr() error {
	if s == nil || s.runtimeState == nil {
		return nil
	}
	return s.runtimeState.TaskInitErr()
}

// StartBackgroundRuntimes 显式初始化 service 依赖的后台 runtime。
func (s *bridgeService) StartBackgroundRuntimes() error {
	if s == nil {
		return nil
	}
	return s.runtimeState.Start(s.configStore, taskExecutorAdapter{service: s})
}

// BootstrapSystemTasks 将系统调度任务同步到 task runtime。
func (s *bridgeService) BootstrapSystemTasks() error {
	if s == nil {
		return nil
	}
	return s.runtimeState.BootstrapSystemTasks()
}

func (s *bridgeService) skillLogFunc() bridgeskills.LogFunc {
	return func(traceID, action, status string, err error) {
		logAction(traceID, action, status, err)
	}
}

// Close 释放 service 级后台资源。
func (s *bridgeService) Close() {
	if s == nil {
		return
	}
	s.lifecycle.Close()
}

// dispatchAction 根据 action 查找处理器；未知 action 返回显式可选列表。
func (s *bridgeService) dispatchAction(ctx context.Context, action string, params json.RawMessage, traceID string) (ServiceResult, error) {
	return s.actionRouter.Dispatch(ctx, action, params, traceID)
}

func (s *bridgeService) registeredActionNames() []string {
	if s == nil || s.actionRouter == nil {
		return nil
	}
	return s.actionRouter.ActionNames()
}

func validateBusRequest(req apiRequest) error {
	return dispatch.ValidateBusRequest(req)
}

func (s *Service) ExecuteConfigGetAction(traceID string) (ServiceResult, error) {
	return s.inner.executeConfigGetAction(traceID)
}

func (s *Service) ExecuteConfigUpdateAction(req ConfigUpdateRequest, traceID string) (ServiceResult, error) {
	return s.inner.executeConfigUpdateAction(req, traceID)
}

func logAction(traceID string, action string, status string, err error) {
	dispatch.LogAction(traceID, action, status, err)
}

func registerDefaultActions(service *bridgeService) {
	if service == nil || service.actionRouter == nil {
		return
	}
	dispatch.RegisterDefaultActions(service.actionRouter, defaultActionHandlers(service))
}

func defaultActionHandlers(service *bridgeService) dispatch.DefaultHandlers {
	return dispatch.DefaultHandlers{
		AgentSend: service.executeAgentAction,
		AgentStop: service.executeAgentStopAction,
		ConfigGet: func(_ context.Context, traceID string) (ServiceResult, error) {
			return service.executeConfigGetAction(traceID)
		},
		ConfigUpdate: func(_ context.Context, params configUpdateRequest, traceID string) (ServiceResult, error) {
			return service.executeConfigUpdateAction(params, traceID)
		},
		ConfigProvidersGet: func(_ context.Context, traceID string) (ServiceResult, error) {
			return service.executeProvidersGetAction(traceID)
		},
		HumanResponse: service.executeHumanResponseAction,
		SessionsList: func(_ context.Context, traceID string) (ServiceResult, error) {
			return service.executeSessionsListAction(traceID)
		},
		SessionGet: func(_ context.Context, params sessionGetParams, traceID string) (ServiceResult, error) {
			return service.executeSessionGetAction(params, traceID)
		},
		TaskCreate: func(_ context.Context, params taskCreateParams, traceID string) (ServiceResult, error) {
			return service.executeTaskCreateActionResult(params, traceID)
		},
		TaskList: service.executeTaskListDispatchAction,
		TaskGet: func(_ context.Context, params taskIDParams, traceID string) (ServiceResult, error) {
			return service.executeTaskGetActionResult(params, traceID)
		},
		TaskUpdate: func(_ context.Context, params taskUpdateParams, traceID string) (ServiceResult, error) {
			return service.executeTaskUpdateActionResult(params, traceID)
		},
		TaskRunNow: func(_ context.Context, params taskRunNowParams, traceID string) (ServiceResult, error) {
			return service.executeTaskRunNowActionResult(params, traceID)
		},
		TaskStop: service.executeTaskStopActionResult,
		TaskLogs: func(_ context.Context, params taskLogsParams, traceID string) (ServiceResult, error) {
			return service.executeTaskLogsActionResult(params, traceID)
		},
		TaskDelete: func(_ context.Context, params taskIDParams, traceID string) (ServiceResult, error) {
			return service.executeTaskDeleteActionResult(params, traceID)
		},
	}
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
