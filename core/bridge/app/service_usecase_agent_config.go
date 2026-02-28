package app

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"ghost-os/bridge/agent"
	"ghost-os/bridge/session"
)

func (s *bridgeService) executeAgentAction(ctx context.Context, params agentParams, traceID string) (any, int, error) {
	trimmed := strings.TrimSpace(params.Message)
	trimmedSessionID := strings.TrimSpace(params.SessionID)
	if trimmed == "" && trimmedSessionID == "" {
		return nil, http.StatusBadRequest, errors.New("message is required")
	}

	logAction(traceID, actionAgentSend, "running", nil)
	response, sessionID, err := s.agentExecutor(ctx, trimmed, trimmedSessionID, traceID, s.configStore, s.sessionStore)
	if err != nil {
		var awaitingErr *agent.ErrAwaitingHuman
		if errors.As(err, &awaitingErr) {
			logAction(traceID, actionAgentSend, "awaiting_human", nil)
			return askHumanAwaitingResponse{
				Status:     "awaiting_human",
				SessionID:  sessionID,
				QuestionID: awaitingErr.QuestionID,
				Prompt:     awaitingErr.Prompt,
			}, http.StatusAccepted, nil
		}

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

func (s *bridgeService) executeHumanResponseAction(_ context.Context, params humanResponseParams, traceID string) (any, int, error) {
	store, code, err := s.requireSessionStore()
	if err != nil {
		return nil, code, err
	}

	sessionID, code, err := requireSessionID(params.SessionID)
	if err != nil {
		return nil, code, err
	}

	questionID := strings.TrimSpace(params.QuestionID)
	if questionID == "" {
		return nil, http.StatusBadRequest, errors.New("question_id is required")
	}

	answer := strings.TrimSpace(params.Answer)
	if answer == "" {
		return nil, http.StatusBadRequest, errors.New("answer is required")
	}

	logAction(traceID, actionHumanResponse, "running", nil)
	sess, err := store.Load(sessionID)
	if err != nil {
		statusCode := mapSessionStorageError(err)
		logAction(traceID, actionHumanResponse, "error", err)
		return nil, statusCode, err
	}

	if !sess.SetHumanAnswer(questionID, answer) {
		err = errors.New("question not found in pending questions")
		logAction(traceID, actionHumanResponse, "error", err)
		return nil, http.StatusNotFound, err
	}

	if err := store.Save(sess); err != nil {
		statusCode := mapSessionStorageError(err)
		logAction(traceID, actionHumanResponse, "error", err)
		return nil, statusCode, err
	}

	logAction(traceID, actionHumanResponse, "success", nil)
	return humanResponseAck{
		SessionID:  sessionID,
		QuestionID: questionID,
		Accepted:   true,
	}, http.StatusOK, nil
}
