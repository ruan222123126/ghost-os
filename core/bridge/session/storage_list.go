package session

import (
	"fmt"
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

	rows, err := s.db.Query(`
SELECT id, created_at, updated_at, message_count, token_count, state_json
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
