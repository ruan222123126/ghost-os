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

func TestStandardGroupExecutorTracksRoundsSessionsAndTranscript(t *testing.T) {
	runner := &sessionRecordingRunner{}
	executor := StandardGroupExecutor{
		Dispatcher: RoundDispatcher{Members: runner},
	}

	result := executor.Execute(context.Background(), StandardGroupCommand{
		GroupNode: bridgeTasks.OrchestrationNode{
			ID: "group-1",
			Group: &bridgeTasks.OrchestrationGroupNode{
				SharedContext: "shared",
				SpeakingMode:  group.SpeakingModeSequential,
				MaxRounds:     2,
			},
		},
		MemberNodes: map[string]bridgeTasks.OrchestrationNode{
			"agent-1": testMemberNode("agent-1", "A"),
		},
		MemberOrder: []string{"agent-1"},
		TraceID:     "trace",
	})

	if result.Status != bridgeTasks.RunStatusSuccess || result.CompletedRounds != 2 {
		t.Fatalf("unexpected standard group result: %#v", result)
	}
	if result.MemberSessions["agent-1"] != "session-agent-1" {
		t.Fatalf("expected member session to be recorded, got %#v", result.MemberSessions)
	}
	if got := strings.Join(runner.sessionInputs(), ","); got != ",session-agent-1" {
		t.Fatalf("expected session reuse inputs, got %q", got)
	}
	if transcript := result.Transcript.Format(); !strings.Contains(transcript, "A: round-2") {
		t.Fatalf("expected completed transcript, got %q", transcript)
	}
}

type sessionRecordingRunner struct {
	mu       sync.Mutex
	sessions []string
}

func (r *sessionRecordingRunner) RunMember(
	_ context.Context,
	req ports.MemberRunRequest,
) (ports.MemberResult, error) {
	r.mu.Lock()
	r.sessions = append(r.sessions, req.SessionID)
	r.mu.Unlock()
	sessionID := req.SessionID
	if sessionID == "" {
		sessionID = "session-" + req.MemberNode.ID
	}
	content := "round-1"
	if req.Round == 2 {
		content = "round-2"
	}
	return ports.MemberResult{
		Round:     req.Round,
		AgentID:   req.MemberNode.ID,
		Title:     req.MemberNode.Agent.Title,
		Status:    bridgeTasks.RunStatusSuccess,
		SessionID: sessionID,
		Content:   content,
		Preview:   content,
	}, nil
}

func (r *sessionRecordingRunner) sessionInputs() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.sessions...)
}
