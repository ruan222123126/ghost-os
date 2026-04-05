package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestBrowserControlLaunchReusesExistingEndpointWithoutExecution(t *testing.T) {
	mockBrowserWSEndpointFetcher(
		t,
		func(base string, _ time.Duration) (string, error) {
			if base != "http://127.0.0.1:9222" {
				t.Fatalf("unexpected probe base: %q", base)
			}
			return "ws://127.0.0.1:9222/devtools/browser/existing", nil
		},
	)
	callCount := 0
	tool := NewBrowserControlTool(mockExecutionClient{
		callFunc: func(_ context.Context, action string, _ map[string]any, _ string) (map[string]any, error) {
			callCount++
			t.Fatalf("launch should reuse existing endpoint, got execution action %q", action)
			return nil, nil
		},
	}).(*BrowserControlTool)

	output, err := tool.Execute(
		context.Background(),
		json.RawMessage(`{"action":"launch","params":{"endpoint":"http://127.0.0.1:9222","session_id":"session-reuse"}}`),
		"trace-launch-reuse",
	)
	if err != nil {
		t.Fatalf("execute returned error: %v", err)
	}
	defer closeBrowserSessionInScope(t, tool, browserSessionGlobalScope, "session-reuse")
	if callCount != 0 {
		t.Fatalf("expected no execution launch call, got %d", callCount)
	}

	var result map[string]any
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("decode output: %v", err)
	}
	if result["reused_existing"] != true {
		t.Fatalf("expected reused_existing=true, got %+v", result)
	}
	if result["session_id"] != "session-reuse" {
		t.Fatalf("unexpected session_id: %+v", result)
	}
	if result["ws_endpoint"] != "ws://127.0.0.1:9222/devtools/browser/existing" {
		t.Fatalf("unexpected ws endpoint: %+v", result)
	}
	if result["debug_port"] != float64(9222) {
		t.Fatalf("unexpected debug port: %+v", result)
	}
}

func TestBrowserControlLaunchProbeFailureStillFallsBackToExecutionLaunch(t *testing.T) {
	mockBrowserWSEndpointFetcher(
		t,
		func(base string, _ time.Duration) (string, error) {
			return "", fmt.Errorf("fetch %s/json/version failed: dial tcp 127.0.0.1:9333: connect: connection refused", base)
		},
	)
	callCount := 0
	tool := NewBrowserControlTool(mockExecutionClient{
		callFunc: func(_ context.Context, action string, _ map[string]any, _ string) (map[string]any, error) {
			callCount++
			if action != browserLaunchAction {
				t.Fatalf("unexpected action: %q", action)
			}
			return nil, nil
		},
	})

	_, err := tool.Execute(
		context.Background(),
		json.RawMessage(`{"action":"launch","params":{"endpoint":"http://127.0.0.1:9333","command":"chrome --remote-debugging-port=9333","wait_timeout_ms":1}}`),
		"trace-launch-fallback",
	)
	if err == nil {
		t.Fatal("expected launch to fail on endpoint discovery")
	}
	if !strings.Contains(err.Error(), "browser cdp preflight failed") {
		t.Fatalf("unexpected error: %v", err)
	}
	if callCount != 1 {
		t.Fatalf("expected exactly one execution launch call, got %d", callCount)
	}
}
