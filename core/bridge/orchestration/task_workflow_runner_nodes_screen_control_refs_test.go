package orchestration

import (
	"context"
	"strings"
	"testing"

	"ghost-os/bridge/tools"
)

func TestExecuteWorkflowToolNodeScreenControlWorkflowStepsResolvesFindIconCoordinateRef(t *testing.T) {
	tool := &workflowScreenControlSequenceTool{
		name: screenControlToolID,
		outputs: []string{
			`{"matches":[{"center":{"x":321,"y":654}}],"display_id":5}`,
			`{"status":"clicked"}`,
		},
	}
	registry := tools.NewRegistry()
	registry.Register(tool)

	outcome := executeWorkflowToolNode(
		context.Background(),
		agentRuntimeDependencies{registry: registry},
		buildWorkflowScreenControlNode(map[string]any{
			"workflow_steps": []any{
				map[string]any{"action": "find_icon", "params": map[string]any{"template_path": "/tmp/icon.png"}},
				map[string]any{"action": "click", "params": map[string]any{"coordinate_ref": workflowScreenControlFindIconCoordinateRef}},
			},
		}),
		"trace-workflow-screen-coordinate-ref",
	)

	if outcome.err != nil {
		t.Fatalf("expected success, got error: %v", outcome.err)
	}
	if len(tool.calls) != 2 {
		t.Fatalf("unexpected tool call count: %d", len(tool.calls))
	}
	params, ok := tool.calls[1]["params"].(map[string]any)
	if !ok {
		t.Fatalf("expected second call params, got: %#v", tool.calls[1])
	}
	if params["x"] != float64(321) || params["y"] != float64(654) {
		t.Fatalf("unexpected resolved click coordinates: %#v", params)
	}
	if params["display_id"] != float64(5) {
		t.Fatalf("unexpected resolved display_id: %#v", params)
	}
	if _, exists := params[workflowScreenControlCoordinateRefKey]; exists {
		t.Fatalf("coordinate_ref should be removed before execution: %#v", params)
	}
}

func TestExecuteWorkflowToolNodeScreenControlWorkflowStepsRejectsMissingFindIconReference(t *testing.T) {
	tool := &workflowScreenControlSequenceTool{name: screenControlToolID}
	registry := tools.NewRegistry()
	registry.Register(tool)

	outcome := executeWorkflowToolNode(
		context.Background(),
		agentRuntimeDependencies{registry: registry},
		buildWorkflowScreenControlNode(map[string]any{
			"workflow_steps": []any{
				map[string]any{"action": "click", "params": map[string]any{"coordinate_ref": workflowScreenControlFindIconCoordinateRef}},
			},
		}),
		"trace-workflow-screen-coordinate-ref-missing",
	)

	if outcome.err == nil {
		t.Fatal("expected missing find_icon reference error")
	}
	if !strings.Contains(outcome.err.Error(), "requires a previous find_icon result") {
		t.Fatalf("unexpected error: %v", outcome.err)
	}
	if len(tool.calls) != 0 {
		t.Fatalf("tool should not execute when coordinate_ref is unresolved: %d", len(tool.calls))
	}
}
