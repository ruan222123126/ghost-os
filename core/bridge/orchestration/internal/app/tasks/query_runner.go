package tasks

import (
	"ghost-os/bridge/orchestration/internal/contracts/api"
)

type QueryRunner struct {
	Store Store
}

func (r QueryRunner) List(scope string) ([]api.TaskPayload, error) {
	tasks, err := r.Store.ListTasks()
	if err != nil {
		return nil, err
	}
	payloads := make([]api.TaskPayload, 0, len(tasks))
	for _, task := range tasks {
		if IncludeInScope(task, scope) {
			payloads = append(payloads, BuildPayload(task))
		}
	}
	return payloads, nil
}

func (r QueryRunner) Get(params api.TaskIDParams) (api.TaskPayload, error) {
	task, err := r.loadScopedTask(params.ID, params.Scope)
	if err != nil {
		return api.TaskPayload{}, err
	}
	return BuildPayload(*task), nil
}

func (r QueryRunner) Logs(params api.TaskLogsParams) ([]api.TaskRunLogPayload, error) {
	id, err := NormalizeID(params.ID)
	if err != nil {
		return nil, err
	}
	if params.Limit < 0 {
		return nil, InvalidConfig("limit must be >= 0")
	}
	task, err := r.loadScopedTask(id, params.Scope)
	if err != nil {
		return nil, err
	}
	runs, err := r.Store.ListRunLogs(task.ID, params.Limit)
	if err != nil {
		return nil, err
	}
	return runLogPayloads(runs), nil
}

func (r QueryRunner) loadScopedTask(id string, scope string) (*api.TaskPayload, error) {
	normalizedID, err := NormalizeID(id)
	if err != nil {
		return nil, err
	}
	task, err := r.Store.LoadTask(normalizedID)
	if err != nil {
		return nil, err
	}
	if err := EnsureMatchesScope(*task, scope); err != nil {
		return nil, err
	}
	payload := BuildPayload(*task)
	return &payload, nil
}

func runLogPayloads(runs []api.TaskRunLogPayload) []api.TaskRunLogPayload {
	payloads := make([]api.TaskRunLogPayload, 0, len(runs))
	for _, run := range runs {
		payloads = append(payloads, BuildRunLogPayload(run))
	}
	return payloads
}
