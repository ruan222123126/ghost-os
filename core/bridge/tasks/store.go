package tasks

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
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
	return &Store{baseDir: resolved, tasksDir: tasksDir, logsDir: logsDir, validator: validator}, nil
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

func (s *Store) SaveTask(task *ScheduledTask) error {
	if task == nil {
		return fmt.Errorf("task is nil")
	}
	if err := NormalizeScheduledTask(task, s.validator); err != nil {
		return err
	}

	path, err := s.PathForTask(task.ID)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if task.CreatedAt.IsZero() {
		task.CreatedAt = time.Now().UTC()
	}
	task.UpdatedAt = time.Now().UTC()

	data, err := json.MarshalIndent(task, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal task %q: %w", task.ID, err)
	}
	data = append(data, '\n')

	tempPath := fmt.Sprintf("%s.tmp-%d", path, time.Now().UnixNano())
	if err := os.WriteFile(tempPath, data, 0o600); err != nil {
		return fmt.Errorf("write temp task file %q: %w", tempPath, err)
	}
	if err := os.Rename(tempPath, path); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("replace task file %q: %w", path, err)
	}
	return nil
}

func (s *Store) LoadTask(taskID string) (*ScheduledTask, error) {
	path, err := s.PathForTask(taskID)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %s", ErrTaskNotFound, strings.TrimSpace(taskID))
		}
		return nil, fmt.Errorf("read task %q: %w", strings.TrimSpace(taskID), err)
	}

	var task ScheduledTask
	if err := json.Unmarshal(data, &task); err != nil {
		return nil, fmt.Errorf("%w: id=%s: %v", ErrTaskCorrupted, strings.TrimSpace(taskID), err)
	}
	if strings.TrimSpace(task.ID) == "" {
		task.ID = strings.TrimSpace(taskID)
	}
	if strings.TrimSpace(task.ID) != strings.TrimSpace(taskID) {
		return nil, fmt.Errorf("%w: id mismatch file=%q payload=%q", ErrTaskCorrupted, strings.TrimSpace(taskID), task.ID)
	}
	if err := NormalizeScheduledTask(&task, s.validator); err != nil {
		return nil, err
	}
	return &task, nil
}

func (s *Store) DeleteTask(taskID string) error {
	path, err := s.PathForTask(taskID)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%w: %s", ErrTaskNotFound, strings.TrimSpace(taskID))
		}
		return fmt.Errorf("delete task %q: %w", strings.TrimSpace(taskID), err)
	}
	logDir := filepath.Join(s.logsDir, strings.TrimSpace(taskID))
	if err := os.RemoveAll(logDir); err != nil {
		return fmt.Errorf("delete task logs %q: %w", strings.TrimSpace(taskID), err)
	}
	return nil
}

func (s *Store) ListTasks() ([]ScheduledTask, error) {
	tasks, _, err := s.scanTasks(false)
	return tasks, err
}

func (s *Store) ListTasksTolerant() ([]ScheduledTask, []LoadIssue, error) {
	return s.scanTasks(true)
}

func (s *Store) AppendRunLog(run RunLog) error {
	if !IsValidTaskID(run.TaskID) {
		return fmt.Errorf("%w: %q", ErrInvalidTaskID, run.TaskID)
	}
	NormalizeRunLog(&run)
	run.TaskID = strings.TrimSpace(run.TaskID)
	run.RunID = strings.TrimSpace(run.RunID)
	if run.RunID == "" {
		run.RunID = NewRunID()
	}
	run.TraceID = strings.TrimSpace(run.TraceID)
	run.TaskKind = NormalizeKind(run.TaskKind)
	run.Action = strings.TrimSpace(run.Action)
	run.Status = strings.TrimSpace(run.Status)
	run.SessionIDInput = strings.TrimSpace(run.SessionIDInput)
	run.SessionIDOutput = strings.TrimSpace(run.SessionIDOutput)
	run.Error = strings.TrimSpace(run.Error)
	run.ResponsePreview = truncateRunes(strings.TrimSpace(run.ResponsePreview), MaxResponsePreviewRunes)

	logDir := filepath.Join(s.logsDir, run.TaskID)
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.MkdirAll(logDir, 0o700); err != nil {
		return fmt.Errorf("create task log directory %q: %w", logDir, err)
	}
	data, err := json.MarshalIndent(run, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal task log %q/%q: %w", run.TaskID, run.RunID, err)
	}
	data = append(data, '\n')
	path := filepath.Join(logDir, run.RunID+".json")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write task log %q/%q: %w", run.TaskID, run.RunID, err)
	}
	if err := s.pruneRunLogsLocked(run.TaskID, DefaultRunLogRetention); err != nil {
		log.Printf("task store: prune task logs task_id=%q keep=%d err=%v", run.TaskID, DefaultRunLogRetention, err)
	}
	return nil
}

func (s *Store) ListRunLogs(taskID string, limit int) ([]RunLog, error) {
	id := strings.TrimSpace(taskID)
	if !IsValidTaskID(id) {
		return nil, fmt.Errorf("%w: %q", ErrInvalidTaskID, taskID)
	}

	logDir := filepath.Join(s.logsDir, id)
	s.mu.Lock()
	defer s.mu.Unlock()

	entries, err := os.ReadDir(logDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []RunLog{}, nil
		}
		return nil, fmt.Errorf("read task log directory %q: %w", logDir, err)
	}

	runs := make([]RunLog, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(logDir, entry.Name()))
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("read task log %q/%q: %w", id, entry.Name(), err)
		}
		var run RunLog
		if err := json.Unmarshal(data, &run); err != nil {
			return nil, fmt.Errorf("decode task log %q/%q: %w", id, entry.Name(), err)
		}
		NormalizeRunLog(&run)
		runs = append(runs, run)
	}

	sort.Slice(runs, func(i, j int) bool {
		left := RunLogSortTime(runs[i])
		right := RunLogSortTime(runs[j])
		return left.After(right)
	})
	if limit > 0 && len(runs) > limit {
		runs = runs[:limit]
	}
	return runs, nil
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

func (s *Store) scanTasks(tolerant bool) ([]ScheduledTask, []LoadIssue, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entries, err := os.ReadDir(s.tasksDir)
	if err != nil {
		return nil, nil, fmt.Errorf("read task directory %q: %w", s.tasksDir, err)
	}
	tasks := make([]ScheduledTask, 0, len(entries))
	issues := make([]LoadIssue, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".json") {
			continue
		}
		id := strings.TrimSuffix(name, ".json")
		if !IsValidTaskID(id) {
			continue
		}
		path := filepath.Join(s.tasksDir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			readErr := fmt.Errorf("read task %q: %w", id, err)
			issue := newLoadIssue(LoadIssueReadError, id, path, readErr)
			if tolerant {
				issues = append(issues, issue)
				continue
			}
			return nil, nil, readErr
		}
		var task ScheduledTask
		if err := json.Unmarshal(data, &task); err != nil {
			decodeErr := fmt.Errorf("%w: id=%s: %v", ErrTaskCorrupted, id, err)
			issue := newLoadIssue(LoadIssueDecodeError, id, path, decodeErr)
			if tolerant {
				issues = append(issues, issue)
				continue
			}
			return nil, nil, decodeErr
		}
		storedID := strings.TrimSpace(task.ID)
		if storedID == "" {
			task.ID = id
		} else if storedID != id {
			mismatchErr := fmt.Errorf("%w: id=%s: payload id %q does not match file name", ErrTaskCorrupted, id, storedID)
			issue := newLoadIssue(LoadIssueIDMismatch, id, path, mismatchErr)
			if tolerant {
				issues = append(issues, issue)
				continue
			}
			return nil, nil, mismatchErr
		}
		if err := NormalizeScheduledTask(&task, s.validator); err != nil {
			issue := newLoadIssue(LoadIssueInvalidConfig, id, path, err)
			if tolerant {
				issues = append(issues, issue)
				continue
			}
			return nil, nil, err
		}
		tasks = append(tasks, task)
	}
	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].CreatedAt.Before(tasks[j].CreatedAt)
	})
	return tasks, issues, nil
}

func (s *Store) pruneRunLogsLocked(taskID string, keep int) error {
	id := strings.TrimSpace(taskID)
	if keep <= 0 {
		return nil
	}

	logDir := filepath.Join(s.logsDir, id)
	entries, err := os.ReadDir(logDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read task log directory %q: %w", logDir, err)
	}

	type logFile struct {
		name     string
		path     string
		sortTime time.Time
	}

	files := make([]logFile, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		path := filepath.Join(logDir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return fmt.Errorf("read task log %q/%q: %w", id, entry.Name(), err)
		}
		item := logFile{name: entry.Name(), path: path}
		var run RunLog
		if err := json.Unmarshal(data, &run); err == nil {
			NormalizeRunLog(&run)
			item.sortTime = RunLogSortTime(run)
		}
		files = append(files, item)
	}

	if len(files) <= keep {
		return nil
	}

	sort.Slice(files, func(i, j int) bool {
		left := files[i].sortTime
		right := files[j].sortTime
		if !left.Equal(right) {
			return left.Before(right)
		}
		return files[i].name < files[j].name
	})

	for _, item := range files[:len(files)-keep] {
		if err := os.Remove(item.path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("delete task log %q/%q: %w", id, item.name, err)
		}
	}
	return nil
}

func newLoadIssue(kind string, taskID string, path string, err error) LoadIssue {
	return LoadIssue{
		Kind:   strings.TrimSpace(kind),
		TaskID: strings.TrimSpace(taskID),
		Path:   strings.TrimSpace(path),
		Error:  strings.TrimSpace(err.Error()),
	}
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
