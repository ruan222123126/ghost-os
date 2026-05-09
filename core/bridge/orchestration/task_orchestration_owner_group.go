package orchestration

import apporchestrations "ghost-os/bridge/orchestration/internal/app/orchestrations"

func (r orchestrationTaskRunner) ownerGroupExecutor() apporchestrations.OwnerGroupExecutor {
	return apporchestrations.OwnerGroupExecutor{
		Decisions: r.ownerDecisionRunner(),
		Dispatches: apporchestrations.DispatchExecutor{
			Dispatcher: r.roundDispatcher(),
		},
	}
}
