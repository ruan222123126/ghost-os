package tools

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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

func TestFetchWebSocketURLDiscoversWSEndpoint(t *testing.T) {
	client := &http.Client{
		Transport: browserControlRoundTripper(func(req *http.Request) (*http.Response, error) {
			if req.URL.String() != "http://browser.test/json/version" {
				t.Fatalf("unexpected discovery url: %q", req.URL.String())
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body: io.NopCloser(strings.NewReader(
					`{"webSocketDebuggerUrl":"ws://127.0.0.1:9222/devtools/browser/discovered"}`,
				)),
			}, nil
		}),
	}

	wsEndpoint, err := fetchWebSocketURLWithClient("http://browser.test", client)
	if err != nil {
		t.Fatalf("fetchWebSocketURLWithClient returned error: %v", err)
	}
	if wsEndpoint != "ws://127.0.0.1:9222/devtools/browser/discovered" {
		t.Fatalf("unexpected ws endpoint: %q", wsEndpoint)
	}
}

func TestLaunchEndpointDefaultsToLocalDebugPort(t *testing.T) {
	endpoint, err := launchEndpoint(map[string]any{})
	if err != nil {
		t.Fatalf("launchEndpoint returned error: %v", err)
	}
	if endpoint != "http://127.0.0.1:9222" {
		t.Fatalf("unexpected endpoint: got %q want %q", endpoint, "http://127.0.0.1:9222")
	}
}

func TestBrowserControlLaunchCommandNotFoundReturnsEarly(t *testing.T) {
	var callCount int
	tool := NewBrowserControlTool(mockExecutionClient{
		callFunc: func(_ context.Context, action string, params map[string]any, traceID string) (map[string]any, error) {
			callCount++
			if action != "BASH_EXEC" {
				t.Fatalf("unexpected action: got %q want %q", action, "BASH_EXEC")
			}
			if traceID != "trace-launch-not-found" {
				t.Fatalf("unexpected trace id: got %q want %q", traceID, "trace-launch-not-found")
			}
			if params["command"] != "google-chrome --headless --remote-debugging-port=9222 &" {
				t.Fatalf("unexpected command payload: %+v", params)
			}
			return map[string]any{
				"stdout": "",
				"stderr": "bash: line 1: google-chrome: command not found",
			}, nil
		},
	})

	_, err := tool.Execute(
		context.Background(),
		json.RawMessage(`{"action":"launch","params":{"command":"google-chrome --headless --remote-debugging-port=9222 &","endpoint":"http://127.0.0.1:9222"}}`),
		"trace-launch-not-found",
	)
	if err == nil {
		t.Fatal("expected missing executable error")
	}
	if !strings.Contains(err.Error(), "launch command references an unavailable executable") {
		t.Fatalf("unexpected error: %v", err)
	}
	if callCount != 1 {
		t.Fatalf("expected one launch command call, got %d", callCount)
	}
}

func TestBrowserLaunchCommandBuildsAutoLaunchScript(t *testing.T) {
	command, err := browserLaunchCommand(map[string]any{"debug_port": 9333}, "")
	if err != nil {
		t.Fatalf("browserLaunchCommand returned error: %v", err)
	}
	for _, snippet := range []string{
		"google-chrome",
		"chromium",
		"--remote-debugging-port=9333",
		"--headless",
		"--no-sandbox",
	} {
		if !strings.Contains(command, snippet) {
			t.Fatalf("auto launch command missing %q: %q", snippet, command)
		}
	}
}

func TestBrowserLaunchCommandRequiresEndpointPortWhenAutoLaunching(t *testing.T) {
	_, err := browserLaunchCommand(map[string]any{}, "http://127.0.0.1")
	if err == nil {
		t.Fatal("expected auto launch port parse error")
	}
	if !strings.Contains(err.Error(), "must include an explicit port") {
		t.Fatalf("unexpected error: %v", err)
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

func TestBrowserControlSessionFromParamsUsesSingleSessionWhenIDOmitted(t *testing.T) {
	tool := NewBrowserControlTool(nil).(*BrowserControlTool)
	tool.storeSession(&browserSession{id: "session-a"})

	session, err := tool.sessionFromParams(map[string]any{})
	if err != nil {
		t.Fatalf("sessionFromParams returned error: %v", err)
	}
	if session.id != "session-a" {
		t.Fatalf("unexpected session id: got %q want %q", session.id, "session-a")
	}
}

func TestBrowserControlSessionFromParamsErrorsWhenMultipleSessionsAndIDOmitted(t *testing.T) {
	tool := NewBrowserControlTool(nil).(*BrowserControlTool)
	tool.storeSession(&browserSession{id: "session-a"})
	tool.storeSession(&browserSession{id: "session-b"})

	_, err := tool.sessionFromParams(map[string]any{})
	if err == nil {
		t.Fatal("expected error when session_id is omitted and multiple sessions exist")
	}
	if !strings.Contains(err.Error(), "session_id is required when multiple browser sessions exist") {
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

type browserControlRoundTripper func(*http.Request) (*http.Response, error)

func (rt browserControlRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return rt(req)
}
