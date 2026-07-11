package orchestration

import (
	"context"
	"encoding/json"
	"errors"
	"ghost-os/bridge/llm"
	"ghost-os/bridge/streaming"
	"strings"
	"sync"
	"testing"
	"time"
)

type workflowTestTool struct {
	name    string
	output  string
	outputs []string
	calls   []map[string]any
}

func (t *workflowTestTool) Name() string { return t.name }

func (t *workflowTestTool) Description() string { return "workflow test tool" }

func (t *workflowTestTool) Parameters() json.RawMessage { return json.RawMessage(`{"type":"object"}`) }

func (t *workflowTestTool) Execute(_ context.Context, args json.RawMessage, _ string) (string, error) {
	decoded := make(map[string]any)
	if err := json.Unmarshal(args, &decoded); err != nil {
		return "", err
	}
	t.calls = append(t.calls, decoded)
	callIndex := len(t.calls) - 1
	if callIndex >= 0 && callIndex < len(t.outputs) {
		return t.outputs[callIndex], nil
	}
	return t.output, nil
}

type workflowParallelProbeTool struct {
	name        string
	startedCh   chan struct{}
	releaseOnce sync.Once
	releaseCh   chan struct{}
	mu          sync.Mutex
	active      int
	maxActive   int
}

func newWorkflowParallelProbeTool(name string) *workflowParallelProbeTool {
	return &workflowParallelProbeTool{
		name:      name,
		startedCh: make(chan struct{}, 8),
		releaseCh: make(chan struct{}),
	}
}

func (t *workflowParallelProbeTool) Name() string { return t.name }

func (t *workflowParallelProbeTool) Description() string { return "workflow parallel probe tool" }

func (t *workflowParallelProbeTool) Parameters() json.RawMessage {
	return json.RawMessage(`{"type":"object"}`)
}

func (t *workflowParallelProbeTool) Execute(_ context.Context, _ json.RawMessage, _ string) (string, error) {
	t.mu.Lock()
	t.active++
	if t.active > t.maxActive {
		t.maxActive = t.active
	}
	t.mu.Unlock()
	t.startedCh <- struct{}{}
	<-t.releaseCh
	t.mu.Lock()
	t.active--
	t.mu.Unlock()
	return `{"status":"ok"}`, nil
}

func (t *workflowParallelProbeTool) waitStarted(count int, timeout time.Duration) bool {
	for i := 0; i < count; i++ {
		select {
		case <-t.startedCh:
		case <-time.After(timeout):
			return false
		}
	}
	return true
}

func (t *workflowParallelProbeTool) release() {
	t.releaseOnce.Do(func() {
		close(t.releaseCh)
	})
}

func (t *workflowParallelProbeTool) maxConcurrency() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.maxActive
}

type workflowTestCompleter struct {
	response *llm.CompletionResponse
	err      error
	requests []llm.CompletionRequest
}

func (c *workflowTestCompleter) Complete(_ context.Context, request llm.CompletionRequest) (*llm.CompletionResponse, error) {
	c.requests = append(c.requests, request)
	if c.err != nil {
		return nil, c.err
	}
	if c.response == nil {
		return nil, errors.New("unexpected llm completion")
	}
	return c.response, nil
}

type workflowRunnerCall struct {
	message          string
	sessionID        string
	runtimeOverrides *TaskRuntimeOverrides
}

type workflowTestRunner struct {
	message   string
	sessionID string
	err       error
	calls     []workflowRunnerCall
}

func (r *workflowTestRunner) RunTurn(_ context.Context, message string, sessionID string, _ string) (string, string, error) {
	r.calls = append(r.calls, workflowRunnerCall{message: message, sessionID: sessionID})
	return r.message, r.sessionID, r.err
}

func (r *workflowTestRunner) RunTurnWithOverrides(
	_ context.Context,
	message string,
	sessionID string,
	_ string,
	runtimeOverrides *TaskRuntimeOverrides,
) (string, string, error) {
	r.calls = append(r.calls, workflowRunnerCall{
		message:          message,
		sessionID:        sessionID,
		runtimeOverrides: cloneTaskRuntimeOverrides(runtimeOverrides),
	})
	return r.message, r.sessionID, r.err
}

func (r *workflowTestRunner) RunTurnStream(
	ctx context.Context,
	message string,
	sessionID string,
	traceID string,
	_ streaming.Sink,
) (string, string, error) {
	return r.RunTurn(ctx, message, sessionID, traceID)
}

func workflowWithToolNode(toolName string) *WorkflowDefinition {
	return &WorkflowDefinition{
		Nodes: []WorkflowNode{
			{ID: "start-node", Type: workflowNodeTypeStart},
			{ID: "tool-node", Type: workflowNodeTypeTool, Tool: &WorkflowToolNode{ToolName: toolName, Arguments: map[string]any{"command": "pwd"}}},
			{ID: "end-node", Type: workflowNodeTypeEnd},
		},
		Edges: []WorkflowEdge{
			{FromNodeID: "start-node", ToNodeID: "tool-node"},
			{FromNodeID: "tool-node", ToNodeID: "end-node"},
		},
	}
}

func assertWorkflowNodeResultsSequence(t *testing.T, nodeResults []RunNodeResult, expectedNodeIDs []string) {
	t.Helper()
	if len(nodeResults) != len(expectedNodeIDs) {
		t.Fatalf("unexpected node result count: got %d want %d", len(nodeResults), len(expectedNodeIDs))
	}
	for index, node := range nodeResults {
		if node.CompletedSeq != index+1 {
			t.Fatalf("unexpected completed_seq at index=%d: %#v", index, node)
		}
		if node.NodeID != expectedNodeIDs[index] {
			t.Fatalf("unexpected node order at index=%d: got %q want %q", index, node.NodeID, expectedNodeIDs[index])
		}
		if strings.TrimSpace(node.Status) == "" || node.StartedAt.IsZero() || node.FinishedAt.IsZero() {
			t.Fatalf("expected non-empty status/timestamps: %#v", node)
		}
	}
}

func assertWorkflowNodeResultsMonotonic(t *testing.T, nodeResults []RunNodeResult) {
	t.Helper()
	if len(nodeResults) == 0 {
		t.Fatal("expected workflow node results")
	}
	lastSeq := 0
	for index, node := range nodeResults {
		if node.CompletedSeq <= lastSeq {
			t.Fatalf("completed_seq must be strictly increasing at index=%d: %#v", index, node)
		}
		lastSeq = node.CompletedSeq
	}
}

func assertRunTranscriptSession(
	t *testing.T,
	service *bridgeService,
	sessionID string,
	disallowed []string,
	required []string,
) {
	t.Helper()
	if strings.TrimSpace(sessionID) == "" {
		t.Fatal("expected run transcript session id")
	}
	for _, blocked := range disallowed {
		if sessionID == blocked {
			t.Fatalf("expected display transcript session, got execution session %q", sessionID)
		}
	}
	if service == nil || service.sessionStore == nil {
		t.Fatal("session store is not configured")
	}
	loaded, err := service.sessionStore.Load(sessionID)
	if err != nil {
		t.Fatalf("load run transcript session %q: %v", sessionID, err)
	}
	transcript := sessionMessagesText(loaded.Messages)
	for _, expected := range required {
		if !strings.Contains(transcript, expected) {
			t.Fatalf("run transcript missing %q in:\n%s", expected, transcript)
		}
	}
}

func sessionMessagesText(messages []llm.Message) string {
	parts := make([]string, 0, len(messages)*2)
	for _, message := range messages {
		parts = append(parts, string(message.Role)+": "+message.Text)
		for _, call := range message.ToolCalls {
			parts = append(parts, "tool_call: "+call.Name+" "+string(call.Arguments))
		}
	}
	return strings.Join(parts, "\n")
}

func workflowWithParallelStartToolBranches(toolName string) *WorkflowDefinition {
	return &WorkflowDefinition{
		Nodes: []WorkflowNode{
			{ID: "start-node", Type: workflowNodeTypeStart},
			{ID: "tool-a", Type: workflowNodeTypeTool, Tool: &WorkflowToolNode{ToolName: toolName, Arguments: map[string]any{"command": "echo a"}}},
			{ID: "tool-b", Type: workflowNodeTypeTool, Tool: &WorkflowToolNode{ToolName: toolName, Arguments: map[string]any{"command": "echo b"}}},
			{ID: "end-node", Type: workflowNodeTypeEnd},
		},
		Edges: []WorkflowEdge{
			{FromNodeID: "start-node", ToNodeID: "tool-a"},
			{FromNodeID: "start-node", ToNodeID: "tool-b"},
			{FromNodeID: "tool-a", ToNodeID: "end-node"},
			{FromNodeID: "tool-b", ToNodeID: "end-node"},
		},
	}
}

func workflowWithAgentNode() *WorkflowDefinition {
	return &WorkflowDefinition{
		Nodes: []WorkflowNode{
			{ID: "start-node", Type: workflowNodeTypeStart},
			{ID: "agent-node", Type: workflowNodeTypeAgent, Agent: &WorkflowAgentNode{Message: "fixed agent message"}},
			{ID: "end-node", Type: workflowNodeTypeEnd},
		},
		Edges: []WorkflowEdge{
			{FromNodeID: "start-node", ToNodeID: "agent-node"},
			{FromNodeID: "agent-node", ToNodeID: "end-node"},
		},
	}
}

func workflowWithIfNode(toolName string) *WorkflowDefinition {
	return &WorkflowDefinition{
		Nodes: []WorkflowNode{
			{ID: "start-node", Type: workflowNodeTypeStart},
			{ID: "tool-node", Type: workflowNodeTypeTool, Tool: &WorkflowToolNode{ToolName: toolName, Arguments: map[string]any{"command": "pwd"}}},
			{
				ID:   "if-node",
				Type: workflowNodeTypeIf,
				If: &WorkflowIfNode{
					SourceNodeID: "tool-node",
					Operator:     workflowIfOperatorContains,
					Value:        "ok",
					TrueNodeID:   "agent-true",
					FalseNodeID:  "agent-false",
				},
			},
			{ID: "agent-true", Type: workflowNodeTypeAgent, Agent: &WorkflowAgentNode{Message: "true branch message"}},
			{ID: "agent-false", Type: workflowNodeTypeAgent, Agent: &WorkflowAgentNode{Message: "false branch message"}},
			{ID: "end-node", Type: workflowNodeTypeEnd},
		},
		Edges: []WorkflowEdge{
			{FromNodeID: "start-node", ToNodeID: "tool-node"},
			{FromNodeID: "tool-node", ToNodeID: "if-node"},
			{FromNodeID: "if-node", ToNodeID: "agent-true"},
			{FromNodeID: "if-node", ToNodeID: "agent-false"},
			{FromNodeID: "agent-true", ToNodeID: "end-node"},
			{FromNodeID: "agent-false", ToNodeID: "end-node"},
		},
	}
}

func workflowWithTemplatedIfNode(toolName string) *WorkflowDefinition {
	return &WorkflowDefinition{
		Nodes: []WorkflowNode{
			{
				ID:   "start-node",
				Type: workflowNodeTypeStart,
				Start: &WorkflowStartNode{Inputs: []WorkflowInputVariable{
					{Name: "token", Type: workflowInputTypeString, Default: []byte(`"ok"`)},
				}},
			},
			{ID: "tool-node", Type: workflowNodeTypeTool, Tool: &WorkflowToolNode{ToolName: toolName, Arguments: map[string]any{"command": "pwd"}}},
			{
				ID:   "if-node",
				Type: workflowNodeTypeIf,
				If: &WorkflowIfNode{
					SourceNodeID: "tool-node",
					Operator:     workflowIfOperatorContains,
					Value:        "${inputs.token}",
					TrueNodeID:   "agent-true",
					FalseNodeID:  "agent-false",
				},
			},
			{ID: "agent-true", Type: workflowNodeTypeAgent, Agent: &WorkflowAgentNode{Message: "true branch message"}},
			{ID: "agent-false", Type: workflowNodeTypeAgent, Agent: &WorkflowAgentNode{Message: "false branch message"}},
			{ID: "end-node", Type: workflowNodeTypeEnd},
		},
		Edges: []WorkflowEdge{
			{FromNodeID: "start-node", ToNodeID: "tool-node"},
			{FromNodeID: "tool-node", ToNodeID: "if-node"},
			{FromNodeID: "if-node", ToNodeID: "agent-true"},
			{FromNodeID: "if-node", ToNodeID: "agent-false"},
			{FromNodeID: "agent-true", ToNodeID: "end-node"},
			{FromNodeID: "agent-false", ToNodeID: "end-node"},
		},
	}
}

func workflowWithLoopNode(toolName string, maxIterations int) *WorkflowDefinition {
	return &WorkflowDefinition{
		Nodes: []WorkflowNode{
			{ID: "start-node", Type: workflowNodeTypeStart},
			{
				ID:   "loop-node",
				Type: workflowNodeTypeLoop,
				Loop: &WorkflowLoopNode{
					MaxIterations: maxIterations,
					BodyNodeID:    "tool-node",
					ExitNodeID:    "end-node",
				},
			},
			{ID: "tool-node", Type: workflowNodeTypeTool, Tool: &WorkflowToolNode{ToolName: toolName, Arguments: map[string]any{"command": "pwd"}}},
			{ID: "end-node", Type: workflowNodeTypeEnd},
		},
		Edges: []WorkflowEdge{
			{FromNodeID: "start-node", ToNodeID: "loop-node"},
			{FromNodeID: "loop-node", ToNodeID: "tool-node"},
			{FromNodeID: "loop-node", ToNodeID: "end-node"},
			{FromNodeID: "tool-node", ToNodeID: "loop-node"},
		},
	}
}

func workflowWithToolLLMAndAgentNodes(toolName string) *WorkflowDefinition {
	return &WorkflowDefinition{
		Nodes: []WorkflowNode{
			{ID: "start-node", Type: workflowNodeTypeStart},
			{ID: "tool-node", Type: workflowNodeTypeTool, Tool: &WorkflowToolNode{ToolName: toolName, Arguments: map[string]any{"command": "pwd"}}},
			{ID: "llm-node", Type: workflowNodeTypeLLM, LLM: &WorkflowLLMNode{Prompt: "Summarize the previous result", SystemPrompt: "Be concise"}},
			{ID: "agent-node", Type: workflowNodeTypeAgent, Agent: &WorkflowAgentNode{Message: "fixed agent message"}},
			{ID: "end-node", Type: workflowNodeTypeEnd},
		},
		Edges: []WorkflowEdge{
			{FromNodeID: "start-node", ToNodeID: "tool-node"},
			{FromNodeID: "tool-node", ToNodeID: "llm-node"},
			{FromNodeID: "llm-node", ToNodeID: "agent-node"},
			{FromNodeID: "agent-node", ToNodeID: "end-node"},
		},
	}
}

func workflowWithTemplatedToolChain(toolName string) *WorkflowDefinition {
	return &WorkflowDefinition{
		Nodes: []WorkflowNode{
			{
				ID:   "start-node",
				Type: workflowNodeTypeStart,
				Start: &WorkflowStartNode{Inputs: []WorkflowInputVariable{
					{
						Name:    "command",
						Type:    workflowInputTypeString,
						Default: []byte(`"pwd"`),
					},
				}},
			},
			{
				ID:   "tool-read",
				Type: workflowNodeTypeTool,
				Tool: &WorkflowToolNode{
					ToolName:  toolName,
					Arguments: map[string]any{"command": "${inputs.command}"},
				},
			},
			{
				ID:   "tool-use",
				Type: workflowNodeTypeTool,
				Tool: &WorkflowToolNode{
					ToolName: toolName,
					Arguments: map[string]any{
						"command": "echo ${outputs.tool-read.cwd}",
						"ok":      "${outputs.tool-read.ok}",
						"text":    "run ${inputs.command}",
					},
				},
			},
			{ID: "end-node", Type: workflowNodeTypeEnd},
		},
		Edges: []WorkflowEdge{
			{FromNodeID: "start-node", ToNodeID: "tool-read"},
			{FromNodeID: "tool-read", ToNodeID: "tool-use"},
			{FromNodeID: "tool-use", ToNodeID: "end-node"},
		},
	}
}

func workflowWithUndefinedTemplate(toolName string) *WorkflowDefinition {
	return &WorkflowDefinition{
		Nodes: []WorkflowNode{
			{ID: "start-node", Type: workflowNodeTypeStart},
			{
				ID:   "tool-node",
				Type: workflowNodeTypeTool,
				Tool: &WorkflowToolNode{
					ToolName:  toolName,
					Arguments: map[string]any{"command": "${outputs.missing.status}"},
				},
			},
			{ID: "end-node", Type: workflowNodeTypeEnd},
		},
		Edges: []WorkflowEdge{
			{FromNodeID: "start-node", ToNodeID: "tool-node"},
			{FromNodeID: "tool-node", ToNodeID: "end-node"},
		},
	}
}

func workflowWithFindIconVariableConsumer(toolName string) *WorkflowDefinition {
	return &WorkflowDefinition{
		Nodes: []WorkflowNode{
			{ID: "start-node", Type: workflowNodeTypeStart},
			{
				ID:   "screen-node",
				Type: workflowNodeTypeTool,
				Tool: &WorkflowToolNode{
					ToolName: "screen_control",
					Arguments: map[string]any{
						"mode":   "atomic",
						"action": "find_icon",
						"params": map[string]any{"template_path": "/tmp/icon.png"},
					},
				},
			},
			{
				ID:   "tool-use",
				Type: workflowNodeTypeTool,
				Tool: &WorkflowToolNode{
					ToolName: toolName,
					Arguments: map[string]any{
						"x":     "${find_icon.x}",
						"y":     "${find_icon.y}",
						"point": "${find_icon}",
						"label": "prefix ${find_icon.x}",
					},
				},
			},
			{ID: "end-node", Type: workflowNodeTypeEnd},
		},
		Edges: []WorkflowEdge{
			{FromNodeID: "start-node", ToNodeID: "screen-node"},
			{FromNodeID: "screen-node", ToNodeID: "tool-use"},
			{FromNodeID: "tool-use", ToNodeID: "end-node"},
		},
	}
}

func workflowWithUndefinedFindIconVariable(toolName string) *WorkflowDefinition {
	return &WorkflowDefinition{
		Nodes: []WorkflowNode{
			{ID: "start-node", Type: workflowNodeTypeStart},
			{
				ID:   "tool-node",
				Type: workflowNodeTypeTool,
				Tool: &WorkflowToolNode{
					ToolName:  toolName,
					Arguments: map[string]any{"point": "${find_icon}"},
				},
			},
			{ID: "end-node", Type: workflowNodeTypeEnd},
		},
		Edges: []WorkflowEdge{
			{FromNodeID: "start-node", ToNodeID: "tool-node"},
			{FromNodeID: "tool-node", ToNodeID: "end-node"},
		},
	}
}

func validWorkflowDefinition() *WorkflowDefinition {
	return &WorkflowDefinition{
		Nodes: []WorkflowNode{
			{ID: "start-node", Type: workflowNodeTypeStart},
			{ID: "end-node", Type: workflowNodeTypeEnd},
		},
		Edges: []WorkflowEdge{{FromNodeID: "start-node", ToNodeID: "end-node"}},
	}
}
