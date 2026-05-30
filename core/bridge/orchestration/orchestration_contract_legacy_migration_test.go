package orchestration

import (
	"strings"
	"testing"
)

func TestOrchestrationLegacyBoundaryNodesRequireMigration(t *testing.T) {
	task := ScheduledTask{
		TaskKind:      taskKindOrchestration,
		Name:          "legacy-orchestration",
		Orchestration: buildLegacyBoundaryDefinition(orchestrationModeSequential, 1),
	}
	err := validateTaskDefinition(&task)
	if err == nil || !strings.Contains(err.Error(), "migrate orchestrations") {
		t.Fatalf("expected migrate-orchestrations validation error, got %v", err)
	}
}

func buildLegacyBoundaryDefinition(mode string, maxRounds int) *OrchestrationDefinition {
	return &OrchestrationDefinition{
		Nodes: []OrchestrationNode{
			{ID: "start-node", Type: orchestrationNodeTypeStart},
			{ID: "group-1", Type: orchestrationNodeTypeGroup, Group: &OrchestrationGroupNode{Title: "Group 1", SharedContext: "", SpeakingMode: mode, MaxRounds: maxRounds}},
			{ID: "agent-1", Type: orchestrationNodeTypeAgent, Agent: &OrchestrationAgentNode{Title: "Looper", Message: "loop-agent"}},
			{ID: "end-node", Type: orchestrationNodeTypeEnd},
		},
		Edges: []OrchestrationEdge{
			{FromNodeID: "start-node", ToNodeID: "group-1", Kind: orchestrationEdgeKindControl},
			{FromNodeID: "group-1", ToNodeID: "end-node", Kind: orchestrationEdgeKindControl},
			{FromNodeID: "agent-1", ToNodeID: "group-1", Kind: orchestrationEdgeKindMember},
		},
	}
}

func TestPrepareWorkflowToolArgumentsRejectsLegacyFindIconUploadKeys(t *testing.T) {
	_, err := prepareWorkflowToolArguments(screenControlToolID, map[string]any{
		"mode":   "atomic",
		"action": "find_icon",
		"params": map[string]any{
			"template_data_url": "data:image/png;base64,R2hvc3Q=",
		},
	})
	if err == nil || !strings.Contains(err.Error(), "legacy find_icon data_url keys are not supported") {
		t.Fatalf("unexpected error for template_data_url: %v", err)
	}

	_, err = prepareWorkflowToolArguments(screenControlToolID, map[string]any{
		"mode":   "atomic",
		"action": "find_icon",
		"params": map[string]any{
			"data_url": "data:image/png;base64,R2hvc3Q=",
		},
	})
	if err == nil || !strings.Contains(err.Error(), "legacy find_icon data_url keys are not supported") {
		t.Fatalf("unexpected error for data_url: %v", err)
	}
}
