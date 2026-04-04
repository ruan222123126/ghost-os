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
