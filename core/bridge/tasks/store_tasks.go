package tasks

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

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

	now := time.Now().UTC()
	if task.CreatedAt.IsZero() {
		task.CreatedAt = now
	}
	task.UpdatedAt = now
	return writeTaskFileAtomic(path, *task)
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

	task, _, err := decodeStoredTask(strings.TrimSpace(taskID), data, s.validator)
	if err != nil {
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
		return fmt.Errorf("delete task logs %q: %w", logDir, err)
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

func writeTaskFileAtomic(path string, task ScheduledTask) error {
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

func decodeStoredTask(
	taskID string,
	data []byte,
	validator DefinitionValidator,
) (ScheduledTask, string, error) {
	var task ScheduledTask
	if err := json.Unmarshal(data, &task); err != nil {
		return ScheduledTask{}, LoadIssueDecodeError, fmt.Errorf("%w: id=%s: %v", ErrTaskCorrupted, taskID, err)
	}

	storedID := strings.TrimSpace(task.ID)
	switch {
	case storedID == "":
		task.ID = taskID
	case storedID != taskID:
		err := fmt.Errorf("%w: id=%s: payload id %q does not match file name", ErrTaskCorrupted, taskID, storedID)
		return ScheduledTask{}, LoadIssueIDMismatch, err
	}
	if err := NormalizeScheduledTask(&task, validator); err != nil {
		return ScheduledTask{}, LoadIssueInvalidConfig, err
	}
	return task, "", nil
}
