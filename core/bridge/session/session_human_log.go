package session

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	sessionHumanLogDirectoryName = "sessions"
	sessionHumanLogFileExt       = ".md"
	sessionHumanLogMetaPrefix    = "<!-- GHOST_SESSION_LOG_META "
	sessionHumanLogMetaSuffix    = " -->"
	sessionHumanLogBodyMarker    = "<!-- GHOST_SESSION_LOG_BODY_START -->\n"
)

type sessionHumanLogMode string

const (
	sessionHumanLogModeSummary sessionHumanLogMode = "summary"
	sessionHumanLogModeFull    sessionHumanLogMode = "full"
)

type sessionHumanLogMeta struct {
	SessionID         string              `json:"session_id"`
	CreatedAt         string              `json:"created_at"`
	UpdatedAt         string              `json:"updated_at"`
	ExportMode        sessionHumanLogMode `json:"export_mode"`
	LastExportedIndex int                 `json:"last_exported_index"`
}

type sessionHumanLogDocument struct {
	Meta sessionHumanLogMeta
	Body string
}

type sessionHumanLogRecord struct {
	SessionID    string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	MessageCount int
}

type sessionHumanLogWriter struct {
	baseDir             string
	humanLogFullEnabled func() bool
}

func newSessionHumanLogWriter(baseDir string, options StoreOptions) (*sessionHumanLogWriter, error) {
	logDir := filepath.Join(baseDir, sessionHumanLogDirectoryName)
	if err := os.MkdirAll(logDir, 0o700); err != nil {
		return nil, fmt.Errorf("create session human log directory: %w", err)
	}
	return &sessionHumanLogWriter{
		baseDir:             logDir,
		humanLogFullEnabled: options.HumanLogFullEnabled,
	}, nil
}

func (w *sessionHumanLogWriter) Sync(db *sql.DB, sessionID string) error {
	id, err := normalizeSessionID(sessionID)
	if err != nil {
		return err
	}
	record, err := loadSessionHumanLogRecord(db, id)
	if err != nil {
		return err
	}

	path := w.pathForSession(id)
	mode := w.mode()
	raw, exists, err := readSessionHumanLogFile(path)
	if err != nil {
		return err
	}
	if !exists {
		return w.rewriteAll(db, path, record, mode)
	}
	return w.syncExistingFile(db, path, raw, record, mode)
}

func (w *sessionHumanLogWriter) syncExistingFile(
	db *sql.DB,
	path string,
	raw string,
	record sessionHumanLogRecord,
	mode sessionHumanLogMode,
) error {
	doc, err := parseSessionHumanLogDocument(raw)
	if err != nil {
		return err
	}
	if doc.Meta.SessionID != record.SessionID {
		return fmt.Errorf("session human log meta mismatch: got %q want %q", doc.Meta.SessionID, record.SessionID)
	}
	if doc.Meta.ExportMode != mode {
		return w.rewriteAll(db, path, record, mode)
	}

	startIndex := nextMissingMessageIndex(doc.Meta)
	if startIndex >= record.MessageCount {
		return nil
	}

	messages, err := loadSessionHumanLogMessagesFrom(db, record.SessionID, startIndex)
	if err != nil {
		return err
	}
	appendedBody := appendSessionHumanLogBody(doc.Body, renderSessionHumanLogBody(messages, mode))
	return writeSessionHumanLogFile(path, renderSessionHumanLogDocument(newSessionHumanLogMeta(record, mode), appendedBody))
}

func (w *sessionHumanLogWriter) rewriteAll(
	db *sql.DB,
	path string,
	record sessionHumanLogRecord,
	mode sessionHumanLogMode,
) error {
	messages, err := loadSessionHumanLogMessagesFrom(db, record.SessionID, 0)
	if err != nil {
		return err
	}
	content := renderSessionHumanLogDocument(newSessionHumanLogMeta(record, mode), renderSessionHumanLogBody(messages, mode))
	return writeSessionHumanLogFile(path, content)
}

func (w *sessionHumanLogWriter) pathForSession(sessionID string) string {
	return filepath.Join(w.baseDir, sessionID+sessionHumanLogFileExt)
}

func (w *sessionHumanLogWriter) mode() sessionHumanLogMode {
	if w == nil || w.humanLogFullEnabled == nil {
		return sessionHumanLogModeSummary
	}
	if w.humanLogFullEnabled() {
		return sessionHumanLogModeFull
	}
	return sessionHumanLogModeSummary
}

func newSessionHumanLogMeta(record sessionHumanLogRecord, mode sessionHumanLogMode) sessionHumanLogMeta {
	return sessionHumanLogMeta{
		SessionID:         record.SessionID,
		CreatedAt:         formatSessionHumanLogTime(record.CreatedAt),
		UpdatedAt:         formatSessionHumanLogTime(record.UpdatedAt),
		ExportMode:        mode,
		LastExportedIndex: lastSessionMessageIndex(record.MessageCount),
	}
}

func formatSessionHumanLogTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}

func lastSessionMessageIndex(messageCount int) int {
	if messageCount <= 0 {
		return -1
	}
	return messageCount - 1
}

func nextMissingMessageIndex(meta sessionHumanLogMeta) int {
	if meta.LastExportedIndex < 0 {
		return 0
	}
	return meta.LastExportedIndex + 1
}

func loadSessionHumanLogRecord(db *sql.DB, sessionID string) (sessionHumanLogRecord, error) {
	row := db.QueryRow(`
SELECT id, created_at, updated_at, message_count
FROM sessions
WHERE id = ?`, sessionID)
	return scanSessionHumanLogRecord(row)
}

func scanSessionHumanLogRecord(scanner interface{ Scan(...any) error }) (sessionHumanLogRecord, error) {
	var (
		record     sessionHumanLogRecord
		createdRaw string
		updatedRaw string
	)
	if err := scanner.Scan(&record.SessionID, &createdRaw, &updatedRaw, &record.MessageCount); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return sessionHumanLogRecord{}, ErrSessionNotFound
		}
		return sessionHumanLogRecord{}, fmt.Errorf("scan session human log record: %w", err)
	}
	createdAt, err := parseSessionTime(createdRaw)
	if err != nil {
		return sessionHumanLogRecord{}, err
	}
	updatedAt, err := parseSessionTime(updatedRaw)
	if err != nil {
		return sessionHumanLogRecord{}, err
	}
	record.CreatedAt = createdAt
	record.UpdatedAt = updatedAt
	return record, nil
}

func loadSessionHumanLogMessagesFrom(db *sql.DB, sessionID string, startIndex int) ([]IndexedMessage, error) {
	rows, err := db.Query(`
SELECT idx, token_count, message_json
FROM session_messages
WHERE session_id = ? AND idx >= ?
ORDER BY idx ASC`, sessionID, startIndex)
	if err != nil {
		return nil, fmt.Errorf("load session human log messages: %w", err)
	}
	defer rows.Close()

	out := make([]IndexedMessage, 0, 32)
	for rows.Next() {
		record, err := scanMessageRecord(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, IndexedMessage{
			Index:   record.Index,
			Message: record.Message,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("load session human log messages: %w", err)
	}
	return out, nil
}

func readSessionHumanLogFile(path string) (string, bool, error) {
	raw, err := os.ReadFile(path)
	switch {
	case err == nil:
		return string(raw), true, nil
	case errors.Is(err, os.ErrNotExist):
		return "", false, nil
	default:
		return "", false, fmt.Errorf("read session human log file: %w", err)
	}
}

func writeSessionHumanLogFile(path string, content string) error {
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		return fmt.Errorf("write session human log file: %w", err)
	}
	return nil
}

func encodeSessionHumanLogMeta(meta sessionHumanLogMeta) (string, error) {
	encoded, err := json.Marshal(meta)
	if err != nil {
		return "", fmt.Errorf("encode session human log meta: %w", err)
	}
	return string(encoded), nil
}
