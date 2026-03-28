package tools

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScreenActionToolExecuteOCRScanUsesAtomicActions(t *testing.T) {
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
					"display_id":   3,
					"scale_x":      2.0,
					"scale_y":      1.5,
					"origin_x":     100,
					"origin_y":     200,
					"region": map[string]any{
						"x": 10, "y": 20, "width": 1, "height": 1,
					},
				}, nil
			case "OCR_IMAGE":
				if params["image_path"] != imagePath {
					t.Fatalf("unexpected OCR_IMAGE image_path: %+v", params)
				}
				if params["origin_x"] != 100 || params["origin_y"] != 200 {
					t.Fatalf("unexpected OCR_IMAGE params: %+v", params)
				}
				return map[string]any{
					"items": []any{
						map[string]any{
							"text":       "文件",
							"confidence": 0.95,
							"bbox": map[string]any{
								"x": 110, "y": 220, "width": 40, "height": 12,
							},
							"center": map[string]any{"x": 130, "y": 226},
						},
					},
				}, nil
			default:
				t.Fatalf("unexpected action: %s", action)
				return nil, nil
			}
		},
	})

	output, err := tool.Execute(
		context.Background(),
		json.RawMessage(`{"action":"ocr_scan","params":{"display_id":3,"languages":["en"],"min_confidence":0.9}}`),
		"trace-ocr-1",
	)
	if err != nil {
		t.Fatalf("execute returned error: %v", err)
	}
	if strings.Join(actions, ",") != "SCREEN_CAPTURE,OCR_IMAGE" {
		t.Fatalf("unexpected action sequence: %v", actions)
	}

	var payload screenOCRPayload
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("decode output: %v", err)
	}
	if payload.DisplayID != 3 || payload.OriginX != 100 || payload.OriginY != 200 {
		t.Fatalf("unexpected OCR payload: %+v", payload)
	}
	if len(payload.Items) != 1 || payload.Items[0].Text != "文件" {
		t.Fatalf("unexpected OCR items: %+v", payload.Items)
	}
}

func TestScreenActionToolExecuteClickTextUsesMouseClickWithoutScale(t *testing.T) {
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
					"display_id":   7,
					"scale_x":      2.0,
					"scale_y":      2.0,
					"origin_x":     10,
					"origin_y":     20,
					"region": map[string]any{
						"x": 0, "y": 0, "width": 1, "height": 1,
					},
				}, nil
			case "OCR_IMAGE":
				if params["image_path"] != imagePath {
					t.Fatalf("unexpected OCR_IMAGE image_path: %+v", params)
				}
				return map[string]any{
					"items": []any{
						map[string]any{
							"text":       "Submit",
							"confidence": 0.99,
							"bbox": map[string]any{
								"x": 40, "y": 60, "width": 20, "height": 10,
							},
							"center": map[string]any{"x": 50, "y": 65},
						},
					},
				}, nil
			case "MOUSE_CLICK":
				if _, ok := params["scale_x"]; ok {
					t.Fatalf("mouse click should not receive scale_x: %+v", params)
				}
				if _, ok := params["scale_y"]; ok {
					t.Fatalf("mouse click should not receive scale_y: %+v", params)
				}
				if params["display_id"] != 7 || params["x"] != 50 || params["y"] != 65 {
					t.Fatalf("unexpected mouse click params: %+v", params)
				}
				return map[string]any{"clicked": true}, nil
			default:
				t.Fatalf("unexpected action: %s", action)
				return nil, nil
			}
		},
	})

	output, err := tool.Execute(
		context.Background(),
		json.RawMessage(`{"action":"click_text","params":{"display_id":7,"text":"Submit"}}`),
		"trace-click-text-1",
	)
	if err != nil {
		t.Fatalf("execute returned error: %v", err)
	}
	if strings.Join(actions, ",") != "SCREEN_CAPTURE,OCR_IMAGE,MOUSE_CLICK" {
		t.Fatalf("unexpected action sequence: %v", actions)
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("decode output: %v", err)
	}
	if payload["action"] != "click_text" || payload["clicked"] != true {
		t.Fatalf("unexpected click_text payload: %+v", payload)
	}
}

func TestScreenActionToolExecuteClickIconUsesAtomicMatcher(t *testing.T) {
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
					"scale_x":      1.25,
					"scale_y":      1.25,
					"origin_x":     30,
					"origin_y":     40,
					"region": map[string]any{
						"x": 3, "y": 4, "width": 1, "height": 1,
					},
				}, nil
			case "TEMPLATE_MATCH_IMAGE":
				if params["image_path"] != imagePath {
					t.Fatalf("unexpected TEMPLATE_MATCH_IMAGE image_path: %+v", params)
				}
				if params["origin_x"] != 30 || params["origin_y"] != 40 {
					t.Fatalf("unexpected TEMPLATE_MATCH_IMAGE params: %+v", params)
				}
				return map[string]any{
					"matches": []any{
						map[string]any{
							"score": 0.93,
							"bbox": map[string]any{
								"x": 80, "y": 90, "width": 12, "height": 12,
							},
							"center": map[string]any{"x": 86, "y": 96},
							"scale":  1.0,
						},
					},
				}, nil
			case "MOUSE_CLICK":
				if _, ok := params["scale_x"]; ok {
					t.Fatalf("mouse click should not receive scale_x: %+v", params)
				}
				if _, ok := params["scale_y"]; ok {
					t.Fatalf("mouse click should not receive scale_y: %+v", params)
				}
				if params["display_id"] != 9 || params["x"] != 86 || params["y"] != 96 {
					t.Fatalf("unexpected mouse click params: %+v", params)
				}
				return map[string]any{"clicked": true}, nil
			default:
				t.Fatalf("unexpected action: %s", action)
				return nil, nil
			}
		},
	})

	output, err := tool.Execute(
		context.Background(),
		json.RawMessage(`{"action":"click_icon","params":{"display_id":9,"template_path":"/tmp/icon.png"}}`),
		"trace-click-icon-1",
	)
	if err != nil {
		t.Fatalf("execute returned error: %v", err)
	}
	if strings.Join(actions, ",") != "SCREEN_CAPTURE,TEMPLATE_MATCH_IMAGE,MOUSE_CLICK" {
		t.Fatalf("unexpected action sequence: %v", actions)
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("decode output: %v", err)
	}
	if payload["action"] != "click_icon" || payload["clicked"] != true {
		t.Fatalf("unexpected click_icon payload: %+v", payload)
	}
}

func writeScreenActionTestPNG(t *testing.T) string {
	t.Helper()
	raw := "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR4nGMAAQAABQABDQottAAAAABJRU5ErkJggg=="
	buf, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		t.Fatalf("decode png fixture: %v", err)
	}
	path := filepath.Join(t.TempDir(), "screen.png")
	if err := os.WriteFile(path, buf, 0o600); err != nil {
		t.Fatalf("write png fixture: %v", err)
	}
	return path
}
