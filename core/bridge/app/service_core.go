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
	configStore         *ConfigStore
	sessionStore        *session.Store
	memoryManager       *memory.MemoryManager
	agentExecutor       agentExecutorFunc
	agentExecutorStream agentStreamExecutorFunc
	runRegistry         *RunRegistry
	actions             map[string]actionHandler
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
	memoryManager := memory.NewMemoryManager(memory.MemoryConfig{
		WarmCapacity:      agentWarmMemoryCapacity,
		WarmPath:          memoryWarmPathFromEnv(),
		ColdBaseDir:       memoryColdPathFromEnv(),
		AutoRecallEnabled: memoryAutoRecallEnabledFromEnv(),
		AutoRecallLimit:   memoryAutoRecallLimitFromEnv(),
		WarmTTL:           memoryWarmTTLFromEnv(),
		EvolutionInterval: memoryEvolutionIntervalFromEnv(),
		EvolutionEnabled:  memoryEvolutionEnabledFromEnv(),
		SessionStore:      sessionStore,
	})
	runRegistry := NewRunRegistry()

	useDefaultExecutor := executor == nil
	if useDefaultExecutor {
		executor = newSessionAgentExecutor(memoryManager, runRegistry)
	}
	if streamExecutor == nil {
		if useDefaultExecutor {
			streamExecutor = newSessionAgentStreamExecutor(memoryManager, runRegistry)
		} else {
			streamExecutor = func(
				ctx context.Context,
				message string,
				sessionID string,
				traceID string,
				store *ConfigStore,
				sessionStore *session.Store,
				_ agent.EventSink,
			) (string, string, error) {
				return executor(ctx, message, sessionID, traceID, store, sessionStore)
			}
		}
	}

	service := &bridgeService{
		configStore:         store,
		sessionStore:        sessionStore,
		memoryManager:       memoryManager,
		agentExecutor:       executor,
		agentExecutorStream: streamExecutor,
		runRegistry:         runRegistry,
		actions:             make(map[string]actionHandler, 7),
	}

	registerAction(service, actionAgentSend, func(ctx context.Context, params agentParams, traceID string) (any, int, error) {
		return service.executeAgentAction(ctx, params, traceID)
	})
	registerAction(service, actionAgentStop, func(ctx context.Context, params agentStopParams, traceID string) (any, int, error) {
		return service.executeAgentStopAction(ctx, params, traceID)
	})
	registerAction(service, actionConfigGet, func(_ context.Context, _ map[string]any, traceID string) (any, int, error) {
		return service.executeConfigGetAction(traceID)
	})
	registerAction(service, actionConfigUpdate, func(_ context.Context, params configUpdateRequest, traceID string) (any, int, error) {
		return service.executeConfigUpdateAction(params, traceID)
	})
	registerAction(service, actionHumanResponse, func(ctx context.Context, params humanResponseParams, traceID string) (any, int, error) {
		return service.executeHumanResponseAction(ctx, params, traceID)
	})
	registerAction(service, actionMemoryQuery, func(ctx context.Context, params memoryQueryParams, traceID string) (any, int, error) {
		return service.executeMemoryQueryAction(ctx, params, traceID)
	})
	registerAction(service, actionMemoryArchive, func(ctx context.Context, params memoryArchiveParams, traceID string) (any, int, error) {
		return service.executeMemoryArchiveAction(ctx, params, traceID)
	})
	return service
}

// Close 释放 service 级后台资源。
func (s *bridgeService) Close() {
	if s == nil || s.memoryManager == nil {
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
