package orchestrations

import (
	"context"

	"ghost-os/bridge/orchestration/internal/domain/group"
	"ghost-os/bridge/orchestration/internal/ports"
)

type RunnerConfig struct {
	Planner Planner
	Members ports.MemberAgentRunner
	Owners  ports.OwnerDecisionRunner
}

func NewRunner(cfg RunnerConfig) Runner {
	dispatcher := RoundDispatcher{Members: cfg.Members}
	return Runner{
		Planner: cfg.Planner,
		Groups:  NewModeGroupExecutor(dispatcher, cfg.Owners),
		Mapper:  ResultMapper{},
	}
}

type ModeGroupExecutor struct {
	Standard StandardGroupExecutor
	Owner    OwnerGroupExecutor
}

func NewModeGroupExecutor(
	dispatcher RoundDispatcher,
	owners ports.OwnerDecisionRunner,
) ModeGroupExecutor {
	return ModeGroupExecutor{
		Standard: StandardGroupExecutor{Dispatcher: dispatcher},
		Owner: OwnerGroupExecutor{
			Decisions: owners,
			Dispatches: DispatchExecutor{
				Dispatcher: dispatcher,
			},
		},
	}
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
