package orchestration

import (
	"context"
	"strings"

	groupdomain "ghost-os/bridge/orchestration/internal/domain/group"
)

func (r orchestrationTaskRunner) executeOwnerGroupNode(
	ctx context.Context,
	node OrchestrationNode,
	state *orchestrationRunState,
) orchestrationGroupResult {
	groupID := node.ID
	publicTranscript := groupdomain.Initial(node, state.lastGroupID, state.lastGroupTranscript)
	memberOrder := r.plan.groupMember[groupID]
	memberSessions := cloneGroupSessionIDs(state.memberSessionIDs[groupID])
	ownerSessionID := strings.TrimSpace(state.ownerSessionIDs[groupID])
	memberResults := make([]orchestrationMemberResult, 0, len(memberOrder)*node.Group.MaxRounds)
	dispatchResults := make([]orchestrationDispatchResult, 0, node.Group.MaxRounds)
	completedRounds := 0
	lastDispatch := orchestrationDispatchResult{}

	for round := 1; round <= node.Group.MaxRounds; round++ {
		dispatch, nextSessionID, dispatchErr := r.runOwnerDispatch(ctx, node, memberOrder, publicTranscript, lastDispatch, ownerSessionID, round)
		if dispatchErr != nil {
			return orchestrationGroupResult{
				status:          taskRunStatusError,
				preview:         truncateRunes(dispatchErr.Error(), maxTaskResponsePreviewRunes),
				errText:         dispatchErr.Error(),
				transcript:      publicTranscript,
				memberResults:   memberResults,
				memberSessions:  memberSessions,
				ownerAgentID:    node.Group.OwnerAgentID,
				ownerSessionID:  ownerSessionID,
				dispatchResults: dispatchResults,
			}
		}
		ownerSessionID = nextSessionID
		dispatchResult := newDispatchResult(dispatch, round, node.Group.OwnerAgentID)
		if dispatch.Action == groupdomain.DispatchActionEndGroup {
			dispatchResults = append(dispatchResults, dispatchResult)
			state.memberSessionIDs[groupID] = cloneGroupSessionIDs(memberSessions)
			state.ownerSessionIDs[groupID] = ownerSessionID
			return orchestrationGroupResult{
				status:          taskRunStatusSuccess,
				preview:         truncateRunes(publicTranscript.Format(), maxTaskResponsePreviewRunes),
				completedRounds: completedRounds,
				transcript:      publicTranscript,
				memberResults:   memberResults,
				memberSessions:  memberSessions,
				ownerAgentID:    node.Group.OwnerAgentID,
				ownerSessionID:  ownerSessionID,
				dispatchResults: dispatchResults,
			}
		}

		dispatchResult, transcriptUpdate, memberSessionUpdate := r.executeOwnerDispatch(ctx, node, publicTranscript, memberSessions, dispatchResult, round)
		memberResults = append(memberResults, dispatchResult.MemberResults...)
		memberSessions = memberSessionUpdate
		publicTranscript = transcriptUpdate
		lastDispatch = dispatchResult

		failure := selectGroupFailure(dispatchResult.MemberResults)
		if failure != nil {
			state.ownerSessionIDs[groupID] = ownerSessionID
			return orchestrationGroupResult{
				status:          failure.Status,
				preview:         truncateRunes(failure.Preview, maxTaskResponsePreviewRunes),
				errText:         strings.TrimSpace(failure.Error),
				completedRounds: completedRounds,
				transcript:      publicTranscript,
				memberResults:   memberResults,
				memberSessions:  memberSessions,
				ownerAgentID:    node.Group.OwnerAgentID,
				ownerSessionID:  ownerSessionID,
				dispatchResults: dispatchResults,
			}
		}
		dispatchResults = append(dispatchResults, dispatchResult)
		completedRounds = round
	}

	state.memberSessionIDs[groupID] = cloneGroupSessionIDs(memberSessions)
	state.ownerSessionIDs[groupID] = ownerSessionID
	return orchestrationGroupResult{
		status:          taskRunStatusSuccess,
		preview:         truncateRunes(publicTranscript.Format(), maxTaskResponsePreviewRunes),
		completedRounds: completedRounds,
		transcript:      publicTranscript,
		memberResults:   memberResults,
		memberSessions:  memberSessions,
		ownerAgentID:    node.Group.OwnerAgentID,
		ownerSessionID:  ownerSessionID,
		dispatchResults: dispatchResults,
	}
}

func (r orchestrationTaskRunner) executeOwnerDispatch(
	ctx context.Context,
	groupNode OrchestrationNode,
	publicTranscript orchestrationTranscript,
	memberSessions map[string]string,
	result orchestrationDispatchResult,
	round int,
) (orchestrationDispatchResult, orchestrationTranscript, map[string]string) {
	participantIDs := append([]string(nil), result.ParticipantIDs...)
	result.ParticipantIDs = append([]string(nil), participantIDs...)
	result.OwnerVisible = containsString(participantIDs, strings.TrimSpace(groupNode.Group.OwnerAgentID))
	updatedSessions := cloneGroupSessionIDs(memberSessions)

	switch result.Action {
	case groupdomain.DispatchActionPublicOnce:
		order := strings.TrimSpace(result.Order)
		results := r.executeOwnerPublicDispatch(ctx, groupNode, participantIDs, publicTranscript, updatedSessions, round, order, strings.TrimSpace(result.Instruction))
		result.Order = order
		result.MemberResults = results
		nextTranscript := publicTranscript.Clone()
		for _, item := range results {
			if item.Status == taskRunStatusSuccess {
				nextTranscript = nextTranscript.AppendMember(round, item.Title, item.AgentID, item.Content)
			}
			if item.SessionID != "" {
				updatedSessions[item.AgentID] = item.SessionID
			}
		}
		return result, nextTranscript, updatedSessions
	case groupdomain.DispatchActionPrivateOnce:
		results, privateTranscript := r.executeOwnerPrivateDispatch(ctx, groupNode, participantIDs, publicTranscript, round, strings.TrimSpace(result.Instruction))
		result.MemberResults = results
		if result.OwnerVisible {
			result.PrivateTranscript = privateTranscript
		}
		return result, publicTranscript.Clone(), updatedSessions
	default:
		return result, publicTranscript.Clone(), updatedSessions
	}
}

func newDispatchResult(
	dispatch orchestrationDispatchRequest,
	round int,
	ownerAgentID string,
) orchestrationDispatchResult {
	participants := append([]string(nil), dispatch.ParticipantIDs...)
	return orchestrationDispatchResult{
		Round:          round,
		Action:         strings.TrimSpace(dispatch.Action),
		Order:          strings.TrimSpace(dispatch.Order),
		Instruction:    strings.TrimSpace(dispatch.Instruction),
		ParticipantIDs: append([]string(nil), participants...),
		OwnerVisible:   containsString(participants, strings.TrimSpace(ownerAgentID)),
	}
}
