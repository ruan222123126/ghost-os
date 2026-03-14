package tools

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const tinyPNGBase64 = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR4nGMAAQAABQABDQottAAAAABJRU5ErkJggg=="

func TestBrowserControlConnectWithWSEndpoint(t *testing.T) {
	tool := NewBrowserControlTool(nil).(*BrowserControlTool)

	output, err := tool.Execute(
		context.Background(),
		json.RawMessage(`{"action":"connect","params":{"endpoint":"ws://127.0.0.1:9222/devtools/browser/test","session_id":"session-a"}}`),
		"trace-connect",
	)
	if err != nil {
		t.Fatalf("execute returned error: %v", err)
	}
	defer tool.popSession("session-a").close()

	var result map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("decode output: %v", err)
	}
	if result["session_id"] != "session-a" {
		t.Fatalf("unexpected session_id: %+v", result)
	}
	if result["ws_endpoint"] != "ws://127.0.0.1:9222/devtools/browser/test" {
		t.Fatalf("unexpected ws endpoint: %+v", result)
	}
}

func TestBrowserControlConnectDiscoversWSEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/json/version" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(`{"webSocketDebuggerUrl":"ws://127.0.0.1:9222/devtools/browser/discovered"}`))
	}))
	defer server.Close()

	tool := NewBrowserControlTool(nil).(*BrowserControlTool)
	output, err := tool.Execute(
		context.Background(),
		json.RawMessage(fmt.Sprintf(`{"action":"connect","params":{"endpoint":"%s"}}`, server.URL)),
		"trace-discover",
	)
	if err != nil {
		t.Fatalf("execute returned error: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("decode output: %v", err)
	}
	sessionID, _ := result["session_id"].(string)
	if sessionID == "" {
		t.Fatalf("expected generated session_id: %+v", result)
	}
	if result["ws_endpoint"] != "ws://127.0.0.1:9222/devtools/browser/discovered" {
		t.Fatalf("unexpected ws endpoint: %+v", result)
	}

	session := tool.popSession(sessionID)
	if session == nil {
		t.Fatalf("expected session to be stored for %q", sessionID)
	}
	session.close()
}

func TestBrowserControlLaunchRequiresEndpointOrDebugPort(t *testing.T) {
	var callCount int
	tool := NewBrowserControlTool(mockExecutionClient{
		callFunc: func(_ context.Context, action string, params map[string]any, traceID string) (map[string]any, error) {
			callCount++
			if action != "BASH_EXEC" {
				t.Fatalf("unexpected action: got %q want %q", action, "BASH_EXEC")
			}
			if traceID != "trace-launch" {
				t.Fatalf("unexpected trace id: got %q want %q", traceID, "trace-launch")
			}
			if params["command"] != "chromium --headless &" {
				t.Fatalf("unexpected params: %+v", params)
			}
			return map[string]any{"ok": true}, nil
		},
	})

	_, err := tool.Execute(
		context.Background(),
		json.RawMessage(`{"action":"launch","params":{"command":"chromium --headless &"}}`),
		"trace-launch",
	)
	if err == nil {
		t.Fatal("expected error for missing debug_port/endpoint")
	}
	if !strings.Contains(err.Error(), "debug_port or endpoint is required for launch") {
		t.Fatalf("unexpected error: %v", err)
	}
	if callCount != 1 {
		t.Fatalf("expected one launch command call, got %d", callCount)
	}
}

func TestBrowserControlCloseMissingSession(t *testing.T) {
	tool := NewBrowserControlTool(nil)

	_, err := tool.Execute(
		context.Background(),
		json.RawMessage(`{"action":"close","params":{"session_id":"missing"}}`),
		"trace-close",
	)
	if err == nil {
		t.Fatal("expected missing session error")
	}
	if !strings.Contains(err.Error(), `browser session "missing" not found`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWriteBrowserScreenshotPersistsArtifact(t *testing.T) {
	imageBytes, err := base64.StdEncoding.DecodeString(tinyPNGBase64)
	if err != nil {
		t.Fatalf("decode png: %v", err)
	}
	tmpDir := t.TempDir()
	t.Setenv("GHOST_SCREENSHOTS_PATH", tmpDir)

	ctx := WithToolCallID(context.Background(), "tool-browser-shot")
	artifact, err := writeBrowserScreenshot(ctx, "browser-session-1", "trace-browser-shot", imageBytes)
	if err != nil {
		t.Fatalf("write browser screenshot: %v", err)
	}
	if artifact.Type != "image" || artifact.VisionMime != "image/png" {
		t.Fatalf("unexpected artifact metadata: %+v", artifact)
	}
	expectedDir := filepath.Join(tmpDir, "browser", "browser-session-1") + string(filepath.Separator)
	if !strings.HasPrefix(artifact.VisionPath, expectedDir) {
		t.Fatalf("unexpected vision path: %q", artifact.VisionPath)
	}

	written, err := os.ReadFile(artifact.VisionPath)
	if err != nil {
		t.Fatalf("read written screenshot: %v", err)
	}
	if string(written) != string(imageBytes) {
		t.Fatal("written browser screenshot content mismatch")
	}

	expectedHash := sha256.Sum256(imageBytes)
	if artifact.SHA256 != strings.ToLower(fmt.Sprintf("%x", expectedHash[:])) {
		t.Fatalf("unexpected sha256: got %q", artifact.SHA256)
	}
}

func TestNormalizeKeyPress(t *testing.T) {
	tests := map[string]string{
		"Enter":     "\n",
		"tab":       "\t",
		"ESC":       "\u001b",
		"backspace": "\b",
		"space":     " ",
		"A":         "A",
	}

	for input, expected := range tests {
		if actual := normalizeKeyPress(input); actual != expected {
			t.Fatalf("normalizeKeyPress(%q): got %q want %q", input, actual, expected)
		}
	}
}
