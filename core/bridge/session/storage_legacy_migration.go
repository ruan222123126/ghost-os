package session

import (
	"fmt"
	"os"
	"strings"
)

// LegacySessionIDs 返回当前 sessions 目录中仍未显式迁移的 legacy JSON 会话文件 ID。
func (s *Store) LegacySessionIDs() ([]string, error) {
	if s == nil {
		return nil, ErrSessionNotFound
	}

	var ids []string
	err := s.withStoreLock(func() error {
		next, scanErr := s.legacySessionIDsLocked()
		if scanErr != nil {
			return scanErr
		}
		ids = next
		return nil
	})
	if err != nil {
		return nil, err
	}
	return ids, nil
}

// MigrateLegacySessions 将 legacy JSON 会话显式导入 SQLite，并在成功后删除源文件。
func (s *Store) MigrateLegacySessions() ([]string, error) {
	if s == nil {
		return nil, ErrSessionNotFound
	}

	migrated := make([]string, 0)
	err := s.withStoreLock(func() error {
		ids, scanErr := s.legacySessionIDsLocked()
		if scanErr != nil {
			return scanErr
		}
		for _, sessionID := range ids {
			if err := s.importLegacySessionLocked(sessionID); err != nil {
				return fmt.Errorf("migrate legacy session %q: %w", sessionID, err)
			}
			migrated = append(migrated, sessionID)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return migrated, nil
}

func (s *Store) legacySessionIDsLocked() ([]string, error) {
	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		return nil, fmt.Errorf("scan legacy sessions: %w", err)
	}

	ids := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".json") {
			continue
		}
		id := strings.TrimSpace(strings.TrimSuffix(name, ".json"))
		if !isValidSessionID(id) {
			continue
		}
		ids = append(ids, id)
	}
	return ids, nil
}
