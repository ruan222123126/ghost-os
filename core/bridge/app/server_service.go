package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

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

// bridgeService 负责 bus action 分发与业务行为实现。
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

func decodeActionParams[T any](raw json.RawMessage) (T, error) {
	var params T
	if err := decodeParams(raw, &params); err != nil {
		return params, err
	}
	return params, nil
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

func (s *bridgeService) executeAgentAction(ctx context.Context, params agentParams, traceID string) (any, int, error) {
	trimmed := strings.TrimSpace(params.Message)
	if trimmed == "" {
		return nil, http.StatusBadRequest, errors.New("message is required")
	}

	trimmedSessionID := strings.TrimSpace(params.SessionID)
	logAction(traceID, actionAgentSend, "running", nil)
	response, sessionID, err := s.agentExecutor(ctx, trimmed, trimmedSessionID, traceID, s.configStore, s.sessionStore)
	if err != nil {
		logAction(traceID, actionAgentSend, "error", err)
		if errors.Is(err, session.ErrInvalidSessionID) {
			return nil, http.StatusBadRequest, err
		}
		return nil, http.StatusInternalServerError, err
	}
	logAction(traceID, actionAgentSend, "success", nil)
	return agentResponse{
		Message:   response,
		SessionID: sessionID,
	}, http.StatusOK, nil
}

func (s *bridgeService) executeConfigGetAction(traceID string) (any, int, error) {
	logAction(traceID, actionConfigGet, "success", nil)
	return s.configStore.Snapshot(), http.StatusOK, nil
}

func (s *bridgeService) executeConfigUpdateAction(req configUpdateRequest, traceID string) (any, int, error) {
	logAction(traceID, actionConfigUpdate, "running", nil)
	if err := s.configStore.Update(req); err != nil {
		logAction(traceID, actionConfigUpdate, "error", err)
		return nil, http.StatusBadRequest, err
	}
	logAction(traceID, actionConfigUpdate, "success", nil)
	return s.configStore.Snapshot(), http.StatusOK, nil
}

func (s *bridgeService) executeSessionsListAction(traceID string) (any, int, error) {
	if s.sessionStore == nil {
		return nil, http.StatusInternalServerError, errors.New("session store is not configured")
	}

	sessionIDs, err := s.sessionStore.List()
	if err != nil {
		logAction(traceID, "SESSIONS_LIST", "error", err)
		return nil, http.StatusInternalServerError, err
	}

	metadata := make([]sessionMetadata, 0, len(sessionIDs))
	for _, id := range sessionIDs {
		sess, loadErr := s.sessionStore.Load(id)
		if loadErr != nil {
			if errors.Is(loadErr, session.ErrSessionNotFound) {
				continue
			}
			logAction(traceID, "SESSIONS_LIST", "error", loadErr)
			return nil, http.StatusInternalServerError, loadErr
		}

		metadata = append(metadata, sessionMetadata{
			ID:           sess.ID,
			CreatedAt:    sess.CreatedAt.UTC().Format(time.RFC3339),
			UpdatedAt:    sess.UpdatedAt.UTC().Format(time.RFC3339),
			MessageCount: len(sess.Messages),
			TokenCount:   sess.TokenCount,
		})
	}

	logAction(traceID, "SESSIONS_LIST", "success", nil)
	return metadata, http.StatusOK, nil
}

func (s *bridgeService) executeSessionGetAction(params sessionIDParams, traceID string) (any, int, error) {
	if s.sessionStore == nil {
		return nil, http.StatusInternalServerError, errors.New("session store is not configured")
	}

	id := strings.TrimSpace(params.ID)
	if id == "" {
		return nil, http.StatusBadRequest, errors.New("session id is required")
	}

	sess, err := s.sessionStore.Load(id)
	if err != nil {
		logAction(traceID, "SESSION_GET", "error", err)
		switch {
		case errors.Is(err, session.ErrInvalidSessionID):
			return nil, http.StatusBadRequest, err
		case errors.Is(err, session.ErrSessionNotFound):
			return nil, http.StatusNotFound, err
		default:
			return nil, http.StatusInternalServerError, err
		}
	}

	logAction(traceID, "SESSION_GET", "success", nil)
	return sessionDetail{
		ID:         sess.ID,
		Messages:   sess.Messages,
		CreatedAt:  sess.CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt:  sess.UpdatedAt.UTC().Format(time.RFC3339),
		TokenCount: sess.TokenCount,
	}, http.StatusOK, nil
}

func (s *bridgeService) executeSessionDeleteAction(params sessionIDParams, traceID string) (any, int, error) {
	if s.sessionStore == nil {
		return nil, http.StatusInternalServerError, errors.New("session store is not configured")
	}

	id := strings.TrimSpace(params.ID)
	if id == "" {
		return nil, http.StatusBadRequest, errors.New("session id is required")
	}

	if err := s.sessionStore.Delete(id); err != nil {
		logAction(traceID, "SESSION_DELETE", "error", err)
		switch {
		case errors.Is(err, session.ErrInvalidSessionID):
			return nil, http.StatusBadRequest, err
		case errors.Is(err, session.ErrSessionNotFound):
			return nil, http.StatusNotFound, err
		default:
			return nil, http.StatusInternalServerError, err
		}
	}

	logAction(traceID, "SESSION_DELETE", "success", nil)
	return sessionDeleteResponse{
		ID:      id,
		Deleted: true,
	}, http.StatusOK, nil
}

func validateBusRequest(req apiRequest) error {
	if strings.TrimSpace(req.Action) == "" {
		return errors.New("action is required")
	}
	if strings.TrimSpace(req.TraceID) == "" {
		return errors.New("trace_id is required")
	}

	source := bytes.TrimSpace(req.Params)
	if len(source) == 0 {
		return errors.New("params is required")
	}
	if bytes.Equal(source, []byte("null")) {
		return errors.New("params must be an object")
	}

	var object map[string]any
	if err := json.Unmarshal(source, &object); err != nil {
		return fmt.Errorf("params must be an object: %w", err)
	}
	if object == nil {
		return errors.New("params must be an object")
	}
	return nil
}

func decodeParams(raw json.RawMessage, target any) error {
	source := bytes.TrimSpace(raw)
	if len(source) == 0 || bytes.Equal(source, []byte("null")) {
		source = []byte("{}")
	}

	decoder := json.NewDecoder(bytes.NewReader(source))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("invalid params: %w", err)
	}

	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err == nil {
		return errors.New("invalid params: multiple JSON values are not allowed")
	} else if !errors.Is(err, io.EOF) {
		return fmt.Errorf("invalid params: %w", err)
	}
	return nil
}
