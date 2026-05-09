package toolregistry

import (
	"context"
	"encoding/json"
	"testing"

	"ghost-os/bridge/orchestration/internal/domain/group"
	bridgeTasks "ghost-os/bridge/tasks"
)

func TestDispatchToolNormalizesValidCommand(t *testing.T) {
	tool := DispatchTool{
		GroupNode:   dispatchGroupNode(),
		MemberOrder: []string{"agent-1", "agent-2"},
	}

	output, err := tool.Execute(context.Background(), json.RawMessage(`{"action":"public_once"}`), "trace")

	if err != nil {
		t.Fatalf("execute dispatch tool: %v", err)
	}
	var got group.DispatchCommand
	if err := json.Unmarshal([]byte(output), &got); err != nil {
		t.Fatalf("decode output: %v", err)
	}
	if got.Action != group.DispatchActionPublicOnce || len(got.ParticipantIDs) != 2 {
		t.Fatalf("expected normalized public dispatch, got %#v", got)
	}
}

func TestDispatchToolInterpretResultReturnsHandoff(t *testing.T) {
	tool := DispatchTool{}

	meta := tool.InterpretResult(`{"action":"end_group"}`)

	if meta.Iteration == nil || meta.Iteration.Remaining != "owner dispatch submitted" {
		t.Fatalf("expected iteration handoff, got %#v", meta)
	}
}

func dispatchGroupNode() bridgeTasks.OrchestrationNode {
	return bridgeTasks.OrchestrationNode{
		ID:    "group-1",
		Type:  bridgeTasks.OrchestrationNodeTypeGroup,
		Group: &bridgeTasks.OrchestrationGroupNode{OwnerAgentID: "agent-1"},
	}
}
