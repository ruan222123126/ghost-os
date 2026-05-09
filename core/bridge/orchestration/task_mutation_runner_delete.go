package orchestration

func (r taskMutationRunner) Delete(params taskIDParams) (taskDeleteResponse, error) {
	id, err := normalizeTaskID(params.ID)
	if err != nil {
		return taskDeleteResponse{}, err
	}
	task, err := r.store.LoadTask(id)
	if err != nil {
		return taskDeleteResponse{}, err
	}
	if err := ensureTaskMatchesScope(*task, params.Scope); err != nil {
		return taskDeleteResponse{}, err
	}
	if err := r.scheduler.Unregister(id); err != nil {
		return taskDeleteResponse{}, err
	}
	if err := r.store.DeleteTask(id); err != nil {
		return taskDeleteResponse{}, wrapTaskMutationRollbackError(err, r.rollbackTaskDeletion(*task))
	}
	return taskDeleteResponse{ID: id, Deleted: true}, nil
}
