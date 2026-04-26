package screen

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

const captureCacheTTLMillis = 5_000

func TestScreenActionToolExecuteOCRScanReusesCapture(t *testing.T) {
	imagePath := writeScreenActionTestPNG(t)
	tool, actions := newCaptureReuseTestTool(t, imagePath)

	args := `{"action":"ocr_scan","params":{"display_id":2,"cache_ttl_ms":5000}}`
	executeScreenActionRaw(t, tool, args, "trace-ocr-cache-1")
	executeScreenActionRaw(t, tool, args, "trace-ocr-cache-2")

	assertActionSequence(t, actions, "SCREEN_CAPTURE,OCR_IMAGE,OCR_IMAGE")
}

func TestScreenActionToolExecuteFindIconReusesCapture(t *testing.T) {
	imagePath := writeScreenActionTestPNG(t)
	tool, actions := newCaptureReuseTestTool(t, imagePath)

	args := `{"action":"find_icon","params":{"display_id":3,"template_path":"/tmp/icon.png","cache_ttl_ms":5000}}`
	executeScreenActionRaw(t, tool, args, "trace-icon-cache-1")
	executeScreenActionRaw(t, tool, args, "trace-icon-cache-2")

	assertActionSequence(t, actions, "SCREEN_CAPTURE,TEMPLATE_MATCH_IMAGE,TEMPLATE_MATCH_IMAGE")
}

func TestScreenActionToolCaptureScreenFreshAlwaysCallsExecution(t *testing.T) {
	imagePath := writeScreenActionTestPNG(t)
	var captureCalls int
	tool := NewScreenActionTool(mockExecutionClient{
		callFunc: func(_ context.Context, action string, _ map[string]any, _ string) (map[string]any, error) {
			if action != "SCREEN_CAPTURE" {
				t.Fatalf("unexpected action: %s", action)
			}
			captureCalls++
			return testScreenCapturePayload(imagePath), nil
		},
	}).(*ScreenActionTool)

	params := map[string]any{
		"display_id":   9,
		"reuse_cache":  true,
		"cache_ttl_ms": captureCacheTTLMillis,
	}
	if _, err := tool.captureScreenFresh(context.Background(), params, "trace-fresh-1"); err != nil {
		t.Fatalf("captureScreenFresh first call failed: %v", err)
	}
	if _, err := tool.captureScreenFresh(context.Background(), params, "trace-fresh-2"); err != nil {
		t.Fatalf("captureScreenFresh second call failed: %v", err)
	}
	if captureCalls != 2 {
		t.Fatalf("expected 2 SCREEN_CAPTURE calls, got %d", captureCalls)
	}
}

func newCaptureReuseTestTool(t *testing.T, imagePath string) (Tool, *[]string) {
	t.Helper()
	actions := &[]string{}
	tool := NewScreenActionTool(mockExecutionClient{
		callFunc: func(_ context.Context, action string, params map[string]any, _ string) (map[string]any, error) {
			*actions = append(*actions, action)
			switch action {
			case "SCREEN_CAPTURE":
				return testScreenCapturePayload(imagePath), nil
			case "OCR_IMAGE":
				if params["image_path"] != imagePath {
					t.Fatalf("unexpected OCR_IMAGE image_path: %+v", params)
				}
				return map[string]any{"items": []any{}}, nil
			case "TEMPLATE_MATCH_IMAGE":
				if params["image_path"] != imagePath {
					t.Fatalf("unexpected TEMPLATE_MATCH_IMAGE image_path: %+v", params)
				}
				return map[string]any{"matches": []any{}}, nil
			default:
				t.Fatalf("unexpected action: %s", action)
				return nil, nil
			}
		},
	})
	return tool, actions
}

func executeScreenActionRaw(t *testing.T, tool Tool, args string, traceID string) string {
	t.Helper()
	output, err := tool.Execute(context.Background(), json.RawMessage(args), traceID)
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}
	return output
}

func assertActionSequence(t *testing.T, actions *[]string, want string) {
	t.Helper()
	if got := strings.Join(*actions, ","); got != want {
		t.Fatalf("unexpected action sequence: got %q want %q", got, want)
	}
}

func testScreenCapturePayload(imagePath string) map[string]any {
	return map[string]any{
		"image_path":   imagePath,
		"image_width":  1,
		"image_height": 1,
		"display_id":   1,
		"scale_x":      1.0,
		"scale_y":      1.0,
		"origin_x":     0,
		"origin_y":     0,
		"region": map[string]any{
			"x": 0, "y": 0, "width": 1, "height": 1,
		},
	}
}
