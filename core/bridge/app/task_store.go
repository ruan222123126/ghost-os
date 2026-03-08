package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

const (
	taskScheduleTypeInterval = "interval"
	taskScheduleTypeCron     = "cron"

	defaultTaskRunLogRetention = 100

	taskRunStatusSuccess        = "success"
	taskRunStatusCancelled      = "cancelled"
	taskRunStatusError          = "error"
	taskRunStatusSkipped        = "skipped"
	taskRunStatusAwaitingHuman  = "awaiting_human"
	maxTaskResponsePreviewRunes = 240
)

var (
	ErrTaskNotFound      = errors.New("task not found")
	ErrTaskCorrupted     = errors.New("task corrupted")
	ErrInvalidTaskID     = errors.New("invalid task id")
	ErrInvalidTaskConfig = errors.New("invalid task config")

	taskIDPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]{0,127}$`)
	taskIDCounter uint64
	runIDCounter  uint64
)

type ScheduledTask struct {
	ID              string         `json:"id"`
	Message         string         `json:"message"`
	SessionID       string         `json:"session_id,omitempty"`
	TaskKind        string         `json:"task_kind,omitempty"`
	Action          string         `json:"action,omitempty"`
	ActionParams    map[string]any `json:"action_params,omitempty"`
	ScheduleType    string         `json:"schedule_type"`
	IntervalSeconds int            `json:"interval_seconds,omitempty"`
	CronExpr        string         `json:"cron_expr,omitempty"`
	Enabled         bool           `json:"enabled"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	LastRunAt       time.Time      `json:"last_run_at,omitempty"`
	NextRunAt       time.Time      `json:"next_run_at,omitempty"`
	LastError       string         `json:"last_error,omitempty"`
}

type TaskRunLog struct {
	TaskID          string    `json:"task_id"`
	RunID           string    `json:"run_id"`
	TraceID         string    `json:"trace_id"`
	TaskKind        string    `json:"task_kind,omitempty"`
	Action          string    `json:"action,omitempty"`
	ScheduledAt     time.Time `json:"scheduled_at"`
	StartedAt       time.Time `json:"started_at,omitempty"`
	FinishedAt      time.Time `json:"finished_at,omitempty"`
	Status          string    `json:"status"`
	SessionIDInput  string    `json:"session_id_input,omitempty"`
	SessionIDOutput string    `json:"session_id_output,omitempty"`
	ResponsePreview string    `json:"response_preview,omitempty"`
	Error           string    `json:"error,omitempty"`
}

type TaskStore struct {
	baseDir  string
	tasksDir string
	logsDir  string
	mu       sync.Mutex
}

type TaskLoadIssue struct {
	Kind   string `json:"kind,omitempty"`
	TaskID string `json:"task_id,omitempty"`
	Path   string `json:"path,omitempty"`
	Error  string `json:"error"`
}

const (
	taskLoadIssueReadError     = "read_error"
	taskLoadIssueDecodeError   = "decode_error"
	taskLoadIssueInvalidConfig = "invalid_config"
	taskLoadIssueIDMismatch    = "id_mismatch"
)

func NewTaskStore(baseDir string) (*TaskStore, error) {
	resolved, err := resolveTaskBaseDir(baseDir)
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
	return &TaskStore{baseDir: resolved, tasksDir: tasksDir, logsDir: logsDir}, nil
}

func (s *TaskStore) SaveTask(task *ScheduledTask) error {
	if task == nil {
		return errors.New("task is nil")
	}
	if err := normalizeScheduledTask(task); err != nil {
		return err
	}

	path, err := s.pathForTask(task.ID)
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

func (s *TaskStore) LoadTask(taskID string) (*ScheduledTask, error) {
	path, err := s.pathForTask(taskID)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
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
	if err := normalizeScheduledTask(&task); err != nil {
		return nil, err
	}
	return &task, nil
}

func (s *TaskStore) DeleteTask(taskID string) error {
	path, err := s.pathForTask(taskID)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.Remove(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
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

func (s *TaskStore) ListTasks() ([]ScheduledTask, error) {
	tasks, _, err := s.scanTasks(false)
	return tasks, err
}

func (s *TaskStore) ListTasksTolerant() ([]ScheduledTask, []TaskLoadIssue, error) {
	return s.scanTasks(true)
}

func (s *TaskStore) scanTasks(tolerant bool) ([]ScheduledTask, []TaskLoadIssue, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	entries, err := os.ReadDir(s.tasksDir)
	if err != nil {
		return nil, nil, fmt.Errorf("read task directory %q: %w", s.tasksDir, err)
	}
	tasks := make([]ScheduledTask, 0, len(entries))
	issues := make([]TaskLoadIssue, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".json") {
			continue
		}
		id := strings.TrimSuffix(name, ".json")
		if !isValidTaskID(id) {
			continue
		}
		path := filepath.Join(s.tasksDir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			readErr := fmt.Errorf("read task %q: %w", id, err)
			issue := newTaskLoadIssue(taskLoadIssueReadError, id, path, readErr)
			if tolerant {
				issues = append(issues, issue)
				continue
			}
			return nil, nil, readErr
		}
		var task ScheduledTask
		if err := json.Unmarshal(data, &task); err != nil {
			decodeErr := fmt.Errorf("%w: id=%s: %v", ErrTaskCorrupted, id, err)
			issue := newTaskLoadIssue(taskLoadIssueDecodeError, id, path, decodeErr)
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
			issue := newTaskLoadIssue(taskLoadIssueIDMismatch, id, path, mismatchErr)
			if tolerant {
				issues = append(issues, issue)
				continue
			}
			return nil, nil, mismatchErr
		}
		if err := normalizeScheduledTask(&task); err != nil {
			issue := newTaskLoadIssue(taskLoadIssueInvalidConfig, id, path, err)
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

func newTaskLoadIssue(kind string, taskID string, path string, err error) TaskLoadIssue {
	return TaskLoadIssue{
		Kind:   strings.TrimSpace(kind),
		TaskID: strings.TrimSpace(taskID),
		Path:   strings.TrimSpace(path),
		Error:  strings.TrimSpace(err.Error()),
	}
}

func (s *TaskStore) AppendRunLog(run TaskRunLog) error {
	if !isValidTaskID(run.TaskID) {
		return fmt.Errorf("%w: %q", ErrInvalidTaskID, run.TaskID)
	}
	normalizeTaskRunLog(&run)
	run.TaskID = strings.TrimSpace(run.TaskID)
	run.RunID = strings.TrimSpace(run.RunID)
	if run.RunID == "" {
		run.RunID = newTaskRunID()
	}
	run.TraceID = strings.TrimSpace(run.TraceID)
	run.TaskKind = normalizeTaskKind(run.TaskKind)
	run.Action = strings.TrimSpace(run.Action)
	run.Status = strings.TrimSpace(run.Status)
	run.SessionIDInput = strings.TrimSpace(run.SessionIDInput)
	run.SessionIDOutput = strings.TrimSpace(run.SessionIDOutput)
	run.Error = strings.TrimSpace(run.Error)
	run.ResponsePreview = truncateRunes(strings.TrimSpace(run.ResponsePreview), maxTaskResponsePreviewRunes)

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
	if err := s.pruneRunLogsLocked(run.TaskID, defaultTaskRunLogRetention); err != nil {
		log.Printf("task store: prune task logs task_id=%q keep=%d err=%v", run.TaskID, defaultTaskRunLogRetention, err)
	}
	return nil
}

func (s *TaskStore) ListRunLogs(taskID string, limit int) ([]TaskRunLog, error) {
	id := strings.TrimSpace(taskID)
	if !isValidTaskID(id) {
		return nil, fmt.Errorf("%w: %q", ErrInvalidTaskID, taskID)
	}

	logDir := filepath.Join(s.logsDir, id)
	s.mu.Lock()
	defer s.mu.Unlock()

	entries, err := os.ReadDir(logDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []TaskRunLog{}, nil
		}
		return nil, fmt.Errorf("read task log directory %q: %w", logDir, err)
	}

	runs := make([]TaskRunLog, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(logDir, entry.Name()))
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, fmt.Errorf("read task log %q/%q: %w", id, entry.Name(), err)
		}
		var run TaskRunLog
		if err := json.Unmarshal(data, &run); err != nil {
			return nil, fmt.Errorf("decode task log %q/%q: %w", id, entry.Name(), err)
		}
		normalizeTaskRunLog(&run)
		runs = append(runs, run)
	}

	sort.Slice(runs, func(i, j int) bool {
		left := taskRunLogSortTime(runs[i])
		right := taskRunLogSortTime(runs[j])
		return left.After(right)
	})
	if limit > 0 && len(runs) > limit {
		runs = runs[:limit]
	}
	return runs, nil
}

func (s *TaskStore) pruneRunLogsLocked(taskID string, keep int) error {
	id := strings.TrimSpace(taskID)
	if keep <= 0 {
		return nil
	}

	logDir := filepath.Join(s.logsDir, id)
	entries, err := os.ReadDir(logDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
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
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return fmt.Errorf("read task log %q/%q: %w", id, entry.Name(), err)
		}
		item := logFile{name: entry.Name(), path: path}
		var run TaskRunLog
		if err := json.Unmarshal(data, &run); err == nil {
			normalizeTaskRunLog(&run)
			item.sortTime = taskRunLogSortTime(run)
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
		if err := os.Remove(item.path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("delete task log %q/%q: %w", id, item.name, err)
		}
	}
	return nil
}

func (s *TaskStore) pathForTask(taskID string) (string, error) {
	id := strings.TrimSpace(taskID)
	if !isValidTaskID(id) {
		return "", fmt.Errorf("%w: %q", ErrInvalidTaskID, taskID)
	}
	return filepath.Join(s.tasksDir, id+".json"), nil
}

func resolveTaskBaseDir(pathValue string) (string, error) {
	trimmed := strings.TrimSpace(pathValue)
	if trimmed == "" {
		return "", errors.New("tasks path is empty")
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

func normalizeScheduledTask(task *ScheduledTask) error {
	if task == nil {
		return errors.New("task is nil")
	}
	task.ID = strings.TrimSpace(task.ID)
	task.Message = strings.TrimSpace(task.Message)
	task.SessionID = strings.TrimSpace(task.SessionID)
	task.TaskKind = normalizeTaskKind(task.TaskKind)
	task.Action = strings.TrimSpace(task.Action)
	task.ActionParams = cloneTaskActionParams(task.ActionParams)
	task.ScheduleType = strings.TrimSpace(task.ScheduleType)
	task.CronExpr = strings.TrimSpace(task.CronExpr)
	task.LastError = strings.TrimSpace(task.LastError)
	if task.ID == "" {
		task.ID = newTaskID()
	}
	if !isValidTaskID(task.ID) {
		return fmt.Errorf("%w: %q", ErrInvalidTaskID, task.ID)
	}
	if err := validateTaskDefinition(task); err != nil {
		return err
	}
	if task.CreatedAt.IsZero() {
		task.CreatedAt = time.Now().UTC()
	} else {
		task.CreatedAt = task.CreatedAt.UTC()
	}
	if !task.LastRunAt.IsZero() {
		task.LastRunAt = task.LastRunAt.UTC()
	}
	if !task.NextRunAt.IsZero() {
		task.NextRunAt = task.NextRunAt.UTC()
	}
	switch task.ScheduleType {
	case taskScheduleTypeInterval:
		if task.IntervalSeconds <= 0 || task.CronExpr != "" {
			return fmt.Errorf("%w: interval task requires interval_seconds only", ErrInvalidTaskConfig)
		}
	case taskScheduleTypeCron:
		if task.IntervalSeconds != 0 || task.CronExpr == "" {
			return fmt.Errorf("%w: cron task requires cron_expr only", ErrInvalidTaskConfig)
		}
	default:
		return fmt.Errorf("%w: schedule_type must be interval or cron", ErrInvalidTaskConfig)
	}
	return nil
}

func normalizeTaskRunLog(run *TaskRunLog) {
	if run == nil {
		return
	}
	run.TaskID = strings.TrimSpace(run.TaskID)
	run.RunID = strings.TrimSpace(run.RunID)
	run.TraceID = strings.TrimSpace(run.TraceID)
	run.TaskKind = normalizeTaskKind(run.TaskKind)
	run.Action = strings.TrimSpace(run.Action)
	run.Status = strings.TrimSpace(run.Status)
	run.SessionIDInput = strings.TrimSpace(run.SessionIDInput)
	run.SessionIDOutput = strings.TrimSpace(run.SessionIDOutput)
	run.ResponsePreview = truncateRunes(strings.TrimSpace(run.ResponsePreview), maxTaskResponsePreviewRunes)
	run.Error = strings.TrimSpace(run.Error)
	if !run.ScheduledAt.IsZero() {
		run.ScheduledAt = run.ScheduledAt.UTC()
	}
	if !run.StartedAt.IsZero() {
		run.StartedAt = run.StartedAt.UTC()
	}
	if !run.FinishedAt.IsZero() {
		run.FinishedAt = run.FinishedAt.UTC()
	}
}

func taskRunLogSortTime(run TaskRunLog) time.Time {
	if !run.StartedAt.IsZero() {
		return run.StartedAt
	}
	return run.ScheduledAt
}

func isValidTaskID(taskID string) bool {
	return taskIDPattern.MatchString(strings.TrimSpace(taskID))
}

func newTaskID() string {
	return fmt.Sprintf("task-%d-%d", time.Now().UnixMilli(), atomic.AddUint64(&taskIDCounter, 1))
}

func newTaskRunID() string {
	return fmt.Sprintf("run-%d-%d", time.Now().UnixMilli(), atomic.AddUint64(&runIDCounter, 1))
}

func truncateRunes(value string, limit int) string {
	if limit <= 0 {
		return ""
	}
	runes := []rune(strings.TrimSpace(value))
	if len(runes) <= limit {
		return string(runes)
	}
	return string(runes[:limit])
}
