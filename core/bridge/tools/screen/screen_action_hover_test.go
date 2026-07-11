package screen

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestScreenActionToolExecuteClickIconHoverOnlyUsesMouseMove(t *testing.T) {
	imagePath := writeScreenActionTestPNG(t)
	var actions []string
	tool := NewScreenActionTool(mockExecutionClient{
		callFunc: func(_ context.Context, action string, params map[string]any, _ string) (map[string]any, error) {
			actions = append(actions, action)
			switch action {
			case "SCREEN_CAPTURE":
				return map[string]any{
					"image_path":   imagePath,
					"image_width":  1,
					"image_height": 1,
					"display_id":   9,
					"scale_x":      1.0,
					"scale_y":      1.0,
					"origin_x":     0,
					"origin_y":     0,
					"region": map[string]any{
						"x": 0, "y": 0, "width": 1, "height": 1,
					},
				}, nil
			case "TEMPLATE_MATCH_IMAGE":
				return map[string]any{
					"matches": []any{
						map[string]any{
							"score": 0.9,
							"bbox": map[string]any{
								"x": 10, "y": 20, "width": 12, "height": 12,
							},
							"center": map[string]any{"x": 16, "y": 26},
						},
					},
				}, nil
			case "MOUSE_MOVE":
				if params["x"] != 16 || params["y"] != 26 || params["display_id"] != 9 {
					t.Fatalf("unexpected mouse move params: %+v", params)
				}
				return map[string]any{"moved": true}, nil
			default:
				t.Fatalf("unexpected action: %s", action)
				return nil, nil
			}
		},
	})

	output, err := tool.Execute(
		context.Background(),
		json.RawMessage(`{"action":"click_icon","params":{"display_id":9,"template_path":"/tmp/icon.png","hover_only":true}}`),
		"trace-hover-icon-1",
	)
	if err != nil {
		t.Fatalf("execute returned error: %v", err)
	}
	if strings.Join(actions, ",") != "SCREEN_CAPTURE,TEMPLATE_MATCH_IMAGE,MOUSE_MOVE" {
		t.Fatalf("unexpected action sequence: %v", actions)
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("decode output: %v", err)
	}
	if payload["action"] != "click_icon" || payload["hovered"] != true {
		t.Fatalf("unexpected hover payload: %+v", payload)
	}
}

func TestScreenActionToolExecuteClickIconHoverOnlyDirectPoint(t *testing.T) {
	var actions []string
	tool := NewScreenActionTool(mockExecutionClient{
		callFunc: func(_ context.Context, action string, params map[string]any, _ string) (map[string]any, error) {
			actions = append(actions, action)
			if action != "MOUSE_MOVE" {
				t.Fatalf("unexpected action: %s", action)
			}
			if params["x"] != 100 || params["y"] != 200 || params["display_id"] != 3 {
				t.Fatalf("unexpected mouse move params: %+v", params)
			}
			return map[string]any{"moved": true}, nil
		},
	})

	output, err := tool.Execute(
		context.Background(),
		json.RawMessage(`{"action":"click_icon","params":{"x":100,"y":200,"display_id":3,"hover_only":true}}`),
		"trace-hover-icon-2",
	)
	if err != nil {
		t.Fatalf("execute returned error: %v", err)
	}
	if strings.Join(actions, ",") != "MOUSE_MOVE" {
		t.Fatalf("unexpected action sequence: %v", actions)
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("decode output: %v", err)
	}
	if payload["hovered"] != true || payload["direct"] != true {
		t.Fatalf("unexpected direct hover payload: %+v", payload)
	}
}

func TestScreenActionToolExecuteClickIconHoverOnlyRelativeDirectPoint(t *testing.T) {
	var actions []string
	tool := NewScreenActionTool(mockExecutionClient{
		callFunc: func(_ context.Context, action string, params map[string]any, _ string) (map[string]any, error) {
			actions = append(actions, action)
			switch action {
			case "MOUSE_POSITION":
				return map[string]any{
					"x":          120,
					"y":          80,
					"display_id": 5,
					"scale_x":    1.0,
					"scale_y":    1.0,
				}, nil
			case "MOUSE_MOVE":
				if params["x"] != 135 || params["y"] != 60 || params["display_id"] != 5 {
					t.Fatalf("unexpected mouse move params: %+v", params)
				}
				return map[string]any{"moved": true}, nil
			default:
				t.Fatalf("unexpected action: %s", action)
				return nil, nil
			}
		},
	})

	output, err := tool.Execute(
		context.Background(),
		json.RawMessage(`{"action":"click_icon","params":{"x":15,"y":-20,"display_id":3,"position_type":"relative","hover_only":true}}`),
		"trace-hover-icon-relative-1",
	)
	if err != nil {
		t.Fatalf("execute returned error: %v", err)
	}
	if strings.Join(actions, ",") != "MOUSE_POSITION,MOUSE_MOVE" {
		t.Fatalf("unexpected action sequence: %v", actions)
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("decode output: %v", err)
	}
	if payload["hovered"] != true || payload["display_id"] != float64(5) || payload["position_type"] != "relative" {
		t.Fatalf("unexpected relative hover payload: %+v", payload)
	}
}

func TestScreenActionToolExecuteClickIconHoverOnlyUnsupportedMouseMoveSuggestsRebuild(t *testing.T) {
	tool := NewScreenActionTool(mockExecutionClient{
		callFunc: func(_ context.Context, action string, _ map[string]any, traceID string) (map[string]any, error) {
			if action != "MOUSE_MOVE" {
				t.Fatalf("unexpected action: %s", action)
			}
			return nil, fmt.Errorf(
				"native execution error: action=MOUSE_MOVE trace_id=%s request_id=: unsupported action: MOUSE_MOVE",
				traceID,
			)
		},
	})

	_, err := tool.Execute(
		context.Background(),
		json.RawMessage(`{"action":"click_icon","params":{"x":100,"y":200,"display_id":3,"hover_only":true}}`),
		"trace-hover-icon-3",
	)
	if err == nil {
		t.Fatal("expected execute to fail")
	}
	if !strings.Contains(err.Error(), "rebuild drivers/native") {
		t.Fatalf("expected rebuild hint, got %v", err)
	}
}
