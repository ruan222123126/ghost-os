package orchestration

import (
	"errors"
	"fmt"
)

func (r taskMutationRunner) Create(params taskCreateParams) (taskPayload, error) {
	task, err := r.newScheduledTask(params)
	if err != nil {
		return taskPayload{}, err
	}
	if err := r.store.SaveTask(&task); err != nil {
		return taskPayload{}, err
	}
	if err := r.scheduler.Upsert(task); err != nil {
		_ = r.store.DeleteTask(task.ID)
		return taskPayload{}, err
	}
	return buildTaskPayload(task), nil
}

func (r taskMutationRunner) Update(params taskUpdateParams) (taskPayload, error) {
	id, err := normalizeTaskID(params.ID)
	if err != nil {
		return taskPayload{}, err
	}
	task, err := r.store.LoadTask(id)
	if err != nil {
		return taskPayload{}, err
	}
	previous := cloneScheduledTask(*task)
	if err := r.applyUpdate(task, params); err != nil {
		return taskPayload{}, err
	}
	if err := r.scheduler.Unregister(id); err != nil {
		return taskPayload{}, err
	}
	if err := r.store.SaveTask(task); err != nil {
		return taskPayload{}, wrapTaskUpdateRollbackError(err, r.restoreTaskRegistration(previous))
	}
	if err := r.scheduler.Upsert(*task); err != nil {
		return taskPayload{}, wrapTaskUpdateRollbackError(err, r.rollbackPersistedTaskUpdate(previous))
	}
	return buildTaskPayload(*task), nil
}

func cloneScheduledTask(task ScheduledTask) ScheduledTask {
	cloned := task
	cloned.ActionParams = cloneTaskActionParams(task.ActionParams)
	cloned.Workflow = cloneTaskWorkflow(task.Workflow)
	return cloned
}

func (r taskMutationRunner) restoreTaskRegistration(task ScheduledTask) error {
	return r.scheduler.Upsert(task)
}

func (r taskMutationRunner) rollbackPersistedTaskUpdate(task ScheduledTask) error {
	rollbackTask := cloneScheduledTask(task)
	return errors.Join(
		r.store.SaveTask(&rollbackTask),
		r.scheduler.Upsert(task),
	)
}

func wrapTaskUpdateRollbackError(updateErr error, rollbackErr error) error {
	if rollbackErr == nil {
		return updateErr
	}
	return fmt.Errorf("%w; rollback failed: %v", updateErr, rollbackErr)
}
