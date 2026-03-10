package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"ghost-os/bridge/session"
	"ghost-os/bridge/streaming"
)

func TestHandleAgentStreamMethodNotAllowed(t *testing.T) {
	handler := newTestHandler(t, nil)
	recorder := serveRequest(handler, http.MethodGet, "/api/agent/stream", "", nil)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("unexpected status: got %d want %d", recorder.Code, http.StatusMethodNotAllowed)
	}
	body := decodeResponseBody(t, recorder)
	if body.Status != "error" {
		t.Fatalf("unexpected status field: got %q want %q", body.Status, "error")
	}
}

func TestHandleAgentStreamReturnsHeadersAndEvents(t *testing.T) {
	streamExecutor := func(
		ctx context.Context,
		message string,
		sessionID string,
		traceID string,
		_ *ConfigStore,
		_ *session.Store,
		sink streaming.Sink,
	) (string, string, error) {
		if message != "hello" {
			t.Fatalf("unexpected message: got %q want %q", message, "hello")
		}
		if sessionID != "" {
			t.Fatalf("unexpected session_id: got %q want empty", sessionID)
		}

		if _, err := sink.Emit(ctx, mustAppEvent(t, traceID, "session-stream", 0, "", streaming.EventRunStarted, map[string]any{
			"session_id": "session-stream",
		})); err != nil {
			return "", "", err
		}
		if _, err := sink.Emit(ctx, mustAppEvent(t, traceID, "session-stream", 0, mustAppToolStepID(t, 0, 0), streaming.EventToolCallStarted, map[string]any{
			"tool":         "web_search",
			"tool_call_id": "call-1",
		})); err != nil {
			return "", "", err
		}
		if _, err := sink.Emit(ctx, mustAppEvent(t, traceID, "session-stream", 0, mustAppToolStepID(t, 0, 0), streaming.EventToolCallFinished, map[string]any{
			"tool":         "web_search",
			"tool_call_id": "call-1",
			"status":       "success",
		})); err != nil {
			return "", "", err
		}
		if _, err := sink.Emit(ctx, mustAppEvent(t, traceID, "session-stream", 1, mustAppAssistantStepID(t, 1), streaming.EventMessage, map[string]any{
			"text":       "stream done",
			"session_id": "session-stream",
		})); err != nil {
			return "", "", err
		}
		if _, err := sink.Emit(ctx, mustAppEvent(t, traceID, "session-stream", 1, "", streaming.EventDone, map[string]any{
			"session_id":    "session-stream",
			"session_ended": false,
		})); err != nil {
			return "", "", err
		}
		return "stream done", "session-stream", nil
	}
	handler, _ := newTestHandlerWithStreamExecutor(t, nil, streamExecutor)

	recorder := serveRequest(
		handler,
		http.MethodPost,
		"/api/agent/stream",
		`{"message":"hello","trace_id":"trace-stream"}`,
		map[string]string{"Content-Type": "application/json"},
	)

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status: got %d want %d", recorder.Code, http.StatusOK)
	}
	if got := recorder.Header().Get("Content-Type"); got != "text/event-stream" {
		t.Fatalf("unexpected content type: got %q want %q", got, "text/event-stream")
	}
	if got := recorder.Header().Get("X-Trace-ID"); got != "trace-stream" {
		t.Fatalf("unexpected trace header: got %q want %q", got, "trace-stream")
	}
	if got := recorder.Header().Get("X-Accel-Buffering"); got != "no" {
		t.Fatalf("unexpected accel header: got %q want %q", got, "no")
	}

	events := decodeSSEEvents(t, recorder)
	if len(events) != 5 {
		t.Fatalf("unexpected event count: got %d want %d", len(events), 5)
	}
	wantOrder := []streaming.EventType{
		streaming.EventRunStarted,
		streaming.EventToolCallStarted,
		streaming.EventToolCallFinished,
		streaming.EventMessage,
		streaming.EventDone,
	}
	for index, want := range wantOrder {
		if events[index].Type != want {
			t.Fatalf("unexpected event[%d]: got %q want %q", index, events[index].Type, want)
		}
	}
	messagePayload, ok := events[3].Payload.(map[string]any)
	if !ok {
		t.Fatalf("unexpected message payload type: %T", events[3].Payload)
	}
	if messagePayload["text"] != "stream done" {
		t.Fatalf("unexpected streamed text: got %v want %q", messagePayload["text"], "stream done")
	}
	donePayload, ok := events[4].Payload.(map[string]any)
	if !ok {
		t.Fatalf("unexpected done payload type: %T", events[4].Payload)
	}
	if donePayload["session_ended"] != false {
		t.Fatalf("unexpected session_ended: got %v want %v", donePayload["session_ended"], false)
	}
}

func TestHandleAgentStreamValidationErrorEmitsErrorEvent(t *testing.T) {
	handler := newTestHandler(t, nil)
	recorder := serveRequest(
		handler,
		http.MethodPost,
		"/api/agent/stream",
		`{"message":"","session_id":"session-empty-message","trace_id":"trace-invalid"}`,
		map[string]string{"Content-Type": "application/json"},
	)

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status: got %d want %d", recorder.Code, http.StatusOK)
	}
	events := decodeSSEEvents(t, recorder)
	if len(events) != 1 {
		t.Fatalf("unexpected event count: got %d want %d", len(events), 1)
	}
	if events[0].Type != streaming.EventError {
		t.Fatalf("unexpected event type: got %q want %q", events[0].Type, streaming.EventError)
	}
	payload, ok := events[0].Payload.(map[string]any)
	if !ok {
		t.Fatalf("unexpected payload type: %T", events[0].Payload)
	}
	if payload["message"] != "message is required" {
		t.Fatalf("unexpected error message: got %v want %q", payload["message"], "message is required")
	}
}

func TestHandleAgentStreamRejectsInflightSessionBeforeSSE(t *testing.T) {
	handler, service, _ := newTestHandlerWithService(t, nil, nil)
	if err := service.runRegistry.Register("session-busy", "trace-busy", func() {}); err != nil {
		t.Fatalf("Register returned error: %v", err)
	}

	recorder := serveRequest(
		handler,
		http.MethodPost,
		"/api/agent/stream",
		`{"message":"hello","session_id":"session-busy","trace_id":"trace-conflict"}`,
		map[string]string{"Content-Type": "application/json"},
	)

	if recorder.Code != http.StatusConflict {
		t.Fatalf("unexpected status: got %d want %d", recorder.Code, http.StatusConflict)
	}
	body := decodeResponseBody(t, recorder)
	if !strings.Contains(body.Error, "session is already running") {
		t.Fatalf("unexpected error: %q", body.Error)
	}
}

func TestHandleAgentStreamClientDisconnectCancelsExecution(t *testing.T) {
	started := make(chan struct{})
	done := make(chan struct{})
	streamExecutor := func(
		ctx context.Context,
		_ string,
		_ string,
		_ string,
		_ *ConfigStore,
		_ *session.Store,
		_ streaming.Sink,
	) (string, string, error) {
		close(started)
		<-ctx.Done()
		close(done)
		return "", "", ctx.Err()
	}
	handler, _ := newTestHandlerWithStreamExecutor(t, nil, streamExecutor)

	ctx, cancel := context.WithCancel(context.Background())
	request := httptest.NewRequest(http.MethodPost, "/api/agent/stream", strings.NewReader(`{"message":"hello"}`)).WithContext(ctx)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()

	handlerDone := make(chan struct{})
	go func() {
		handler.ServeHTTP(recorder, request)
		close(handlerDone)
	}()

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("stream executor did not start")
	}

	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("context cancellation did not reach executor")
	}
	select {
	case <-handlerDone:
	case <-time.After(time.Second):
		t.Fatal("handler did not return after cancellation")
	}
}
