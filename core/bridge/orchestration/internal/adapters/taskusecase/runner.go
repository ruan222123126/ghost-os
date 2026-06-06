package taskusecase

import (
	"context"
	"errors"
	"strings"
	"time"

	bridgeconfig "ghost-os/bridge/config"
	apptasks "ghost-os/bridge/orchestration/internal/app/tasks"
	"ghost-os/bridge/orchestration/internal/contracts/api"
	"ghost-os/bridge/session"
	bridgeTasks "ghost-os/bridge/tasks"
)

var ErrSessionEnded = errors.New("session has already ended")

type MutationStore interface {
	LoadTask(taskID string) (*bridgeTasks.ScheduledTask, error)
	SaveTask(task *bridgeTasks.ScheduledTask) error
	DeleteTask(taskID string) error
}

type MutationScheduler interface {
	Upsert(task bridgeTasks.ScheduledTask) error
	Unregister(taskID string) error
	RunNow(task bridgeTasks.ScheduledTask, traceID string) (bridgeTasks.RunLog, error)
	StartNow(task bridgeTasks.ScheduledTask, traceID string) (bridgeTasks.RunLog, error)
	StopRun(ctx context.Context, taskID string, runID string) (bridgeTasks.RunLog, error)
}

type MutationOptions struct {
	Store             MutationStore
	Scheduler         MutationScheduler
	ConfigStore       bridgeconfig.Store
	SessionStore      *session.Store
	SessionEndedError error
	Now               func() time.Time
}

type MutationRunner struct {
	store             MutationStore
	scheduler         MutationScheduler
	configStore       bridgeconfig.Store
	sessionStore      *session.Store
	sessionEndedError error
	now               func() time.Time
}

func NewQueryRunner(store apptasks.Store) apptasks.QueryRunner {
	return apptasks.QueryRunner{Store: store}
}

func NewMutationRunner(options MutationOptions) MutationRunner {
	sessionEndedError := options.SessionEndedError
	if sessionEndedError == nil {
		sessionEndedError = ErrSessionEnded
	}
	return MutationRunner{
		store:             options.Store,
		scheduler:         options.Scheduler,
		configStore:       options.ConfigStore,
		sessionStore:      options.SessionStore,
		sessionEndedError: sessionEndedError,
		now:               options.Now,
	}
}

func (r MutationRunner) RunNow(params api.TaskIDParams, traceID string) (api.TaskRunPayload, error) {
	return r.inner().RunNow(params, traceID)
}

func (r MutationRunner) Stop(ctx context.Context, params api.TaskStopParams) (api.TaskStopResponse, error) {
	return r.inner().Stop(ctx, params)
}

func (r MutationRunner) Delete(params api.TaskIDParams) (api.TaskDeleteResponse, error) {
	return r.inner().Delete(params)
}

func (r MutationRunner) Create(params api.TaskCreateParams) (api.TaskPayload, error) {
	return r.inner().Create(params)
}

func (r MutationRunner) Update(params api.TaskUpdateParams) (api.TaskPayload, error) {
	return r.inner().Update(params)
}

func (r MutationRunner) EnsureTaskSessionActive(taskKind string, sessionID string) error {
	if bridgeTasks.NormalizeKind(taskKind) != bridgeTasks.KindAgentMessage {
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
		return r.sessionEndedError
	}
	return nil
}

func (r MutationRunner) inner() apptasks.MutationRunner {
	builder := r.builder()
	return apptasks.MutationRunner{
		Store:       mutationStoreAdapter{inner: r.store},
		Scheduler:   r.scheduler,
		BuildTask:   builder.NewScheduledTask,
		ApplyUpdate: builder.ApplyUpdate,
	}
}

func (r MutationRunner) builder() apptasks.TaskMutationBuilder {
	return apptasks.TaskMutationBuilder{
		ConfigStore:  r.configStore,
		SessionGuard: r,
		Now:          r.now,
	}
}

type mutationStoreAdapter struct {
	inner MutationStore
}

func (s mutationStoreAdapter) ListTasks() ([]bridgeTasks.ScheduledTask, error) {
	return nil, errors.New("list tasks is unavailable for mutation runner")
}

func (s mutationStoreAdapter) LoadTask(taskID string) (*bridgeTasks.ScheduledTask, error) {
	return s.inner.LoadTask(taskID)
}

func (s mutationStoreAdapter) SaveTask(task *bridgeTasks.ScheduledTask) error {
	return s.inner.SaveTask(task)
}

func (s mutationStoreAdapter) DeleteTask(taskID string) error {
	return s.inner.DeleteTask(taskID)
}

func (s mutationStoreAdapter) ListRunLogs(taskID string, limit int) ([]bridgeTasks.RunLog, error) {
	return nil, errors.New("list run logs is unavailable for mutation runner")
}
