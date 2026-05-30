package session

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
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

func removeLegacySessionFile(store *Store, sessionID string) error {
	if store == nil {
		return nil
	}
	path, err := store.legacyPathForSession(sessionID)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove legacy session %q: %w", sessionID, err)
	}
	return nil
}

func deleteLegacySessionFile(store *Store, sessionID string) (bool, error) {
	if store == nil {
		return false, nil
	}
	path, err := store.legacyPathForSession(sessionID)
	if err != nil {
		return false, err
	}
	if err := os.Remove(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("delete legacy session %q: %w", sessionID, err)
	}
	return true, nil
}

func (s *Store) withStoreLock(fn func() error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return fn()
}

func (s *Store) withSessionLock(sessionID string, allowMissingLegacy bool, fn func() error) error {
	_ = sessionID
	_ = allowMissingLegacy
	return s.withStoreLock(fn)
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
