package orchestration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"

	bridgeconfig "ghost-os/bridge/config"
	bridgerss "ghost-os/bridge/rss"
	"ghost-os/bridge/session"
	bridgeskills "ghost-os/bridge/skills"
	"ghost-os/bridge/streaming"
	"ghost-os/bridge/tools"
)

type actionHandler func(ctx context.Context, params json.RawMessage, traceID string) (any, int, error)
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
	sessionPush    *sessionPushHub
	taskStore      *TaskStore
	taskScheduler  *TaskScheduler
	taskInitErr    error
	rssHandler     *bridgerss.ActionHandler
	skillHandler   *bridgeskills.ActionHandler
	agentRunner    SessionTurnRunner
	runRegistry    *RunRegistry
	runtimeFactory AgentRuntimeFactory
	actions        map[string]actionHandler
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
	return &bridgeService{
		configStore:  store,
		sessionStore: sessionStore,
		sessionPush:  newSessionPushHub(),
		runRegistry:  NewRunRegistry(),
		actions:      make(map[string]actionHandler, 21),
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

// StartBackgroundRuntimes 显式初始化 service 依赖的后台 runtime。
func (s *bridgeService) StartBackgroundRuntimes() error {
	if s == nil {
		return nil
	}
	if err := s.initRSSInboxRuntime(); err != nil {
		return err
	}
	return s.initTaskRuntime()
}

// BootstrapSystemTasks 将系统调度任务同步到 task runtime。
func (s *bridgeService) BootstrapSystemTasks() error {
	if s == nil {
		return nil
	}
	coordinator := bridgerss.NewSystemTaskCoordinator(
		s.configStore,
		s.taskStore,
		s.taskScheduler,
		s.rssHandlerInitErr(),
	)
	if err := coordinator.SyncPollTask(); err != nil {
		return err
	}
	if err := coordinator.SyncBriefingTask(); err != nil {
		return err
	}
	return nil
}

func (s *bridgeService) initTaskRuntime() error {
	if s == nil {
		return nil
	}
	if s.taskStore != nil && s.taskScheduler != nil {
		s.taskInitErr = nil
		return nil
	}
	taskCfg, err := loadTaskRuntimeConfig(s.configStore)
	if err != nil {
		s.taskInitErr = err
		return err
	}
	taskStore, err := NewTaskStore(taskCfg.TasksPath)
	if err != nil {
		s.taskInitErr = err
		return err
	}
	scheduler := NewTaskScheduler(taskStore, s)
	if err := scheduler.Start(); err != nil {
		s.taskInitErr = err
		return err
	}
	s.taskStore = taskStore
	s.taskScheduler = scheduler
	s.taskInitErr = nil
	return nil
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
	service, err := newRSSInboxServiceFromConfig(s.configStore)
	if err != nil {
		s.rssHandler = bridgerss.NewActionHandler(nil, err, s.rssLogFunc())
		return err
	}
	s.rssHandler = bridgerss.NewActionHandler(service, nil, s.rssLogFunc())
	return nil
}

func (s *bridgeService) rssHandlerInitErr() error {
	if s == nil || s.rssHandler == nil {
		return nil
	}
	return s.rssHandler.InitErr()
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
	if s.taskScheduler != nil {
		s.taskScheduler.Stop()
	}
	if s.sessionPush != nil {
		s.sessionPush.Close()
	}
}

// dispatchAction 根据 action 查找处理器；未知 action 返回显式可选列表。
func (s *bridgeService) dispatchAction(ctx context.Context, action string, params json.RawMessage, traceID string) (any, int, error) {
	handler, ok := s.actions[action]
	if !ok {
		return nil, http.StatusBadRequest, s.unsupportedActionError(action)
	}
	return handler(ctx, params, traceID)
}

// unsupportedActionError 构造稳定错误消息，便于客户端快速定位拼写/版本问题。
func (s *bridgeService) unsupportedActionError(action string) error {
	registered := make([]string, 0, len(s.actions))
	for name := range s.actions {
		registered = append(registered, name)
	}
	sort.Strings(registered)
	return fmt.Errorf(
		"unsupported action %q, expected one of: %s",
		action,
		strings.Join(registered, "|"),
	)
}
