package agent

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestRunAbortsAfterTwoConsecutiveBrowserSessionInvalidFailures(t *testing.T) {
	browserTool := newErrorTool(
		"browser_control",
		errors.New(`{"code":"browser_session_invalid","reason":"target_closed","message":"target closed"}`),
	)
	completer := newFakeCompleter(
		newToolCallsResponse(newToolCall("call-browser-1", "browser_control", `{"action":"info"}`)),
		newToolCallsResponse(newToolCall("call-browser-2", "browser_control", `{"action":"info"}`)),
		newStopResponse("should-not-reach"),
	)
	agent := newTestAgent(completer, newFakeToolCatalog(browserTool), 5)

	_, err := agent.Run(context.Background(), "open browser")
	if err == nil {
		t.Fatal("expected repeated browser session invalid error")
	}
	if !strings.Contains(err.Error(), `"code":"browser_session_invalid_repeated"`) {
		t.Fatalf("unexpected error: %v", err)
	}
	if browserTool.callCount != 2 {
		t.Fatalf("expected browser_control to run twice, got %d", browserTool.callCount)
	}
	if len(completer.requests) != 2 {
		t.Fatalf("expected completion loop to stop at turn 2, got %d requests", len(completer.requests))
	}

	var payload map[string]any
	if decodeErr := json.Unmarshal([]byte(err.Error()), &payload); decodeErr != nil {
		t.Fatalf("expected structured error payload, decode failed: %v", decodeErr)
	}
	if payload["tool"] != "browser_control" {
		t.Fatalf("unexpected structured error payload: %+v", payload)
	}
	if payload["consecutive_failures"] != float64(2) {
		t.Fatalf("unexpected consecutive_failures: %+v", payload)
	}
}

func TestRunBrowserSessionInvalidCounterResetsAfterHealthyToolTurn(t *testing.T) {
	browserTool := newErrorTool(
		"browser_control",
		errors.New(`{"code":"browser_session_invalid","reason":"websocket_closed","message":"websocket is closed"}`),
	)
	echoTool := newStaticTool("echo", `{"status":"ok"}`)
	completer := newFakeCompleter(
		newToolCallsResponse(newToolCall("call-browser-1", "browser_control", `{"action":"info"}`)),
		newToolCallsResponse(newToolCall("call-echo-1", "echo", `{"input":"ok"}`)),
		newToolCallsResponse(newToolCall("call-browser-2", "browser_control", `{"action":"info"}`)),
		newStopResponse("done"),
	)
	agent := newTestAgent(completer, newFakeToolCatalog(browserTool, echoTool), 6)

	got, err := agent.Run(context.Background(), "keep going")
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if got != "done" {
		t.Fatalf("unexpected output: got %q want %q", got, "done")
	}
	if browserTool.callCount != 2 {
		t.Fatalf("expected browser_control to run twice, got %d", browserTool.callCount)
	}
	if echoTool.callCount != 1 {
		t.Fatalf("expected echo to run once, got %d", echoTool.callCount)
	}
}
