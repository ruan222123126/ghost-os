// Session service helper methods shared across session use cases.

package app

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"ghost-os/bridge/session"
)

var errSessionEnded = errors.New("session has already ended")

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
	case errors.Is(err, errSessionEnded):
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}

// ensureSessionActive 在继续已有会话前校验其可续跑状态。
func (s *bridgeService) ensureSessionActive(sessionID string) (int, error) {
	store := s.sessionStore
	if store == nil {
		return http.StatusOK, nil
	}
	id := strings.TrimSpace(sessionID)
	if id == "" {
		return http.StatusOK, nil
	}

	sess, err := store.Load(id)
	if err != nil {
		if errors.Is(err, session.ErrSessionNotFound) {
			return http.StatusOK, nil
		}
		return mapSessionStorageError(err), err
	}
	if !sess.IsEnded() {
		return http.StatusOK, nil
	}
	return http.StatusConflict, fmt.Errorf("%w: session_id=%s", errSessionEnded, id)
}

func (s *bridgeService) ensureSessionNotInflight(sessionID string) (int, error) {
	if s == nil || s.runRegistry == nil {
		return http.StatusOK, nil
	}

	id := strings.TrimSpace(sessionID)
	if id == "" {
		return http.StatusOK, nil
	}
	if !s.runRegistry.IsInflight(id) {
		return http.StatusOK, nil
	}
	return http.StatusConflict, fmt.Errorf("%w: session_id=%s", ErrSessionInflight, id)
}

// markSessionEnded 在收到结构化结束信号后把会话状态持久化为 ended。
func (s *bridgeService) markSessionEnded(sessionID string) (int, error) {
	store := s.sessionStore
	if store == nil {
		return http.StatusInternalServerError, errors.New("session store is not configured")
	}
	id := strings.TrimSpace(sessionID)
	if id == "" {
		return http.StatusInternalServerError, errors.New("session id is empty")
	}

	sess, err := store.Load(id)
	if err != nil {
		return mapSessionStorageError(err), err
	}
	if sess.IsEnded() {
		return http.StatusOK, nil
	}

	sess.MarkEnded(time.Now().UTC())
	if err := store.Save(sess); err != nil {
		return mapSessionStorageError(err), err
	}
	return http.StatusOK, nil
}
