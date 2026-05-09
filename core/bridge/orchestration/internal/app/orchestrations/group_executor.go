package orchestrations

import (
	"context"

	"ghost-os/bridge/orchestration/internal/domain/group"
)

type ModeGroupExecutor struct {
	Standard StandardGroupExecutor
	Owner    OwnerGroupExecutor
}

func (e ModeGroupExecutor) ExecuteGroup(
	ctx context.Context,
	cmd GroupExecuteCommand,
) GroupResult {
	if cmd.GroupNode.Group.SpeakingMode == group.SpeakingModeOwner {
		return ownerGroupResult(e.Owner.Execute(ctx, toOwnerGroupCommand(cmd)))
	}
	return standardGroupResult(e.Standard.Execute(ctx, toStandardGroupCommand(cmd)))
}

func toStandardGroupCommand(cmd GroupExecuteCommand) StandardGroupCommand {
	return StandardGroupCommand{
		GroupNode:          cmd.GroupNode,
		MemberNodes:        cmd.MemberNodes,
		MemberOrder:        append([]string(nil), cmd.MemberOrder...),
		PreviousGroupID:    cmd.PreviousGroupID,
		PreviousTranscript: cmd.PreviousTranscript,
		MemberSessions:     CloneSessionIDs(cmd.MemberSessions),
		TraceID:            cmd.TraceID,
	}
}

func toOwnerGroupCommand(cmd GroupExecuteCommand) OwnerGroupCommand {
	return OwnerGroupCommand{
		GroupNode:          cmd.GroupNode,
		MemberNodes:        cmd.MemberNodes,
		MemberOrder:        append([]string(nil), cmd.MemberOrder...),
		PreviousGroupID:    cmd.PreviousGroupID,
		PreviousTranscript: cmd.PreviousTranscript,
		MemberSessions:     CloneSessionIDs(cmd.MemberSessions),
		OwnerSessionID:     cmd.OwnerSessionID,
		TraceID:            cmd.TraceID,
	}
}

func standardGroupResult(result StandardGroupResult) GroupResult {
	return GroupResult{
		Status:          result.Status,
		Preview:         result.Preview,
		Error:           result.Error,
		CompletedRounds: result.CompletedRounds,
		Transcript:      result.Transcript,
		MemberResults:   result.MemberResults,
		MemberSessions:  result.MemberSessions,
	}
}

func ownerGroupResult(result OwnerGroupResult) GroupResult {
	return GroupResult{
		Status:          result.Status,
		Preview:         result.Preview,
		Error:           result.Error,
		CompletedRounds: result.CompletedRounds,
		Transcript:      result.Transcript,
		MemberResults:   result.MemberResults,
		MemberSessions:  result.MemberSessions,
		OwnerAgentID:    result.OwnerAgentID,
		OwnerSessionID:  result.OwnerSessionID,
		DispatchResults: result.DispatchResults,
	}
}
