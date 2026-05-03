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
