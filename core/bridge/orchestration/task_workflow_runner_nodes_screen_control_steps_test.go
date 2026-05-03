package orchestration

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"ghost-os/bridge/tools"
)

func TestExecuteWorkflowToolNodeScreenControlWorkflowStepsSequential(t *testing.T) {
	tool := &workflowScreenControlSequenceTool{
		name: screenControlToolID,
		outputs: []string{
			`{"step":"one"}`,
			`{"step":"two"}`,
		},
	}
	registry := tools.NewRegistry()
	registry.Register(tool)

	outcome := executeWorkflowToolNode(
		context.Background(),
		agentRuntimeDependencies{registry: registry},
		buildWorkflowScreenControlNode(map[string]any{
			"mode": "atomic",
			"workflow_steps": []any{
				map[string]any{"action": "screenshot"},
				map[string]any{"action": "click", "params": map[string]any{"x": 10, "y": 20}},
			},
		}),
		"trace-workflow-screen-steps",
	)

	if outcome.err != nil {
		t.Fatalf("expected success, got error: %v", outcome.err)
	}
	if outcome.status != taskRunStatusSuccess {
		t.Fatalf("unexpected status: %q", outcome.status)
	}
	if len(tool.calls) != 2 {
		t.Fatalf("unexpected tool call count: %d", len(tool.calls))
	}
	if tool.calls[0]["action"] != "screenshot" || tool.calls[1]["action"] != "click_icon" {
		t.Fatalf("unexpected call sequence: %#v", tool.calls)
	}
	if len(tool.callTimes) != 2 {
		t.Fatalf("unexpected call time count: %d", len(tool.callTimes))
	}
	const minimumObservedDelay = 180 * time.Millisecond
	if gap := tool.callTimes[1].Sub(tool.callTimes[0]); gap < minimumObservedDelay {
		t.Fatalf("expected inter-step gap >= %s, got %s", minimumObservedDelay, gap)
	}

	payload, ok := outcome.outputValue.(map[string]any)
	if !ok {
		t.Fatalf("expected map output, got %T", outcome.outputValue)
	}
	if payload["step_count"] != 2 {
		t.Fatalf("unexpected step_count: %#v", payload["step_count"])
	}
	steps, ok := payload["steps"].([]map[string]any)
	if !ok || len(steps) != 2 {
		t.Fatalf("unexpected steps payload: %#v", payload["steps"])
	}
	finalOutput, ok := payload["final_output"].(map[string]any)
	if !ok || finalOutput["step"] != "two" {
		t.Fatalf("unexpected final_output: %#v", payload["final_output"])
	}
	if !strings.Contains(outcome.outputText, `"step_count":2`) {
		t.Fatalf("expected structured output_text, got %q", outcome.outputText)
	}
}

func TestExecuteWorkflowToolNodeScreenControlWorkflowStepsFailFast(t *testing.T) {
	tool := &workflowScreenControlSequenceTool{
		name:    screenControlToolID,
		outputs: []string{`{"step":"one"}`},
		failAt:  2,
	}
	registry := tools.NewRegistry()
	registry.Register(tool)

	outcome := executeWorkflowToolNode(
		context.Background(),
		agentRuntimeDependencies{registry: registry},
		buildWorkflowScreenControlNode(map[string]any{
			"workflow_steps": []any{
				map[string]any{"action": "screenshot"},
				map[string]any{"action": "find_text", "params": map[string]any{"text": "Submit"}},
				map[string]any{"action": "click", "params": map[string]any{"x": 8, "y": 9}},
			},
		}),
		"trace-workflow-screen-fail-fast",
	)

	if outcome.err == nil {
		t.Fatal("expected fail-fast error")
	}
	if !strings.Contains(outcome.err.Error(), "step 2 (find_text)") {
		t.Fatalf("unexpected fail-fast error: %v", outcome.err)
	}
	if len(tool.calls) != 2 {
		t.Fatalf("expected stop after second step, got calls=%d", len(tool.calls))
	}
}

func TestExecuteWorkflowToolNodeScreenControlWorkflowStepsRejectsConflict(t *testing.T) {
	tool := &workflowScreenControlSequenceTool{name: screenControlToolID}
	registry := tools.NewRegistry()
	registry.Register(tool)

	outcome := executeWorkflowToolNode(
		context.Background(),
		agentRuntimeDependencies{registry: registry},
		buildWorkflowScreenControlNode(map[string]any{
			"action": "find_text",
			"params": map[string]any{"text": "Submit"},
			"workflow_steps": []any{
				map[string]any{"action": "screenshot"},
			},
		}),
		"trace-workflow-screen-conflict",
	)

	if outcome.err == nil {
		t.Fatal("expected conflict error")
	}
	if !strings.Contains(outcome.err.Error(), "conflicts with action/params") {
		t.Fatalf("unexpected conflict error: %v", outcome.err)
	}
	if len(tool.calls) != 0 {
		t.Fatalf("tool should not execute on conflict, calls=%d", len(tool.calls))
	}
}

func TestExecuteWorkflowToolNodeScreenControlSingleActionStillWorks(t *testing.T) {
	tool := &workflowScreenControlSequenceTool{
		name:    screenControlToolID,
		outputs: []string{`{"status":"ok"}`},
	}
	registry := tools.NewRegistry()
	registry.Register(tool)

	outcome := executeWorkflowToolNode(
		context.Background(),
		agentRuntimeDependencies{registry: registry},
		buildWorkflowScreenControlNode(map[string]any{
			"action": "find_text",
			"params": map[string]any{"text": "Submit"},
		}),
		"trace-workflow-screen-single",
	)

	if outcome.err != nil {
		t.Fatalf("expected single-action success, got error: %v", outcome.err)
	}
	if outcome.status != taskRunStatusSuccess {
		t.Fatalf("unexpected status: %q", outcome.status)
	}
	if len(tool.calls) != 1 || tool.calls[0]["action"] != "find_text" {
		t.Fatalf("unexpected single-action call: %#v", tool.calls)
	}
}

func TestRandomWorkflowScreenControlStepDelayWithinDefaultRange(t *testing.T) {
	minimum := time.Duration(workflowScreenControlStepDelayMinMS) * time.Millisecond
	maximum := time.Duration(workflowScreenControlStepDelayMaxMS) * time.Millisecond
	for range 64 {
		delay := randomWorkflowScreenControlStepDelay()
		if delay < minimum || delay > maximum {
			t.Fatalf("expected delay in [%s, %s], got %s", minimum, maximum, delay)
		}
	}
}

func buildWorkflowScreenControlNode(arguments map[string]any) WorkflowNode {
	return WorkflowNode{
		ID:   "tool-node",
		Type: workflowNodeTypeTool,
		Tool: &WorkflowToolNode{
			ToolName:  screenControlToolID,
			Arguments: arguments,
		},
	}
}

type workflowScreenControlSequenceTool struct {
	name      string
	outputs   []string
	failAt    int
	calls     []map[string]any
	callTimes []time.Time
}

func (t *workflowScreenControlSequenceTool) Name() string { return t.name }
func (t *workflowScreenControlSequenceTool) Description() string {
	return "workflow screen control sequence test tool"
}
func (t *workflowScreenControlSequenceTool) Parameters() json.RawMessage {
	return json.RawMessage(`{"type":"object"}`)
}

func (t *workflowScreenControlSequenceTool) Execute(_ context.Context, args json.RawMessage, _ string) (string, error) {
	decoded := map[string]any{}
	if err := json.Unmarshal(args, &decoded); err != nil {
		return "", err
	}
	t.calls = append(t.calls, decoded)
	t.callTimes = append(t.callTimes, time.Now())
	if t.failAt > 0 && len(t.calls) == t.failAt {
		return "", errors.New("forced step failure")
	}
	callIndex := len(t.calls) - 1
	if callIndex >= 0 && callIndex < len(t.outputs) {
		return t.outputs[callIndex], nil
	}
	return `{"status":"ok"}`, nil
}
