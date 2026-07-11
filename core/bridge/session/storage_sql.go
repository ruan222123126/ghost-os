package session

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"ghost-os/bridge/llm"
)

func initSessionSchema(db *sql.DB) error {
	statements := []string{
		`PRAGMA foreign_keys = ON;`,
		`CREATE TABLE IF NOT EXISTS sessions (
			id TEXT PRIMARY KEY,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			ended_at TEXT NOT NULL,
			turn_index INTEGER NOT NULL,
			token_count INTEGER NOT NULL,
			message_count INTEGER NOT NULL,
			window_start INTEGER NOT NULL,
			window_token_count INTEGER NOT NULL,
			state_json TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS session_messages (
			session_id TEXT NOT NULL,
			idx INTEGER NOT NULL,
			token_count INTEGER NOT NULL,
			message_json TEXT NOT NULL,
			PRIMARY KEY (session_id, idx),
			FOREIGN KEY (session_id) REFERENCES sessions(id) ON DELETE CASCADE
		);`,
	}
	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			return fmt.Errorf("init session schema: %w", err)
		}
	}
	return nil
}

func loadSessionTx(tx *sql.Tx, sessionID string) (*Session, error) {
	record, err := loadSessionRecordTx(tx, sessionID)
	if err != nil {
		return nil, err
	}
	messages, windowStart, windowTokens, err := loadHotWindowMessagesTx(tx, sessionID)
	if err != nil {
		return nil, err
	}
	record.WindowStart = windowStart
	record.WindowTokenCount = windowTokens
	return sessionFromRecord(*record, messages), nil
}

func loadSessionRecordTx(tx *sql.Tx, sessionID string) (*sessionRecord, error) {
	row := tx.QueryRow(`
SELECT id, created_at, updated_at, ended_at, turn_index, token_count, message_count, window_start, window_token_count, state_json
FROM sessions
WHERE id = ?`, sessionID)
	record, err := scanSessionRecord(row)
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func scanSessionRecord(scanner interface{ Scan(...any) error }) (sessionRecord, error) {
	var (
		record    sessionRecord
		createdAt string
		updatedAt string
		endedAt   string
		stateJSON string
	)
	if err := scanner.Scan(
		&record.ID,
		&createdAt,
		&updatedAt,
		&endedAt,
		&record.TurnIndex,
		&record.TokenCount,
		&record.MessageCount,
		&record.WindowStart,
		&record.WindowTokenCount,
		&stateJSON,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return sessionRecord{}, ErrSessionNotFound
		}
		return sessionRecord{}, fmt.Errorf("scan session: %w", err)
	}

	var err error
	record.CreatedAt, err = parseSessionTime(createdAt)
	if err != nil {
		return sessionRecord{}, err
	}
	record.UpdatedAt, err = parseSessionTime(updatedAt)
	if err != nil {
		return sessionRecord{}, err
	}
	record.EndedAt, err = parseSessionTime(endedAt)
	if err != nil {
		return sessionRecord{}, err
	}
	record.State, err = decodeSessionState(stateJSON)
	if err != nil {
		return sessionRecord{}, err
	}
	return record, nil
}

func upsertSessionTx(
	tx *sql.Tx,
	sess *Session,
	existing *sessionRecord,
	appended []llm.Message,
	appendTokens int,
) error {
	totals, err := computeSessionUpsertTotals(sess, existing, appendTokens, len(appended))
	if err != nil {
		return err
	}
	stateJSON, err := encodeSessionState(sess)
	if err != nil {
		return err
	}
	if existing == nil {
		if err := insertSessionPlaceholderTx(
			tx,
			sess,
			totals.createdAt,
			totals.totalTokenCount,
			totals.totalMessageCount,
			stateJSON,
		); err != nil {
			return err
		}
	}
	if err := insertMessagesTx(tx, sess.ID, totals.baseMessageCount, appended); err != nil {
		return err
	}
	messages, windowStart, windowTokens, err := loadHotWindowMessagesTx(tx, sess.ID)
	if err != nil {
		return err
	}
	if err := upsertSessionRowTx(
		tx,
		sess,
		totals.createdAt,
		totals.totalTokenCount,
		totals.totalMessageCount,
		windowStart,
		windowTokens,
		stateJSON,
	); err != nil {
		return err
	}
	applyPersistedSessionState(sess, totals, messages, windowStart, windowTokens)
	return nil
}

func insertSessionPlaceholderTx(
	tx *sql.Tx,
	sess *Session,
	createdAt time.Time,
	totalTokens int,
	totalMessages int,
	stateJSON string,
) error {
	_, err := tx.Exec(`
INSERT INTO sessions (
	id, created_at, updated_at, ended_at, turn_index, token_count, message_count, window_start, window_token_count, state_json
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		sess.ID,
		formatSessionTime(createdAt),
		formatSessionTime(sess.UpdatedAt),
		formatSessionTime(sess.EndedAt),
		sess.TurnIndex,
		totalTokens,
		totalMessages,
		0,
		0,
		stateJSON,
	)
	if err != nil {
		return fmt.Errorf("insert session row: %w", err)
	}
	return nil
}

func insertMessagesTx(tx *sql.Tx, sessionID string, startIndex int, messages []llm.Message) error {
	for index, message := range messages {
		encoded, err := encodeMessageJSON(message)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(`
INSERT INTO session_messages (session_id, idx, token_count, message_json)
VALUES (?, ?, ?, ?)`,
			sessionID,
			startIndex+index,
			EstimateTokens(message),
			encoded,
		); err != nil {
			return fmt.Errorf("insert session message[%d]: %w", startIndex+index, err)
		}
	}
	return nil
}

func deleteSessionTx(tx *sql.Tx, sessionID string) (bool, error) {
	result, err := tx.Exec(`DELETE FROM sessions WHERE id = ?`, sessionID)
	if err != nil {
		return false, fmt.Errorf("delete session %q: %w", sessionID, err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("delete session %q: %w", sessionID, err)
	}
	return affected > 0, nil
}

func scanSessionMetadata(scanner interface{ Scan(...any) error }) (SessionMetadata, error) {
	var (
		item      SessionMetadata
		createdAt string
		updatedAt string
		stateJSON string
	)
	if err := scanner.Scan(&item.ID, &createdAt, &updatedAt, &item.MessageCount, &item.TokenCount, &stateJSON); err != nil {
		return SessionMetadata{}, fmt.Errorf("scan session metadata: %w", err)
	}

	var err error
	item.CreatedAt, err = parseSessionTime(createdAt)
	if err != nil {
		return SessionMetadata{}, err
	}
	item.UpdatedAt, err = parseSessionTime(updatedAt)
	if err != nil {
		return SessionMetadata{}, err
	}
	state, err := decodeSessionState(stateJSON)
	if err != nil {
		return SessionMetadata{}, err
	}
	item.Title = strings.TrimSpace(state.Title)
	return item, nil
}
