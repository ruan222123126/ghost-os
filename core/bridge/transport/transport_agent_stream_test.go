package transport

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	bridgeconfig "ghost-os/bridge/config"
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
		_ bridgeconfig.Store,
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

func TestHandleAgentStreamPassesRuntimeOverrides(t *testing.T) {
	var capturedRootModel string
	streamExecutor := func(
		ctx context.Context,
		_ string,
		_ string,
		traceID string,
		store bridgeconfig.Store,
		_ *session.Store,
		sink streaming.Sink,
	) (string, string, error) {
		cfg, err := store.Config()
		if err != nil {
			return "", "", err
		}
		capturedRootModel = cfg.Provider.Model
		sessionID := "session-stream-runtime"
		if _, err := sink.Emit(ctx, mustAppEvent(t, traceID, sessionID, 1, mustAppAssistantStepID(t, 1), streaming.EventMessage, map[string]any{
			"text":       "ok",
			"session_id": sessionID,
		})); err != nil {
			return "", "", err
		}
		if _, err := sink.Emit(ctx, mustAppEvent(t, traceID, sessionID, 1, "", streaming.EventDone, map[string]any{
			"session_id":    sessionID,
			"session_ended": false,
		})); err != nil {
			return "", "", err
		}
		return "ok", sessionID, nil
	}
	handler, _ := newTestHandlerWithStreamExecutor(t, nil, streamExecutor)

	recorder := serveRequest(
		handler,
		http.MethodPost,
		"/api/agent/stream",
		`{"message":"hello","runtime_overrides":{"provider_name":"openai","model":"gpt-5.4"},"trace_id":"trace-runtime-stream"}`,
		map[string]string{"Content-Type": "application/json"},
	)
	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status: got %d want %d body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if capturedRootModel != "gpt-5.4" {
		t.Fatalf("unexpected runtime override model: got %q want %q", capturedRootModel, "gpt-5.4")
	}
}

func TestHandleAgentStreamTreatsProPrefixAsStandardMessage(t *testing.T) {
	streamExecutor := func(
		ctx context.Context,
		message string,
		sessionID string,
		traceID string,
		_ bridgeconfig.Store,
		_ *session.Store,
		sink streaming.Sink,
	) (string, string, error) {
		if message != "pro fix config" {
			t.Fatalf("unexpected message: got %q want %q", message, "pro fix config")
		}
		if sessionID != "" {
			t.Fatalf("unexpected session_id: got %q want empty", sessionID)
		}
		if _, err := sink.Emit(ctx, mustAppEvent(t, traceID, "session-stream", 1, mustAppAssistantStepID(t, 1), streaming.EventMessage, map[string]any{
			"text":       "standard streamed done",
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
		return "standard streamed done", "session-stream", nil
	}
	handler, _ := newTestHandlerWithStreamExecutor(t, nil, streamExecutor)

	recorder := serveRequest(
		handler,
		http.MethodPost,
		"/api/agent/stream",
		`{"message":"pro fix config","trace_id":"trace-pro-stream"}`,
		map[string]string{"Content-Type": "application/json"},
	)

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status: got %d want %d", recorder.Code, http.StatusOK)
	}
	events := decodeSSEEvents(t, recorder)
	if len(events) != 2 {
		t.Fatalf("unexpected event count: got %d want %d", len(events), 2)
	}
	if events[0].Type != streaming.EventMessage || events[1].Type != streaming.EventDone {
		t.Fatalf("unexpected event types: %+v", events)
	}
	payload, ok := events[0].Payload.(map[string]any)
	if !ok {
		t.Fatalf("unexpected message payload type: %T", events[0].Payload)
	}
	if payload["text"] != "standard streamed done" {
		t.Fatalf("unexpected text: %v", payload["text"])
	}
}

func TestHandleAgentStreamRejectsPlanMode(t *testing.T) {
	handler := newTestHandler(t, nil)
	recorder := serveRequest(
		handler,
		http.MethodPost,
		"/api/agent/stream",
		`{"mode":"plan","message":"pro fix config","trace_id":"trace-plan-stream"}`,
		map[string]string{"Content-Type": "application/json"},
	)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status: got %d want %d", recorder.Code, http.StatusBadRequest)
	}
	body := decodeResponseBody(t, recorder)
	if body.Status != "error" {
		t.Fatalf("unexpected response status: got %q want %q", body.Status, "error")
	}
	if body.Error != `unsupported agent mode: "plan"` {
		t.Fatalf("unexpected error message: got %q", body.Error)
	}
}

func TestHandleAgentStreamValidationErrorReturnsEnvelope(t *testing.T) {
	handler := newTestHandler(t, nil)
	recorder := serveRequest(
		handler,
		http.MethodPost,
		"/api/agent/stream",
		`{"message":"","session_id":"session-empty-message","trace_id":"trace-invalid"}`,
		map[string]string{"Content-Type": "application/json"},
	)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status: got %d want %d", recorder.Code, http.StatusBadRequest)
	}
	if got := recorder.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("unexpected content type: got %q want %q", got, "application/json")
	}
	body := decodeResponseBody(t, recorder)
	if body.Status != "error" {
		t.Fatalf("unexpected response status: got %q want %q", body.Status, "error")
	}
	if body.Error != "message or images is required" {
		t.Fatalf("unexpected error message: got %q want %q", body.Error, "message or images is required")
	}
}

func TestHandleAgentStreamInflightSessionReturnsEnvelope(t *testing.T) {
	handler, service, _ := newTestHandlerWithService(t, nil, nil)
	if err := service.RunRegistry().Register("session-busy", "trace-busy", func() {}); err != nil {
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
	if got := recorder.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("unexpected content type: got %q want %q", got, "application/json")
	}
	body := decodeResponseBody(t, recorder)
	if body.Status != "error" {
		t.Fatalf("unexpected response status: got %q want %q", body.Status, "error")
	}
	if !strings.Contains(body.Error, "session is already running") {
		t.Fatalf("unexpected error: %v", body.Error)
	}
}

func TestHandleAgentStreamClientDisconnectKeepsExecutionRunning(t *testing.T) {
	started := make(chan struct{})
	done := make(chan struct{})
	release := make(chan struct{})
	streamExecutor := func(
		ctx context.Context,
		_ string,
		_ string,
		_ string,
		_ bridgeconfig.Store,
		_ *session.Store,
		_ streaming.Sink,
	) (string, string, error) {
		close(started)
		select {
		case <-ctx.Done():
			close(done)
			return "", "", ctx.Err()
		case <-release:
			close(done)
			return "ok", "session-background", nil
		}
	}
	_, service, _ := newTestHandlerWithService(t, nil, streamExecutor)
	runCtx, stopRunContext := context.WithCancel(context.Background())
	defer stopRunContext()
	options, err := newServerOptionsFromEnv(8080)
	if err != nil {
		t.Fatalf("new server options: %v", err)
	}
	handler := newHTTPHandlerWithContext(runCtx, service, options)

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
	case <-handlerDone:
	case <-time.After(time.Second):
		t.Fatal("handler did not return after cancellation")
	}
	select {
	case <-done:
		t.Fatal("stream executor stopped after client disconnect")
	default:
	}

	close(release)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("stream executor did not finish after release")
	}
}
