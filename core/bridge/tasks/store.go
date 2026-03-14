package tasks

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type Store struct {
	baseDir   string
	tasksDir  string
	logsDir   string
	validator DefinitionValidator
	mu        sync.Mutex
}

func NewStore(baseDir string, validator DefinitionValidator) (*Store, error) {
	resolved, err := resolveBaseDir(baseDir)
	if err != nil {
		return nil, err
	}

	tasksDir := filepath.Join(resolved, "tasks")
	logsDir := filepath.Join(resolved, "logs")
	if err := os.MkdirAll(tasksDir, 0o700); err != nil {
		return nil, fmt.Errorf("create task directory %q: %w", tasksDir, err)
	}
	if err := os.MkdirAll(logsDir, 0o700); err != nil {
		return nil, fmt.Errorf("create task logs directory %q: %w", logsDir, err)
	}

	return &Store{
		baseDir:   resolved,
		tasksDir:  tasksDir,
		logsDir:   logsDir,
		validator: validator,
	}, nil
}

func (s *Store) BaseDir() string {
	if s == nil {
		return ""
	}
	return s.baseDir
}

func (s *Store) TasksDir() string {
	if s == nil {
		return ""
	}
	return s.tasksDir
}

func (s *Store) LogsDir() string {
	if s == nil {
		return ""
	}
	return s.logsDir
}

func (s *Store) PathForTask(taskID string) (string, error) {
	id := strings.TrimSpace(taskID)
	if !IsValidTaskID(id) {
		return "", fmt.Errorf("%w: %q", ErrInvalidTaskID, taskID)
	}
	return filepath.Join(s.tasksDir, id+".json"), nil
}

func (s *Store) PruneRunLogs(taskID string, keep int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.pruneRunLogsLocked(taskID, keep)
}

func resolveBaseDir(pathValue string) (string, error) {
	trimmed := strings.TrimSpace(pathValue)
	if trimmed == "" {
		return "", fmt.Errorf("tasks path is empty")
	}
	if strings.HasPrefix(trimmed, "~/") || trimmed == "~" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve user home directory: %w", err)
		}
		if trimmed == "~" {
			return homeDir, nil
		}
		return filepath.Join(homeDir, strings.TrimPrefix(trimmed, "~/")), nil
	}
	return filepath.Clean(trimmed), nil
}
