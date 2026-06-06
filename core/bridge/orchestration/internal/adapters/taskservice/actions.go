package taskservice

import (
	"context"
	"errors"
	"net/http"

	bridgeconfig "ghost-os/bridge/config"
	taskusecase "ghost-os/bridge/orchestration/internal/adapters/taskusecase"
	apptasks "ghost-os/bridge/orchestration/internal/app/tasks"
	"ghost-os/bridge/orchestration/internal/contracts/api"
	"ghost-os/bridge/session"
	bridgeTasks "ghost-os/bridge/tasks"
)

type Config struct {
	Store             *bridgeTasks.Store
	Scheduler         *bridgeTasks.TaskScheduler
	InitErr           error
	ConfigStore       bridgeconfig.Store
	SessionStore      *session.Store
	SessionEndedError error
	Logger            apptasks.Logger
}

type Actions struct {
	config Config
}

func New(config Config) Actions {
	return Actions{config: config}
}

func (a Actions) RequireStore() (*bridgeTasks.Store, int, error) {
	if a.config.Store != nil {
		return a.config.Store, http.StatusOK, nil
	}
	if a.config.InitErr != nil {
		return nil, http.StatusInternalServerError, a.config.InitErr
	}
	return nil, http.StatusInternalServerError, errors.New("task store is not configured")
}

func (a Actions) RequireScheduler() (*bridgeTasks.TaskScheduler, int, error) {
	if a.config.Scheduler != nil {
		return a.config.Scheduler, http.StatusOK, nil
	}
	if a.config.InitErr != nil {
		return nil, http.StatusInternalServerError, a.config.InitErr
	}
	return nil, http.StatusInternalServerError, errors.New("task scheduler is not configured")
}

func (a Actions) Create(params api.TaskCreateParams, traceID string) (any, int, error) {
	usecase, code, err := a.requireMutationService()
	if err != nil {
		return nil, code, err
	}
	payload, err := usecase.Create(params, traceID)
	if err != nil {
		return nil, a.mapError(err), err
	}
	return payload, http.StatusCreated, nil
}

func (a Actions) Update(params api.TaskUpdateParams, traceID string) (any, int, error) {
	usecase, code, err := a.requireMutationService()
	if err != nil {
		return nil, code, err
	}
	payload, err := usecase.Update(params, traceID)
	if err != nil {
		return nil, a.mapError(err), err
	}
	return payload, http.StatusOK, nil
}

func (a Actions) List(scope string, traceID string) (any, int, error) {
	usecase, code, err := a.requireQueryService()
	if err != nil {
		return nil, code, err
	}
	payload, err := usecase.List(scope, traceID)
	if err != nil {
		return nil, a.mapError(err), err
	}
	return payload, http.StatusOK, nil
}

func (a Actions) Get(params api.TaskIDParams, traceID string) (any, int, error) {
	usecase, code, err := a.requireQueryService()
	if err != nil {
		return nil, code, err
	}
	payload, err := usecase.Get(params, traceID)
	if err != nil {
		return nil, a.mapError(err), err
	}
	return payload, http.StatusOK, nil
}

func (a Actions) Logs(params api.TaskLogsParams, traceID string) (any, int, error) {
	usecase, code, err := a.requireQueryService()
	if err != nil {
		return nil, code, err
	}
	payload, err := usecase.Logs(params, traceID)
	if err != nil {
		return nil, a.mapError(err), err
	}
	return payload, http.StatusOK, nil
}

func (a Actions) RunNow(params api.TaskIDParams, traceID string) (any, int, error) {
	usecase, code, err := a.requireMutationService()
	if err != nil {
		return nil, code, err
	}
	payload, err := usecase.RunNow(params, traceID)
	if err != nil {
		return nil, a.mapError(err), err
	}
	return payload, http.StatusOK, nil
}

func (a Actions) Stop(ctx context.Context, params api.TaskStopParams, traceID string) (any, int, error) {
	usecase, code, err := a.requireMutationService()
	if err != nil {
		return nil, code, err
	}
	payload, err := usecase.Stop(ctx, params, traceID)
	if err != nil {
		return nil, a.mapError(err), err
	}
	return payload, http.StatusOK, nil
}

func (a Actions) Delete(params api.TaskIDParams, traceID string) (any, int, error) {
	usecase, code, err := a.requireMutationService()
	if err != nil {
		return nil, code, err
	}
	payload, err := usecase.Delete(params, traceID)
	if err != nil {
		return nil, a.mapError(err), err
	}
	return payload, http.StatusOK, nil
}

func (a Actions) requireQueryService() (apptasks.Service, int, error) {
	runner, code, err := a.RequireQueryRunner()
	if err != nil {
		return apptasks.Service{}, code, err
	}
	return apptasks.Service{
		Query:  runner,
		Logger: a.config.Logger,
	}, http.StatusOK, nil
}

func (a Actions) requireMutationService() (apptasks.Service, int, error) {
	runner, code, err := a.RequireMutationRunner()
	if err != nil {
		return apptasks.Service{}, code, err
	}
	return apptasks.Service{
		Mutation: runner,
		Logger:   a.config.Logger,
	}, http.StatusOK, nil
}

func (a Actions) RequireQueryRunner() (apptasks.Query, int, error) {
	store, code, err := a.RequireStore()
	if err != nil {
		return nil, code, err
	}
	return taskusecase.NewQueryRunner(store), http.StatusOK, nil
}

func (a Actions) RequireMutationRunner() (apptasks.Mutation, int, error) {
	store, code, err := a.RequireStore()
	if err != nil {
		return nil, code, err
	}
	scheduler, code, err := a.RequireScheduler()
	if err != nil {
		return nil, code, err
	}
	return taskusecase.NewMutationRunner(taskusecase.MutationOptions{
		Store:             store,
		Scheduler:         scheduler,
		ConfigStore:       a.config.ConfigStore,
		SessionStore:      a.config.SessionStore,
		SessionEndedError: a.config.SessionEndedError,
	}), http.StatusOK, nil
}

func (a Actions) mapError(err error) int {
	switch {
	case errors.Is(err, apptasks.ErrQueryRunnerRequired),
		errors.Is(err, apptasks.ErrMutationRunnerRequired):
		return http.StatusInternalServerError
	case errors.Is(err, bridgeTasks.ErrInvalidTaskID),
		errors.Is(err, bridgeTasks.ErrInvalidTaskConfig),
		errors.Is(err, session.ErrInvalidSessionID):
		return http.StatusBadRequest
	case errors.Is(err, bridgeTasks.ErrTaskNotFound),
		errors.Is(err, session.ErrSessionNotFound):
		return http.StatusNotFound
	case errors.Is(err, a.config.SessionEndedError),
		errors.Is(err, taskusecase.ErrSessionEnded):
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}
