package sessions

import (
	"fmt"

	"ghost-os/bridge/orchestration/internal/contracts/api"
	bridgeTasks "ghost-os/bridge/tasks"
)

type TaskStore interface {
	ListTasks() ([]bridgeTasks.ScheduledTask, error)
	ListRunLogs(taskID string, limit int) ([]bridgeTasks.RunLog, error)
}

func (s Service) Sources(traceID string) (api.SessionSourceResolution, error) {
	tasks, err := LoadSessionSourceTasks(s.taskStore())
	if err != nil {
		s.log(traceID, ActionSources, "error", err)
		return api.SessionSourceResolution{}, err
	}

	resolution, err := BuildSessionSourceResolution(tasks)
	if err != nil {
		s.log(traceID, ActionSources, "error", err)
		return api.SessionSourceResolution{}, err
	}

	s.log(traceID, ActionSources, "success", nil)
	return resolution, nil
}

func LoadSessionSourceTasks(store TaskStore) ([]SessionSourceTask, error) {
	if store == nil {
		return nil, ErrTaskStoreRequired
	}

	tasks, err := store.ListTasks()
	if err != nil {
		return nil, err
	}
	sources := make([]SessionSourceTask, 0, len(tasks))
	for _, task := range tasks {
		runs, err := store.ListRunLogs(task.ID, 0)
		if err != nil {
			return nil, fmt.Errorf("list run logs for %s: %w", task.ID, err)
		}
		sources = append(sources, BuildSessionSourceTask(task, runs))
	}
	return sources, nil
}
