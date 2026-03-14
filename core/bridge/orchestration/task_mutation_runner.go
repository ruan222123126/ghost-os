package orchestration

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
	if err := r.applyUpdate(task, params); err != nil {
		return taskPayload{}, err
	}
	if err := r.store.SaveTask(task); err != nil {
		return taskPayload{}, err
	}
	if err := r.scheduler.Upsert(*task); err != nil {
		return taskPayload{}, err
	}
	return buildTaskPayload(*task), nil
}
