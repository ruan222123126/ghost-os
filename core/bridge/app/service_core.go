package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"

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

// bridgeService 负责 action 分发，不承载 transport 细节。
type bridgeService struct {
	configStore   *ConfigStore
	sessionStore  *session.Store
	agentExecutor agentExecutorFunc
	actions       map[string]actionHandler
}

func newBridgeService(store *ConfigStore, sessionStore *session.Store, executor agentExecutorFunc) *bridgeService {
	service := &bridgeService{
		configStore:   store,
		sessionStore:  sessionStore,
		agentExecutor: executor,
		actions:       make(map[string]actionHandler, 3),
	}

	registerAction(service, actionAgentSend, func(ctx context.Context, params agentParams, traceID string) (any, int, error) {
		return service.executeAgentAction(ctx, params, traceID)
	})
	registerAction(service, actionConfigGet, func(_ context.Context, _ map[string]any, traceID string) (any, int, error) {
		return service.executeConfigGetAction(traceID)
	})
	registerAction(service, actionConfigUpdate, func(_ context.Context, params configUpdateRequest, traceID string) (any, int, error) {
		return service.executeConfigUpdateAction(params, traceID)
	})
	return service
}

func registerAction[T any](service *bridgeService, action string, handler func(context.Context, T, string) (any, int, error)) {
	service.actions[action] = func(ctx context.Context, rawParams json.RawMessage, traceID string) (any, int, error) {
		params, err := decodeActionParams[T](rawParams)
		if err != nil {
			return nil, http.StatusBadRequest, err
		}
		return handler(ctx, params, traceID)
	}
}

func (s *bridgeService) dispatchAction(ctx context.Context, action string, params json.RawMessage, traceID string) (any, int, error) {
	handler, ok := s.actions[action]
	if !ok {
		return nil, http.StatusBadRequest, s.unsupportedActionError(action)
	}
	return handler(ctx, params, traceID)
}

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
