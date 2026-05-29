package orchestration

import (
	"fmt"

	internaltrace "ghost-os/bridge/orchestration/internal/trace"
)

func (s *bridgeService) executeSessionSourcesAction(traceID string) (ServiceResult, error) {
	store, code, err := s.requireTaskStore()
	if err != nil {
		logAction(traceID, internaltrace.ActionSessionSources, "error", err)
		return ServiceResult{}, wrapServiceError(serviceErrorKindFromStatus(code), err)
	}

	tasks, err := loadSessionSourceTasks(store)
	if err != nil {
		logAction(traceID, internaltrace.ActionSessionSources, "error", err)
		return ServiceResult{}, wrapServiceError(ServiceErrorInternal, err)
	}

	resolution, err := internaltrace.BuildSessionSourceResolution(tasks)
	if err != nil {
		logAction(traceID, internaltrace.ActionSessionSources, "error", err)
		return ServiceResult{}, wrapServiceError(ServiceErrorInternal, err)
	}

	logAction(traceID, internaltrace.ActionSessionSources, "success", nil)
	return serviceResultSuccess(resolution), nil
}

func loadSessionSourceTasks(store *TaskStore) ([]internaltrace.SessionSourceTask, error) {
	tasks, err := store.ListTasks()
	if err != nil {
		return nil, err
	}
	sources := make([]internaltrace.SessionSourceTask, 0, len(tasks))
	for _, task := range tasks {
		runs, err := store.ListRunLogs(task.ID, 0)
		if err != nil {
			return nil, fmt.Errorf("list run logs for %s: %w", task.ID, err)
		}
		sources = append(sources, internaltrace.BuildSessionSourceTask(task, runs))
	}
	return sources, nil
}
