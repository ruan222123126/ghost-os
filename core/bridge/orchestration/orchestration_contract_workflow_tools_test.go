package orchestration

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	apptools "ghost-os/bridge/orchestration/internal/app/tools"
	"ghost-os/bridge/tools"
)

func TestSchemaByToolNameReturnsMissingWhenToolHasNoSchema(t *testing.T) {
	schema := apptools.SchemaByToolName(map[string]map[string]any{
		"script_exec": {"type": "object"},
	}, "web_search")
	if schema != nil {
		t.Fatalf("expected nil schema for tool without schema, got %#v", schema)
	}
}

func TestSchemaByToolNameReturnsSchemaWhenPresent(t *testing.T) {
	schema := apptools.SchemaByToolName(map[string]map[string]any{
		"script_exec": {"type": "object"},
	}, "script_exec")
	if schema == nil {
		t.Fatal("expected schema to exist")
	}
	if got, ok := schema["type"].(string); !ok || got != "object" {
		t.Fatalf("expected object schema type, got %#v", schema["type"])
	}
}

func TestExecuteFindIconTemplateUploadStoresTemplate(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	payload, err := apptools.ExecuteFindIconTemplateUpload(findIconTemplateUploadRequest{
		Filename: "icon.png",
		MimeType: "image/png",
		DataURL:  "data:image/png;base64,R2hvc3Q=",
	})
	if err != nil {
		t.Fatalf("executeFindIconTemplateUpload returned error: %v", err)
	}
	if !strings.HasPrefix(payload.TemplatePath, filepath.Join(homeDir, ".ghost-os")) {
		t.Fatalf("unexpected template path: %q", payload.TemplatePath)
	}
	if len(payload.SHA256) != 64 {
		t.Fatalf("unexpected sha256 length: %q", payload.SHA256)
	}
	if _, err := os.Stat(payload.TemplatePath); err != nil {
		t.Fatalf("template path should exist: %v", err)
	}
}

func TestNormalizeFindIconPreviewRequestRejectsInvalidThreshold(t *testing.T) {
	_, err := apptools.NormalizeFindIconPreviewRequest(findIconPreviewRequest{
		TemplatePath: "/tmp/icon.png",
		Threshold:    floatPtr(1.1),
	})
	if err == nil || !strings.Contains(err.Error(), "threshold must be between 0 and 1") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDecodeFindIconPreviewPayloadParsesMatches(t *testing.T) {
	payload, err := apptools.DecodeFindIconPreviewPayload(`{"display_id":1,"matches":[{"score":0.95}]}`)
	if err != nil {
		t.Fatalf("decodeFindIconPreviewPayload returned error: %v", err)
	}
	if !payload.Exists || payload.MatchCount != 1 {
		t.Fatalf("unexpected payload: %+v", payload)
	}
	if payload.DisplayID == nil || *payload.DisplayID != 1 {
		t.Fatalf("unexpected display id: %+v", payload.DisplayID)
	}
}

func floatPtr(value float64) *float64 {
	return &value
}

func TestPrepareWorkflowToolArgumentsUploadsFindIconTemplate(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	input := map[string]any{
		"mode":   "atomic",
		"action": "find_icon",
		"params": map[string]any{
			"workflow_template_data_url": "data:image/png;base64,R2hvc3Q=",
			"template_filename":          "icon.png",
			"template_mime_type":         "image/png",
			"threshold":                  0.91,
		},
	}
	original := cloneTaskActionParams(input)

	prepared, err := prepareWorkflowToolArguments(screenControlToolID, input)
	if err != nil {
		t.Fatalf("prepareWorkflowToolArguments returned error: %v", err)
	}

	params := decodeWorkflowToolParams(t, prepared)
	templatePath := strings.TrimSpace(workflowMapString(params, "template_path"))
	if templatePath == "" {
		t.Fatalf("expected template_path to be generated, got: %+v", params)
	}
	if !strings.HasPrefix(templatePath, filepath.Join(homeDir, ".ghost-os")) {
		t.Fatalf("unexpected template path: %q", templatePath)
	}
	if _, err := os.Stat(templatePath); err != nil {
		t.Fatalf("template path should exist: %v", err)
	}
	if _, exists := params["workflow_template_data_url"]; exists {
		t.Fatalf("workflow_template_data_url should be removed after upload: %+v", params)
	}
	if params["threshold"] != 0.91 {
		t.Fatalf("expected threshold to be preserved, got: %+v", params)
	}
	if !reflect.DeepEqual(input, original) {
		t.Fatalf("input arguments should remain immutable: before=%+v after=%+v", original, input)
	}
}

func TestPrepareWorkflowToolArgumentsRejectsInvalidFindIconUploadData(t *testing.T) {
	_, err := prepareWorkflowToolArguments(screenControlToolID, map[string]any{
		"mode":   "atomic",
		"action": "find_icon",
		"params": map[string]any{
			"workflow_template_data_url": "invalid-data-url",
			"template_mime_type":         "image/png",
		},
	})
	if err == nil || !strings.Contains(err.Error(), "data_url must start with data:") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPrepareWorkflowToolArgumentsSkipsNonScreenControlTools(t *testing.T) {
	input := map[string]any{
		"command": "pwd",
	}
	prepared, err := prepareWorkflowToolArguments("script_exec", input)
	if err != nil {
		t.Fatalf("prepareWorkflowToolArguments returned error: %v", err)
	}
	if !reflect.DeepEqual(prepared, input) {
		t.Fatalf("unexpected prepared arguments: got=%+v want=%+v", prepared, input)
	}
}

func decodeWorkflowToolParams(t *testing.T, arguments map[string]any) map[string]any {
	t.Helper()
	params, ok := arguments["params"].(map[string]any)
	if !ok {
		t.Fatalf("params should be an object, got: %T", arguments["params"])
	}
	return params
}

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
