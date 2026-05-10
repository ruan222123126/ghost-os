package orchestration

import (
	"errors"
	"net/http"
	"time"

	bridgeconfig "ghost-os/bridge/config"
	apptasks "ghost-os/bridge/orchestration/internal/app/tasks"
	"ghost-os/bridge/session"
)

const (
	taskListScopeUser          = apptasks.ScopeUser
	taskListScopeSystem        = apptasks.ScopeSystem
	taskListScopeOrchestration = apptasks.ScopeOrchestration
)

type taskQueryRunner struct {
	inner apptasks.QueryRunner
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
	return apptasks.NormalizeID(id)
}

func invalidTaskConfig(message string) error {
	return apptasks.InvalidConfig(message)
}

func wrapTaskConfigError(err error) error {
	return apptasks.WrapConfigError(err)
}

func ensureTaskMatchesScope(task ScheduledTask, scope string) error {
	return apptasks.EnsureMatchesScope(task, scope)
}

func (r taskMutationRunner) RunNow(params taskIDParams, traceID string) (taskRunPayload, error) {
	return r.inner().RunNow(params, traceID)
}

func (r taskMutationRunner) inner() apptasks.MutationRunner {
	return apptasks.MutationRunner{
		Store:       taskMutationStoreAdapter{inner: r.store},
		Scheduler:   r.scheduler,
		BuildTask:   r.newScheduledTask,
		ApplyUpdate: r.applyUpdate,
	}
}

type taskMutationStoreAdapter struct {
	inner taskMutationStore
}

func (s taskMutationStoreAdapter) ListTasks() ([]ScheduledTask, error) {
	return nil, errors.New("list tasks is unavailable for mutation runner")
}

func (s taskMutationStoreAdapter) LoadTask(taskID string) (*ScheduledTask, error) {
	return s.inner.LoadTask(taskID)
}

func (s taskMutationStoreAdapter) SaveTask(task *ScheduledTask) error {
	return s.inner.SaveTask(task)
}

func (s taskMutationStoreAdapter) DeleteTask(taskID string) error {
	return s.inner.DeleteTask(taskID)
}

func (s taskMutationStoreAdapter) ListRunLogs(taskID string, limit int) ([]TaskRunLog, error) {
	return nil, errors.New("list run logs is unavailable for mutation runner")
}
