package orchestrations

import (
	"strings"

	"ghost-os/bridge/orchestration/internal/domain/group"
	"ghost-os/bridge/orchestration/internal/ports"
	sharedtext "ghost-os/bridge/orchestration/internal/shared/text"
	bridgeTasks "ghost-os/bridge/tasks"
)

type ownerDecisionFailure struct {
	parent         OwnerGroupCommand
	transcript     group.Transcript
	memberResults  []ports.MemberResult
	memberSessions map[string]string
	ownerSessionID string
	err            error
}

func (s ownerExecutionState) decisionFailure(
	cmd OwnerGroupCommand,
	err error,
) ownerDecisionFailure {
	return ownerDecisionFailure{
		parent:         cmd,
		transcript:     s.transcript,
		memberResults:  s.memberResults,
		memberSessions: s.sessions,
		ownerSessionID: s.ownerSessionID,
		err:            err,
	}
}

func failedOwnerDecision(failure ownerDecisionFailure) OwnerGroupResult {
	return OwnerGroupResult{
		Status:         bridgeTasks.RunStatusError,
		Preview:        sharedtext.TruncateRunes(failure.err.Error(), bridgeTasks.MaxResponsePreviewRunes),
		Error:          failure.err.Error(),
		Transcript:     failure.transcript,
		MemberResults:  failure.memberResults,
		MemberSessions: failure.memberSessions,
		OwnerAgentID:   failure.parent.GroupNode.Group.OwnerAgentID,
		OwnerSessionID: failure.ownerSessionID,
	}
}

type ownerMemberFailure struct {
	parent          OwnerGroupCommand
	failure         ports.MemberResult
	completedRounds int
	transcript      group.Transcript
	memberResults   []ports.MemberResult
	memberSessions  map[string]string
	ownerSessionID  string
	dispatchResults []DispatchResult
}

func (s ownerExecutionState) memberFailure(
	cmd OwnerGroupCommand,
	failure ports.MemberResult,
	dispatchResults []DispatchResult,
) ownerMemberFailure {
	return ownerMemberFailure{
		parent:          cmd,
		failure:         failure,
		completedRounds: s.completedRounds,
		transcript:      s.transcript,
		memberResults:   s.memberResults,
		memberSessions:  s.sessions,
		ownerSessionID:  s.ownerSessionID,
		dispatchResults: dispatchResults,
	}
}

func failedOwnerMember(cmd ownerMemberFailure) OwnerGroupResult {
	return OwnerGroupResult{
		Status:          cmd.failure.Status,
		Preview:         sharedtext.TruncateRunes(cmd.failure.Preview, bridgeTasks.MaxResponsePreviewRunes),
		Error:           strings.TrimSpace(cmd.failure.Error),
		CompletedRounds: cmd.completedRounds,
		Transcript:      cmd.transcript,
		MemberResults:   cmd.memberResults,
		MemberSessions:  cmd.memberSessions,
		OwnerAgentID:    cmd.parent.GroupNode.Group.OwnerAgentID,
		OwnerSessionID:  cmd.ownerSessionID,
		DispatchResults: cmd.dispatchResults,
	}
}

type ownerSuccess struct {
	parent          OwnerGroupCommand
	completedRounds int
	transcript      group.Transcript
	memberResults   []ports.MemberResult
	memberSessions  map[string]string
	ownerSessionID  string
	dispatchResults []DispatchResult
}

func (s ownerExecutionState) success(cmd OwnerGroupCommand) ownerSuccess {
	return ownerSuccess{
		parent:          cmd,
		completedRounds: s.completedRounds,
		transcript:      s.transcript,
		memberResults:   s.memberResults,
		memberSessions:  s.sessions,
		ownerSessionID:  s.ownerSessionID,
		dispatchResults: s.dispatchResults,
	}
}

func successfulOwnerGroup(cmd ownerSuccess) OwnerGroupResult {
	return OwnerGroupResult{
		Status:          bridgeTasks.RunStatusSuccess,
		Preview:         sharedtext.TruncateRunes(cmd.transcript.Format(), bridgeTasks.MaxResponsePreviewRunes),
		CompletedRounds: cmd.completedRounds,
		Transcript:      cmd.transcript,
		MemberResults:   cmd.memberResults,
		MemberSessions:  cmd.memberSessions,
		OwnerAgentID:    cmd.parent.GroupNode.Group.OwnerAgentID,
		OwnerSessionID:  cmd.ownerSessionID,
		DispatchResults: cmd.dispatchResults,
	}
}

func ownerResultCapacity(cmd OwnerGroupCommand) int {
	return len(cmd.MemberOrder) * cmd.GroupNode.Group.MaxRounds
}

func appendMemberResults(
	base []ports.MemberResult,
	items []ports.MemberResult,
) []ports.MemberResult {
	out := append([]ports.MemberResult(nil), base...)
	return append(out, items...)
}

func appendDispatchResults(base []DispatchResult, item DispatchResult) []DispatchResult {
	out := append([]DispatchResult(nil), base...)
	return append(out, item)
}
