package session

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"ghost-os/bridge/llm"
)

func (s *Store) importLegacySessionLocked(sessionID string) error {
	if s == nil {
		return ErrSessionNotFound
	}
	if legacyExists, err := s.legacySessionExistsLocked(sessionID); err != nil {
		return err
	} else if !legacyExists {
		return nil
	}

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin legacy session import: %w", err)
	}
	defer rollbackTx(tx)

	if _, err := loadSessionRecordTx(tx, sessionID); err == nil {
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit legacy session import: %w", err)
		}
		return removeLegacySessionFile(s, sessionID)
	} else if !errors.Is(err, ErrSessionNotFound) {
		return err
	}

	legacy, err := s.readLegacySessionLocked(sessionID)
	if err != nil {
		return err
	}
	if err := importLegacySessionTx(tx, legacy); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit legacy session import: %w", err)
	}
	return removeLegacySessionFile(s, sessionID)
}

func (s *Store) legacySessionExistsLocked(sessionID string) (bool, error) {
	path, err := s.legacyPathForSession(sessionID)
	if err != nil {
		return false, err
	}
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("stat legacy session %q: %w", sessionID, err)
	}
	return !info.IsDir(), nil
}

func (s *Store) readLegacySessionLocked(sessionID string) (*Session, error) {
	path, err := s.legacyPathForSession(sessionID)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("%w: %s", ErrSessionNotFound, strings.TrimSpace(sessionID))
		}
		return nil, fmt.Errorf("read legacy session %q: %w", strings.TrimSpace(sessionID), err)
	}
	loaded, err := decodeStoredSession(strings.TrimSpace(sessionID), data, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	loaded.persistedMessageCount = 0
	loaded.persistedMessages = nil
	return loaded, nil
}

func importLegacySessionTx(tx *sql.Tx, sess *Session) error {
	if sess == nil {
		return errors.New("legacy session is nil")
	}
	sess.RecalculateTokenCount()
	sess.TokenCount = sess.WindowTokenCount
	sess.MessageCount = len(sess.Messages)
	sess.WindowStart = 0
	sess.WindowTokenCount = sess.TokenCount
	if sess.CreatedAt.IsZero() {
		sess.CreatedAt = time.Now().UTC()
	}
	if sess.UpdatedAt.IsZero() {
		sess.UpdatedAt = sess.CreatedAt
	}
	return upsertSessionTx(tx, sess, nil, llm.CloneMessages(sess.Messages), sess.TokenCount)
}
