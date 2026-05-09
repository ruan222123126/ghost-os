package orchestrations

import (
	"context"
	"strings"
	"testing"

	"ghost-os/bridge/orchestration/internal/domain/group"
	bridgeTasks "ghost-os/bridge/tasks"
)

func TestDispatchExecutorPublicAppendsSharedTranscriptAndSessions(t *testing.T) {
	executor := DispatchExecutor{Dispatcher: RoundDispatcher{Members: newRecordingMemberRunner()}}

	result := executor.Execute(context.Background(), dispatchCommand(group.DispatchCommand{
		Action:         group.DispatchActionPublicOnce,
		ParticipantIDs: []string{"agent-2"},
		Order:          group.SpeakingModeSequential,
	}))

	if !strings.Contains(result.PublicTranscript.Format(), "B: bravo") {
		t.Fatalf("expected public transcript update, got %#v", result.PublicTranscript)
	}
	if result.MemberSessions["agent-2"] != "session-agent-2" {
		t.Fatalf("expected public session update, got %#v", result.MemberSessions)
	}
	if result.Dispatch.OwnerVisible {
		t.Fatalf("expected owner_visible=false for member-only public dispatch")
	}
}

func TestDispatchExecutorPrivateDoesNotPolluteSharedTranscript(t *testing.T) {
	executor := DispatchExecutor{Dispatcher: RoundDispatcher{Members: newRecordingMemberRunner()}}

	result := executor.Execute(context.Background(), dispatchCommand(group.DispatchCommand{
		Action:         group.DispatchActionPrivateOnce,
		ParticipantIDs: []string{"agent-2", "agent-3"},
		Instruction:    "private chat",
	}))

	if strings.Contains(result.PublicTranscript.Format(), "bravo") {
		t.Fatalf("expected private dispatch to keep shared transcript clean: %#v", result.PublicTranscript)
	}
	if len(result.MemberSessions) != 0 {
		t.Fatalf("expected private dispatch not to update public sessions, got %#v", result.MemberSessions)
	}
	if result.Dispatch.PrivateTranscript != nil || result.Dispatch.OwnerVisible {
		t.Fatalf("expected private transcript hidden from absent owner, got %#v", result.Dispatch)
	}
}

func TestDispatchExecutorPrivateRecordsTranscriptWhenOwnerVisible(t *testing.T) {
	executor := DispatchExecutor{Dispatcher: RoundDispatcher{Members: newRecordingMemberRunner()}}

	result := executor.Execute(context.Background(), dispatchCommand(group.DispatchCommand{
		Action:         group.DispatchActionPrivateOnce,
		ParticipantIDs: []string{"agent-1", "agent-2"},
		Instruction:    "private chat",
	}))

	if !result.Dispatch.OwnerVisible || len(result.Dispatch.PrivateTranscript) == 0 {
		t.Fatalf("expected owner-visible private transcript, got %#v", result.Dispatch)
	}
}

func dispatchCommand(dispatch group.DispatchCommand) DispatchExecuteCommand {
	memberNodes := map[string]bridgeTasks.OrchestrationNode{
		"agent-1": testMemberNode("agent-1", "A"),
		"agent-2": testMemberNode("agent-2", "B"),
		"agent-3": testMemberNode("agent-3", "C"),
	}
	return DispatchExecuteCommand{
		GroupNode: bridgeTasks.OrchestrationNode{
			ID: "group-1",
			Group: &bridgeTasks.OrchestrationGroupNode{
				OwnerAgentID: "agent-1",
				MaxRounds:    1,
			},
		},
		MemberNodes:      memberNodes,
		PublicTranscript: group.Transcript{},
		MemberSessions:   map[string]string{},
		Dispatch:         dispatch,
		Round:            1,
		TraceID:          "trace",
	}
}
