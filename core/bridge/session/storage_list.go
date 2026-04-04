package session

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
)

// List 返回当前存储目录下的全部会话 ID。
func (s *Store) List() ([]string, error) {
	metadata, err := s.ListMetadata()
	if err != nil {
		return nil, err
	}

	ids := make([]string, 0, len(metadata))
	for _, item := range metadata {
		ids = append(ids, item.ID)
	}
	return ids, nil
}

// ListMetadata 返回全部会话的轻量元数据。
func (s *Store) ListMetadata() ([]SessionMetadata, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.importLegacySessionsForListingLocked()
	rows, err := s.db.Query(`
SELECT id, created_at, updated_at, message_count, token_count
FROM sessions
ORDER BY id ASC`)
	if err != nil {
		return nil, fmt.Errorf("list session metadata: %w", err)
	}
	defer rows.Close()

	metadata := make([]SessionMetadata, 0, 16)
	for rows.Next() {
		item, err := scanSessionMetadata(rows)
		if err != nil {
			return nil, err
		}
		metadata = append(metadata, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list session metadata: %w", err)
	}
	return metadata, nil
}

func (s *Store) importLegacySessionsForListingLocked() {
	entries, err := os.ReadDir(s.baseDir)
	if err != nil {
		log.Printf("[SESSION] ListMetadata failed to scan legacy files: err=%v", err)
		return
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".json") {
			continue
		}
		id := strings.TrimSuffix(name, ".json")
		if !isValidSessionID(id) {
			continue
		}
		if err := s.importLegacySessionLocked(id); err != nil && !errors.Is(err, ErrSessionNotFound) {
			log.Printf("[SESSION] ListMetadata skipped legacy session import: id=%s err=%v", id, err)
		}
	}
}
