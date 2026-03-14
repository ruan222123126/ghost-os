package orchestration

import (
	"errors"
	"net/http"
	"strings"

	"ghost-os/bridge/session"
)

func (s *bridgeService) requireTaskStore() (*TaskStore, int, error) {
	if s.taskStore == nil {
		if s.taskInitErr != nil {
			return nil, http.StatusInternalServerError, s.taskInitErr
		}
		return nil, http.StatusInternalServerError, errors.New("task store is not configured")
	}
	return s.taskStore, http.StatusOK, nil
}

func (s *bridgeService) requireTaskScheduler() (*TaskScheduler, int, error) {
	if s.taskScheduler == nil {
		if s.taskInitErr != nil {
			return nil, http.StatusInternalServerError, s.taskInitErr
		}
		return nil, http.StatusInternalServerError, errors.New("task scheduler is not configured")
	}
	return s.taskScheduler, http.StatusOK, nil
}

func requireTaskID(id string) (string, int, error) {
	trimmed, err := normalizeTaskID(id)
	if err != nil {
		return "", http.StatusBadRequest, err
	}
	return trimmed, http.StatusOK, nil
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
	return taskPayload{
		ID:              task.ID,
		Message:         task.Message,
		SessionID:       task.SessionID,
		TaskKind:        task.TaskKind,
		Action:          task.Action,
		ActionParams:    cloneTaskActionParams(task.ActionParams),
		ScheduleType:    task.ScheduleType,
		IntervalSeconds: task.IntervalSeconds,
		CronExpr:        task.CronExpr,
		Enabled:         task.Enabled,
		CreatedAt:       task.CreatedAt,
		UpdatedAt:       task.UpdatedAt,
		LastRunAt:       task.LastRunAt,
		NextRunAt:       task.NextRunAt,
		LastError:       task.LastError,
	}
}

func buildTaskRunLogPayload(run TaskRunLog) taskRunLogPayload {
	return taskRunLogPayload{
		TaskID:          run.TaskID,
		RunID:           run.RunID,
		TraceID:         run.TraceID,
		TaskKind:        run.TaskKind,
		Action:          run.Action,
		ScheduledAt:     run.ScheduledAt,
		StartedAt:       run.StartedAt,
		FinishedAt:      run.FinishedAt,
		Status:          run.Status,
		SessionIDInput:  run.SessionIDInput,
		SessionIDOutput: run.SessionIDOutput,
		ResponsePreview: run.ResponsePreview,
		Error:           run.Error,
	}
}

func includeTaskInScope(task ScheduledTask, scope string) bool {
	switch strings.TrimSpace(scope) {
	case "", taskListScopeUser:
		return normalizeTaskKind(task.TaskKind) == taskKindAgentMessage
	case taskListScopeSystem:
		return normalizeTaskKind(task.TaskKind) == taskKindSystemAction
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
