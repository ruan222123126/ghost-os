package app

import (
	"errors"
	"net/http"
	"time"

	"ghost-os/bridge/session"
)

func (s *bridgeService) executeSessionsListAction(traceID string) (any, int, error) {
	store, code, err := s.requireSessionStore()
	if err != nil {
		return nil, code, err
	}

	sessionIDs, err := store.List()
	if err != nil {
		logAction(traceID, "SESSIONS_LIST", "error", err)
		return nil, http.StatusInternalServerError, err
	}

	metadata := make([]sessionMetadata, 0, len(sessionIDs))
	for _, id := range sessionIDs {
		sess, loadErr := store.Load(id)
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
	store, code, err := s.requireSessionStore()
	if err != nil {
		return nil, code, err
	}

	id, code, err := requireSessionID(params.ID)
	if err != nil {
		return nil, code, err
	}

	sess, err := store.Load(id)
	if err != nil {
		logAction(traceID, "SESSION_GET", "error", err)
		return nil, mapSessionStorageError(err), err
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
	store, code, err := s.requireSessionStore()
	if err != nil {
		return nil, code, err
	}

	id, code, err := requireSessionID(params.ID)
	if err != nil {
		return nil, code, err
	}

	if err := store.Delete(id); err != nil {
		logAction(traceID, "SESSION_DELETE", "error", err)
		return nil, mapSessionStorageError(err), err
	}

	logAction(traceID, "SESSION_DELETE", "success", nil)
	return sessionDeleteResponse{
		ID:      id,
		Deleted: true,
	}, http.StatusOK, nil
}
