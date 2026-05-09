package orchestration

import (
	"context"

	apporchestrations "ghost-os/bridge/orchestration/internal/app/orchestrations"
	groupdomain "ghost-os/bridge/orchestration/internal/domain/group"
)

func (r orchestrationTaskRunner) roundDispatcher() apporchestrations.RoundDispatcher {
	return apporchestrations.RoundDispatcher{Members: r.memberRunner()}
}

func (r orchestrationTaskRunner) executeOwnerPublicDispatch(
	ctx context.Context,
	groupNode OrchestrationNode,
	participantIDs []string,
	transcript orchestrationTranscript,
	memberSessions map[string]string,
	round int,
	order string,
	instruction string,
) []orchestrationMemberResult {
	result := r.roundDispatcher().Dispatch(ctx, apporchestrations.RoundDispatchCommand{
		GroupNode:        groupNode,
		MemberNodes:      r.plan.nodes,
		ParticipantIDs:   append([]string(nil), participantIDs...),
		Transcript:       transcript,
		MemberSessionIDs: memberSessions,
		Round:            round,
		Order:            order,
		Instruction:      instruction,
		TraceID:          r.traceID,
	})
	return result.MemberResults
}

func (r orchestrationTaskRunner) executeOwnerPrivateDispatch(
	ctx context.Context,
	groupNode OrchestrationNode,
	participantIDs []string,
	publicTranscript orchestrationTranscript,
	round int,
	instruction string,
) ([]orchestrationMemberResult, orchestrationTranscript) {
	result := r.roundDispatcher().Dispatch(ctx, apporchestrations.RoundDispatchCommand{
		GroupNode:      groupNode,
		MemberNodes:    r.plan.nodes,
		ParticipantIDs: append([]string(nil), participantIDs...),
		Transcript:     publicTranscript,
		Round:          round,
		Order:          groupdomain.SpeakingModeSequential,
		Instruction:    instruction,
		Private:        true,
		TraceID:        r.traceID,
	})
	return result.MemberResults, result.Transcript
}

func (r orchestrationTaskRunner) executeGroupRound(
	ctx context.Context,
	groupID string,
	groupNode OrchestrationNode,
	memberOrder []string,
	transcript orchestrationTranscript,
	memberSessions map[string]string,
	round int,
) []orchestrationMemberResult {
	_ = groupID
	result := r.roundDispatcher().Dispatch(ctx, apporchestrations.RoundDispatchCommand{
		GroupNode:        groupNode,
		MemberNodes:      r.plan.nodes,
		ParticipantIDs:   append([]string(nil), memberOrder...),
		Transcript:       transcript,
		MemberSessionIDs: memberSessions,
		Round:            round,
		Order:            groupNode.Group.SpeakingMode,
		TraceID:          r.traceID,
	})
	return result.MemberResults
}

func (r orchestrationTaskRunner) executeSequentialRound(
	ctx context.Context,
	groupID string,
	groupNode OrchestrationNode,
	memberOrder []string,
	transcript orchestrationTranscript,
	memberSessions map[string]string,
	round int,
) []orchestrationMemberResult {
	_ = groupID
	result := r.dispatchMemberRound(ctx, groupNode, memberOrder, transcript, memberSessions, round, groupdomain.SpeakingModeSequential)
	return result.MemberResults
}

func (r orchestrationTaskRunner) executeParallelRound(
	ctx context.Context,
	groupID string,
	groupNode OrchestrationNode,
	memberOrder []string,
	transcript orchestrationTranscript,
	memberSessions map[string]string,
	round int,
) []orchestrationMemberResult {
	_ = groupID
	result := r.dispatchMemberRound(ctx, groupNode, memberOrder, transcript, memberSessions, round, groupdomain.SpeakingModeParallel)
	return result.MemberResults
}

func (r orchestrationTaskRunner) dispatchMemberRound(
	ctx context.Context,
	groupNode OrchestrationNode,
	memberOrder []string,
	transcript orchestrationTranscript,
	memberSessions map[string]string,
	round int,
	order string,
) apporchestrations.RoundDispatchResult {
	return r.roundDispatcher().Dispatch(ctx, apporchestrations.RoundDispatchCommand{
		GroupNode:        groupNode,
		MemberNodes:      r.plan.nodes,
		ParticipantIDs:   append([]string(nil), memberOrder...),
		Transcript:       transcript,
		MemberSessionIDs: memberSessions,
		Round:            round,
		Order:            order,
		TraceID:          r.traceID,
	})
}
