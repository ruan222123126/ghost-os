package orchestration

import (
	apptasks "ghost-os/bridge/orchestration/internal/app/tasks"
)

func (r taskMutationRunner) Create(params taskCreateParams) (taskPayload, error) {
	return r.inner().Create(params)
}

func (r taskMutationRunner) Update(params taskUpdateParams) (taskPayload, error) {
	return r.inner().Update(params)
}

func cloneScheduledTask(task ScheduledTask) ScheduledTask {
	return apptasks.CloneScheduledTask(task)
}
