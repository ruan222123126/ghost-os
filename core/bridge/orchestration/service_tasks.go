package orchestration

import (
	"errors"
	"net/http"
	"strings"

	"ghost-os/bridge/session"
)

func (s *bridgeService) requireTaskStore() (*TaskStore, int, error) {
	store := s.taskStore()
	if store == nil {
		if initErr := s.taskInitErr(); initErr != nil {
			return nil, http.StatusInternalServerError, initErr
		}
		return nil, http.StatusInternalServerError, errors.New("task store is not configured")
	}
	return store, http.StatusOK, nil
}

func (s *bridgeService) requireTaskScheduler() (*TaskScheduler, int, error) {
	scheduler := s.taskScheduler()
	if scheduler == nil {
		if initErr := s.taskInitErr(); initErr != nil {
			return nil, http.StatusInternalServerError, initErr
		}
		return nil, http.StatusInternalServerError, errors.New("task scheduler is not configured")
	}
	return scheduler, http.StatusOK, nil
}

func mapTaskError(err error) int {
	switch {
	case errors.Is(err, ErrInvalidTaskID), errors.Is(err, ErrInvalidTaskConfig), errors.Is(err, session.ErrInvalidSessionID):
		return http.StatusBadRequest
	case errors.Is(err, ErrTaskNotFound), errors.Is(err, session.ErrSessionNotFound):
		return http.StatusNotFound
	case errors.Is(err, errSessionEnded):
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}

func buildTaskPayload(task ScheduledTask) taskPayload {
	payload := cloneScheduledTask(task)
	payload.TaskKind = normalizeTaskKind(task.TaskKind)
	return payload
}

func buildTaskRunLogPayload(run TaskRunLog) taskRunLogPayload {
	return run
}

func includeTaskInScope(task ScheduledTask, scope string) bool {
	kind := normalizeTaskKind(task.TaskKind)
	switch strings.TrimSpace(scope) {
	case "", taskListScopeUser:
		return kind == taskKindAgentMessage || kind == taskKindWorkflow
	case taskListScopeSystem:
		return kind == taskKindSystemAction
	default:
		return false
	}
}

func (s *bridgeService) executeTaskCreateAction(params taskCreateParams, traceID string) (any, int, error) {
	runner, code, err := s.requireTaskMutationRunner()
	if err != nil {
		return nil, code, err
	}
	payload, err := runner.Create(params)
	if err != nil {
		logAction(traceID, busActionTaskCreate, "error", err)
		return nil, mapTaskError(err), err
	}
	logAction(traceID, busActionTaskCreate, "success", nil)
	return payload, http.StatusCreated, nil
}

func (s *bridgeService) executeTaskUpdateAction(params taskUpdateParams, traceID string) (any, int, error) {
	runner, code, err := s.requireTaskMutationRunner()
	if err != nil {
		return nil, code, err
	}
	payload, err := runner.Update(params)
	if err != nil {
		logAction(traceID, busActionTaskUpdate, "error", err)
		return nil, mapTaskError(err), err
	}
	logAction(traceID, busActionTaskUpdate, "success", nil)
	return payload, http.StatusOK, nil
}

func (s *bridgeService) executeTaskListAction(scope string, traceID string) (any, int, error) {
	runner, code, err := s.requireTaskQueryRunner()
	if err != nil {
		return nil, code, err
	}
	payload, err := runner.List(scope)
	if err != nil {
		logAction(traceID, busActionTaskList, "error", err)
		return nil, mapTaskError(err), err
	}
	logAction(traceID, busActionTaskList, "success", nil)
	return payload, http.StatusOK, nil
}

func (s *bridgeService) executeTaskGetAction(params taskIDParams, traceID string) (any, int, error) {
	runner, code, err := s.requireTaskQueryRunner()
	if err != nil {
		return nil, code, err
	}
	payload, err := runner.Get(params)
	if err != nil {
		logAction(traceID, busActionTaskGet, "error", err)
		return nil, mapTaskError(err), err
	}
	logAction(traceID, busActionTaskGet, "success", nil)
	return payload, http.StatusOK, nil
}

func (s *bridgeService) executeTaskLogsAction(params taskLogsParams, traceID string) (any, int, error) {
	runner, code, err := s.requireTaskQueryRunner()
	if err != nil {
		return nil, code, err
	}
	payload, err := runner.Logs(params)
	if err != nil {
		logAction(traceID, busActionTaskLogs, "error", err)
		return nil, mapTaskError(err), err
	}
	logAction(traceID, busActionTaskLogs, "success", nil)
	return payload, http.StatusOK, nil
}

func (s *bridgeService) executeTaskRunNowAction(params taskIDParams, traceID string) (any, int, error) {
	runner, code, err := s.requireTaskMutationRunner()
	if err != nil {
		return nil, code, err
	}
	payload, err := runner.RunNow(params, traceID)
	if err != nil {
		logAction(traceID, busActionTaskRunNow, "error", err)
		return nil, mapTaskError(err), err
	}
	logAction(traceID, busActionTaskRunNow, "success", nil)
	return payload, http.StatusOK, nil
}

func (s *bridgeService) executeTaskDeleteAction(params taskIDParams, traceID string) (any, int, error) {
	runner, code, err := s.requireTaskMutationRunner()
	if err != nil {
		return nil, code, err
	}
	payload, err := runner.Delete(params)
	if err != nil {
		logAction(traceID, busActionTaskDelete, "error", err)
		return nil, mapTaskError(err), err
	}
	logAction(traceID, busActionTaskDelete, "success", nil)
	return payload, http.StatusOK, nil
}
