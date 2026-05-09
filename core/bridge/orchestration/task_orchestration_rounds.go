package orchestration

import (
	apporchestrations "ghost-os/bridge/orchestration/internal/app/orchestrations"
)

func (r orchestrationTaskRunner) roundDispatcher() apporchestrations.RoundDispatcher {
	return apporchestrations.RoundDispatcher{Members: r.memberRunner()}
}
