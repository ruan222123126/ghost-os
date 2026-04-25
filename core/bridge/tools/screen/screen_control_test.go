package screen

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestScreenControlExecuteAtomicRoutesToScreenAction(t *testing.T) {
	imagePath := writeScreenActionTestPNG(t)
	screenshotsDir := t.TempDir()
	t.Setenv("GHOST_SCREENSHOTS_PATH", screenshotsDir)

	var calls []string
	tool := NewScreenControlTool(
		mockExecutionClient{
			callFunc: func(_ context.Context, action string, _ map[string]any, _ string) (map[string]any, error) {
				calls = append(calls, action)
				if action != "SCREEN_CAPTURE" {
					t.Fatalf("unexpected action: %s", action)
				}
				return map[string]any{
					"image_path":   imagePath,
					"image_width":  1,
					"image_height": 1,
					"display_id":   2,
				}, nil
			},
		},
		nil,
		nil,
	).(*ScreenControlTool)

	output, err := tool.Execute(context.Background(), json.RawMessage(`{"mode":"atomic","action":"screenshot"}`), "trace-screen-control-atomic")
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	if strings.Join(calls, ",") != "SCREEN_CAPTURE" {
		t.Fatalf("unexpected action sequence: %v", calls)
	}

	var result screenActionResult
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("decode output: %v", err)
	}
	if result.Action != "screenshot" || result.Artifact == nil {
		t.Fatalf("unexpected atomic output: %s", output)
	}
}

func TestScreenControlRejectsRemovedAgentMode(t *testing.T) {
	tool := NewScreenControlTool(nil, nil, nil)

	_, err := tool.Execute(
		context.Background(),
		json.RawMessage(`{"mode":"agent","goal":"x"}`),
		"trace-screen-control-validate-1",
	)
	if err == nil || !strings.Contains(err.Error(), "mode=\"agent\" is removed") {
		t.Fatalf("expected removed agent mode error, got %v", err)
	}
}

func TestScreenControlRejectsAgentFields(t *testing.T) {
	tool := NewScreenControlTool(nil, nil, nil)

	_, err := tool.Execute(
		context.Background(),
		json.RawMessage(`{"action":"screenshot","goal":"x"}`),
		"trace-screen-control-validate-2",
	)
	if err == nil || !strings.Contains(err.Error(), "does not support goal") {
		t.Fatalf("expected goal rejection error, got %v", err)
	}

	_, err = tool.Execute(
		context.Background(),
		json.RawMessage(`{"mode":"atomic","action":"screenshot","target":{"window_title":"A"}}`),
		"trace-screen-control-validate-3",
	)
	if err == nil || !strings.Contains(err.Error(), "does not support target") {
		t.Fatalf("expected target rejection error, got %v", err)
	}
}

func TestScreenControlModeValidation(t *testing.T) {
	tool := NewScreenControlTool(nil, nil, nil)

	_, err := tool.Execute(
		context.Background(),
		json.RawMessage(`{"mode":"desktop","action":"screenshot"}`),
		"trace-screen-control-validate-4",
	)
	if err == nil || !strings.Contains(err.Error(), "only supports mode=\"atomic\"") {
		t.Fatalf("expected mode validation error, got %v", err)
	}

	_, err = tool.Execute(
		context.Background(),
		json.RawMessage(`{"mode":"atomic"}`),
		"trace-screen-control-validate-5",
	)
	if err == nil || !strings.Contains(err.Error(), "action is required") {
		t.Fatalf("expected missing action error, got %v", err)
	}
}

func TestScreenControlRejectsWorkflowOnlyUploadKey(t *testing.T) {
	tool := NewScreenControlTool(nil, nil, nil)

	_, err := tool.Execute(
		context.Background(),
		json.RawMessage(`{"mode":"atomic","action":"find_icon","params":{"workflow_template_data_url":"data:image/png;base64,R2hvc3Q="}}`),
		"trace-screen-control-validate-6",
	)
	if err == nil || !strings.Contains(err.Error(), "workflow-only") {
		t.Fatalf("expected workflow-only param error, got %v", err)
	}
}

func TestScreenControlTextInputMapsToNativeTextInput(t *testing.T) {
	var gotAction string
	var gotParams map[string]any
	tool := NewScreenControlTool(
		mockExecutionClient{
			callFunc: func(_ context.Context, action string, params map[string]any, traceID string) (map[string]any, error) {
				gotAction = action
				gotParams = params
				if traceID != "trace-screen-control-text-1" {
					t.Fatalf("unexpected trace id: got %q want %q", traceID, "trace-screen-control-text-1")
				}
				return map[string]any{"typed": true, "submitted": true, "characters": 11}, nil
			},
		},
		nil,
		nil,
	).(*ScreenControlTool)

	output, err := tool.Execute(
		context.Background(),
		json.RawMessage(`{"action":"text_input","params":{"text":"hello\nworld","submit":true}}`),
		"trace-screen-control-text-1",
	)
	if err != nil {
		t.Fatalf("execute returned error: %v", err)
	}
	if gotAction != "TEXT_INPUT" {
		t.Fatalf("unexpected native action: got %q want %q", gotAction, "TEXT_INPUT")
	}
	if gotParams["text"] != "hello\nworld" || gotParams["submit"] != true {
		t.Fatalf("unexpected text_input forwarding: %+v", gotParams)
	}
	if !strings.Contains(output, `"action":"text_input"`) || !strings.Contains(output, `"typed":true`) {
		t.Fatalf("unexpected output: %s", output)
	}
}

func TestScreenControlTextInputRejectsInvalidParams(t *testing.T) {
	tool := NewScreenControlTool(mockExecutionClient{
		callFunc: func(_ context.Context, _ string, _ map[string]any, _ string) (map[string]any, error) {
			t.Fatal("execution should not be called")
			return nil, nil
		},
	}, nil, nil)

	_, err := tool.Execute(
		context.Background(),
		json.RawMessage(`{"action":"text_input","display_id":1,"params":{"text":"ghost"}}`),
		"trace-screen-control-text-2",
	)
	if err == nil || !strings.Contains(err.Error(), "display_id is not supported") {
		t.Fatalf("expected display_id rejection error, got %v", err)
	}

	_, err = tool.Execute(
		context.Background(),
		json.RawMessage(`{"action":"text_input","params":{"text":"ghost","display_id":1}}`),
		"trace-screen-control-text-3",
	)
	if err == nil || !strings.Contains(err.Error(), "params.display_id is not supported") {
		t.Fatalf("expected params key rejection error, got %v", err)
	}
}
