package orchestration

import (
	"context"
	"encoding/json"

	bridgeconfig "ghost-os/bridge/config"
	bridgerss "ghost-os/bridge/rss"
	"ghost-os/bridge/session"
	bridgeskills "ghost-os/bridge/skills"
	"ghost-os/bridge/streaming"
	"ghost-os/bridge/tools"
)

type actionHandler func(ctx context.Context, params json.RawMessage, traceID string) (ServiceResult, error)
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
	configStore    bridgeconfig.Store
	sessionStore   *session.Store
	lifecycle      *serviceLifecycle
	runtimeState   *serviceRuntimeState
	actionRouter   *serviceActionRouter
	skillHandler   *bridgeskills.ActionHandler
	agentRunner    SessionTurnRunner
	runRegistry    *RunRegistry
	runtimeFactory AgentRuntimeFactory
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
	service.skillHandler = NewSkillActionHandler(store, service.skillLogFunc())
	service.runtimeFactory = newAgentRuntimeFactoryWithTaskManager(service.taskToolManager())
	service.agentRunner = newServiceAgentRunner(service, executor, streamExecutor)
	registerDefaultActions(service)
	return service
}

func newBridgeServiceState(store bridgeconfig.Store, sessionStore *session.Store) *bridgeService {
	runtimeState := newServiceRuntimeState()
	sessionPush := newSessionPushHub()
	return &bridgeService{
		configStore:  store,
		sessionStore: sessionStore,
		lifecycle:    newServiceLifecycle(runtimeState, sessionPush),
		runtimeState: runtimeState,
		actionRouter: newServiceActionRouter(21),
		runRegistry:  NewRunRegistry(),
	}
}

func newServiceAgentRunner(service *bridgeService, executor agentExecutorFunc, streamExecutor agentStreamExecutorFunc) SessionTurnRunner {
	runner := newSessionTurnRunnerAdapter(service.configStore, service.sessionStore, executor, streamExecutor)
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

func (s *bridgeService) taskToolManager() tools.TaskManager {
	if s == nil {
		return nil
	}
	return s
}

func (s *bridgeService) sessionPushHub() *sessionPushHub {
	if s == nil || s.lifecycle == nil {
		return nil
	}
	return s.lifecycle.sessionPushHub()
}

func (s *bridgeService) taskStore() *TaskStore {
	if s == nil || s.runtimeState == nil {
		return nil
	}
	return s.runtimeState.taskStore()
}

func (s *bridgeService) taskScheduler() *TaskScheduler {
	if s == nil || s.runtimeState == nil {
		return nil
	}
	return s.runtimeState.taskScheduler()
}

func (s *bridgeService) taskInitErr() error {
	if s == nil || s.runtimeState == nil {
		return nil
	}
	return s.runtimeState.taskInitErr()
}

func (s *bridgeService) rssActionHandler() *bridgerss.ActionHandler {
	if s == nil || s.runtimeState == nil {
		return nil
	}
	return s.runtimeState.rssHandler()
}

// StartBackgroundRuntimes 显式初始化 service 依赖的后台 runtime。
func (s *bridgeService) StartBackgroundRuntimes() error {
	if s == nil {
		return nil
	}
	return s.runtimeState.start(s.configStore, s, s.rssLogFunc())
}

// BootstrapSystemTasks 将系统调度任务同步到 task runtime。
func (s *bridgeService) BootstrapSystemTasks() error {
	if s == nil {
		return nil
	}
	return s.runtimeState.bootstrapSystemTasks(s.configStore)
}

func (s *bridgeService) initTaskRuntime() error {
	if s == nil {
		return nil
	}
	return s.runtimeState.initTaskRuntime(s.configStore, s)
}

func (s *bridgeService) initRSSInboxRuntime() error {
	if s == nil {
		return nil
	}
	return s.reloadRSSInboxRuntime()
}

func (s *bridgeService) reloadRSSInboxRuntime() error {
	if s == nil {
		return nil
	}
	return s.runtimeState.reloadRSSInbox(s.configStore, s.rssLogFunc())
}

func (s *bridgeService) rssHandlerInitErr() error {
	if s == nil || s.runtimeState == nil {
		return nil
	}
	return s.runtimeState.rssInitErr()
}

func (s *bridgeService) rssLogFunc() bridgerss.LogFunc {
	return func(traceID, action, status string, err error) {
		logAction(traceID, action, status, err)
	}
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
	s.lifecycle.close()
}

// dispatchAction 根据 action 查找处理器；未知 action 返回显式可选列表。
func (s *bridgeService) dispatchAction(ctx context.Context, action string, params json.RawMessage, traceID string) (ServiceResult, error) {
	return s.actionRouter.dispatch(ctx, action, params, traceID)
}

// unsupportedActionError 构造稳定错误消息，便于客户端快速定位拼写/版本问题。
func (s *bridgeService) unsupportedActionError(action string) error {
	return s.actionRouter.unsupportedActionError(action)
}

func (s *bridgeService) registerAction(action string, handler actionHandler) {
	if s == nil || s.actionRouter == nil {
		return
	}
	s.actionRouter.register(action, handler)
}

func (s *bridgeService) actionHandler(action string) (actionHandler, bool) {
	if s == nil || s.actionRouter == nil {
		return nil, false
	}
	return s.actionRouter.handler(action)
}

func (s *bridgeService) registeredActionNames() []string {
	if s == nil || s.actionRouter == nil {
		return nil
	}
	return s.actionRouter.actionNames()
}

func (s *bridgeService) setRSSHandler(service *bridgerss.RSSInboxService, initErr error) {
	if s == nil || s.runtimeState == nil {
		return
	}
	s.runtimeState.setRSSHandler(service, initErr, s.rssLogFunc())
}
