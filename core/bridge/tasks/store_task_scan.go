package tasks

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

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
		task, issue, handled, err := scanStoredTaskEntry(s.tasksDir, entry, s.validator)
		if !handled {
			continue
		}
		if err != nil && !tolerant {
			return nil, nil, err
		}
		if err != nil {
			issues = append(issues, issue)
			continue
		}
		tasks = append(tasks, task)
	}

	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].CreatedAt.Before(tasks[j].CreatedAt)
	})
	return tasks, issues, nil
}

func scanStoredTaskEntry(
	tasksDir string,
	entry os.DirEntry,
	validator DefinitionValidator,
) (ScheduledTask, LoadIssue, bool, error) {
	taskID, ok := taskIDFromEntry(entry)
	if !ok {
		return ScheduledTask{}, LoadIssue{}, false, nil
	}

	path := filepath.Join(tasksDir, entry.Name())
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return ScheduledTask{}, LoadIssue{}, false, nil
		}
		wrapped := fmt.Errorf("read task %q: %w", taskID, err)
		return ScheduledTask{}, newLoadIssue(LoadIssueReadError, taskID, path, wrapped), true, wrapped
	}

	task, kind, err := decodeStoredTask(taskID, data, validator)
	if err != nil {
		return ScheduledTask{}, newLoadIssue(kind, taskID, path, err), true, err
	}
	return task, LoadIssue{}, true, nil
}

func taskIDFromEntry(entry os.DirEntry) (string, bool) {
	if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
		return "", false
	}
	id := strings.TrimSuffix(entry.Name(), ".json")
	if !IsValidTaskID(id) {
		return "", false
	}
	return id, true
}

func newLoadIssue(kind string, taskID string, path string, err error) LoadIssue {
	return LoadIssue{
		Kind:   strings.TrimSpace(kind),
		TaskID: strings.TrimSpace(taskID),
		Path:   strings.TrimSpace(path),
		Error:  strings.TrimSpace(err.Error()),
	}
}
