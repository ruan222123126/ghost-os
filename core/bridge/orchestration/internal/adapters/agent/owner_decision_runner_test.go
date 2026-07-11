package agent

import (
	"context"
	"strings"
	"testing"

	"ghost-os/bridge/orchestration/internal/adapters/toolregistry"
	"ghost-os/bridge/orchestration/internal/domain/group"
	"ghost-os/bridge/orchestration/internal/ports"
	bridgeTasks "ghost-os/bridge/tasks"
)

func TestOwnerDecisionRunnerBuildsScopedDispatchTurn(t *testing.T) {
	executor := &capturingOwnerTurnExecutor{
		command: group.DispatchCommand{Action: group.DispatchActionEndGroup},
	}
	runner := OwnerDecisionRunner{Executor: executor}
	allowlistOnly := true

	got, sessionID, err := runner.Decide(context.Background(), ownerDecisionRequest(&allowlistOnly))

	if err != nil {
		t.Fatalf("decide owner turn: %v", err)
	}
	if got.Action != group.DispatchActionEndGroup || sessionID != "owner-session-next" {
		t.Fatalf("unexpected decision result: %#v session=%q", got, sessionID)
	}
	req := executor.request
	if req.RuntimeOverrides == nil || req.RuntimeOverrides.ToolAllowlistOnly == nil {
		t.Fatalf("expected runtime overrides to be forwarded, got %#v", req.RuntimeOverrides)
	}
	if len(req.Catalog.ToolDefs()) != 1 || req.Catalog.ToolDefs()[0].Name != toolregistry.DispatchToolName {
		t.Fatalf("expected scoped dispatch catalog, got %#v", req.Catalog.ToolDefs())
	}
	if !strings.Contains(req.UserPrompt, toolregistry.DispatchToolName) || req.Round != 2 {
		t.Fatalf("expected owner prompt metadata, got %#v", req)
	}
}

type capturingOwnerTurnExecutor struct {
	command group.DispatchCommand
	request ports.OwnerDecisionTurnRequest
}

func (e *capturingOwnerTurnExecutor) RunOwnerDecisionTurn(
	_ context.Context,
	req ports.OwnerDecisionTurnRequest,
) (group.DispatchCommand, string, error) {
	e.request = req
	return e.command, "owner-session-next", nil
}

func ownerDecisionRequest(allowlistOnly *bool) ports.OwnerDecisionRequest {
	memberNodes := map[string]bridgeTasks.OrchestrationNode{
		"agent-1": {
			ID:   "agent-1",
			Type: bridgeTasks.OrchestrationNodeTypeAgent,
			Agent: &bridgeTasks.OrchestrationAgentNode{
				Title:   "Owner",
				Message: "lead",
				RuntimeOverrides: &bridgeTasks.TaskRuntimeOverrides{
					ToolAllowlistOnly: allowlistOnly,
					ToolAllowlist:     []string{"script_exec"},
				},
			},
		},
		"agent-2": {
			ID:    "agent-2",
			Type:  bridgeTasks.OrchestrationNodeTypeAgent,
			Agent: &bridgeTasks.OrchestrationAgentNode{Title: "B", Message: "member"},
		},
	}
	return ports.OwnerDecisionRequest{
		OwnerNode:   memberNodes["agent-1"],
		GroupNode:   ownerDecisionGroupNode(),
		MemberNodes: memberNodes,
		MemberOrder: []string{"agent-1", "agent-2"},
		PublicTranscript: group.Transcript{
			{Round: 1, Speaker: "B", AgentID: "agent-2", Content: "hello"},
		},
		LastDispatch:   group.DispatchCommand{Action: group.DispatchActionPublicOnce},
		OwnerSessionID: "owner-session",
		Round:          2,
		TraceID:        "trace",
	}
}

func ownerDecisionGroupNode() bridgeTasks.OrchestrationNode {
	return bridgeTasks.OrchestrationNode{
		ID: "group-1",
		Group: &bridgeTasks.OrchestrationGroupNode{
			SpeakingMode: group.SpeakingModeOwner,
			OwnerAgentID: "agent-1",
			MaxRounds:    3,
		},
	}
}
