// Session service helper methods shared across session use cases.

package app

import (
	"errors"
	"net/http"
	"strings"

	"ghost-os/bridge/session"
)

// requireSessionStore 确保当前 service 已配置持久化会话存储。
func (s *bridgeService) requireSessionStore() (*session.Store, int, error) {
	if s.sessionStore == nil {
		return nil, http.StatusInternalServerError, errors.New("session store is not configured")
	}
	return s.sessionStore, http.StatusOK, nil
}

// requireSessionID 对输入 id 做最小合法性校验并返回 trim 后值。
func requireSessionID(id string) (string, int, error) {
	trimmed := strings.TrimSpace(id)
	if trimmed == "" {
		return "", http.StatusBadRequest, errors.New("session id is required")
	}
	return trimmed, http.StatusOK, nil
}

// mapSessionStorageError 将存储层错误映射到稳定的 HTTP 状态码。
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
