package tasks

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"ghost-os/bridge/orchestration/internal/contracts/api"
	bridgeTasks "ghost-os/bridge/tasks"
)

type BuildTaskFunc func(api.TaskCreateParams) (bridgeTasks.ScheduledTask, error)
type ApplyUpdateFunc func(*bridgeTasks.ScheduledTask, api.TaskUpdateParams) error

type MutationRunner struct {
	Store       Store
	Scheduler   Scheduler
	BuildTask   BuildTaskFunc
	ApplyUpdate ApplyUpdateFunc
}

func (r MutationRunner) Create(params api.TaskCreateParams) (api.TaskPayload, error) {
	task, err := r.BuildTask(params)
	if err != nil {
		return api.TaskPayload{}, err
	}
	if err := EnsureMatchesScope(task, params.Scope); err != nil {
		return api.TaskPayload{}, err
	}
	if err := r.Store.SaveTask(&task); err != nil {
		return api.TaskPayload{}, err
	}
	if err := r.Scheduler.Upsert(task); err != nil {
		_ = r.Store.DeleteTask(task.ID)
		return api.TaskPayload{}, err
	}
	return BuildPayload(task), nil
}

func (r MutationRunner) Update(params api.TaskUpdateParams) (api.TaskPayload, error) {
	id, task, err := r.loadForMutation(params.ID, params.Scope)
	if err != nil {
		return api.TaskPayload{}, err
	}
	previous := CloneScheduledTask(*task)
	if err := r.ApplyUpdate(task, params); err != nil {
		return api.TaskPayload{}, err
	}
	if err := EnsureMatchesScope(*task, params.Scope); err != nil {
		return api.TaskPayload{}, err
	}
	return r.persistUpdatedTask(id, task, previous)
}

func (r MutationRunner) Delete(params api.TaskIDParams) (api.TaskDeleteResponse, error) {
	id, task, err := r.loadForMutation(params.ID, params.Scope)
	if err != nil {
		return api.TaskDeleteResponse{}, err
	}
	if err := r.Scheduler.Unregister(id); err != nil {
		return api.TaskDeleteResponse{}, err
	}
	if err := r.Store.DeleteTask(id); err != nil {
		return api.TaskDeleteResponse{}, wrapRollbackError(err, r.rollbackTaskDeletion(*task))
	}
	return api.TaskDeleteResponse{ID: id, Deleted: true}, nil
}

func (r MutationRunner) RunNow(params api.TaskRunNowParams, traceID string) (api.TaskRunPayload, error) {
	id, task, err := r.loadForMutation(params.ID, params.Scope)
	if err != nil {
		return api.TaskRunPayload{}, err
	}
	if StartOnlyEnabled(params.StartOnly) {
		return r.startNow(*task, traceID)
	}
	run, err := r.Scheduler.RunNow(*task, traceID)
	if err != nil {
		return api.TaskRunPayload{}, err
	}
	updatedTask, err := r.Store.LoadTask(id)
	if err != nil {
		return api.TaskRunPayload{}, err
	}
	return api.TaskRunPayload{Task: BuildPayload(*updatedTask), Run: BuildRunLogPayload(run)}, nil
}

func (r MutationRunner) startNow(task bridgeTasks.ScheduledTask, traceID string) (api.TaskRunPayload, error) {
	run, err := r.Scheduler.StartNow(task, traceID)
	if err != nil {
		return api.TaskRunPayload{}, err
	}
	return api.TaskRunPayload{Task: BuildPayload(task), Run: BuildRunLogPayload(run)}, nil
}

func (r MutationRunner) Stop(ctx context.Context, params api.TaskStopParams) (api.TaskStopResponse, error) {
	id, task, err := r.loadForMutation(params.ID, params.Scope)
	if err != nil {
		return api.TaskStopResponse{}, err
	}
	runID := strings.TrimSpace(params.RunID)
	if runID == "" {
		return api.TaskStopResponse{}, InvalidConfig("run_id is required")
	}
	run, err := r.Scheduler.StopRun(ctx, id, runID)
	if errors.Is(err, bridgeTasks.ErrTaskRunNotRunning) {
		return api.TaskStopResponse{
			Status:  "not_running",
			Message: "task run is not running",
			TaskID:  task.ID,
			RunID:   runID,
		}, nil
	}
	if err != nil {
		return api.TaskStopResponse{}, err
	}
	payload := BuildRunLogPayload(run)
	return api.TaskStopResponse{
		Status:  "stopped",
		Message: "task run cancelled successfully",
		TaskID:  task.ID,
		RunID:   run.RunID,
		Run:     &payload,
	}, nil
}

func (r MutationRunner) loadForMutation(id string, scope string) (string, *bridgeTasks.ScheduledTask, error) {
	normalizedID, err := NormalizeID(id)
	if err != nil {
		return "", nil, err
	}
	task, err := r.Store.LoadTask(normalizedID)
	if err != nil {
		return "", nil, err
	}
	if err := EnsureMatchesScope(*task, scope); err != nil {
		return "", nil, err
	}
	return normalizedID, task, nil
}

func (r MutationRunner) persistUpdatedTask(
	id string,
	task *bridgeTasks.ScheduledTask,
	previous bridgeTasks.ScheduledTask,
) (api.TaskPayload, error) {
	if err := r.Scheduler.Unregister(id); err != nil {
		return api.TaskPayload{}, err
	}
	if err := r.Store.SaveTask(task); err != nil {
		return api.TaskPayload{}, wrapRollbackError(err, r.restoreTaskRegistration(previous))
	}
	if err := r.Scheduler.Upsert(*task); err != nil {
		return api.TaskPayload{}, wrapRollbackError(err, r.rollbackPersistedTaskUpdate(previous))
	}
	return BuildPayload(*task), nil
}

func (r MutationRunner) restoreTaskRegistration(task bridgeTasks.ScheduledTask) error {
	return r.Scheduler.Upsert(task)
}

func (r MutationRunner) rollbackPersistedTaskUpdate(task bridgeTasks.ScheduledTask) error {
	rollbackTask := CloneScheduledTask(task)
	return errors.Join(r.Store.SaveTask(&rollbackTask), r.Scheduler.Upsert(task))
}

func (r MutationRunner) rollbackTaskDeletion(task bridgeTasks.ScheduledTask) error {
	rollbackTask := CloneScheduledTask(task)
	return errors.Join(r.Store.SaveTask(&rollbackTask), r.Scheduler.Upsert(task))
}

func wrapRollbackError(updateErr error, rollbackErr error) error {
	if rollbackErr == nil {
		return updateErr
	}
	return fmt.Errorf("%w; rollback failed: %v", updateErr, rollbackErr)
}
