package orchestration

import (
	"net/http"
	"strings"
	"testing"
)

func TestValidateTaskDefinitionWorkflowAcceptsLinearNodeChain(t *testing.T) {
	task := ScheduledTask{
		TaskKind: taskKindWorkflow,
		Workflow: &WorkflowDefinition{
			Nodes: []WorkflowNode{
				{ID: "start-node", Type: workflowNodeTypeStart},
				{ID: "tool-node", Type: workflowNodeTypeTool, Tool: &WorkflowToolNode{ToolName: "script_exec"}},
				{ID: "llm-node", Type: workflowNodeTypeLLM, LLM: &WorkflowLLMNode{Prompt: "summarize"}},
				{ID: "agent-node", Type: workflowNodeTypeAgent, Agent: &WorkflowAgentNode{Message: "reply"}},
				{ID: "end-node", Type: workflowNodeTypeEnd},
			},
			Edges: []WorkflowEdge{
				{FromNodeID: "start-node", ToNodeID: "tool-node"},
				{FromNodeID: "tool-node", ToNodeID: "llm-node"},
				{FromNodeID: "llm-node", ToNodeID: "agent-node"},
				{FromNodeID: "agent-node", ToNodeID: "end-node"},
			},
		},
	}
	if err := validateTaskDefinition(&task); err != nil {
		t.Fatalf("validate workflow chain: %v", err)
	}
}

func TestValidateTaskDefinitionWorkflowRejectsNodePayloadMismatch(t *testing.T) {
	task := ScheduledTask{
		TaskKind: taskKindWorkflow,
		Workflow: &WorkflowDefinition{
			Nodes: []WorkflowNode{
				{ID: "start-node", Type: workflowNodeTypeStart},
				{ID: "tool-node", Type: workflowNodeTypeTool, LLM: &WorkflowLLMNode{Prompt: "bad"}},
				{ID: "end-node", Type: workflowNodeTypeEnd},
			},
			Edges: []WorkflowEdge{
				{FromNodeID: "start-node", ToNodeID: "tool-node"},
				{FromNodeID: "tool-node", ToNodeID: "end-node"},
			},
		},
	}
	err := validateTaskDefinition(&task)
	if err == nil || !strings.Contains(err.Error(), "payload does not match type") {
		t.Fatalf("expected payload mismatch, got %v", err)
	}
}

func TestValidateTaskDefinitionWorkflowRejectsBranchingChain(t *testing.T) {
	task := ScheduledTask{
		TaskKind: taskKindWorkflow,
		Workflow: &WorkflowDefinition{
			Nodes: []WorkflowNode{
				{ID: "start-node", Type: workflowNodeTypeStart},
				{ID: "tool-node", Type: workflowNodeTypeTool, Tool: &WorkflowToolNode{ToolName: "script_exec"}},
				{ID: "llm-node", Type: workflowNodeTypeLLM, LLM: &WorkflowLLMNode{Prompt: "branch"}},
				{ID: "end-node", Type: workflowNodeTypeEnd},
			},
			Edges: []WorkflowEdge{
				{FromNodeID: "start-node", ToNodeID: "tool-node"},
				{FromNodeID: "start-node", ToNodeID: "llm-node"},
				{FromNodeID: "llm-node", ToNodeID: "end-node"},
			},
		},
	}
	err := validateTaskDefinition(&task)
	if err == nil || !strings.Contains(err.Error(), "branching is not supported") {
		t.Fatalf("expected branching failure, got %v", err)
	}
}

func TestTaskWorkflowCreateRejectsToolOutsideAllowlist(t *testing.T) {
	t.Setenv("GHOST_WORKFLOW_TOOL_ALLOWLIST", "script_exec")
	_, service, _ := newTestHandlerWithService(t, nil, nil)

	_, code, err := service.executeTaskCreateAction(taskCreateParams{
		TaskKind:        taskKindWorkflow,
		Workflow:        workflowWithToolNode("rss_fetch"),
		IntervalSeconds: 60,
	}, "trace-workflow-tool-reject")
	if err == nil {
		t.Fatal("expected workflow tool allowlist validation to fail")
	}
	if code != http.StatusBadRequest {
		t.Fatalf("unexpected status code: got %d want %d", code, http.StatusBadRequest)
	}
	if !strings.Contains(err.Error(), `workflow tool "rss_fetch" is not allowed`) {
		t.Fatalf("unexpected error: %v", err)
	}
}
