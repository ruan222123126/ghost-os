package orchestrations

import (
	"context"
	"strings"
	"sync"
	"testing"

	"ghost-os/bridge/orchestration/internal/domain/group"
	"ghost-os/bridge/orchestration/internal/ports"
	bridgeTasks "ghost-os/bridge/tasks"
)

func TestRoundDispatcherSequentialSeesPreviousMemberOutput(t *testing.T) {
	runner := newRecordingMemberRunner()
	dispatcher := RoundDispatcher{Members: runner}

	result := dispatcher.Dispatch(context.Background(), roundCommand(group.SpeakingModeSequential))

	if len(result.MemberResults) != 2 {
		t.Fatalf("expected two results, got %#v", result.MemberResults)
	}
	if got := runner.transcriptFor("agent-2"); !strings.Contains(got, "A: alpha") {
		t.Fatalf("expected sequential transcript to include alpha, got %q", got)
	}
}

func TestRoundDispatcherParallelUsesSharedSnapshotAndStableOrder(t *testing.T) {
	runner := newRecordingMemberRunner()
	dispatcher := RoundDispatcher{Members: runner}

	result := dispatcher.Dispatch(context.Background(), roundCommand(group.SpeakingModeParallel))

	if got := runner.transcriptFor("agent-2"); strings.Contains(got, "A: alpha") {
		t.Fatalf("expected parallel transcript snapshot isolation, got %q", got)
	}
	if result.MemberResults[0].AgentID != "agent-1" || result.MemberResults[1].AgentID != "agent-2" {
		t.Fatalf("expected stable member order, got %#v", result.MemberResults)
	}
}

func TestRoundDispatcherParallelSelectsBusinessFailureBeforeCanceled(t *testing.T) {
	dispatcher := RoundDispatcher{Members: cancelRecordingMemberRunner{}}
	result := dispatcher.Dispatch(context.Background(), roundCommand(group.SpeakingModeParallel))

	failure := SelectGroupFailure(result.MemberResults)
	if failure == nil || failure.AgentID != "agent-1" {
		t.Fatalf("expected business failure before context cancellation, got %#v", failure)
	}
}

type recordingMemberRunner struct {
	mu          sync.Mutex
	transcripts map[string]string
}

func newRecordingMemberRunner() *recordingMemberRunner {
	return &recordingMemberRunner{transcripts: map[string]string{}}
}

func (r *recordingMemberRunner) RunMember(
	_ context.Context,
	req ports.MemberRunRequest,
) (ports.MemberResult, error) {
	r.mu.Lock()
	r.transcripts[req.MemberNode.ID] = req.TranscriptText
	r.mu.Unlock()
	content := "alpha"
	if req.MemberNode.ID == "agent-2" {
		content = "bravo"
	}
	return ports.MemberResult{
		Round:     req.Round,
		AgentID:   req.MemberNode.ID,
		Title:     req.MemberNode.Agent.Title,
		Status:    bridgeTasks.RunStatusSuccess,
		SessionID: "session-" + req.MemberNode.ID,
		Content:   content,
		Preview:   content,
	}, nil
}

func (r *recordingMemberRunner) transcriptFor(agentID string) string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.transcripts[agentID]
}

type cancelRecordingMemberRunner struct{}

func (cancelRecordingMemberRunner) RunMember(
	ctx context.Context,
	req ports.MemberRunRequest,
) (ports.MemberResult, error) {
	if req.MemberNode.ID == "agent-2" {
		<-ctx.Done()
		return memberFailure(req, ctx.Err().Error()), nil
	}
	return memberFailure(req, "agent failed"), nil
}

func memberFailure(req ports.MemberRunRequest, message string) ports.MemberResult {
	return ports.MemberResult{
		Round:   req.Round,
		AgentID: req.MemberNode.ID,
		Title:   req.MemberNode.Agent.Title,
		Status:  bridgeTasks.RunStatusError,
		Preview: message,
		Error:   message,
	}
}

func roundCommand(order string) RoundDispatchCommand {
	memberNodes := map[string]bridgeTasks.OrchestrationNode{
		"agent-1": testMemberNode("agent-1", "A"),
		"agent-2": testMemberNode("agent-2", "B"),
	}
	return RoundDispatchCommand{
		GroupNode: bridgeTasks.OrchestrationNode{
			ID: "group-1",
			Group: &bridgeTasks.OrchestrationGroupNode{
				SpeakingMode: order,
				MaxRounds:    1,
			},
		},
		MemberNodes:      memberNodes,
		ParticipantIDs:   []string{"agent-1", "agent-2"},
		Transcript:       group.Transcript{},
		MemberSessionIDs: map[string]string{},
		Round:            1,
		Order:            order,
	}
}

func testMemberNode(id string, title string) bridgeTasks.OrchestrationNode {
	return bridgeTasks.OrchestrationNode{
		ID:   id,
		Type: bridgeTasks.OrchestrationNodeTypeAgent,
		Agent: &bridgeTasks.OrchestrationAgentNode{
			Title:   title,
			Message: "member " + title,
		},
	}
}
