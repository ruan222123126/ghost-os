package orchestrations

import (
	"context"
	"strings"
	"testing"

	"ghost-os/bridge/orchestration/internal/domain/group"
	"ghost-os/bridge/orchestration/internal/ports"
	bridgeTasks "ghost-os/bridge/tasks"
)

func TestOwnerGroupExecutorDispatchesAndStopsOnEndGroup(t *testing.T) {
	decisions := &scriptedOwnerDecisions{
		commands: []group.DispatchCommand{
			{Action: group.DispatchActionPublicOnce, ParticipantIDs: []string{"agent-2"}, Order: group.SpeakingModeSequential},
			{Action: group.DispatchActionEndGroup},
		},
	}
	executor := OwnerGroupExecutor{
		Decisions:  decisions,
		Dispatches: DispatchExecutor{Dispatcher: RoundDispatcher{Members: newRecordingMemberRunner()}},
	}

	result := executor.Execute(context.Background(), ownerGroupCommand(3))

	if result.Status != bridgeTasks.RunStatusSuccess || result.CompletedRounds != 1 {
		t.Fatalf("unexpected owner group result: %#v", result)
	}
	if len(result.DispatchResults) != 2 || result.DispatchResults[1].Action != group.DispatchActionEndGroup {
		t.Fatalf("expected public dispatch plus end_group, got %#v", result.DispatchResults)
	}
	if !strings.Contains(result.Transcript.Format(), "B: bravo") {
		t.Fatalf("expected public dispatch in transcript, got %#v", result.Transcript)
	}
	if decisions.requests[1].LastDispatch.Action != group.DispatchActionPublicOnce {
		t.Fatalf("expected last dispatch to feed next owner turn, got %#v", decisions.requests)
	}
	if result.OwnerSessionID != "owner-session-2" {
		t.Fatalf("expected owner session update, got %#v", result)
	}
}

func TestOwnerGroupExecutorImmediateEndGroupSkipsMembers(t *testing.T) {
	decisions := &scriptedOwnerDecisions{
		commands: []group.DispatchCommand{{Action: group.DispatchActionEndGroup}},
	}
	members := newRecordingMemberRunner()
	executor := OwnerGroupExecutor{
		Decisions:  decisions,
		Dispatches: DispatchExecutor{Dispatcher: RoundDispatcher{Members: members}},
	}

	result := executor.Execute(context.Background(), ownerGroupCommand(2))

	if result.CompletedRounds != 0 || len(result.MemberResults) != 0 {
		t.Fatalf("expected immediate end_group to skip members, got %#v", result)
	}
	if got := members.transcriptFor("agent-2"); got != "" {
		t.Fatalf("expected no member dispatch, got transcript %q", got)
	}
}

type scriptedOwnerDecisions struct {
	commands []group.DispatchCommand
	requests []ports.OwnerDecisionRequest
}

func (r *scriptedOwnerDecisions) Decide(
	_ context.Context,
	req ports.OwnerDecisionRequest,
) (group.DispatchCommand, string, error) {
	r.requests = append(r.requests, req)
	index := len(r.requests) - 1
	return r.commands[index], "owner-session-" + string(rune('1'+index)), nil
}

func ownerGroupCommand(maxRounds int) OwnerGroupCommand {
	memberNodes := map[string]bridgeTasks.OrchestrationNode{
		"agent-1": testMemberNode("agent-1", "A"),
		"agent-2": testMemberNode("agent-2", "B"),
	}
	return OwnerGroupCommand{
		GroupNode: bridgeTasks.OrchestrationNode{
			ID: "group-1",
			Group: &bridgeTasks.OrchestrationGroupNode{
				SpeakingMode:  group.SpeakingModeOwner,
				OwnerAgentID:  "agent-1",
				MaxRounds:     maxRounds,
				SharedContext: "shared",
			},
		},
		MemberNodes:    memberNodes,
		MemberOrder:    []string{"agent-1", "agent-2"},
		TraceID:        "trace",
		MemberSessions: map[string]string{},
	}
}
