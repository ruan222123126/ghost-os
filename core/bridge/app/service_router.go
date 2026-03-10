package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"ghost-os/bridge/agent"
	"ghost-os/bridge/memory"
	"ghost-os/bridge/session"
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
	sink agent.EventSink,
) (string, string, error)

// bridgeService 负责 action 分发，不承载 transport 细节。
type bridgeService struct {
	configStore   *ConfigStore
	sessionStore  *session.Store
	sessionPush   *sessionPushHub
	taskStore     *TaskStore
	taskScheduler *TaskScheduler
	taskInitErr   error
	rssInbox      *RSSInboxService
	rssInitErr    error
	memoryManager *memory.MemoryManager
	agentRunner   SessionTurnRunner
	runRegistry   *RunRegistry
	feedStore     *tools.FeedStore
	actions       map[string]actionHandler
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
	memoryManager := memory.NewMemoryManager(memoryManagerConfigFromStore(store, sessionStore))
	runRegistry := NewRunRegistry()

	service := &bridgeService{
		configStore:   store,
		sessionStore:  sessionStore,
		sessionPush:   newSessionPushHub(),
		memoryManager: memoryManager,
		runRegistry:   runRegistry,
		actions:       make(map[string]actionHandler, 21),
	}

	runner := newSessionTurnRunnerAdapter(store, sessionStore, executor, streamExecutor)
	if runner == nil {
		runner = NewSessionAgentRunner(newAgentRuntimeFactoryWithTaskManager(service.taskToolManager()), store, sessionStore, memoryManager, runRegistry)
	}
	service.agentRunner = runner

	registerAction(service, busActionAgentSend, func(ctx context.Context, params agentParams, traceID string) (any, int, error) {
		return service.executeAgentAction(ctx, params, traceID)
	})
	registerAction(service, busActionAgentStop, func(ctx context.Context, params agentStopParams, traceID string) (any, int, error) {
		return service.executeAgentStopAction(ctx, params, traceID)
	})
	registerAction(service, busActionConfigGet, func(_ context.Context, _ map[string]any, traceID string) (any, int, error) {
		return service.executeConfigGetAction(traceID)
	})
	registerAction(service, busActionConfigUpdate, func(_ context.Context, params configUpdateRequest, traceID string) (any, int, error) {
		return service.executeConfigUpdateAction(params, traceID)
	})
	registerAction(service, busActionHumanResponse, func(ctx context.Context, params humanResponseParams, traceID string) (any, int, error) {
		return service.executeHumanResponseAction(ctx, params, traceID)
	})
	registerAction(service, busActionMemoryQuery, func(ctx context.Context, params memoryQueryParams, traceID string) (any, int, error) {
		return service.executeMemoryQueryAction(ctx, params, traceID)
	})
	registerAction(service, busActionMemoryArchive, func(ctx context.Context, params memoryArchiveParams, traceID string) (any, int, error) {
		return service.executeMemoryArchiveAction(ctx, params, traceID)
	})
	registerAction(service, busActionMemoryDecisionQuery, func(ctx context.Context, params memoryDecisionQueryParams, traceID string) (any, int, error) {
		return service.executeMemoryDecisionQueryAction(ctx, params, traceID)
	})
	registerAction(service, busActionMemoryDecisionStats, func(ctx context.Context, params memoryDecisionStatsParams, traceID string) (any, int, error) {
		return service.executeMemoryDecisionStatsAction(ctx, params, traceID)
	})
	registerAction(service, busActionMemoryDecisionRebuild, func(ctx context.Context, params memoryDecisionRebuildParams, traceID string) (any, int, error) {
		return service.executeMemoryDecisionRebuildAction(ctx, params, traceID)
	})
	registerAction(service, busActionMemoryHygieneRun, func(_ context.Context, params memoryHygieneRunParams, traceID string) (any, int, error) {
		return service.executeMemoryHygieneRunAction(params, traceID)
	})
	registerAction(service, busActionMemoryDecisionRelationRun, func(_ context.Context, params memoryDecisionRelationRunParams, traceID string) (any, int, error) {
		return service.executeMemoryDecisionRelationRunAction(params, traceID)
	})
	registerAction(service, busActionMemoryPalaceCurateRun, func(_ context.Context, params memoryPalaceCurateRunParams, traceID string) (any, int, error) {
		return service.executeMemoryPalaceCurateRunAction(params, traceID)
	})
	registerAction(service, busActionTaskCreate, func(_ context.Context, params taskCreateParams, traceID string) (any, int, error) {
		return service.executeTaskCreateAction(params, traceID)
	})
	registerAction(service, busActionTaskList, func(_ context.Context, _ map[string]any, traceID string) (any, int, error) {
		return service.executeTaskListAction(taskListScopeUser, traceID)
	})
	registerAction(service, busActionTaskGet, func(_ context.Context, params taskIDParams, traceID string) (any, int, error) {
		return service.executeTaskGetAction(params, traceID)
	})
	registerAction(service, busActionTaskUpdate, func(_ context.Context, params taskUpdateParams, traceID string) (any, int, error) {
		return service.executeTaskUpdateAction(params, traceID)
	})
	registerAction(service, busActionTaskRunNow, func(_ context.Context, params taskIDParams, traceID string) (any, int, error) {
		return service.executeTaskRunNowAction(params, traceID)
	})
	registerAction(service, busActionTaskLogs, func(_ context.Context, params taskLogsParams, traceID string) (any, int, error) {
		return service.executeTaskLogsAction(params, traceID)
	})
	registerAction(service, busActionTaskDelete, func(_ context.Context, params taskIDParams, traceID string) (any, int, error) {
		return service.executeTaskDeleteAction(params, traceID)
	})
	registerAction(service, busActionRSSInboxPoll, func(ctx context.Context, params rssInboxPollParams, traceID string) (any, int, error) {
		return service.executeRSSInboxPollAction(ctx, params, traceID)
	})
	registerAction(service, busActionRSSInboxList, func(_ context.Context, params rssInboxListParams, traceID string) (any, int, error) {
		return service.executeRSSInboxListAction(params, traceID)
	})
	registerAction(service, busActionRSSInboxGet, func(_ context.Context, params rssInboxGetParams, traceID string) (any, int, error) {
		return service.executeRSSInboxGetAction(params, traceID)
	})
	registerAction(service, busActionRSSInboxGroups, func(_ context.Context, params rssInboxGroupsParams, traceID string) (any, int, error) {
		return service.executeRSSInboxGroupsAction(params, traceID)
	})
	registerAction(service, busActionRSSBriefingBuild, func(ctx context.Context, params rssBriefingParams, traceID string) (any, int, error) {
		return service.executeRSSBriefingBuildAction(ctx, params, traceID)
	})
	registerAction(service, busActionRSSBriefingGet, func(_ context.Context, _ map[string]any, traceID string) (any, int, error) {
		return service.executeRSSBriefingGetAction(traceID)
	})
	return service
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
	taskStore, err := NewTaskStore(tasksPathFromEnv())
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
	s.feedStore = service.feedStore
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
	if s.memoryManager == nil {
		return
	}
	s.memoryManager.StopDreaming()
}

// registerAction 负责“先解码参数，再调用用例”，避免每个 action 重复样板代码。
func registerAction[T any](service *bridgeService, action string, handler func(context.Context, T, string) (any, int, error)) {
	service.actions[action] = func(ctx context.Context, rawParams json.RawMessage, traceID string) (any, int, error) {
		params, err := decodeActionParams[T](rawParams)
		if err != nil {
			return nil, http.StatusBadRequest, err
		}
		return handler(ctx, params, traceID)
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
