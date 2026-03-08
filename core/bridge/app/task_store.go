package app

import (
	"encoding/json"
	"errors"
	"fmt"
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

	taskRunStatusSuccess        = "success"
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
	ID              string    `json:"id"`
	Message         string    `json:"message"`
	SessionID       string    `json:"session_id,omitempty"`
	ScheduleType    string    `json:"schedule_type"`
	IntervalSeconds int       `json:"interval_seconds,omitempty"`
	CronExpr        string    `json:"cron_expr,omitempty"`
	Enabled         bool      `json:"enabled"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
	LastRunAt       time.Time `json:"last_run_at,omitempty"`
	NextRunAt       time.Time `json:"next_run_at,omitempty"`
	LastError       string    `json:"last_error,omitempty"`
}

type TaskRunLog struct {
	TaskID          string    `json:"task_id"`
	RunID           string    `json:"run_id"`
	TraceID         string    `json:"trace_id"`
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
	s.mu.Lock()
	defer s.mu.Unlock()

	entries, err := os.ReadDir(s.tasksDir)
	if err != nil {
		return nil, fmt.Errorf("read task directory %q: %w", s.tasksDir, err)
	}
	tasks := make([]ScheduledTask, 0, len(entries))
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
		data, err := os.ReadFile(filepath.Join(s.tasksDir, name))
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return nil, fmt.Errorf("read task %q: %w", id, err)
		}
		var task ScheduledTask
		if err := json.Unmarshal(data, &task); err != nil {
			return nil, fmt.Errorf("%w: id=%s: %v", ErrTaskCorrupted, id, err)
		}
		if strings.TrimSpace(task.ID) == "" {
			task.ID = id
		}
		if err := normalizeScheduledTask(&task); err != nil {
			return nil, err
		}
		tasks = append(tasks, task)
	}
	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].CreatedAt.Before(tasks[j].CreatedAt)
	})
	return tasks, nil
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
		left := runs[i].StartedAt
		if left.IsZero() {
			left = runs[i].ScheduledAt
		}
		right := runs[j].StartedAt
		if right.IsZero() {
			right = runs[j].ScheduledAt
		}
		return left.After(right)
	})
	if limit > 0 && len(runs) > limit {
		runs = runs[:limit]
	}
	return runs, nil
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
	task.ScheduleType = strings.TrimSpace(task.ScheduleType)
	task.CronExpr = strings.TrimSpace(task.CronExpr)
	task.LastError = strings.TrimSpace(task.LastError)
	if task.ID == "" {
		task.ID = newTaskID()
	}
	if !isValidTaskID(task.ID) {
		return fmt.Errorf("%w: %q", ErrInvalidTaskID, task.ID)
	}
	if task.Message == "" {
		return fmt.Errorf("%w: message is required", ErrInvalidTaskConfig)
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
