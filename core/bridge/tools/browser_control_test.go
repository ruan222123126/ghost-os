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
	"time"

	"ghost-os/bridge/session"
)

const tinyPNGBase64 = "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR4nGMAAQAABQABDQottAAAAABJRU5ErkJggg=="

func TestBrowserControlConnectWithWSEndpoint(t *testing.T) {
	mockBrowserWSEndpointFetcher(
		t,
		func(base string, _ time.Duration) (string, error) {
			return strings.Replace(base, "http://", "ws://", 1) + "/devtools/browser/test", nil
		},
	)
	tool := NewBrowserControlTool(nil).(*BrowserControlTool)

	output, err := tool.Execute(
		context.Background(),
		json.RawMessage(`{"action":"connect","params":{"endpoint":"ws://127.0.0.1:9222/devtools/browser/test","session_id":"session-a"}}`),
		"trace-connect",
	)
	if err != nil {
		t.Fatalf("execute returned error: %v", err)
	}
	defer closeBrowserSessionInScope(t, tool, browserSessionGlobalScope, "session-a")

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
	if result["debug_port"] != float64(9222) {
		t.Fatalf("unexpected debug_port: %+v", result)
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
	mockBrowserWSEndpointFetcher(
		t,
		func(base string, _ time.Duration) (string, error) {
			return "", fmt.Errorf("fetch %s/json/version failed: dial tcp 127.0.0.1:9222: connect: connection refused", base)
		},
	)
	var callCount int
	tool := NewBrowserControlTool(mockExecutionClient{
		callFunc: func(_ context.Context, action string, params map[string]any, traceID string) (map[string]any, error) {
			callCount++
			if action != browserLaunchAction {
				t.Fatalf("unexpected action: got %q want %q", action, browserLaunchAction)
			}
			if traceID != "trace-launch-not-found" {
				t.Fatalf("unexpected trace id: got %q want %q", traceID, "trace-launch-not-found")
			}
			if params["command"] != "google-chrome --headless --remote-debugging-port=9222 &" {
				t.Fatalf("unexpected command payload: %+v", params)
			}
			if params["display_mode"] != "background" {
				t.Fatalf("unexpected display_mode payload: %+v", params)
			}
			return nil, fmt.Errorf("command failed: bash: line 1: google-chrome: command not found")
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
	if !strings.Contains(err.Error(), "execution BROWSER_LAUNCH failed") {
		t.Fatalf("unexpected error: %v", err)
	}
	if callCount != 1 {
		t.Fatalf("expected one launch command call, got %d", callCount)
	}
}

func TestBrowserLaunchPayloadUsesExplicitCommand(t *testing.T) {
	payload, err := browserLaunchPayload(
		map[string]any{
			"command":    "google-chrome --headless --remote-debugging-port=9222 &",
			"debug_port": 9222,
		},
		"http://127.0.0.1:9222",
	)
	if err != nil {
		t.Fatalf("browserLaunchPayload returned error: %v", err)
	}
	if payload["command"] != "google-chrome --headless --remote-debugging-port=9222 &" {
		t.Fatalf("unexpected command payload: %+v", payload)
	}
	if payload["debug_port"] != 9222 {
		t.Fatalf("unexpected debug_port payload: %+v", payload)
	}
	if payload["display_mode"] != "background" {
		t.Fatalf("unexpected display_mode payload: %+v", payload)
	}
}

func TestBrowserLaunchPayloadUsesForegroundDisplayMode(t *testing.T) {
	payload, err := browserLaunchPayload(
		map[string]any{
			"debug_port":   9222,
			"display_mode": "foreground",
		},
		"http://127.0.0.1:9222",
	)
	if err != nil {
		t.Fatalf("browserLaunchPayload returned error: %v", err)
	}
	if payload["display_mode"] != "foreground" {
		t.Fatalf("unexpected display_mode payload: %+v", payload)
	}
}

func TestBrowserLaunchPayloadRejectsInvalidDisplayMode(t *testing.T) {
	_, err := browserLaunchPayload(
		map[string]any{
			"display_mode": "invalid",
		},
		"http://127.0.0.1:9222",
	)
	if err == nil {
		t.Fatal("expected display_mode validation error")
	}
	if !strings.Contains(err.Error(), "display_mode must be one of") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBrowserLaunchPayloadRequiresEndpointPortWhenAutoLaunching(t *testing.T) {
	_, err := browserLaunchPayload(map[string]any{}, "http://127.0.0.1")
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
	tool.storeSession(browserSessionGlobalScope, &browserSession{id: "session-a"})
	defer closeBrowserSessionInScope(t, tool, browserSessionGlobalScope, "session-a")

	session, err := tool.sessionFromParams(context.Background(), map[string]any{})
	if err != nil {
		t.Fatalf("sessionFromParams returned error: %v", err)
	}
	if session.id != "session-a" {
		t.Fatalf("unexpected session id: got %q want %q", session.id, "session-a")
	}
}

func TestBrowserControlSessionFromParamsUsesActiveSessionWhenMultipleSessionsExist(t *testing.T) {
	tool := NewBrowserControlTool(nil).(*BrowserControlTool)
	tool.storeSession(browserSessionGlobalScope, &browserSession{id: "session-a"})
	tool.storeSession(browserSessionGlobalScope, &browserSession{id: "session-b"})
	defer closeBrowserSessionInScope(t, tool, browserSessionGlobalScope, "session-a")
	defer closeBrowserSessionInScope(t, tool, browserSessionGlobalScope, "session-b")

	session, err := tool.sessionFromParams(context.Background(), map[string]any{})
	if err != nil {
		t.Fatalf("sessionFromParams returned error: %v", err)
	}
	if session.id != "session-b" {
		t.Fatalf("expected active session-b, got %q", session.id)
	}
}

func TestBrowserControlSessionIDCannotCrossConversationScope(t *testing.T) {
	tool := NewBrowserControlTool(nil).(*BrowserControlTool)
	scopeA := "session-a"
	scopeB := "session-b"
	tool.storeSession(scopeA, &browserSession{id: "browser-a"})
	defer closeBrowserSessionInScope(t, tool, scopeA, "browser-a")

	ctx := WithSession(context.Background(), &session.Session{ID: scopeB})
	_, err := tool.sessionFromParams(ctx, map[string]any{"session_id": "browser-a"})
	if err == nil {
		t.Fatal("expected scope mismatch error")
	}
	if !strings.Contains(err.Error(), "does not belong to current conversation session") {
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

func closeBrowserSessionInScope(
	t *testing.T,
	tool *BrowserControlTool,
	scopeKey string,
	sessionID string,
) {
	t.Helper()
	session, err := tool.popSession(scopeKey, sessionID)
	if err != nil {
		return
	}
	if session != nil {
		session.close()
	}
}
