package app

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"ghost-os/bridge/session"
)

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
