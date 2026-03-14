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
	if err := createIndexes(db); err != nil {
		return err
	}
	return recreateRecallView(db)
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
	}
	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			return fmt.Errorf("create memory index: %w", err)
		}
	}
	return nil
}

func recreateRecallView(db *sql.DB) error {
	if _, err := db.Exec(`DROP VIEW IF EXISTS memory_recall_view`); err != nil {
		return fmt.Errorf("drop recall view: %w", err)
	}
	view := fmt.Sprintf(`CREATE VIEW memory_recall_view AS
		SELECT
			'explicit:' || uri AS id,
			COALESCE(json_extract(metadata_json, '$.scope_type'), '%s') AS scope_type,
			COALESCE(json_extract(metadata_json, '$.scope_id'), '%s') AS scope_id,
			'%s' AS source_kind,
			COALESCE(json_extract(metadata_json, '$.memory_type'), '%s') AS memory_type,
			NULL AS memory_key,
			content AS content,
			substr(content, 1, 240) AS summary,
			metadata_json AS metadata_json,
			1.0 AS confidence,
			'%s' AS status,
			created_at AS created_at,
			updated_at AS updated_at,
			last_used_at AS last_used_at
		FROM memories
		UNION ALL
		SELECT
			id,
			scope_type,
			scope_id,
			source_kind,
			memory_type,
			memory_key,
			content,
			summary,
			metadata_json,
			confidence,
			status,
			created_at,
			updated_at,
			last_used_at
		FROM learned_memories`,
		ScopeTypeUser,
		DefaultUserScopeID,
		SourceKindExplicit,
		MemoryTypeFact,
		MemoryStatusActive,
	)
	if _, err := db.Exec(view); err != nil {
		return fmt.Errorf("create recall view: %w", err)
	}
	return nil
}

func ensureMemoriesColumn(db *sql.DB, name string, definition string) error {
	return ensureTableColumn(db, "memories", name, definition)
}

func ensureLearnedColumn(db *sql.DB, name string, definition string) error {
	return ensureTableColumn(db, "learned_memories", name, definition)
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
