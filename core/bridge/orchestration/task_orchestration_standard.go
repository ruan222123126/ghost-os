package orchestration

import apporchestrations "ghost-os/bridge/orchestration/internal/app/orchestrations"

func (r orchestrationTaskRunner) standardGroupExecutor() apporchestrations.StandardGroupExecutor {
	return apporchestrations.StandardGroupExecutor{Dispatcher: r.roundDispatcher()}
}
