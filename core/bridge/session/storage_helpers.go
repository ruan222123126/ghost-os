package session

import (
	"database/sql"
	"fmt"
	"strings"
)

func normalizeSessionID(sessionID string) (string, error) {
	id := strings.TrimSpace(sessionID)
	if !isValidSessionID(id) {
		return "", fmt.Errorf("%w: %q", ErrInvalidSessionID, sessionID)
	}
	return id, nil
}

func rollbackTx(tx *sql.Tx) {
	if tx != nil {
		_ = tx.Rollback()
	}
}

func replaceLoadedSession(dst *Session, src *Session) {
	if dst == nil || src == nil {
		return
	}
	*dst = *src
	dst.setPersistedSnapshot()
}

func (s *Store) withStoreLock(fn func() error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return fn()
}

func (s *Store) withTx(action string, fn func(tx *sql.Tx) error) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin %s: %w", action, err)
	}
	defer rollbackTx(tx)

	if err := fn(tx); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit %s: %w", action, err)
	}
	return nil
}
