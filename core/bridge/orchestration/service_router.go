package orchestration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"

	rsssubscriptions "ghost-os/bridge/rss/subscriptions"
	"ghost-os/bridge/session"
	"ghost-os/bridge/streaming"
	"ghost-os/bridge/tools"
)

type actionHandler func(ctx context.Context, params json.RawMessage, traceID string) (any, int, error)
type agentExecutorFunc func(
	ctx context.Context,
	message string,
	sessionID string,
	traceID string,
	store *ConfigStore,
	sessionStore *session.Store,
) (string, string, error)
type agentStreamExecutorFunc func(
	ctx context.Context,
	message string,
	sessionID string,
	traceID string,
	store *ConfigStore,
	sessionStore *session.Store,
	sink streaming.Sink,
) (string, string, error)

// bridgeService 负责 action 分发，不承载 transport 细节。
type bridgeService struct {
	configStore    *ConfigStore
	sessionStore   *session.Store
	sessionPush    *sessionPushHub
	taskStore      *TaskStore
	taskScheduler  *TaskScheduler
	taskInitErr    error
	rssInbox       *RSSInboxService
	rssInitErr     error
	agentRunner    SessionTurnRunner
	runRegistry    *RunRegistry
	feedStore      *rsssubscriptions.FeedStore
	runtimeFactory AgentRuntimeFactory
	actions        map[string]actionHandler
}

// newBridgeService 组装 action -> handler 映射，并初始化会话与记忆依赖。
func newBridgeService(store *ConfigStore, sessionStore *session.Store, executor agentExecutorFunc) *bridgeService {
	return newBridgeServiceWithStreamExecutor(store, sessionStore, executor, nil)
}

func newBridgeServiceWithStreamExecutor(
	store *ConfigStore,
	sessionStore *session.Store,
	executor agentExecutorFunc,
	streamExecutor agentStreamExecutorFunc,
) *bridgeService {
	service := newBridgeServiceState(store, sessionStore)
	service.runtimeFactory = newAgentRuntimeFactoryWithTaskManager(service.taskToolManager())
	service.agentRunner = newServiceAgentRunner(service, executor, streamExecutor)
	registerDefaultActions(service)
	return service
}

func newBridgeServiceState(store *ConfigStore, sessionStore *session.Store) *bridgeService {
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
	if err := s.ensureRSSPollTask(); err != nil {
		return err
	}
	if err := s.ensureRSSBriefingTask(); err != nil {
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
		s.rssInitErr = err
		s.rssInbox = nil
		s.feedStore = nil
		return err
	}
	s.rssInbox = service
	s.feedStore = service.FeedStore()
	s.rssInitErr = nil
	return nil
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
