package group

import (
	"strings"
	"testing"

	bridgeTasks "ghost-os/bridge/tasks"
)

const testMaxRounds = 1

func TestPlanBuilderAcceptsEmptyDraft(t *testing.T) {
	plan, err := PlanBuilder{}.Build(&bridgeTasks.OrchestrationDefinition{})
	if err != nil {
		t.Fatalf("build empty draft: %v", err)
	}
	if len(plan.Nodes) != 0 || plan.EntryGroupID != "" {
		t.Fatalf("unexpected empty draft plan: %#v", plan)
	}
}

func TestPlanBuilderAcceptsLegacyBoundariesWithoutResultNodes(t *testing.T) {
	plan, err := PlanBuilder{}.Build(legacyBoundaryDefinition())
	if err != nil {
		t.Fatalf("build legacy boundary plan: %v", err)
	}
	if plan.EntryGroupID != "group-1" {
		t.Fatalf("unexpected entry group: %#v", plan)
	}
	if len(plan.ControlNext) != 0 {
		t.Fatalf("legacy start/end should not become group control next: %#v", plan.ControlNext)
	}
}

func TestPlanBuilderBuildsGroupControlFlow(t *testing.T) {
	plan, err := PlanBuilder{}.Build(twoGroupDefinition())
	if err != nil {
		t.Fatalf("build group control flow: %v", err)
	}
	if plan.EntryGroupID != "group-1" || plan.ControlNext["group-1"] != "group-2" {
		t.Fatalf("unexpected control plan: %#v", plan)
	}
}

func TestPlanBuilderOwnerModeRequiresOwnerToBeMember(t *testing.T) {
	_, err := PlanBuilder{}.Build(ownerDefinition("ghost-owner"))
	if err == nil || !strings.Contains(err.Error(), `owner_agent_id "ghost-owner" must be an existing member`) {
		t.Fatalf("expected owner member error, got %v", err)
	}
}

func TestPlanBuilderRejectsAgentOnlyGraph(t *testing.T) {
	definition := &bridgeTasks.OrchestrationDefinition{
		Nodes: []bridgeTasks.OrchestrationNode{agentNode("agent-1", "Solo")},
	}
	_, err := PlanBuilder{}.Build(definition)
	if err == nil || !strings.Contains(err.Error(), "requires at least 1 group node") {
		t.Fatalf("expected missing group error, got %v", err)
	}
}

func legacyBoundaryDefinition() *bridgeTasks.OrchestrationDefinition {
	return &bridgeTasks.OrchestrationDefinition{
		Nodes: []bridgeTasks.OrchestrationNode{
			{ID: "start-node", Type: NodeTypeStart},
			groupNode("group-1", SpeakingModeSequential),
			agentNode("agent-1", "Member"),
			{ID: "end-node", Type: NodeTypeEnd},
		},
		Edges: []bridgeTasks.OrchestrationEdge{
			controlEdge("start-node", "group-1"),
			controlEdge("group-1", "end-node"),
			memberEdge("agent-1", "group-1"),
		},
	}
}

func twoGroupDefinition() *bridgeTasks.OrchestrationDefinition {
	return &bridgeTasks.OrchestrationDefinition{
		Nodes: []bridgeTasks.OrchestrationNode{
			groupNode("group-1", SpeakingModeSequential),
			groupNode("group-2", SpeakingModeParallel),
			agentNode("agent-1", "One"),
			agentNode("agent-2", "Two"),
		},
		Edges: []bridgeTasks.OrchestrationEdge{
			controlEdge("group-1", "group-2"),
			memberEdge("agent-1", "group-1"),
			memberEdge("agent-2", "group-2"),
		},
	}
}

func ownerDefinition(ownerID string) *bridgeTasks.OrchestrationDefinition {
	definition := twoGroupDefinition()
	definition.Nodes = []bridgeTasks.OrchestrationNode{
		ownerGroupNode(ownerID),
		agentNode("agent-1", "Owner"),
		agentNode("agent-2", "Member"),
	}
	definition.Edges = []bridgeTasks.OrchestrationEdge{
		memberEdge("agent-1", "group-1"),
		memberEdge("agent-2", "group-1"),
	}
	return definition
}

func groupNode(id string, mode string) bridgeTasks.OrchestrationNode {
	return bridgeTasks.OrchestrationNode{
		ID:   id,
		Type: NodeTypeGroup,
		Group: &bridgeTasks.OrchestrationGroupNode{
			Title:        id,
			SpeakingMode: mode,
			MaxRounds:    testMaxRounds,
		},
	}
}

func ownerGroupNode(ownerID string) bridgeTasks.OrchestrationNode {
	node := groupNode("group-1", SpeakingModeOwner)
	node.Group.OwnerAgentID = ownerID
	return node
}

func agentNode(id string, title string) bridgeTasks.OrchestrationNode {
	return bridgeTasks.OrchestrationNode{
		ID:    id,
		Type:  NodeTypeAgent,
		Agent: &bridgeTasks.OrchestrationAgentNode{Title: title, Message: title},
	}
}

func controlEdge(fromID string, toID string) bridgeTasks.OrchestrationEdge {
	return bridgeTasks.OrchestrationEdge{FromNodeID: fromID, ToNodeID: toID, Kind: EdgeKindControl}
}

func memberEdge(agentID string, groupID string) bridgeTasks.OrchestrationEdge {
	return bridgeTasks.OrchestrationEdge{FromNodeID: agentID, ToNodeID: groupID, Kind: EdgeKindMember}
}
