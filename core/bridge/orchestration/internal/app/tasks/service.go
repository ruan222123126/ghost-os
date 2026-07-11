package tasks

import (
	"context"
	"errors"

	"ghost-os/bridge/orchestration/internal/contracts/api"
	"ghost-os/bridge/orchestration/internal/contracts/bus"
)

var (
	ErrQueryRunnerRequired    = errors.New("task query runner is not configured")
	ErrMutationRunnerRequired = errors.New("task mutation runner is not configured")
)

type Query interface {
	List(scope string) ([]api.TaskPayload, error)
	Get(params api.TaskIDParams) (api.TaskPayload, error)
	Logs(params api.TaskLogsParams) ([]api.TaskRunLogPayload, error)
}

type Mutation interface {
	Create(params api.TaskCreateParams) (api.TaskPayload, error)
	Update(params api.TaskUpdateParams) (api.TaskPayload, error)
	RunNow(params api.TaskRunNowParams, traceID string) (api.TaskRunPayload, error)
	Stop(ctx context.Context, params api.TaskStopParams) (api.TaskStopResponse, error)
	Delete(params api.TaskIDParams) (api.TaskDeleteResponse, error)
}

type Logger interface {
	Log(traceID string, action string, status string, err error)
}

type Service struct {
	Query    Query
	Mutation Mutation
	Logger   Logger
}

func (s Service) Create(params api.TaskCreateParams, traceID string) (api.TaskPayload, error) {
	mutation, err := s.requireMutation()
	if err != nil {
		s.log(traceID, bus.ActionTaskCreate, "error", err)
		return api.TaskPayload{}, err
	}
	payload, err := mutation.Create(params)
	s.logResult(traceID, bus.ActionTaskCreate, err)
	return payload, err
}

func (s Service) Update(params api.TaskUpdateParams, traceID string) (api.TaskPayload, error) {
	mutation, err := s.requireMutation()
	if err != nil {
		s.log(traceID, bus.ActionTaskUpdate, "error", err)
		return api.TaskPayload{}, err
	}
	payload, err := mutation.Update(params)
	s.logResult(traceID, bus.ActionTaskUpdate, err)
	return payload, err
}

func (s Service) List(scope string, traceID string) ([]api.TaskPayload, error) {
	query, err := s.requireQuery()
	if err != nil {
		s.log(traceID, bus.ActionTaskList, "error", err)
		return nil, err
	}
	payload, err := query.List(scope)
	s.logResult(traceID, bus.ActionTaskList, err)
	return payload, err
}

func (s Service) Get(params api.TaskIDParams, traceID string) (api.TaskPayload, error) {
	query, err := s.requireQuery()
	if err != nil {
		s.log(traceID, bus.ActionTaskGet, "error", err)
		return api.TaskPayload{}, err
	}
	payload, err := query.Get(params)
	s.logResult(traceID, bus.ActionTaskGet, err)
	return payload, err
}

func (s Service) Logs(params api.TaskLogsParams, traceID string) ([]api.TaskRunLogPayload, error) {
	query, err := s.requireQuery()
	if err != nil {
		s.log(traceID, bus.ActionTaskLogs, "error", err)
		return nil, err
	}
	payload, err := query.Logs(params)
	s.logResult(traceID, bus.ActionTaskLogs, err)
	return payload, err
}

func (s Service) RunNow(params api.TaskRunNowParams, traceID string) (api.TaskRunPayload, error) {
	mutation, err := s.requireMutation()
	if err != nil {
		s.log(traceID, bus.ActionTaskRunNow, "error", err)
		return api.TaskRunPayload{}, err
	}
	payload, err := mutation.RunNow(params, traceID)
	s.logResult(traceID, bus.ActionTaskRunNow, err)
	return payload, err
}

func (s Service) Stop(
	ctx context.Context,
	params api.TaskStopParams,
	traceID string,
) (api.TaskStopResponse, error) {
	mutation, err := s.requireMutation()
	if err != nil {
		s.log(traceID, bus.ActionTaskStop, "error", err)
		return api.TaskStopResponse{}, err
	}
	payload, err := mutation.Stop(ctx, params)
	if err != nil {
		s.log(traceID, bus.ActionTaskStop, "error", err)
		return payload, err
	}
	status := "success"
	if payload.Status == "not_running" {
		status = "not_running"
	}
	s.log(traceID, bus.ActionTaskStop, status, nil)
	return payload, nil
}

func (s Service) Delete(params api.TaskIDParams, traceID string) (api.TaskDeleteResponse, error) {
	mutation, err := s.requireMutation()
	if err != nil {
		s.log(traceID, bus.ActionTaskDelete, "error", err)
		return api.TaskDeleteResponse{}, err
	}
	payload, err := mutation.Delete(params)
	s.logResult(traceID, bus.ActionTaskDelete, err)
	return payload, err
}

func (s Service) requireQuery() (Query, error) {
	if s.Query == nil {
		return nil, ErrQueryRunnerRequired
	}
	return s.Query, nil
}

func (s Service) requireMutation() (Mutation, error) {
	if s.Mutation == nil {
		return nil, ErrMutationRunnerRequired
	}
	return s.Mutation, nil
}

func (s Service) logResult(traceID string, action string, err error) {
	if err != nil {
		s.log(traceID, action, "error", err)
		return
	}
	s.log(traceID, action, "success", nil)
}

func (s Service) log(traceID string, action string, status string, err error) {
	if s.Logger != nil {
		s.Logger.Log(traceID, action, status, err)
	}
}
