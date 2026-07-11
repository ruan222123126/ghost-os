package screen

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestScreenActionToolExecuteFindIconHoverAfterMatchUsesMouseMove(t *testing.T) {
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
					"display_id":   5,
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
							"score": 0.91,
							"bbox": map[string]any{
								"x": 10, "y": 20, "width": 12, "height": 12,
							},
							"center": map[string]any{"x": 16, "y": 26},
						},
					},
				}, nil
			case "MOUSE_MOVE":
				if params["x"] != 16 || params["y"] != 26 || params["display_id"] != 5 {
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
		json.RawMessage(`{"action":"find_icon","params":{"display_id":5,"template_path":"/tmp/icon.png","hover_after_match":true}}`),
		"trace-find-icon-hover-1",
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
	if payload["action"] != "find_icon" || payload["hovered"] != true || payload["exists"] != true {
		t.Fatalf("unexpected find_icon hover payload: %+v", payload)
	}
}

func TestScreenActionToolExecuteFindIconHoverAfterMatchSkipsMouseMoveWhenNoMatch(t *testing.T) {
	imagePath := writeScreenActionTestPNG(t)
	var actions []string
	tool := NewScreenActionTool(mockExecutionClient{
		callFunc: func(_ context.Context, action string, _ map[string]any, _ string) (map[string]any, error) {
			actions = append(actions, action)
			switch action {
			case "SCREEN_CAPTURE":
				return map[string]any{
					"image_path":   imagePath,
					"image_width":  1,
					"image_height": 1,
					"display_id":   5,
					"scale_x":      1.0,
					"scale_y":      1.0,
					"origin_x":     0,
					"origin_y":     0,
					"region": map[string]any{
						"x": 0, "y": 0, "width": 1, "height": 1,
					},
				}, nil
			case "TEMPLATE_MATCH_IMAGE":
				return map[string]any{"matches": []any{}}, nil
			default:
				t.Fatalf("unexpected action: %s", action)
				return nil, nil
			}
		},
	})

	output, err := tool.Execute(
		context.Background(),
		json.RawMessage(`{"action":"find_icon","params":{"display_id":5,"template_path":"/tmp/icon.png","hover_after_match":true}}`),
		"trace-find-icon-hover-2",
	)
	if err != nil {
		t.Fatalf("execute returned error: %v", err)
	}
	if strings.Join(actions, ",") != "SCREEN_CAPTURE,TEMPLATE_MATCH_IMAGE" {
		t.Fatalf("unexpected action sequence: %v", actions)
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("decode output: %v", err)
	}
	if payload["exists"] != false || payload["match_count"] != float64(0) {
		t.Fatalf("unexpected find_icon hover no-match payload: %+v", payload)
	}
	if _, ok := payload["hovered"]; ok {
		t.Fatalf("expected hovered to be omitted, got %+v", payload)
	}
}
