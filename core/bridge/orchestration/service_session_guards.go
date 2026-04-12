// Session service helper methods shared across session use cases.

package orchestration

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"ghost-os/bridge/session"
)

var errSessionEnded = errors.New("session has already ended")

// requireSessionStore 确保当前 service 已配置持久化会话存储。
func (s *bridgeService) requireSessionStore() (*session.Store, int, error) {
	if s.sessionStore == nil {
		return nil, legacyStatusFromServiceErrorKind(ServiceErrorInternal), errors.New("session store is not configured")
	}
	return s.sessionStore, legacyStatusFromServiceOutcome(ServiceOutcomeSuccess), nil
}

// requireSessionID 对输入 id 做最小合法性校验并返回 trim 后值。
func requireSessionID(id string) (string, int, error) {
	trimmed := strings.TrimSpace(id)
	if trimmed == "" {
		return "", legacyStatusFromServiceErrorKind(ServiceErrorInvalidInput), errors.New("session id is required")
	}
	return trimmed, legacyStatusFromServiceOutcome(ServiceOutcomeSuccess), nil
}

// mapSessionStorageError 将存储层错误映射到稳定的 HTTP 状态码。
func mapSessionStorageError(err error) int {
	switch {
	case errors.Is(err, session.ErrInvalidSessionID):
		return legacyStatusFromServiceErrorKind(ServiceErrorInvalidInput)
	case errors.Is(err, session.ErrSessionNotFound):
		return legacyStatusFromServiceErrorKind(ServiceErrorNotFound)
	case errors.Is(err, errSessionEnded):
		return legacyStatusFromServiceErrorKind(ServiceErrorConflict)
	default:
		return legacyStatusFromServiceErrorKind(ServiceErrorInternal)
	}
}

func mapSessionStorageErrorKind(err error) ServiceErrorKind {
	switch {
	case errors.Is(err, session.ErrInvalidSessionID):
		return ServiceErrorInvalidInput
	case errors.Is(err, session.ErrSessionNotFound):
		return ServiceErrorNotFound
	case errors.Is(err, errSessionEnded):
		return ServiceErrorConflict
	default:
		return ServiceErrorInternal
	}
}

// ensureSessionActive 在继续已有会话前校验其可续跑状态。
func (s *bridgeService) ensureSessionActive(sessionID string) error {
	id := strings.TrimSpace(sessionID)
	if id == "" {
		return nil
	}
	if s == nil || s.sessionStore == nil {
		return wrapServiceError(ServiceErrorInternal, errors.New("session store is not configured"))
	}

	sess, err := s.sessionStore.Load(id)
	if err != nil {
		if errors.Is(err, session.ErrSessionNotFound) {
			return wrapServiceError(
				ServiceErrorNotFound,
				fmt.Errorf("%w: session_id=%s (omit session_id to start a new session)", session.ErrSessionNotFound, id),
			)
		}
		return wrapServiceError(mapSessionStorageErrorKind(err), err)
	}
	if !sess.IsEnded() {
		return nil
	}
	return wrapServiceError(ServiceErrorConflict, fmt.Errorf("%w: session_id=%s", errSessionEnded, id))
}

func (s *bridgeService) ensureSessionNotInflight(sessionID string) error {
	id := strings.TrimSpace(sessionID)
	if id == "" {
		return nil
	}
	if s == nil || s.runRegistry == nil {
		return wrapServiceError(ServiceErrorInternal, errors.New("run registry is not configured"))
	}
	if !s.runRegistry.IsInflight(id) {
		return nil
	}
	return wrapServiceError(ServiceErrorConflict, fmt.Errorf("%w: session_id=%s", ErrSessionInflight, id))
}

// markSessionEnded 在收到结构化结束信号后把会话状态持久化为 ended。
func (s *bridgeService) markSessionEnded(sessionID string) error {
	store := s.sessionStore
	if store == nil {
		return wrapServiceError(ServiceErrorInternal, errors.New("session store is not configured"))
	}
	id := strings.TrimSpace(sessionID)
	if id == "" {
		return wrapServiceError(ServiceErrorInternal, errors.New("session id is empty"))
	}

	sess, err := store.Load(id)
	if err != nil {
		return wrapServiceError(mapSessionStorageErrorKind(err), err)
	}
	if sess.IsEnded() {
		return nil
	}

	sess.MarkEnded(time.Now().UTC())
	if err := store.Save(sess); err != nil {
		return wrapServiceError(mapSessionStorageErrorKind(err), err)
	}
	return nil
}
