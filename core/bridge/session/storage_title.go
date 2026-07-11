package session

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

// UpdateTitle updates only the session title inside state_json.
func (s *Store) UpdateTitle(sessionID string, title string) error {
	id, err := normalizeSessionID(sessionID)
	if err != nil {
		return err
	}
	normalizedTitle := strings.TrimSpace(title)
	if normalizedTitle == "" {
		return errors.New("session title cannot be empty")
	}
	return s.withSessionLock(id, true, func() error {
		return s.withTx("update session title", func(tx *sql.Tx) error {
			return updateSessionTitleTx(tx, id, normalizedTitle)
		})
	})
}

func updateSessionTitleTx(tx *sql.Tx, sessionID string, title string) error {
	record, err := loadSessionRecordTx(tx, sessionID)
	if err != nil {
		return err
	}
	record.State.Title = title
	stateJSON, err := encodeSessionRecordState(record.State)
	if err != nil {
		return err
	}
	if _, err := tx.Exec(`UPDATE sessions SET state_json = ? WHERE id = ?`, stateJSON, sessionID); err != nil {
		return fmt.Errorf("update session title: %w", err)
	}
	return nil
}
