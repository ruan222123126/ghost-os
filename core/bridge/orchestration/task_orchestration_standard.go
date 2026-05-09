package orchestration

import (
	"context"

	apporchestrations "ghost-os/bridge/orchestration/internal/app/orchestrations"
)

func (r orchestrationTaskRunner) standardGroupExecutor() apporchestrations.StandardGroupExecutor {
	return apporchestrations.StandardGroupExecutor{Dispatcher: r.roundDispatcher()}
}

func (r orchestrationTaskRunner) executeStandardGroupNode(
	ctx context.Context,
	node OrchestrationNode,
	state *orchestrationRunState,
) orchestrationGroupResult {
	groupID := node.ID
	result := r.standardGroupExecutor().Execute(ctx, apporchestrations.StandardGroupCommand{
		GroupNode:          node,
		MemberNodes:        r.plan.nodes,
		MemberOrder:        append([]string(nil), r.plan.groupMember[groupID]...),
		PreviousGroupID:    state.lastGroupID,
		PreviousTranscript: state.lastGroupTranscript,
		MemberSessions:     apporchestrations.CloneSessionIDs(state.memberSessionIDs[groupID]),
		TraceID:            r.traceID,
	})
	if result.Status == taskRunStatusSuccess {
		state.memberSessionIDs[groupID] = apporchestrations.CloneSessionIDs(result.MemberSessions)
	}
	return orchestrationGroupResult{
		status:          result.Status,
		preview:         result.Preview,
		errText:         result.Error,
		completedRounds: result.CompletedRounds,
		transcript:      result.Transcript,
		memberResults:   result.MemberResults,
		memberSessions:  result.MemberSessions,
	}
}
