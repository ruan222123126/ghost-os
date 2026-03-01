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

func TestBrowserActionToolMapsActionAndReturnsJSON(t *testing.T) {
	var gotAction string
	tool := NewBrowserActionTool(mockExecutionClient{
		callFunc: func(_ context.Context, action string, params map[string]any, traceID string) (map[string]any, error) {
			gotAction = action
			if traceID != "trace-browser-1" {
				t.Fatalf("unexpected trace id: got %q want %q", traceID, "trace-browser-1")
			}
			if params["selector"] != "#submit" {
				t.Fatalf("unexpected params selector: %v", params["selector"])
			}
			return map[string]any{"ok": true}, nil
		},
	})

	output, err := tool.Execute(context.Background(), json.RawMessage(`{"action":"query","params":{"selector":"#submit"}}`), "trace-browser-1")
	if err != nil {
		t.Fatalf("execute returned error: %v", err)
	}
	if gotAction != "BROWSER_QUERY" {
		t.Fatalf("unexpected native action: got %q want %q", gotAction, "BROWSER_QUERY")
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("decode output: %v", err)
	}
	if payload["ok"] != true {
		t.Fatalf("unexpected payload: %+v", payload)
	}
}

func TestBrowserActionToolRejectsUnknownAction(t *testing.T) {
	tool := NewBrowserActionTool(mockExecutionClient{
		callFunc: func(_ context.Context, _ string, _ map[string]any, _ string) (map[string]any, error) {
			return map[string]any{}, nil
		},
	})

	_, err := tool.Execute(context.Background(), json.RawMessage(`{"action":"scroll"}`), "trace-browser-2")
	if err == nil {
		t.Fatal("expected unsupported action error")
	}
}

func TestBrowserActionScreenshotReturnsArtifactReference(t *testing.T) {
	tempDir := t.TempDir()
	if err := os.Setenv("GHOST_ARTIFACTS_PATH", tempDir); err != nil {
		t.Fatalf("set GHOST_ARTIFACTS_PATH: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Unsetenv("GHOST_ARTIFACTS_PATH")
	})

	encodedImage := base64.StdEncoding.EncodeToString([]byte("fake-png-bytes"))
	tool := NewBrowserActionTool(mockExecutionClient{
		callFunc: func(_ context.Context, action string, _ map[string]any, _ string) (map[string]any, error) {
			if action != "SCREEN_SHOT" {
				t.Fatalf("unexpected action: got %q want %q", action, "SCREEN_SHOT")
			}
			return map[string]any{
				"image_base64": encodedImage,
				"width":        800,
				"height":       600,
				"display_id":   1,
			}, nil
		},
	})

	output, err := tool.Execute(context.Background(), json.RawMessage(`{"action":"screenshot"}`), "trace-shot-1")
	if err != nil {
		t.Fatalf("execute returned error: %v", err)
	}
	if strings.Contains(output, "image_base64") {
		t.Fatalf("screenshot output should not contain image_base64, got: %s", output)
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(output), &payload); err != nil {
		t.Fatalf("decode output: %v", err)
	}
	artifact, ok := payload["artifact"].(map[string]any)
	if !ok {
		t.Fatalf("artifact missing in payload: %+v", payload)
	}

	path, _ := artifact["path"].(string)
	visionPath, _ := artifact["vision_path"].(string)
	if strings.TrimSpace(path) == "" || strings.TrimSpace(visionPath) == "" {
		t.Fatalf("unexpected artifact paths: path=%q vision_path=%q", path, visionPath)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected artifact file at %q: %v", path, err)
	}
	if _, err := os.Stat(visionPath); err != nil {
		t.Fatalf("expected vision artifact file at %q: %v", visionPath, err)
	}
	if !strings.HasPrefix(path, filepath.Join(tempDir, "screenshots")) {
		t.Fatalf("artifact path should be in configured dir, got %q", path)
	}
}

func TestBrowserActionToolInterpretResultBuildsImageContent(t *testing.T) {
	tool := NewBrowserActionTool(mockExecutionClient{
		callFunc: func(_ context.Context, _ string, _ map[string]any, _ string) (map[string]any, error) {
			return map[string]any{}, nil
		},
	})
	interpreter, ok := tool.(ResultInterpreter)
	if !ok {
		t.Fatal("browser_action should implement ResultInterpreter")
	}

	meta := interpreter.InterpretResult(`{
		"action":"screenshot",
		"artifact":{
			"type":"image",
			"path":"/tmp/a.png",
			"vision_path":"/tmp/a.vision.jpg",
			"mime_type":"image/png",
			"vision_mime_type":"image/jpeg",
			"width":1280,
			"height":720,
			"sha256":"abc123",
			"bytes":1024,
			"vision_bytes":640
		}
	}`)
	if len(meta.Content) != 1 {
		t.Fatalf("unexpected content part count: got %d want %d", len(meta.Content), 1)
	}
	if meta.Content[0].Image == nil {
		t.Fatal("expected image content")
	}
	if meta.Content[0].Image.Path != "/tmp/a.vision.jpg" {
		t.Fatalf("unexpected image path: got %q want %q", meta.Content[0].Image.Path, "/tmp/a.vision.jpg")
	}
	if meta.Content[0].Image.MimeType != "image/jpeg" {
		t.Fatalf("unexpected image mime type: got %q want %q", meta.Content[0].Image.MimeType, "image/jpeg")
	}
}
