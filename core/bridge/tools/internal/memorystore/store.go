package memorystore

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

const (
	defaultMemoryPath = "~/.ghost-os/memory/memory.db"
	memoryPathEnv     = "GHOST_MEMORY_PATH"
)

var (
	ErrNotFound      = errors.New("memory not found")
	ErrAlreadyExists = errors.New("memory already exists")
	ErrInvalidURI    = errors.New("invalid uri")
)

type Record struct {
	URI       string         `json:"uri"`
	Content   string         `json:"content"`
	Metadata  map[string]any `json:"metadata,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

type Store struct {
	db  *sql.DB
	now func() time.Time
}

func NewStoreFromEnv() (*Store, error) {
	return NewStore(strings.TrimSpace(os.Getenv(memoryPathEnv)))
}

func NewStore(path string) (*Store, error) {
	resolved, err := ResolveMemoryPath(path)
	if err != nil {
		return nil, err
	}
	dir := filepath.Dir(resolved)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("create memory directory: %w", err)
	}
	db, err := sql.Open("sqlite", resolved)
	if err != nil {
		return nil, fmt.Errorf("open memory database: %w", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if err := initSchema(db); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &Store{db: db, now: time.Now}, nil
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func ResolveMemoryPath(path string) (string, error) {
	resolved := strings.TrimSpace(path)
	if resolved == "" {
		resolved = defaultMemoryPath
	}
	if resolved == "~" || strings.HasPrefix(resolved, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home for memory database: %w", err)
		}
		if resolved == "~" {
			resolved = homeDir
		} else {
			resolved = filepath.Join(homeDir, strings.TrimPrefix(resolved, "~/"))
		}
	}
	absolute, err := filepath.Abs(resolved)
	if err != nil {
		return "", fmt.Errorf("resolve memory database path: %w", err)
	}
	return filepath.Clean(absolute), nil
}

func (s *Store) Create(ctx context.Context, uri string, content string, metadata map[string]any) (Record, error) {
	if s == nil || s.db == nil {
		return Record{}, errors.New("memory store is not configured")
	}
	normalized, err := normalizeURI(uri)
	if err != nil {
		return Record{}, err
	}
	exists, err := s.exists(ctx, normalized)
	if err != nil {
		return Record{}, err
	}
	if exists {
		return Record{}, ErrAlreadyExists
	}
	createdAt := s.currentTime()
	metadataValue, err := encodeMetadata(metadata)
	if err != nil {
		return Record{}, err
	}
	if _, err := s.db.ExecContext(ctx, `INSERT INTO memories (uri, content, metadata_json, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`, normalized, content, metadataValue, formatTime(createdAt), formatTime(createdAt)); err != nil {
		return Record{}, fmt.Errorf("insert memory: %w", err)
	}
	return Record{
		URI:       normalized,
		Content:   content,
		Metadata:  cloneMetadata(metadata),
		CreatedAt: createdAt,
		UpdatedAt: createdAt,
	}, nil
}

func (s *Store) Update(ctx context.Context, uri string, content string, metadata map[string]any) (Record, error) {
	if s == nil || s.db == nil {
		return Record{}, errors.New("memory store is not configured")
	}
	normalized, err := normalizeURI(uri)
	if err != nil {
		return Record{}, err
	}
	existing, err := s.Read(ctx, normalized)
	if err != nil {
		return Record{}, err
	}
	updatedAt := s.currentTime()
	metadataValue, err := encodeMetadata(resolveUpdatedMetadata(existing.Metadata, metadata))
	if err != nil {
		return Record{}, err
	}
	result, err := s.db.ExecContext(ctx, `UPDATE memories SET content = ?, metadata_json = ?, updated_at = ? WHERE uri = ?`, content, metadataValue, formatTime(updatedAt), normalized)
	if err != nil {
		return Record{}, fmt.Errorf("update memory: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return Record{}, fmt.Errorf("update memory: %w", err)
	}
	if affected == 0 {
		return Record{}, ErrNotFound
	}
	return s.Read(ctx, normalized)
}

func (s *Store) Delete(ctx context.Context, uri string) (bool, error) {
	if s == nil || s.db == nil {
		return false, errors.New("memory store is not configured")
	}
	normalized, err := normalizeURI(uri)
	if err != nil {
		return false, err
	}
	result, err := s.db.ExecContext(ctx, `DELETE FROM memories WHERE uri = ?`, normalized)
	if err != nil {
		return false, fmt.Errorf("delete memory: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("delete memory: %w", err)
	}
	return affected > 0, nil
}

func (s *Store) Read(ctx context.Context, uri string) (Record, error) {
	if s == nil || s.db == nil {
		return Record{}, errors.New("memory store is not configured")
	}
	normalized, err := normalizeURI(uri)
	if err != nil {
		return Record{}, err
	}
	row := s.db.QueryRowContext(ctx, `SELECT uri, content, metadata_json, created_at, updated_at FROM memories WHERE uri = ?`, normalized)
	return scanRecord(row)
}

func (s *Store) List(ctx context.Context, prefix string, limit int, offset int) ([]Record, int, error) {
	if s == nil || s.db == nil {
		return nil, 0, errors.New("memory store is not configured")
	}
	filter := strings.TrimSpace(prefix)
	pattern := filter + "%"
	total, err := s.count(ctx, `SELECT COUNT(1) FROM memories WHERE uri LIKE ?`, pattern)
	if err != nil {
		return nil, 0, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT uri, content, metadata_json, created_at, updated_at FROM memories WHERE uri LIKE ? ORDER BY uri ASC LIMIT ? OFFSET ?`, pattern, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list memories: %w", err)
	}
	defer rows.Close()
	items, err := scanRecords(rows)
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (s *Store) Search(ctx context.Context, query string, limit int, offset int) ([]Record, int, error) {
	if s == nil || s.db == nil {
		return nil, 0, errors.New("memory store is not configured")
	}
	term := strings.TrimSpace(query)
	pattern := "%" + term + "%"
	total, err := s.count(ctx, `SELECT COUNT(1) FROM memories WHERE content LIKE ?`, pattern)
	if err != nil {
		return nil, 0, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT uri, content, metadata_json, created_at, updated_at FROM memories WHERE content LIKE ? ORDER BY updated_at DESC LIMIT ? OFFSET ?`, pattern, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("search memories: %w", err)
	}
	defer rows.Close()
	items, err := scanRecords(rows)
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}
