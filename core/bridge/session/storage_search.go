package session

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"ghost-os/bridge/llm"
)

// SearchMetadata returns session metadata matched by metadata, sidebar partition names, or user-visible messages.
func (s *Store) SearchMetadata(query string, limit int) ([]SessionMetadata, error) {
	normalizedQuery := normalizeSessionSearchQuery(query)

	var matches []SessionMetadata
	err := s.withStoreLock(func() error {
		partitionState, err := s.loadSearchPartitionStateLocked(normalizedQuery)
		if err != nil {
			return err
		}
		return s.withTx("search session metadata", func(tx *sql.Tx) error {
			loaded, searchErr := searchMetadataTx(tx, normalizedQuery, limit, partitionState)
			if searchErr != nil {
				return searchErr
			}
			matches = loaded
			return nil
		})
	})
	if err != nil {
		return nil, err
	}
	return matches, nil
}

func (s *Store) loadSearchPartitionStateLocked(normalizedQuery string) (SessionSidebarPartitionState, error) {
	if normalizedQuery == "" {
		return SessionSidebarPartitionState{}, nil
	}

	state, changed, err := s.loadSidebarPartitionStateLocked()
	if err != nil {
		return SessionSidebarPartitionState{}, err
	}
	if !changed {
		return state, nil
	}
	return state, s.writeSidebarPartitionStateLocked(state)
}

func searchMetadataTx(
	tx *sql.Tx,
	normalizedQuery string,
	limit int,
	partitionState SessionSidebarPartitionState,
) ([]SessionMetadata, error) {
	metadata, err := loadSearchMetadataTx(tx)
	if err != nil {
		return nil, err
	}
	if normalizedQuery == "" {
		return limitSessionMetadata(metadata, limit), nil
	}

	partitionSessionIDs := matchingSearchPartitionSessionIDs(partitionState, normalizedQuery)
	messageSessionIDs, err := matchingSearchMessageSessionIDsTx(tx, normalizedQuery)
	if err != nil {
		return nil, err
	}

	matches := make([]SessionMetadata, 0, len(metadata))
	for _, item := range metadata {
		if !searchMetadataMatches(item, normalizedQuery, partitionSessionIDs, messageSessionIDs) {
			continue
		}
		matches = append(matches, item)
		if limit > 0 && len(matches) >= limit {
			return matches, nil
		}
	}
	return matches, nil
}

func loadSearchMetadataTx(tx *sql.Tx) ([]SessionMetadata, error) {
	rows, err := tx.Query(`
SELECT id, created_at, updated_at, message_count, token_count, state_json
FROM sessions
ORDER BY updated_at DESC, created_at DESC, id DESC`)
	if err != nil {
		return nil, fmt.Errorf("search session metadata: %w", err)
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
		return nil, fmt.Errorf("search session metadata: %w", err)
	}
	return metadata, nil
}

func matchingSearchMessageSessionIDsTx(tx *sql.Tx, normalizedQuery string) (map[string]struct{}, error) {
	rows, err := tx.Query(`
SELECT session_id, idx, message_json
FROM session_messages
ORDER BY session_id ASC, idx ASC`)
	if err != nil {
		return nil, fmt.Errorf("search session messages: %w", err)
	}
	defer rows.Close()

	matches := map[string]struct{}{}
	for rows.Next() {
		sessionID, message, err := scanSearchMessage(rows)
		if err != nil {
			return nil, err
		}
		if searchMessageMatches(message, normalizedQuery) {
			matches[sessionID] = struct{}{}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("search session messages: %w", err)
	}
	return matches, nil
}

func scanSearchMessage(scanner interface{ Scan(...any) error }) (string, llm.Message, error) {
	var (
		sessionID  string
		index      int
		rawMessage string
	)
	if err := scanner.Scan(&sessionID, &index, &rawMessage); err != nil {
		return "", llm.Message{}, fmt.Errorf("scan session search message: %w", err)
	}

	var message llm.Message
	if err := json.Unmarshal([]byte(rawMessage), &message); err != nil {
		return "", llm.Message{}, fmt.Errorf("%w: session message[%d]: %v", ErrSessionCorrupted, index, err)
	}
	return sessionID, message, nil
}

func searchMetadataMatches(
	item SessionMetadata,
	normalizedQuery string,
	partitionSessionIDs map[string]struct{},
	messageSessionIDs map[string]struct{},
) bool {
	if _, ok := partitionSessionIDs[item.ID]; ok {
		return true
	}
	if _, ok := messageSessionIDs[item.ID]; ok {
		return true
	}
	return searchTextMatches(item.ID, normalizedQuery) || searchTextMatches(item.Title, normalizedQuery)
}

func searchMessageMatches(message llm.Message, normalizedQuery string) bool {
	if message.Role != llm.RoleUser && message.Role != llm.RoleAssistant {
		return false
	}
	if searchTextMatches(message.Text, normalizedQuery) {
		return true
	}
	for _, part := range message.Content {
		if part.Type == llm.ContentTypeText && searchTextMatches(part.Text, normalizedQuery) {
			return true
		}
	}
	return false
}

func matchingSearchPartitionSessionIDs(
	state SessionSidebarPartitionState,
	normalizedQuery string,
) map[string]struct{} {
	if normalizedQuery == "" {
		return nil
	}

	partitionIDs := make(map[string]struct{})
	for _, partition := range state.Partitions {
		if searchTextMatches(partition.Name, normalizedQuery) {
			id := strings.TrimSpace(partition.ID)
			if id != "" {
				partitionIDs[id] = struct{}{}
			}
		}
	}
	if len(partitionIDs) == 0 {
		return nil
	}

	sessionIDs := make(map[string]struct{})
	for sessionID, partitionID := range state.Assignments {
		if _, ok := partitionIDs[strings.TrimSpace(partitionID)]; !ok {
			continue
		}
		id := strings.TrimSpace(sessionID)
		if id != "" {
			sessionIDs[id] = struct{}{}
		}
	}
	return sessionIDs
}

func limitSessionMetadata(metadata []SessionMetadata, limit int) []SessionMetadata {
	if limit <= 0 || len(metadata) <= limit {
		return metadata
	}
	return metadata[:limit]
}

func normalizeSessionSearchQuery(query string) string {
	return strings.ToLower(strings.TrimSpace(query))
}

func searchTextMatches(value string, normalizedQuery string) bool {
	return strings.Contains(normalizeSessionSearchQuery(value), normalizedQuery)
}
