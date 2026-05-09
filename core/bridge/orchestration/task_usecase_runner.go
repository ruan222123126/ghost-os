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
	taskListScopeUser          = "user"
	taskListScopeSystem        = "system"
	taskListScopeOrchestration = "orchestration"
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

func ensureTaskMatchesScope(task ScheduledTask, scope string) error {
	if includeTaskInScope(task, scope) {
		return nil
	}
	return ErrTaskNotFound
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
	if err := ensureTaskMatchesScope(*task, params.Scope); err != nil {
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
