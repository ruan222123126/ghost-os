package session

import (
	"database/sql"
	"fmt"
	"time"

	"ghost-os/bridge/llm"
)

type sessionUpsertTotals struct {
	baseMessageCount  int
	totalTokenCount   int
	totalMessageCount int
	createdAt         time.Time
}

func computeSessionUpsertTotals(
	sess *Session,
	existing *sessionRecord,
	appendTokens int,
	appendedCount int,
) (sessionUpsertTotals, error) {
	totals := sessionUpsertTotals{
		createdAt: sess.CreatedAt.UTC(),
	}
	baseTokenCount := 0
	if existing != nil {
		baseTokenCount = existing.TokenCount
		totals.baseMessageCount = existing.MessageCount
		totals.createdAt = existing.CreatedAt.UTC()
	}
	totals.totalTokenCount = baseTokenCount + appendTokens
	totals.totalMessageCount = totals.baseMessageCount + appendedCount
	if totals.totalMessageCount != sess.MessageCount {
		return sessionUpsertTotals{}, ErrSessionNotAppendOnly
	}
	return totals, nil
}

func upsertSessionRowTx(
	tx *sql.Tx,
	sess *Session,
	createdAt time.Time,
	totalTokenCount int,
	totalMessageCount int,
	windowStart int,
	windowTokens int,
	stateJSON string,
) error {
	if _, err := tx.Exec(`
INSERT INTO sessions (
	id, created_at, updated_at, ended_at, turn_index, token_count, message_count, window_start, window_token_count, state_json
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
	created_at = excluded.created_at,
	updated_at = excluded.updated_at,
	ended_at = excluded.ended_at,
	turn_index = excluded.turn_index,
	token_count = excluded.token_count,
	message_count = excluded.message_count,
	window_start = excluded.window_start,
	window_token_count = excluded.window_token_count,
	state_json = excluded.state_json`,
		sess.ID,
		formatSessionTime(createdAt),
		formatSessionTime(sess.UpdatedAt),
		formatSessionTime(sess.EndedAt),
		sess.TurnIndex,
		totalTokenCount,
		totalMessageCount,
		windowStart,
		windowTokens,
		stateJSON,
	); err != nil {
		return fmt.Errorf("upsert session row: %w", err)
	}
	return nil
}

func applyPersistedSessionState(
	sess *Session,
	totals sessionUpsertTotals,
	messages []llm.Message,
	windowStart int,
	windowTokens int,
) {
	sess.CreatedAt = totals.createdAt
	sess.TokenCount = totals.totalTokenCount
	sess.MessageCount = totals.totalMessageCount
	sess.WindowStart = windowStart
	sess.WindowTokenCount = windowTokens
	sess.Messages = messages
	sess.setPersistedSnapshot()
}
