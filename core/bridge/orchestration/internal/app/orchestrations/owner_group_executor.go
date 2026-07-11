package orchestrations

import (
	"context"
	"errors"
	"strings"

	"ghost-os/bridge/orchestration/internal/domain/group"
	"ghost-os/bridge/orchestration/internal/ports"
	bridgeTasks "ghost-os/bridge/tasks"
)

type OwnerGroupExecutor struct {
	Decisions  ports.OwnerDecisionRunner
	Dispatches DispatchExecutor
}

type OwnerGroupCommand struct {
	GroupNode          bridgeTasks.OrchestrationNode
	MemberNodes        map[string]bridgeTasks.OrchestrationNode
	MemberOrder        []string
	PreviousGroupID    string
	PreviousTranscript group.Transcript
	MemberSessions     map[string]string
	OwnerSessionID     string
	TraceID            string
}

type OwnerGroupResult struct {
	Status          string
	Preview         string
	Error           string
	CompletedRounds int
	Transcript      group.Transcript
	MemberResults   []ports.MemberResult
	MemberSessions  map[string]string
	OwnerAgentID    string
	OwnerSessionID  string
	DispatchResults []DispatchResult
}

func (e OwnerGroupExecutor) Execute(
	ctx context.Context,
	cmd OwnerGroupCommand,
) OwnerGroupResult {
	state := newOwnerExecutionState(cmd)
	for round := 1; round <= cmd.GroupNode.Group.MaxRounds; round++ {
		next, result, done := e.advanceRound(ctx, ownerAdvanceCommand{
			parent: cmd,
			state:  state,
			round:  round,
		})
		if done {
			return result
		}
		state = next
	}
	return successfulOwnerGroup(state.success(cmd))
}

type ownerExecutionState struct {
	transcript      group.Transcript
	sessions        map[string]string
	privateInboxes  map[string][]group.PrivateMessage
	memberResults   []ports.MemberResult
	dispatchResults []DispatchResult
	ownerSessionID  string
	completedRounds int
	lastDispatch    group.DispatchCommand
}

func newOwnerExecutionState(cmd OwnerGroupCommand) ownerExecutionState {
	return ownerExecutionState{
		transcript:      group.Initial(cmd.GroupNode, cmd.PreviousGroupID, cmd.PreviousTranscript),
		sessions:        CloneSessionIDs(cmd.MemberSessions),
		privateInboxes:  map[string][]group.PrivateMessage{},
		memberResults:   make([]ports.MemberResult, 0, ownerResultCapacity(cmd)),
		dispatchResults: make([]DispatchResult, 0, cmd.GroupNode.Group.MaxRounds),
		ownerSessionID:  strings.TrimSpace(cmd.OwnerSessionID),
	}
}

type ownerAdvanceCommand struct {
	parent OwnerGroupCommand
	state  ownerExecutionState
	round  int
}

func (e OwnerGroupExecutor) advanceRound(
	ctx context.Context,
	cmd ownerAdvanceCommand,
) (ownerExecutionState, OwnerGroupResult, bool) {
	roundResult := e.executeRound(ctx, cmd.state.roundCommand(cmd.parent, cmd.round))
	if roundResult.decisionErr != nil {
		next := cmd.state.withOwnerSession(roundResult.ownerSessionID)
		result := failedOwnerDecision(next.decisionFailure(cmd.parent, roundResult.decisionErr))
		return next, result, true
	}
	if roundResult.ended {
		next := cmd.state.withOwnerSession(roundResult.ownerSessionID)
		next.dispatchResults = appendDispatchResults(cmd.state.dispatchResults, roundResult.dispatch)
		return next, successfulOwnerGroup(next.success(cmd.parent)), true
	}
	next := cmd.state.withDispatchResult(roundResult)
	if failure := SelectGroupFailure(roundResult.dispatch.MemberResults); failure != nil {
		result := failedOwnerMember(next.memberFailure(cmd.parent, *failure, cmd.state.dispatchResults))
		return next, result, true
	}
	next.dispatchResults = appendDispatchResults(cmd.state.dispatchResults, roundResult.dispatch)
	next.completedRounds = cmd.round
	return next, OwnerGroupResult{}, false
}

func (s ownerExecutionState) roundCommand(cmd OwnerGroupCommand, round int) ownerRoundCommand {
	return ownerRoundCommand{
		parent:         cmd,
		ownerSessionID: s.ownerSessionID,
		lastDispatch:   s.lastDispatch,
		transcript:     s.transcript,
		sessions:       s.sessions,
		privateInboxes: s.privateInboxes,
		round:          round,
	}
}

func (s ownerExecutionState) withDispatchResult(result ownerRoundResult) ownerExecutionState {
	return ownerExecutionState{
		transcript:      result.transcript,
		sessions:        result.memberSessions,
		privateInboxes:  result.privateInboxes,
		memberResults:   appendMemberResults(s.memberResults, result.dispatch.MemberResults),
		dispatchResults: s.dispatchResults,
		ownerSessionID:  result.ownerSessionID,
		completedRounds: s.completedRounds,
		lastDispatch:    DispatchCommandFromResult(result.dispatch),
	}
}

func (s ownerExecutionState) withOwnerSession(ownerSessionID string) ownerExecutionState {
	return ownerExecutionState{
		transcript:      s.transcript,
		sessions:        s.sessions,
		privateInboxes:  s.privateInboxes,
		memberResults:   s.memberResults,
		dispatchResults: s.dispatchResults,
		ownerSessionID:  ownerSessionID,
		completedRounds: s.completedRounds,
		lastDispatch:    s.lastDispatch,
	}
}

type ownerRoundResult struct {
	dispatch       DispatchResult
	transcript     group.Transcript
	memberSessions map[string]string
	privateInboxes map[string][]group.PrivateMessage
	ownerSessionID string
	decisionErr    error
	ended          bool
}

type ownerRoundCommand struct {
	parent         OwnerGroupCommand
	ownerSessionID string
	lastDispatch   group.DispatchCommand
	transcript     group.Transcript
	sessions       map[string]string
	privateInboxes map[string][]group.PrivateMessage
	round          int
}

func (e OwnerGroupExecutor) executeRound(
	ctx context.Context,
	cmd ownerRoundCommand,
) ownerRoundResult {
	decision, nextSessionID, err := e.decide(ctx, ownerDecisionCommand{
		parent:         cmd.parent,
		ownerSessionID: cmd.ownerSessionID,
		lastDispatch:   cmd.lastDispatch,
		transcript:     cmd.transcript,
		round:          cmd.round,
	})
	if err != nil {
		return ownerRoundResult{decisionErr: err, ownerSessionID: nextSessionID}
	}
	dispatch := NewDispatchResult(decision, cmd.round, cmd.parent.GroupNode.Group.OwnerAgentID)
	if dispatch.Action == group.DispatchActionEndGroup {
		return ownerRoundResult{dispatch: dispatch, ownerSessionID: nextSessionID, ended: true}
	}
	executed := e.Dispatches.Execute(ctx, DispatchExecuteCommand{
		GroupNode:        cmd.parent.GroupNode,
		MemberNodes:      cmd.parent.MemberNodes,
		PublicTranscript: cmd.transcript,
		MemberSessions:   cmd.sessions,
		PrivateInboxes:   cmd.privateInboxes,
		Dispatch:         decision,
		Round:            cmd.round,
		TraceID:          cmd.parent.TraceID,
	})
	return ownerRoundResult{
		dispatch:       executed.Dispatch,
		transcript:     executed.PublicTranscript,
		memberSessions: executed.MemberSessions,
		privateInboxes: executed.PrivateInboxes,
		ownerSessionID: nextSessionID,
	}
}

type ownerDecisionCommand struct {
	parent         OwnerGroupCommand
	ownerSessionID string
	lastDispatch   group.DispatchCommand
	transcript     group.Transcript
	round          int
}

func (e OwnerGroupExecutor) decide(
	ctx context.Context,
	cmd ownerDecisionCommand,
) (group.DispatchCommand, string, error) {
	if e.Decisions == nil {
		return group.DispatchCommand{}, cmd.ownerSessionID, errors.New("owner decision runner is not configured")
	}
	return e.Decisions.Decide(ctx, ports.OwnerDecisionRequest{
		OwnerNode:        cmd.parent.MemberNodes[cmd.parent.GroupNode.Group.OwnerAgentID],
		GroupNode:        cmd.parent.GroupNode,
		MemberNodes:      cmd.parent.MemberNodes,
		MemberOrder:      append([]string(nil), cmd.parent.MemberOrder...),
		PublicTranscript: cmd.transcript,
		LastDispatch:     cmd.lastDispatch,
		OwnerSessionID:   cmd.ownerSessionID,
		Round:            cmd.round,
		TraceID:          cmd.parent.TraceID,
	})
}
