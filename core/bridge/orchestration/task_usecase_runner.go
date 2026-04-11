package orchestration

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	bridgeconfig "ghost-os/bridge/config"
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
	configStore  bridgeconfig.Store
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
		configStore:  s.configStore,
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
	if err := r.validateTaskRuntime(&task); err != nil {
		return ScheduledTask{}, err
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
		Message:          strings.TrimSpace(params.Message),
		SessionID:        strings.TrimSpace(params.SessionID),
		RuntimeOverrides: cloneTaskRuntimeOverrides(params.RuntimeOverrides),
		TaskKind:         strings.TrimSpace(params.TaskKind),
		Action:           strings.TrimSpace(params.Action),
		ActionParams:     cloneTaskActionParams(params.ActionParams),
		Workflow:         cloneTaskWorkflow(params.Workflow),
		Enabled:          true,
		CreatedAt:        now,
		ScheduleType:     taskScheduleTypeInterval,
		IntervalSeconds:  params.IntervalSeconds,
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
	if err := r.validateTaskRuntime(task); err != nil {
		return err
	}
	if err := r.ensureTaskSessionExists(task.TaskKind, task.SessionID); err != nil {
		return err
	}
	return r.refreshNextRunAt(task, scheduleChanged, previousEnabled)
}

func applyTaskPatch(task *ScheduledTask, params taskUpdateParams) (bool, error) {
	applyTaskCoreTextFields(task, params)
	applyTaskRuntimePatch(task, params.RuntimeOverrides)
	taskKind := applyTaskKindActionPatch(task, params)
	applyTaskActionParamsPatch(task, params.ActionParams)
	if err := applyTaskWorkflowPatch(task, taskKind, params.Workflow); err != nil {
		return false, err
	}
	applyTaskEnabledPatch(task, params.Enabled)
	return applyTaskSchedulePatch(task, params.IntervalSeconds, params.CronExpr)
}

func applyTaskCoreTextFields(task *ScheduledTask, params taskUpdateParams) {
	if params.Message != nil {
		task.Message = strings.TrimSpace(*params.Message)
	}
	if params.SessionID != nil {
		task.SessionID = strings.TrimSpace(*params.SessionID)
	}
}

func applyTaskRuntimePatch(task *ScheduledTask, runtimeOverrides *TaskRuntimeOverrides) {
	if runtimeOverrides != nil {
		task.RuntimeOverrides = cloneTaskRuntimeOverrides(runtimeOverrides)
	}
}

func applyTaskKindActionPatch(task *ScheduledTask, params taskUpdateParams) string {
	if params.TaskKind != nil {
		task.TaskKind = strings.TrimSpace(*params.TaskKind)
	}
	if params.Action != nil {
		task.Action = strings.TrimSpace(*params.Action)
	}
	return normalizeTaskKind(task.TaskKind)
}

func applyTaskActionParamsPatch(task *ScheduledTask, actionParams *map[string]any) {
	if actionParams != nil {
		task.ActionParams = cloneTaskActionParams(*actionParams)
	}
}

func applyTaskWorkflowPatch(task *ScheduledTask, taskKind string, workflow *WorkflowDefinition) error {
	if workflow != nil {
		if err := ensureWorkflowAllowedForTaskKind(taskKind, workflow); err != nil {
			return err
		}
		task.Workflow = cloneTaskWorkflow(workflow)
	}
	if taskKind != taskKindWorkflow {
		task.Workflow = nil
	}
	return nil
}

func applyTaskEnabledPatch(task *ScheduledTask, enabled *bool) {
	if enabled != nil {
		task.Enabled = *enabled
	}
}

func applyTaskSchedulePatch(task *ScheduledTask, intervalSeconds *int, cronExpr *string) (bool, error) {
	if intervalSeconds != nil && cronExpr != nil {
		return false, invalidTaskConfig("exactly one schedule field can be updated at a time")
	}
	if intervalSeconds != nil {
		task.ScheduleType = taskScheduleTypeInterval
		task.IntervalSeconds = *intervalSeconds
		task.CronExpr = ""
		return true, nil
	}
	if cronExpr != nil {
		task.ScheduleType = taskScheduleTypeCron
		task.CronExpr = strings.TrimSpace(*cronExpr)
		task.IntervalSeconds = 0
		return true, nil
	}
	return false, nil
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
	normalized := normalizeTaskKind(taskKind)
	if workflow == nil || normalized == taskKindWorkflow {
		return nil
	}
	if !isSupportedTaskKind(normalized) {
		return invalidTaskConfig(fmt.Sprintf("unsupported task_kind %q", normalized))
	}
	return invalidTaskConfig(normalized + " does not allow workflow")
}
