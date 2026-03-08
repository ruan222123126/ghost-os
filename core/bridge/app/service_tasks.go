package app

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"ghost-os/bridge/session"
)

const (
	busActionTaskCreate = "TASK_CREATE"
	busActionTaskList   = "TASK_LIST"
	busActionTaskGet    = "TASK_GET"
	busActionTaskUpdate = "TASK_UPDATE"
	busActionTaskRunNow = "TASK_RUN_NOW"
	busActionTaskLogs   = "TASK_LOGS"
	busActionTaskDelete = "TASK_DELETE"
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
	trimmed := strings.TrimSpace(id)
	if trimmed == "" {
		return "", http.StatusBadRequest, errors.New("task id is required")
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

func (s *bridgeService) ensureTaskSessionExists(sessionID string) (int, error) {
	id := strings.TrimSpace(sessionID)
	if id == "" {
		return http.StatusOK, nil
	}
	store, code, err := s.requireSessionStore()
	if err != nil {
		return code, err
	}
	sess, err := store.Load(id)
	if err != nil {
		return mapSessionStorageError(err), err
	}
	if sess.IsEnded() {
		return http.StatusConflict, errSessionEnded
	}
	return http.StatusOK, nil
}

func buildTaskPayload(task ScheduledTask) taskPayload {
	return taskPayload{
		ID:              task.ID,
		Message:         task.Message,
		SessionID:       task.SessionID,
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

func (s *bridgeService) executeTaskCreateAction(params taskCreateParams, traceID string) (any, int, error) {
	store, code, err := s.requireTaskStore()
	if err != nil {
		return nil, code, err
	}
	scheduler, code, err := s.requireTaskScheduler()
	if err != nil {
		return nil, code, err
	}
	message := strings.TrimSpace(params.Message)
	sessionID := strings.TrimSpace(params.SessionID)
	cronExpr := strings.TrimSpace(params.CronExpr)
	if message == "" {
		return nil, http.StatusBadRequest, errors.New("message is required")
	}
	if (params.IntervalSeconds > 0 && cronExpr != "") || (params.IntervalSeconds <= 0 && cronExpr == "") {
		return nil, http.StatusBadRequest, errors.New("exactly one of interval_seconds or cron_expr is required")
	}
	if code, err := s.ensureTaskSessionExists(sessionID); err != nil {
		return nil, code, err
	}
	task := ScheduledTask{
		Message:         message,
		SessionID:       sessionID,
		Enabled:         true,
		CreatedAt:       time.Now().UTC(),
		ScheduleType:    taskScheduleTypeInterval,
		IntervalSeconds: params.IntervalSeconds,
	}
	if cronExpr != "" {
		task.ScheduleType = taskScheduleTypeCron
		task.IntervalSeconds = 0
		task.CronExpr = cronExpr
	}
	nextRunAt, err := nextTaskRunAt(task, time.Now().UTC())
	if err != nil {
		return nil, http.StatusBadRequest, err
	}
	task.NextRunAt = nextRunAt
	if err := store.SaveTask(&task); err != nil {
		logAction(traceID, busActionTaskCreate, "error", err)
		return nil, mapTaskError(err), err
	}
	if err := scheduler.Upsert(task); err != nil {
		_ = store.DeleteTask(task.ID)
		logAction(traceID, busActionTaskCreate, "error", err)
		return nil, mapTaskError(err), err
	}
	logAction(traceID, busActionTaskCreate, "success", nil)
	return buildTaskPayload(task), http.StatusCreated, nil
}

func (s *bridgeService) executeTaskUpdateAction(params taskUpdateParams, traceID string) (any, int, error) {
	store, code, err := s.requireTaskStore()
	if err != nil {
		return nil, code, err
	}
	scheduler, code, err := s.requireTaskScheduler()
	if err != nil {
		return nil, code, err
	}
	id, code, err := requireTaskID(params.ID)
	if err != nil {
		return nil, code, err
	}
	task, err := store.LoadTask(id)
	if err != nil {
		logAction(traceID, busActionTaskUpdate, "error", err)
		return nil, mapTaskError(err), err
	}
	previousEnabled := task.Enabled
	scheduleChanged := false
	if params.Message != nil {
		task.Message = strings.TrimSpace(*params.Message)
	}
	if params.SessionID != nil {
		task.SessionID = strings.TrimSpace(*params.SessionID)
	}
	if params.Enabled != nil {
		task.Enabled = *params.Enabled
	}
	if params.IntervalSeconds != nil && params.CronExpr != nil {
		err = errors.New("exactly one schedule field can be updated at a time")
		logAction(traceID, busActionTaskUpdate, "error", err)
		return nil, http.StatusBadRequest, err
	}
	if params.IntervalSeconds != nil {
		task.ScheduleType = taskScheduleTypeInterval
		task.IntervalSeconds = *params.IntervalSeconds
		task.CronExpr = ""
		scheduleChanged = true
	}
	if params.CronExpr != nil {
		task.ScheduleType = taskScheduleTypeCron
		task.CronExpr = strings.TrimSpace(*params.CronExpr)
		task.IntervalSeconds = 0
		scheduleChanged = true
	}
	if code, err := s.ensureTaskSessionExists(task.SessionID); err != nil {
		logAction(traceID, busActionTaskUpdate, "error", err)
		return nil, code, err
	}
	if !task.Enabled {
		task.NextRunAt = time.Time{}
	} else if scheduleChanged || !previousEnabled || task.NextRunAt.IsZero() {
		nextRunAt, err := nextTaskRunAt(*task, time.Now().UTC())
		if err != nil {
			logAction(traceID, busActionTaskUpdate, "error", err)
			return nil, http.StatusBadRequest, err
		}
		task.NextRunAt = nextRunAt
	}
	if err := store.SaveTask(task); err != nil {
		logAction(traceID, busActionTaskUpdate, "error", err)
		return nil, mapTaskError(err), err
	}
	if err := scheduler.Upsert(*task); err != nil {
		logAction(traceID, busActionTaskUpdate, "error", err)
		return nil, mapTaskError(err), err
	}
	logAction(traceID, busActionTaskUpdate, "success", nil)
	return buildTaskPayload(*task), http.StatusOK, nil
}

func (s *bridgeService) executeTaskListAction(traceID string) (any, int, error) {
	store, code, err := s.requireTaskStore()
	if err != nil {
		return nil, code, err
	}
	tasks, err := store.ListTasks()
	if err != nil {
		logAction(traceID, busActionTaskList, "error", err)
		return nil, mapTaskError(err), err
	}
	res := make([]taskPayload, 0, len(tasks))
	for _, task := range tasks {
		res = append(res, buildTaskPayload(task))
	}
	logAction(traceID, busActionTaskList, "success", nil)
	return res, http.StatusOK, nil
}

func (s *bridgeService) executeTaskGetAction(params taskIDParams, traceID string) (any, int, error) {
	store, code, err := s.requireTaskStore()
	if err != nil {
		return nil, code, err
	}
	id, code, err := requireTaskID(params.ID)
	if err != nil {
		return nil, code, err
	}
	task, err := store.LoadTask(id)
	if err != nil {
		logAction(traceID, busActionTaskGet, "error", err)
		return nil, mapTaskError(err), err
	}
	logAction(traceID, busActionTaskGet, "success", nil)
	return buildTaskPayload(*task), http.StatusOK, nil
}

func (s *bridgeService) executeTaskLogsAction(params taskLogsParams, traceID string) (any, int, error) {
	store, code, err := s.requireTaskStore()
	if err != nil {
		return nil, code, err
	}
	id, code, err := requireTaskID(params.ID)
	if err != nil {
		return nil, code, err
	}
	if params.Limit < 0 {
		return nil, http.StatusBadRequest, errors.New("limit must be >= 0")
	}
	if _, err := store.LoadTask(id); err != nil {
		logAction(traceID, busActionTaskLogs, "error", err)
		return nil, mapTaskError(err), err
	}
	runs, err := store.ListRunLogs(id, params.Limit)
	if err != nil {
		logAction(traceID, busActionTaskLogs, "error", err)
		return nil, mapTaskError(err), err
	}
	res := make([]taskRunLogPayload, 0, len(runs))
	for _, run := range runs {
		res = append(res, buildTaskRunLogPayload(run))
	}
	logAction(traceID, busActionTaskLogs, "success", nil)
	return res, http.StatusOK, nil
}

func (s *bridgeService) executeTaskRunNowAction(params taskIDParams, traceID string) (any, int, error) {
	store, code, err := s.requireTaskStore()
	if err != nil {
		return nil, code, err
	}
	scheduler, code, err := s.requireTaskScheduler()
	if err != nil {
		return nil, code, err
	}
	id, code, err := requireTaskID(params.ID)
	if err != nil {
		return nil, code, err
	}
	task, err := store.LoadTask(id)
	if err != nil {
		logAction(traceID, busActionTaskRunNow, "error", err)
		return nil, mapTaskError(err), err
	}
	run, err := scheduler.RunNow(*task, traceID)
	if err != nil {
		logAction(traceID, busActionTaskRunNow, "error", err)
		return nil, mapTaskError(err), err
	}
	updatedTask, err := store.LoadTask(id)
	if err != nil {
		logAction(traceID, busActionTaskRunNow, "error", err)
		return nil, mapTaskError(err), err
	}
	logAction(traceID, busActionTaskRunNow, "success", nil)
	return taskRunPayload{Task: buildTaskPayload(*updatedTask), Run: buildTaskRunLogPayload(run)}, http.StatusOK, nil
}

func (s *bridgeService) executeTaskDeleteAction(params taskIDParams, traceID string) (any, int, error) {
	store, code, err := s.requireTaskStore()
	if err != nil {
		return nil, code, err
	}
	scheduler, code, err := s.requireTaskScheduler()
	if err != nil {
		return nil, code, err
	}
	id, code, err := requireTaskID(params.ID)
	if err != nil {
		return nil, code, err
	}
	if err := scheduler.Unregister(id); err != nil {
		logAction(traceID, busActionTaskDelete, "error", err)
		return nil, mapTaskError(err), err
	}
	if err := store.DeleteTask(id); err != nil {
		logAction(traceID, busActionTaskDelete, "error", err)
		return nil, mapTaskError(err), err
	}
	logAction(traceID, busActionTaskDelete, "success", nil)
	return taskDeleteResponse{ID: id, Deleted: true}, http.StatusOK, nil
}
