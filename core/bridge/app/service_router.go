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
	taskStore     *TaskStore
	taskScheduler *TaskScheduler
	taskInitErr   error
	memoryManager *memory.MemoryManager
	agentRunner   SessionTurnRunner
	runRegistry   *RunRegistry
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

	runner := newSessionTurnRunnerAdapter(store, sessionStore, executor, streamExecutor)
	if runner == nil {
		runner = NewSessionAgentRunner(newAgentRuntimeFactory(), store, sessionStore, memoryManager, runRegistry)
	}

	service := &bridgeService{
		configStore:   store,
		sessionStore:  sessionStore,
		memoryManager: memoryManager,
		agentRunner:   runner,
		runRegistry:   runRegistry,
		actions:       make(map[string]actionHandler, 18),
	}
	service.initTaskRuntime()

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
	registerAction(service, busActionTaskCreate, func(_ context.Context, params taskCreateParams, traceID string) (any, int, error) {
		return service.executeTaskCreateAction(params, traceID)
	})
	registerAction(service, busActionTaskList, func(_ context.Context, _ map[string]any, traceID string) (any, int, error) {
		return service.executeTaskListAction(traceID)
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
	return service
}

func (s *bridgeService) initTaskRuntime() {
	if s == nil {
		return
	}
	taskStore, err := NewTaskStore(tasksPathFromEnv())
	if err != nil {
		s.taskInitErr = err
		return
	}
	scheduler := NewTaskScheduler(taskStore, s)
	if err := scheduler.Start(); err != nil {
		s.taskInitErr = err
		return
	}
	s.taskStore = taskStore
	s.taskScheduler = scheduler
}

// Close 释放 service 级后台资源。
func (s *bridgeService) Close() {
	if s == nil {
		return
	}
	if s.taskScheduler != nil {
		s.taskScheduler.Stop()
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
