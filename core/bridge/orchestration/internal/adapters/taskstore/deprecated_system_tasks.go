package taskstore

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	bridgeTasks "ghost-os/bridge/tasks"
)

const deprecatedSystemActionKind = "system_action"

type Store interface {
	TasksDir() string
	DeleteTask(string) error
}

type Scheduler interface {
	Unregister(string) error
}

func RemoveDeprecatedSystemTasks(store Store, scheduler Scheduler) error {
	if store == nil || scheduler == nil {
		return nil
	}
	tasksDir := strings.TrimSpace(store.TasksDir())
	if tasksDir == "" {
		return nil
	}
	entries, err := os.ReadDir(tasksDir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		taskID, ok := parseTaskEntryID(entry.Name(), entry.IsDir())
		if !ok {
			continue
		}
		path := filepath.Join(tasksDir, entry.Name())
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		if !isDeprecatedSystemTaskDocument(data) {
			continue
		}
		_ = scheduler.Unregister(taskID)
		if deleteErr := store.DeleteTask(taskID); deleteErr != nil && !errors.Is(deleteErr, bridgeTasks.ErrTaskNotFound) {
			return deleteErr
		}
	}
	return nil
}

func parseTaskEntryID(name string, isDir bool) (string, bool) {
	if isDir || filepath.Ext(name) != ".json" {
		return "", false
	}
	id := strings.TrimSpace(strings.TrimSuffix(name, ".json"))
	if id == "" {
		return "", false
	}
	return id, true
}

func isDeprecatedSystemTaskDocument(data []byte) bool {
	var task struct {
		TaskKind string `json:"task_kind"`
	}
	if err := json.Unmarshal(data, &task); err != nil {
		return false
	}
	return strings.TrimSpace(task.TaskKind) == deprecatedSystemActionKind
}
