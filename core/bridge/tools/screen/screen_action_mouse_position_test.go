package screen

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestScreenActionToolExecuteMousePositionUsesAtomicAction(t *testing.T) {
	var actions []string
	tool := NewScreenActionTool(mockExecutionClient{
		callFunc: func(_ context.Context, action string, _ map[string]any, _ string) (map[string]any, error) {
			actions = append(actions, action)
			if action != "MOUSE_POSITION" {
				t.Fatalf("unexpected action: %s", action)
			}
			return map[string]any{
				"x":          320,
				"y":          640,
				"display_id": 3,
				"scale_x":    2.0,
				"scale_y":    2.0,
			}, nil
		},
	})

	output, err := tool.Execute(
		context.Background(),
		json.RawMessage(`{"action":"mouse_position"}`),
		"trace-mouse-position-1",
	)
	if err != nil {
		t.Fatalf("execute returned error: %v", err)
	}
	if len(actions) != 1 || actions[0] != "MOUSE_POSITION" {
		t.Fatalf("unexpected action sequence: %v", actions)
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("decode output: %v", err)
	}
	if payload["action"] != "mouse_position" || payload["x"] != float64(320) || payload["y"] != float64(640) {
		t.Fatalf("unexpected mouse_position payload: %+v", payload)
	}
}

func TestScreenActionToolExecuteMousePositionUnsupportedActionSuggestsRebuild(t *testing.T) {
	tool := NewScreenActionTool(mockExecutionClient{
		callFunc: func(_ context.Context, action string, _ map[string]any, traceID string) (map[string]any, error) {
			if action != "MOUSE_POSITION" {
				t.Fatalf("unexpected action: %s", action)
			}
			return nil, fmt.Errorf(
				"native execution error: action=MOUSE_POSITION trace_id=%s request_id=: unsupported action: MOUSE_POSITION",
				traceID,
			)
		},
	})

	_, err := tool.Execute(
		context.Background(),
		json.RawMessage(`{"action":"mouse_position"}`),
		"trace-mouse-position-2",
	)
	if err == nil {
		t.Fatal("expected execute to fail")
	}
	if !strings.Contains(err.Error(), "rebuild drivers/native") {
		t.Fatalf("expected rebuild hint, got %v", err)
	}
}
