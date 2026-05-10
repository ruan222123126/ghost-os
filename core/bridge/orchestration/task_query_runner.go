package orchestration

import (
	"net/http"

	apptasks "ghost-os/bridge/orchestration/internal/app/tasks"
)

func (r taskQueryRunner) List(scope string) ([]taskPayload, error) {
	return r.inner.List(scope)
}

func (r taskQueryRunner) Get(params taskIDParams) (taskPayload, error) {
	return r.inner.Get(params)
}

func (r taskQueryRunner) Logs(params taskLogsParams) ([]taskRunLogPayload, error) {
	return r.inner.Logs(params)
}

func (s *bridgeService) requireTaskQueryRunner() (taskQueryRunner, int, error) {
	store, code, err := s.requireTaskStore()
	if err != nil {
		return taskQueryRunner{}, code, err
	}
	return taskQueryRunner{inner: apptasks.QueryRunner{Store: store}}, http.StatusOK, nil
}
