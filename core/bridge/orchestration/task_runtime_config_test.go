package orchestration

import (
	"net/http"
	"strings"
	"testing"

	bridgeconfig "ghost-os/bridge/config"
)

func TestValidateWorkflowTaskRuntimeAllowsAllToolsWhenAllowlistEmpty(t *testing.T) {
	definition := workflowWithToolNode("web_search")

	err := validateWorkflowTaskRuntime(definition, bridgeconfig.TaskConfig{})
	if err != nil {
		t.Fatalf("expected empty allowlist to allow all tools, got %v", err)
	}
}

func TestValidateWorkflowTaskRuntimeRejectsToolOutsideExplicitAllowlist(t *testing.T) {
	definition := workflowWithToolNode("web_search")
	cfg := bridgeconfig.TaskConfig{WorkflowToolAllowlist: []string{"script_exec"}}

	err := validateWorkflowTaskRuntime(definition, cfg)
	if err == nil {
		t.Fatal("expected workflow tool validation to fail")
	}
	if !strings.Contains(err.Error(), `workflow tool "web_search" is not allowed`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWorkflowTaskCreateRejectsInvalidAgentRuntimeOverrides(t *testing.T) {
	_, service, _ := newTestHandlerWithService(t, nil, nil)
	configureRuntimeOverrideProviders(t, service)

	workflow := &WorkflowDefinition{
		Nodes: []WorkflowNode{
			{ID: "start-node", Type: workflowNodeTypeStart},
			{
				ID:   "agent-node",
				Type: workflowNodeTypeAgent,
				Agent: &WorkflowAgentNode{
					Message: "run agent",
					RuntimeOverrides: &TaskRuntimeOverrides{
						ProviderName: "openai-main",
					},
				},
			},
			{ID: "end-node", Type: workflowNodeTypeEnd},
		},
		Edges: []WorkflowEdge{
			{FromNodeID: "start-node", ToNodeID: "agent-node"},
			{FromNodeID: "agent-node", ToNodeID: "end-node"},
		},
	}

	_, code, err := service.executeTaskCreateAction(taskCreateParams{
		TaskKind:        taskKindWorkflow,
		Workflow:        workflow,
		IntervalSeconds: 60,
	}, "trace-workflow-agent-runtime-invalid")
	if err == nil {
		t.Fatal("expected workflow agent runtime validation to fail")
	}
	if code != http.StatusBadRequest {
		t.Fatalf("unexpected status code: got %d want %d", code, http.StatusBadRequest)
	}
	if !strings.Contains(err.Error(), "provider_name requires model") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRelayTaskCreateAppliesConfiguredDefaults(t *testing.T) {
	_, service, _ := newTestHandlerWithService(t, nil, nil)
	_, err := service.executeConfigUpdateAction(configUpdateRequest{
		RelayDefaultStopPolicy:         stringPointer(taskRelayStopPolicyMaxRounds),
		RelayDefaultMaxRounds:          intPointer(7),
		RelayDefaultExecutionTimeoutMs: intPointer(0),
	}, "trace-relay-defaults-config")
	if err != nil {
		t.Fatalf("config update failed: %v", err)
	}

	createdRaw, code, err := service.executeTaskCreateAction(taskCreateParams{
		TaskKind:        taskKindAgentMessage,
		Message:         "run relay task",
		AgentMode:       taskAgentModeRelay,
		IntervalSeconds: 60,
	}, "trace-relay-defaults-task")
	if err != nil {
		t.Fatalf("task create failed: %v", err)
	}
	if code != http.StatusCreated {
		t.Fatalf("unexpected status code: got %d want %d", code, http.StatusCreated)
	}
	created := createdRaw.(taskPayload)
	if created.Relay == nil {
		t.Fatal("expected relay defaults")
	}
	if created.Relay.StopPolicy != taskRelayStopPolicyMaxRounds || created.Relay.MaxRounds != 7 {
		t.Fatalf("unexpected relay defaults: %+v", created.Relay)
	}
	if created.Relay.ExecutionTimeoutMS == nil || *created.Relay.ExecutionTimeoutMS != 0 {
		t.Fatalf("unexpected relay execution timeout: %+v", created.Relay.ExecutionTimeoutMS)
	}
}

func TestRelayTaskCreateAppliesAIDecidesMaxRoundDefault(t *testing.T) {
	_, service, _ := newTestHandlerWithService(t, nil, nil)
	_, err := service.executeConfigUpdateAction(configUpdateRequest{
		RelayDefaultStopPolicy:         stringPointer(taskRelayStopPolicyAIDecides),
		RelayDefaultMaxRounds:          intPointer(6),
		RelayDefaultExecutionTimeoutMs: intPointer(0),
	}, "trace-relay-ai-defaults-config")
	if err != nil {
		t.Fatalf("config update failed: %v", err)
	}

	createdRaw, code, err := service.executeTaskCreateAction(taskCreateParams{
		TaskKind:        taskKindAgentMessage,
		Message:         "run relay task",
		AgentMode:       taskAgentModeRelay,
		IntervalSeconds: 60,
	}, "trace-relay-ai-defaults-task")
	if err != nil {
		t.Fatalf("task create failed: %v", err)
	}
	if code != http.StatusCreated {
		t.Fatalf("unexpected status code: got %d want %d", code, http.StatusCreated)
	}
	created := createdRaw.(taskPayload)
	if created.Relay == nil {
		t.Fatal("expected relay defaults")
	}
	if created.Relay.StopPolicy != taskRelayStopPolicyAIDecides || created.Relay.MaxRounds != 6 {
		t.Fatalf("unexpected relay defaults: %+v", created.Relay)
	}
}
