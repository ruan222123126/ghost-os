package screen

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ghost-os/bridge/session"
	"ghost-os/bridge/tools/internal/tooljson"
)

func TestScreenActionToolExecuteScreenshotPersistsImage(t *testing.T) {
	imageBase64 := "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR4nGMAAQAABQABDQottAAAAABJRU5ErkJggg=="
	imageBytes, err := base64.StdEncoding.DecodeString(imageBase64)
	if err != nil {
		t.Fatalf("decode test png: %v", err)
	}
	tmpDir := t.TempDir()
	t.Setenv("GHOST_SCREENSHOTS_PATH", tmpDir)
	sourcePath := filepath.Join(tmpDir, "native-shot.png")
	if err := os.WriteFile(sourcePath, imageBytes, 0o600); err != nil {
		t.Fatalf("write source screenshot: %v", err)
	}

	var capturedParams map[string]any
	tool := NewScreenActionTool(mockExecutionClient{
		callFunc: func(_ context.Context, action string, params map[string]any, traceID string) (map[string]any, error) {
			if action != "SCREEN_CAPTURE" {
				t.Fatalf("unexpected action: got %q want %q", action, "SCREEN_CAPTURE")
			}
			if traceID != "trace-shot-1" {
				t.Fatalf("unexpected trace id: got %q want %q", traceID, "trace-shot-1")
			}
			capturedParams = params
			return map[string]any{
				"image_path":   sourcePath,
				"image_width":  1,
				"image_height": 1,
				"display_id":   2,
			}, nil
		},
	})

	ctx := WithToolCallID(
		WithSession(context.Background(), &session.Session{ID: "session-1"}),
		"call-shot-1",
	)
	output, err := tool.Execute(ctx, json.RawMessage(`{"action":"screenshot","params":{"display_id":2}}`), "trace-shot-1")
	if err != nil {
		t.Fatalf("execute returned error: %v", err)
	}
	if capturedParams["display_id"] != 2 {
		t.Fatalf("unexpected display_id param: %+v", capturedParams)
	}

	var result screenActionResult
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("decode output: %v", err)
	}
	if result.Action != "screenshot" {
		t.Fatalf("unexpected action: got %q", result.Action)
	}
	if result.DisplayID != 2 {
		t.Fatalf("unexpected display id: got %d", result.DisplayID)
	}
	if result.Artifact == nil {
		t.Fatal("expected artifact payload")
	}

	sessionDir := filepath.Join(tmpDir, "session-1") + string(filepath.Separator)
	if !strings.HasPrefix(result.Artifact.VisionPath, sessionDir) {
		t.Fatalf("unexpected vision path: %q", result.Artifact.VisionPath)
	}
	if !strings.HasSuffix(result.Artifact.VisionPath, ".png") {
		t.Fatalf("expected png path, got %q", result.Artifact.VisionPath)
	}
	if result.Artifact.VisionMime != "image/png" {
		t.Fatalf("unexpected mime type: %q", result.Artifact.VisionMime)
	}
	if result.Artifact.Width != 1 || result.Artifact.Height != 1 {
		t.Fatalf("unexpected dimensions: %dx%d", result.Artifact.Width, result.Artifact.Height)
	}
	if result.Artifact.VisionBytes != len(imageBytes) {
		t.Fatalf("unexpected byte count: got %d want %d", result.Artifact.VisionBytes, len(imageBytes))
	}

	written, err := os.ReadFile(result.Artifact.VisionPath)
	if err != nil {
		t.Fatalf("read written screenshot: %v", err)
	}
	if string(written) != string(imageBytes) {
		t.Fatalf("written screenshot content mismatch")
	}

	expectedHash := sha256.Sum256(imageBytes)
	if result.Artifact.SHA256 != strings.ToLower(fmt.Sprintf("%x", expectedHash[:])) {
		t.Fatalf("unexpected sha256: got %q", result.Artifact.SHA256)
	}
}

func TestScreenActionToolInterpretResultBuildsImageContent(t *testing.T) {
	result := screenActionResult{
		Action: "screenshot",
		Artifact: &screenActionArtifact{
			Type:        "image",
			VisionPath:  "/tmp/screenshot.png",
			VisionMime:  "image/png",
			Width:       120,
			Height:      80,
			SHA256:      "abc123",
			VisionBytes: 2048,
		},
	}
	output, err := tooljson.Encode(result)
	if err != nil {
		t.Fatalf("encode result: %v", err)
	}

	tool := NewScreenActionTool(nil)
	interpreter, ok := tool.(ResultInterpreter)
	if !ok {
		t.Fatal("screen_action should implement ResultInterpreter")
	}
	meta := interpreter.InterpretResult(output)
	if len(meta.Content) != 1 {
		t.Fatalf("unexpected content length: %d", len(meta.Content))
	}
	part := meta.Content[0]
	if part.Image == nil {
		t.Fatal("expected image content")
	}
	if part.Image.Path != "/tmp/screenshot.png" {
		t.Fatalf("unexpected image path: %q", part.Image.Path)
	}
	if part.Image.MimeType != "image/png" {
		t.Fatalf("unexpected image mime: %q", part.Image.MimeType)
	}
	if part.Image.Width != 120 || part.Image.Height != 80 {
		t.Fatalf("unexpected image dimensions: %dx%d", part.Image.Width, part.Image.Height)
	}
	if part.Image.SHA256 != "abc123" {
		t.Fatalf("unexpected sha256: %q", part.Image.SHA256)
	}
	if part.Image.Bytes != 2048 {
		t.Fatalf("unexpected byte count: %d", part.Image.Bytes)
	}
}

func TestScreenActionToolExecuteScreenshotRejectsDeprecatedCapturePayload(t *testing.T) {
	tool := NewScreenActionTool(mockExecutionClient{
		callFunc: func(_ context.Context, action string, _ map[string]any, _ string) (map[string]any, error) {
			if action != "SCREEN_CAPTURE" {
				t.Fatalf("unexpected action: got %q", action)
			}
			return map[string]any{
				"image_base64": "ZmFrZS1wbmc=",
				"image_width":  100,
				"image_height": 80,
				"display_id":   67,
			}, nil
		},
	})

	_, err := tool.Execute(context.Background(), json.RawMessage(`{"action":"screenshot"}`), "trace-shot-legacy")
	if err == nil {
		t.Fatal("expected error but got nil")
	}
	if !strings.Contains(err.Error(), "deprecated image_base64 payload") {
		t.Fatalf("unexpected error: %v", err)
	}
}
