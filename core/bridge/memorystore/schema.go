package memorystore

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

func initSchema(db *sql.DB) error {
	if db == nil {
		return errors.New("memory database is nil")
	}
	if err := createTables(db); err != nil {
		return err
	}
	if err := ensureMemoriesColumn(db, "last_used_at", "DATETIME"); err != nil {
		return err
	}
	if err := ensureLearnedColumn(db, "superseded_by", "TEXT"); err != nil {
		return err
	}
	if err := ensureLearnedColumn(db, "memory_key", "TEXT"); err != nil {
		return err
	}
	if err := ensureEventMemoryColumn(db, "superseded_by", "TEXT"); err != nil {
		return err
	}
	if err := createIndexes(db); err != nil {
		return err
	}
	return nil
}

func createTables(db *sql.DB) error {
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS memories (
		uri TEXT PRIMARY KEY,
		content TEXT NOT NULL,
		metadata_json TEXT,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL,
		last_used_at DATETIME
	)`); err != nil {
		return fmt.Errorf("create memories table: %w", err)
	}
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS learned_memories (
		id TEXT PRIMARY KEY,
		scope_type TEXT NOT NULL,
		scope_id TEXT NOT NULL,
		source_kind TEXT NOT NULL,
		memory_type TEXT NOT NULL,
		memory_key TEXT,
		content TEXT NOT NULL,
		summary TEXT NOT NULL,
		metadata_json TEXT,
		confidence REAL NOT NULL,
		status TEXT NOT NULL,
		superseded_by TEXT,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL,
		last_used_at DATETIME
	)`); err != nil {
		return fmt.Errorf("create learned memories table: %w", err)
	}
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS memory_learning_events (
		id TEXT PRIMARY KEY,
		session_id TEXT NOT NULL,
		trace_id TEXT,
		status TEXT NOT NULL,
		input_json TEXT,
		filtered_json TEXT,
		candidates_json TEXT,
		result_json TEXT,
		error_text TEXT,
		created_at DATETIME NOT NULL
	)`); err != nil {
		return fmt.Errorf("create learning events table: %w", err)
	}
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS event_nodes (
		id TEXT PRIMARY KEY,
		session_id TEXT NOT NULL,
		title TEXT NOT NULL,
		summary TEXT NOT NULL,
		status TEXT NOT NULL,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL,
		last_activated_at DATETIME
	)`); err != nil {
		return fmt.Errorf("create event nodes table: %w", err)
	}
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS event_edges (
		session_id TEXT NOT NULL,
		from_event_id TEXT NOT NULL,
		to_event_id TEXT NOT NULL,
		edge_type TEXT NOT NULL,
		confidence REAL NOT NULL,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL,
		PRIMARY KEY (session_id, from_event_id, to_event_id, edge_type)
	)`); err != nil {
		return fmt.Errorf("create event edges table: %w", err)
	}
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS event_memories (
		id TEXT PRIMARY KEY,
		event_id TEXT NOT NULL,
		memory_type TEXT NOT NULL,
		memory_key TEXT,
		content TEXT NOT NULL,
		summary TEXT NOT NULL,
		metadata_json TEXT,
		confidence REAL NOT NULL,
		status TEXT NOT NULL,
		superseded_by TEXT,
		created_at DATETIME NOT NULL,
		updated_at DATETIME NOT NULL,
		last_used_at DATETIME
	)`); err != nil {
		return fmt.Errorf("create event memories table: %w", err)
	}
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS session_event_state (
		session_id TEXT PRIMARY KEY,
		primary_event_id TEXT,
		active_event_ids_json TEXT NOT NULL,
		planner_snapshot_json TEXT,
		updated_at DATETIME NOT NULL
	)`); err != nil {
		return fmt.Errorf("create session event state table: %w", err)
	}
	return nil
}

func createIndexes(db *sql.DB) error {
	statements := []string{
		`CREATE INDEX IF NOT EXISTS idx_memories_updated_at ON memories(updated_at)`,
		`CREATE INDEX IF NOT EXISTS idx_memories_last_used_at ON memories(last_used_at)`,
		`CREATE INDEX IF NOT EXISTS idx_learned_scope_status ON learned_memories(scope_type, scope_id, status)`,
		`CREATE INDEX IF NOT EXISTS idx_learned_scope_key_status ON learned_memories(scope_type, scope_id, memory_key, status)`,
		`CREATE INDEX IF NOT EXISTS idx_learned_updated_at ON learned_memories(updated_at)`,
		`CREATE INDEX IF NOT EXISTS idx_learned_last_used_at ON learned_memories(last_used_at)`,
		`CREATE INDEX IF NOT EXISTS idx_learning_events_session_created ON memory_learning_events(session_id, created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_event_nodes_session_status_activation ON event_nodes(session_id, status, last_activated_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_event_nodes_session_updated ON event_nodes(session_id, updated_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_event_edges_from ON event_edges(session_id, from_event_id, edge_type)`,
		`CREATE INDEX IF NOT EXISTS idx_event_edges_to ON event_edges(session_id, to_event_id, edge_type)`,
		`CREATE INDEX IF NOT EXISTS idx_event_memories_event_status ON event_memories(event_id, status, updated_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_event_memories_event_key_status ON event_memories(event_id, memory_key, status)`,
		`CREATE INDEX IF NOT EXISTS idx_event_memories_updated_at ON event_memories(updated_at)`,
		`CREATE INDEX IF NOT EXISTS idx_event_memories_last_used_at ON event_memories(last_used_at)`,
	}
	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			return fmt.Errorf("create memory index: %w", err)
		}
	}
	return nil
}

func ensureMemoriesColumn(db *sql.DB, name string, definition string) error {
	return ensureTableColumn(db, "memories", name, definition)
}

func ensureLearnedColumn(db *sql.DB, name string, definition string) error {
	return ensureTableColumn(db, "learned_memories", name, definition)
}

func ensureEventMemoryColumn(db *sql.DB, name string, definition string) error {
	return ensureTableColumn(db, "event_memories", name, definition)
}

func ensureTableColumn(db *sql.DB, table string, name string, definition string) error {
	exists, err := tableHasColumn(db, table, name)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	statement := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, name, definition)
	if _, err := db.Exec(statement); err != nil {
		return fmt.Errorf("alter %s add %s: %w", table, name, err)
	}
	return nil
}

func tableHasColumn(db *sql.DB, table string, column string) (bool, error) {
	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return false, fmt.Errorf("inspect %s columns: %w", table, err)
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name string
		var columnType string
		var notNull int
		var defaultValue sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &pk); err != nil {
			return false, fmt.Errorf("scan %s columns: %w", table, err)
		}
		if strings.EqualFold(strings.TrimSpace(name), column) {
			return true, nil
		}
	}
	if err := rows.Err(); err != nil {
		return false, fmt.Errorf("scan %s columns: %w", table, err)
	}
	return false, nil
}
