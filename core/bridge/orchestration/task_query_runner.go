package orchestration

import "net/http"

func (r taskQueryRunner) List(scope string) ([]taskPayload, error) {
	tasks, err := r.store.ListTasks()
	if err != nil {
		return nil, err
	}
	payloads := make([]taskPayload, 0, len(tasks))
	for _, task := range tasks {
		if includeTaskInScope(task, scope) {
			payloads = append(payloads, buildTaskPayload(task))
		}
	}
	return payloads, nil
}

func (r taskQueryRunner) Get(params taskIDParams) (taskPayload, error) {
	id, err := normalizeTaskID(params.ID)
	if err != nil {
		return taskPayload{}, err
	}
	task, err := r.store.LoadTask(id)
	if err != nil {
		return taskPayload{}, err
	}
	return buildTaskPayload(*task), nil
}

func (r taskQueryRunner) Logs(params taskLogsParams) ([]taskRunLogPayload, error) {
	id, err := normalizeTaskID(params.ID)
	if err != nil {
		return nil, err
	}
	if params.Limit < 0 {
		return nil, invalidTaskConfig("limit must be >= 0")
	}
	if _, err := r.store.LoadTask(id); err != nil {
		return nil, err
	}
	runs, err := r.store.ListRunLogs(id, params.Limit)
	if err != nil {
		return nil, err
	}
	payloads := make([]taskRunLogPayload, 0, len(runs))
	for _, run := range runs {
		payloads = append(payloads, buildTaskRunLogPayload(run))
	}
	return payloads, nil
}

func (s *bridgeService) requireTaskQueryRunner() (taskQueryRunner, int, error) {
	store, code, err := s.requireTaskStore()
	if err != nil {
		return taskQueryRunner{}, code, err
	}
	return taskQueryRunner{store: store}, http.StatusOK, nil
}
