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
	memoryUserScopeID = "GHOST_MEMORY_USER_SCOPE_ID"
)

var (
	ErrNotFound      = errors.New("memory not found")
	ErrAlreadyExists = errors.New("memory already exists")
	ErrInvalidURI    = errors.New("invalid uri")
)

type StoreOptions struct {
	Path               string
	DefaultUserScopeID string
}

type Store struct {
	db                 *sql.DB
	now                func() time.Time
	defaultUserScopeID string
}

func NewStoreFromEnv() (*Store, error) {
	return NewStoreWithOptions(StoreOptions{
		Path:               strings.TrimSpace(os.Getenv(memoryPathEnv)),
		DefaultUserScopeID: strings.TrimSpace(os.Getenv(memoryUserScopeID)),
	})
}

func NewStore(path string) (*Store, error) {
	return NewStoreWithOptions(StoreOptions{Path: path})
}

func NewStoreWithOptions(opts StoreOptions) (*Store, error) {
	resolved, err := ResolveMemoryPath(opts.Path)
	if err != nil {
		return nil, err
	}
	if err := ensureMemoryDir(resolved); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", resolved)
	if err != nil {
		return nil, fmt.Errorf("open memory database: %w", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	store := &Store{
		db:                 db,
		now:                time.Now,
		defaultUserScopeID: normalizeDefaultUserScopeID(opts.DefaultUserScopeID),
	}
	if err := initSchema(db); err != nil {
		_ = db.Close()
		return nil, err
	}
	return store, nil
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

func (s *Store) currentTime() time.Time {
	if s == nil || s.now == nil {
		return time.Now().UTC()
	}
	return s.now().UTC()
}

func (s *Store) count(ctx context.Context, query string, args ...any) (int, error) {
	var total int
	if err := s.db.QueryRowContext(ctx, query, args...).Scan(&total); err != nil {
		return 0, fmt.Errorf("count memories: %w", err)
	}
	return total, nil
}

func ensureMemoryDir(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create memory directory: %w", err)
	}
	return nil
}

func normalizeDefaultUserScopeID(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return DefaultUserScopeID
	}
	return value
}
