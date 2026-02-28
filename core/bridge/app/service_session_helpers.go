package app

import (
	"errors"
	"net/http"
	"strings"

	"ghost-os/bridge/session"
)

func (s *bridgeService) requireSessionStore() (*session.Store, int, error) {
	if s.sessionStore == nil {
		return nil, http.StatusInternalServerError, errors.New("session store is not configured")
	}
	return s.sessionStore, http.StatusOK, nil
}

func requireSessionID(id string) (string, int, error) {
	trimmed := strings.TrimSpace(id)
	if trimmed == "" {
		return "", http.StatusBadRequest, errors.New("session id is required")
	}
	return trimmed, http.StatusOK, nil
}

func mapSessionStorageError(err error) int {
	switch {
	case errors.Is(err, session.ErrInvalidSessionID):
		return http.StatusBadRequest
	case errors.Is(err, session.ErrSessionNotFound):
		return http.StatusNotFound
	default:
		return http.StatusInternalServerError
	}
}
