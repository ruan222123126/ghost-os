package orchestration

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"ghost-os/bridge/session"
)

const (
	taskListScopeUser   = "user"
	taskListScopeSystem = "system"
)

type taskQueryRunner struct {
	store *TaskStore
}

type taskMutationStore interface {
	LoadTask(taskID string) (*ScheduledTask, error)
	SaveTask(task *ScheduledTask) error
	DeleteTask(taskID string) error
}

type taskMutationScheduler interface {
	Upsert(task ScheduledTask) error
	Unregister(taskID string) error
	RunNow(task ScheduledTask, traceID string) (TaskRunLog, error)
}

type taskMutationRunner struct {
	store        taskMutationStore
	scheduler    taskMutationScheduler
	sessionStore *session.Store
	now          func() time.Time
}

func (s *bridgeService) requireTaskMutationRunner() (taskMutationRunner, int, error) {
	store, code, err := s.requireTaskStore()
	if err != nil {
		return taskMutationRunner{}, code, err
	}
	scheduler, code, err := s.requireTaskScheduler()
	if err != nil {
		return taskMutationRunner{}, code, err
	}
	return taskMutationRunner{
		store:        store,
		scheduler:    scheduler,
		sessionStore: s.sessionStore,
		now:          func() time.Time { return time.Now().UTC() },
	}, http.StatusOK, nil
}

func normalizeTaskID(id string) (string, error) {
	trimmed := strings.TrimSpace(id)
	if trimmed == "" {
		return "", fmt.Errorf("%w: task id is required", ErrInvalidTaskID)
	}
	return trimmed, nil
}

func invalidTaskConfig(message string) error {
	return fmt.Errorf("%w: %s", ErrInvalidTaskConfig, message)
}

func wrapTaskConfigError(err error) error {
	if err == nil || errors.Is(err, ErrInvalidTaskConfig) {
		return err
	}
	return fmt.Errorf("%w: %v", ErrInvalidTaskConfig, err)
}

func (r taskMutationRunner) RunNow(params taskIDParams, traceID string) (taskRunPayload, error) {
	id, err := normalizeTaskID(params.ID)
	if err != nil {
		return taskRunPayload{}, err
	}
	task, err := r.store.LoadTask(id)
	if err != nil {
		return taskRunPayload{}, err
	}
	run, err := r.scheduler.RunNow(*task, traceID)
	if err != nil {
		return taskRunPayload{}, err
	}
	updatedTask, err := r.store.LoadTask(id)
	if err != nil {
		return taskRunPayload{}, err
	}
	return taskRunPayload{
		Task: buildTaskPayload(*updatedTask),
		Run:  buildTaskRunLogPayload(run),
	}, nil
}

func (r taskMutationRunner) Delete(params taskIDParams) (taskDeleteResponse, error) {
	id, err := normalizeTaskID(params.ID)
	if err != nil {
		return taskDeleteResponse{}, err
	}
	if err := r.scheduler.Unregister(id); err != nil {
		return taskDeleteResponse{}, err
	}
	if err := r.store.DeleteTask(id); err != nil {
		return taskDeleteResponse{}, err
	}
	return taskDeleteResponse{ID: id, Deleted: true}, nil
}

func (r taskMutationRunner) newScheduledTask(params taskCreateParams) (ScheduledTask, error) {
	task, err := r.buildScheduledTask(params)
	if err != nil {
		return ScheduledTask{}, err
	}
	nextRunAt, err := nextTaskRunAt(task, r.now())
	if err != nil {
		return ScheduledTask{}, wrapTaskConfigError(err)
	}
	task.NextRunAt = nextRunAt
	return task, nil
}

func (r taskMutationRunner) buildScheduledTask(params taskCreateParams) (ScheduledTask, error) {
	task, err := buildTaskFromCreateParams(params, r.now())
	if err != nil {
		return ScheduledTask{}, err
	}
	if err := validateTaskDefinition(&task); err != nil {
		return ScheduledTask{}, wrapTaskConfigError(err)
	}
	if err := r.ensureTaskSessionExists(task.TaskKind, task.SessionID); err != nil {
		return ScheduledTask{}, err
	}
	return task, nil
}

func buildTaskFromCreateParams(params taskCreateParams, now time.Time) (ScheduledTask, error) {
	cronExpr := strings.TrimSpace(params.CronExpr)
	if invalidScheduleFields(params.IntervalSeconds, cronExpr) {
		return ScheduledTask{}, invalidTaskConfig("exactly one of interval_seconds or cron_expr is required")
	}
	taskKind := normalizeTaskKind(params.TaskKind)
	if err := ensureWorkflowAllowedForTaskKind(taskKind, params.Workflow); err != nil {
		return ScheduledTask{}, err
	}

	task := ScheduledTask{
		Message:         strings.TrimSpace(params.Message),
		SessionID:       strings.TrimSpace(params.SessionID),
		TaskKind:        strings.TrimSpace(params.TaskKind),
		Action:          strings.TrimSpace(params.Action),
		ActionParams:    cloneTaskActionParams(params.ActionParams),
		Workflow:        cloneTaskWorkflow(params.Workflow),
		Enabled:         true,
		CreatedAt:       now,
		ScheduleType:    taskScheduleTypeInterval,
		IntervalSeconds: params.IntervalSeconds,
	}
	if cronExpr != "" {
		task.ScheduleType = taskScheduleTypeCron
		task.IntervalSeconds = 0
		task.CronExpr = cronExpr
	}
	return task, nil
}

func invalidScheduleFields(intervalSeconds int, cronExpr string) bool {
	return (intervalSeconds > 0 && cronExpr != "") || (intervalSeconds <= 0 && cronExpr == "")
}

func (r taskMutationRunner) applyUpdate(task *ScheduledTask, params taskUpdateParams) error {
	if task == nil {
		return invalidTaskConfig("task is nil")
	}
	previousEnabled := task.Enabled
	scheduleChanged, err := applyTaskPatch(task, params)
	if err != nil {
		return err
	}
	if err := validateTaskDefinition(task); err != nil {
		return wrapTaskConfigError(err)
	}
	if err := r.ensureTaskSessionExists(task.TaskKind, task.SessionID); err != nil {
		return err
	}
	return r.refreshNextRunAt(task, scheduleChanged, previousEnabled)
}

func applyTaskPatch(task *ScheduledTask, params taskUpdateParams) (bool, error) {
	scheduleChanged := false
	if params.Message != nil {
		task.Message = strings.TrimSpace(*params.Message)
	}
	if params.SessionID != nil {
		task.SessionID = strings.TrimSpace(*params.SessionID)
	}
	if params.TaskKind != nil {
		task.TaskKind = strings.TrimSpace(*params.TaskKind)
	}
	if params.Action != nil {
		task.Action = strings.TrimSpace(*params.Action)
	}
	if params.ActionParams != nil {
		task.ActionParams = cloneTaskActionParams(*params.ActionParams)
	}
	taskKind := normalizeTaskKind(task.TaskKind)
	if params.Workflow != nil {
		if err := ensureWorkflowAllowedForTaskKind(taskKind, params.Workflow); err != nil {
			return false, err
		}
		task.Workflow = cloneTaskWorkflow(params.Workflow)
	}
	if taskKind != taskKindWorkflow {
		task.Workflow = nil
	}
	if params.Enabled != nil {
		task.Enabled = *params.Enabled
	}
	if params.IntervalSeconds != nil && params.CronExpr != nil {
		return false, invalidTaskConfig("exactly one schedule field can be updated at a time")
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
	return scheduleChanged, nil
}

func (r taskMutationRunner) refreshNextRunAt(task *ScheduledTask, scheduleChanged bool, previousEnabled bool) error {
	if !task.Enabled {
		task.NextRunAt = time.Time{}
		return nil
	}
	if !scheduleChanged && previousEnabled && !task.NextRunAt.IsZero() {
		return nil
	}
	nextRunAt, err := nextTaskRunAt(*task, r.now())
	if err != nil {
		return wrapTaskConfigError(err)
	}
	task.NextRunAt = nextRunAt
	return nil
}

func (r taskMutationRunner) ensureTaskSessionExists(taskKind string, sessionID string) error {
	if normalizeTaskKind(taskKind) != taskKindAgentMessage {
		return nil
	}
	id := strings.TrimSpace(sessionID)
	if id == "" || r.sessionStore == nil {
		return nil
	}
	sess, err := r.sessionStore.Load(id)
	if err != nil {
		return err
	}
	if sess.IsEnded() {
		return errSessionEnded
	}
	return nil
}

func ensureWorkflowAllowedForTaskKind(taskKind string, workflow *WorkflowDefinition) error {
	if workflow == nil || normalizeTaskKind(taskKind) == taskKindWorkflow {
		return nil
	}
	return invalidTaskConfig(normalizeTaskKind(taskKind) + " does not allow workflow")
}
