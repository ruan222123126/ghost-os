package group

import (
	"strings"
	"testing"

	taskdefs "ghost-os/bridge/taskdefs"
)

const testMaxRounds = 1

func TestPlanBuilderAcceptsEmptyDraft(t *testing.T) {
	plan, err := PlanBuilder{}.Build(&taskdefs.OrchestrationDefinition{})
	if err != nil {
		t.Fatalf("build empty draft: %v", err)
	}
	if len(plan.Nodes) != 0 || plan.EntryGroupID != "" {
		t.Fatalf("unexpected empty draft plan: %#v", plan)
	}
}

func TestPlanBuilderRejectsLegacyBoundaries(t *testing.T) {
	_, err := PlanBuilder{}.Build(legacyBoundaryDefinition())
	if err == nil || !strings.Contains(err.Error(), "migrate orchestrations") {
		t.Fatalf("expected legacy-boundary migration error, got %v", err)
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
	definition := &taskdefs.OrchestrationDefinition{
		Nodes: []taskdefs.OrchestrationNode{agentNode("agent-1", "Solo")},
	}
	_, err := PlanBuilder{}.Build(definition)
	if err == nil || !strings.Contains(err.Error(), "requires at least 1 group node") {
		t.Fatalf("expected missing group error, got %v", err)
	}
}

func legacyBoundaryDefinition() *taskdefs.OrchestrationDefinition {
	return &taskdefs.OrchestrationDefinition{
		Nodes: []taskdefs.OrchestrationNode{
			{ID: "start-node", Type: NodeTypeStart},
			groupNode("group-1", SpeakingModeSequential),
			agentNode("agent-1", "Member"),
			{ID: "end-node", Type: NodeTypeEnd},
		},
		Edges: []taskdefs.OrchestrationEdge{
			controlEdge("start-node", "group-1"),
			controlEdge("group-1", "end-node"),
			memberEdge("agent-1", "group-1"),
		},
	}
}

func twoGroupDefinition() *taskdefs.OrchestrationDefinition {
	return &taskdefs.OrchestrationDefinition{
		Nodes: []taskdefs.OrchestrationNode{
			groupNode("group-1", SpeakingModeSequential),
			groupNode("group-2", SpeakingModeParallel),
			agentNode("agent-1", "One"),
			agentNode("agent-2", "Two"),
		},
		Edges: []taskdefs.OrchestrationEdge{
			controlEdge("group-1", "group-2"),
			memberEdge("agent-1", "group-1"),
			memberEdge("agent-2", "group-2"),
		},
	}
}

func ownerDefinition(ownerID string) *taskdefs.OrchestrationDefinition {
	definition := twoGroupDefinition()
	definition.Nodes = []taskdefs.OrchestrationNode{
		ownerGroupNode(ownerID),
		agentNode("agent-1", "Owner"),
		agentNode("agent-2", "Member"),
	}
	definition.Edges = []taskdefs.OrchestrationEdge{
		memberEdge("agent-1", "group-1"),
		memberEdge("agent-2", "group-1"),
	}
	return definition
}

func groupNode(id string, mode string) taskdefs.OrchestrationNode {
	return taskdefs.OrchestrationNode{
		ID:   id,
		Type: NodeTypeGroup,
		Group: &taskdefs.OrchestrationGroupNode{
			Title:        id,
			SpeakingMode: mode,
			MaxRounds:    testMaxRounds,
		},
	}
}

func ownerGroupNode(ownerID string) taskdefs.OrchestrationNode {
	node := groupNode("group-1", SpeakingModeOwner)
	node.Group.OwnerAgentID = ownerID
	return node
}

func agentNode(id string, title string) taskdefs.OrchestrationNode {
	return taskdefs.OrchestrationNode{
		ID:    id,
		Type:  NodeTypeAgent,
		Agent: &taskdefs.OrchestrationAgentNode{Title: title, Message: title},
	}
}

func controlEdge(fromID string, toID string) taskdefs.OrchestrationEdge {
	return taskdefs.OrchestrationEdge{FromNodeID: fromID, ToNodeID: toID, Kind: EdgeKindControl}
}

func memberEdge(agentID string, groupID string) taskdefs.OrchestrationEdge {
	return taskdefs.OrchestrationEdge{FromNodeID: agentID, ToNodeID: groupID, Kind: EdgeKindMember}
}
