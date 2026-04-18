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

func TestValidateTaskDefinitionWorkflowAcceptsStartParallelBranches(t *testing.T) {
	task := ScheduledTask{
		TaskKind: taskKindWorkflow,
		Workflow: &WorkflowDefinition{
			Nodes: []WorkflowNode{
				{ID: "start-node", Type: workflowNodeTypeStart},
				{ID: "agent-a", Type: workflowNodeTypeAgent, Agent: &WorkflowAgentNode{Message: "a"}},
				{ID: "agent-b", Type: workflowNodeTypeAgent, Agent: &WorkflowAgentNode{Message: "b"}},
				{ID: "end-node", Type: workflowNodeTypeEnd},
			},
			Edges: []WorkflowEdge{
				{FromNodeID: "start-node", ToNodeID: "agent-a"},
				{FromNodeID: "start-node", ToNodeID: "agent-b"},
				{FromNodeID: "agent-a", ToNodeID: "end-node"},
				{FromNodeID: "agent-b", ToNodeID: "end-node"},
			},
		},
	}
	if err := validateTaskDefinition(&task); err != nil {
		t.Fatalf("validate workflow start parallel branches: %v", err)
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

func TestValidateTaskDefinitionWorkflowRejectsBranchingFromNonControlNode(t *testing.T) {
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
				{FromNodeID: "tool-node", ToNodeID: "llm-node"},
				{FromNodeID: "tool-node", ToNodeID: "end-node"},
				{FromNodeID: "llm-node", ToNodeID: "end-node"},
			},
		},
	}
	err := validateTaskDefinition(&task)
	if err == nil || !strings.Contains(err.Error(), `must have in>=1 and out=1`) {
		t.Fatalf("expected non-control branching failure, got %v", err)
	}
}

func TestValidateTaskDefinitionWorkflowAcceptsIfAndLoopNodes(t *testing.T) {
	task := ScheduledTask{
		TaskKind: taskKindWorkflow,
		Workflow: &WorkflowDefinition{
			Nodes: []WorkflowNode{
				{ID: "start-node", Type: workflowNodeTypeStart},
				{
					ID:   "if-node",
					Type: workflowNodeTypeIf,
					If: &WorkflowIfNode{
						Operator:    workflowIfOperatorIsEmpty,
						TrueNodeID:  "loop-node",
						FalseNodeID: "end-node",
					},
				},
				{
					ID:   "loop-node",
					Type: workflowNodeTypeLoop,
					Loop: &WorkflowLoopNode{
						MaxIterations: 2,
						BodyNodeID:    "tool-node",
						ExitNodeID:    "end-node",
					},
				},
				{ID: "tool-node", Type: workflowNodeTypeTool, Tool: &WorkflowToolNode{ToolName: "script_exec"}},
				{ID: "end-node", Type: workflowNodeTypeEnd},
			},
			Edges: []WorkflowEdge{
				{FromNodeID: "start-node", ToNodeID: "if-node"},
				{FromNodeID: "if-node", ToNodeID: "loop-node"},
				{FromNodeID: "if-node", ToNodeID: "end-node"},
				{FromNodeID: "loop-node", ToNodeID: "tool-node"},
				{FromNodeID: "loop-node", ToNodeID: "end-node"},
				{FromNodeID: "tool-node", ToNodeID: "loop-node"},
			},
		},
	}
	if err := validateTaskDefinition(&task); err != nil {
		t.Fatalf("validate workflow with if/loop: %v", err)
	}
}

func TestValidateTaskDefinitionWorkflowRejectsLoopWithoutCycle(t *testing.T) {
	task := ScheduledTask{
		TaskKind: taskKindWorkflow,
		Workflow: &WorkflowDefinition{
			Nodes: []WorkflowNode{
				{ID: "start-node", Type: workflowNodeTypeStart},
				{
					ID:   "loop-node",
					Type: workflowNodeTypeLoop,
					Loop: &WorkflowLoopNode{
						MaxIterations: 3,
						BodyNodeID:    "tool-node",
						ExitNodeID:    "end-node",
					},
				},
				{ID: "tool-node", Type: workflowNodeTypeTool, Tool: &WorkflowToolNode{ToolName: "script_exec"}},
				{ID: "end-node", Type: workflowNodeTypeEnd},
			},
			Edges: []WorkflowEdge{
				{FromNodeID: "start-node", ToNodeID: "loop-node"},
				{FromNodeID: "loop-node", ToNodeID: "tool-node"},
				{FromNodeID: "loop-node", ToNodeID: "end-node"},
				{FromNodeID: "tool-node", ToNodeID: "end-node"},
			},
		},
	}
	err := validateTaskDefinition(&task)
	if err == nil || !strings.Contains(err.Error(), "body path must return to the loop node") {
		t.Fatalf("expected loop cycle validation failure, got %v", err)
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

func TestValidateTaskDefinitionWorkflowAcceptsStartInputs(t *testing.T) {
	task := ScheduledTask{
		TaskKind: taskKindWorkflow,
		Workflow: workflowWithStartInputs([]WorkflowInputVariable{
			{
				Name:        "topic",
				Type:        workflowInputTypeString,
				Required:    true,
				Default:     []byte(`"release notes"`),
				Description: "summary topic",
			},
			{
				Name:    "threshold",
				Type:    workflowInputTypeNumber,
				Default: []byte(`0.7`),
			},
			{
				Name:    "flags",
				Type:    workflowInputTypeObject,
				Default: []byte(`{"urgent":true}`),
			},
			{
				Name:    "tags",
				Type:    workflowInputTypeArray,
				Default: []byte(`["ops","weekly"]`),
			},
		}),
	}
	if err := validateTaskDefinition(&task); err != nil {
		t.Fatalf("validate workflow with start inputs: %v", err)
	}
}

func TestValidateTaskDefinitionWorkflowRejectsInvalidStartInputs(t *testing.T) {
	tests := []struct {
		name   string
		inputs []WorkflowInputVariable
		want   string
	}{
		{
			name: "invalid variable name",
			inputs: []WorkflowInputVariable{
				{Name: "1topic", Type: workflowInputTypeString},
			},
			want: "must match ^[A-Za-z_][A-Za-z0-9_]*$",
		},
		{
			name: "duplicate variable names",
			inputs: []WorkflowInputVariable{
				{Name: "topic", Type: workflowInputTypeString},
				{Name: "topic", Type: workflowInputTypeNumber},
			},
			want: "duplicate name",
		},
		{
			name: "default type mismatch",
			inputs: []WorkflowInputVariable{
				{Name: "retry_count", Type: workflowInputTypeNumber, Default: []byte(`"3"`)},
			},
			want: "must match declared type",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			task := ScheduledTask{
				TaskKind: taskKindWorkflow,
				Workflow: workflowWithStartInputs(test.inputs),
			}
			err := validateTaskDefinition(&task)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("expected error containing %q, got %v", test.want, err)
			}
		})
	}
}

func workflowWithStartInputs(inputs []WorkflowInputVariable) *WorkflowDefinition {
	return &WorkflowDefinition{
		Nodes: []WorkflowNode{
			{
				ID:    "start-node",
				Type:  workflowNodeTypeStart,
				Start: &WorkflowStartNode{Inputs: inputs},
			},
			{ID: "end-node", Type: workflowNodeTypeEnd},
		},
		Edges: []WorkflowEdge{{FromNodeID: "start-node", ToNodeID: "end-node"}},
	}
}
