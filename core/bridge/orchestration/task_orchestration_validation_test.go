package orchestration

import (
	"strings"
	"testing"
)

func TestValidateTaskDefinitionOrchestrationAcceptsEmptyDraft(t *testing.T) {
	task := ScheduledTask{
		TaskKind:      taskKindOrchestration,
		Name:          "empty-orchestration",
		Orchestration: &OrchestrationDefinition{},
	}
	if err := validateTaskDefinition(&task); err != nil {
		t.Fatalf("validate empty orchestration: %v", err)
	}
}

func TestValidateTaskDefinitionOrchestrationAcceptsLegacyBoundaries(t *testing.T) {
	task := ScheduledTask{
		TaskKind:      taskKindOrchestration,
		Name:          "legacy-orchestration",
		Orchestration: buildLegacyBoundaryDefinition(orchestrationModeSequential, 1),
	}
	if err := validateTaskDefinition(&task); err != nil {
		t.Fatalf("validate legacy orchestration: %v", err)
	}
}

func TestValidateTaskDefinitionOrchestrationRejectsAgentOnlyGraph(t *testing.T) {
	task := ScheduledTask{
		TaskKind: taskKindOrchestration,
		Name:     "broken-orchestration",
		Orchestration: &OrchestrationDefinition{
			Nodes: []OrchestrationNode{
				{ID: "agent-1", Type: orchestrationNodeTypeAgent, Agent: &OrchestrationAgentNode{Title: "Solo", Message: "hello"}},
			},
		},
	}
	err := validateTaskDefinition(&task)
	if err == nil || !strings.Contains(err.Error(), "requires at least 1 group node") {
		t.Fatalf("expected missing-group error, got %v", err)
	}
}

func TestValidateTaskDefinitionOrchestrationOwnerModeRequiresOwnerAgentID(t *testing.T) {
	task := ScheduledTask{
		TaskKind: taskKindOrchestration,
		Name:     "owner-missing",
		Orchestration: &OrchestrationDefinition{
			Nodes: []OrchestrationNode{
				{
					ID:   "group-1",
					Type: orchestrationNodeTypeGroup,
					Group: &OrchestrationGroupNode{
						Title:        "Group 1",
						SpeakingMode: orchestrationModeOwner,
						MaxRounds:    1,
					},
				},
				{ID: "agent-1", Type: orchestrationNodeTypeAgent, Agent: &OrchestrationAgentNode{Title: "Owner", Message: "owner"}},
			},
			Edges: []OrchestrationEdge{
				{FromNodeID: "agent-1", ToNodeID: "group-1", Kind: orchestrationEdgeKindMember},
			},
		},
	}
	err := validateTaskDefinition(&task)
	if err == nil || !strings.Contains(err.Error(), `requires owner_agent_id in owner mode`) {
		t.Fatalf("expected owner_agent_id validation error, got %v", err)
	}
}

func TestValidateTaskDefinitionOrchestrationOwnerModeRequiresOwnerToBeMember(t *testing.T) {
	task := ScheduledTask{
		TaskKind:      taskKindOrchestration,
		Name:          "owner-not-member",
		Orchestration: buildOwnerDefinition("ghost-owner", 1),
	}
	err := validateTaskDefinition(&task)
	if err == nil || !strings.Contains(err.Error(), `owner_agent_id "ghost-owner" must be an existing member`) {
		t.Fatalf("expected owner membership validation error, got %v", err)
	}
}

func TestOrchestrationTaskCreateStripsMemberMaxTurns(t *testing.T) {
	_, service, _ := newTestHandlerWithService(t, nil, nil)

	createdRaw, code, err := service.executeTaskCreateAction(taskCreateParams{
		TaskKind:        taskKindOrchestration,
		Name:            "strip-member-max-turns",
		Orchestration:   buildSingleMemberDefinitionWithOverrides(&TaskRuntimeOverrides{MaxTurns: intPointer(3)}),
		IntervalSeconds: 60,
		Scope:           taskListScopeOrchestration,
	}, "trace-orchestration-strip-max-turns")
	if err != nil || code != 201 {
		t.Fatalf("create orchestration task: code=%d err=%v", code, err)
	}
	created := createdRaw.(taskPayload)

	task, err := service.taskStore().LoadTask(created.ID)
	if err != nil {
		t.Fatalf("load saved orchestration task: %v", err)
	}
	agentNode := task.Orchestration.Nodes[1]
	if agentNode.Agent == nil {
		t.Fatalf("expected agent node payload, got %#v", task.Orchestration.Nodes)
	}
	if agentNode.Agent.RuntimeOverrides != nil && agentNode.Agent.RuntimeOverrides.MaxTurns != nil {
		t.Fatalf("expected orchestration member max_turns to be stripped, got %#v", agentNode.Agent.RuntimeOverrides)
	}
}

func buildSingleMemberDefinitionWithOverrides(overrides *TaskRuntimeOverrides) *OrchestrationDefinition {
	definition := buildSingleMemberDefinition(1)
	definition.Nodes[1].Agent.RuntimeOverrides = overrides
	return definition
}

func buildOwnerDefinition(ownerAgentID string, maxRounds int) *OrchestrationDefinition {
	return &OrchestrationDefinition{
		Nodes: []OrchestrationNode{
			{
				ID:   "group-1",
				Type: orchestrationNodeTypeGroup,
				Group: &OrchestrationGroupNode{
					Title:         "Group 1",
					SharedContext: "",
					SpeakingMode:  orchestrationModeOwner,
					OwnerAgentID:  ownerAgentID,
					MaxRounds:     maxRounds,
				},
			},
			{ID: "agent-1", Type: orchestrationNodeTypeAgent, Agent: &OrchestrationAgentNode{Title: "Owner", Message: "owner-agent"}},
			{ID: "agent-2", Type: orchestrationNodeTypeAgent, Agent: &OrchestrationAgentNode{Title: "Member", Message: "member-agent"}},
		},
		Edges: []OrchestrationEdge{
			{FromNodeID: "agent-1", ToNodeID: "group-1", Kind: orchestrationEdgeKindMember},
			{FromNodeID: "agent-2", ToNodeID: "group-1", Kind: orchestrationEdgeKindMember},
		},
	}
}
