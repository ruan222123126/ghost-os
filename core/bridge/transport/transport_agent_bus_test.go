package transport

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"ghost-os/bridge/session"
	"ghost-os/bridge/streaming"
)

func TestBusAgentStopCancelsRunBySessionID(t *testing.T) {
	const sessionID = "session-stop"
	handler, service, sessionStore := newTestHandlerWithService(t, nil, nil)
	sess := session.NewSession("system")
	sess.ID = sessionID
	if err := sessionStore.Save(sess); err != nil {
		t.Fatalf("save session: %v", err)
	}
	started := make(chan struct{})
	stopped := make(chan struct{})
	service.SetAgentRunner(newSessionTurnRunnerAdapter(service.ConfigStore(), service.SessionStore(), func(
		_ context.Context,
		_ string,
		_ string,
		_ string,
		_ *ConfigStore,
		_ *session.Store,
	) (string, string, error) {
		return "", "", errors.New("unexpected sync run")
	}, func(
		ctx context.Context,
		_ string,
		sessionID string,
		traceID string,
		_ *ConfigStore,
		_ *session.Store,
		_ streaming.Sink,
	) (string, string, error) {
		execCtx, cancel := context.WithCancel(ctx)
		if err := service.RunRegistry().Register(sessionID, traceID, cancel); err != nil {
			cancel()
			return "", sessionID, err
		}
		defer service.RunRegistry().Unregister(sessionID)
		close(started)
		<-execCtx.Done()
		close(stopped)
		return "", sessionID, context.Canceled
	}))

	request := httptest.NewRequest(http.MethodPost, "/api/agent/stream", strings.NewReader(`{"message":"hello","session_id":"session-stop","trace_id":"trace-stop"}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	streamDone := make(chan struct{})
	go func() {
		handler.ServeHTTP(recorder, request)
		close(streamDone)
	}()

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("stream run did not start")
	}

	stopRecorder := serveRequest(
		handler,
		http.MethodPost,
		"/api/bus",
		`{"action":"AGENT_STOP","params":{"session_id":"session-stop"},"trace_id":"trace-stop-request"}`,
		map[string]string{"Content-Type": "application/json"},
	)
	if stopRecorder.Code != http.StatusOK {
		t.Fatalf("unexpected stop status: got %d want %d", stopRecorder.Code, http.StatusOK)
	}
	body := decodeResponseBody(t, stopRecorder)
	payload, ok := body.Payload.(map[string]any)
	if !ok {
		t.Fatalf("unexpected payload type: %T", body.Payload)
	}
	if payload["status"] != "stopped" {
		t.Fatalf("unexpected stop status payload: got %v want %q", payload["status"], "stopped")
	}

	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("run was not cancelled")
	}
	select {
	case <-streamDone:
	case <-time.After(time.Second):
		t.Fatal("stream handler did not exit")
	}
}

func TestBusAgentStopReturnsNotRunning(t *testing.T) {
	handler := newTestHandler(t, nil)

	recorder := serveRequest(
		handler,
		http.MethodPost,
		"/api/bus",
		`{"action":"AGENT_STOP","params":{"session_id":"missing-session"},"trace_id":"trace-stop-missing"}`,
		map[string]string{"Content-Type": "application/json"},
	)
	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status: got %d want %d", recorder.Code, http.StatusOK)
	}
	body := decodeResponseBody(t, recorder)
	payload, ok := body.Payload.(map[string]any)
	if !ok {
		t.Fatalf("unexpected payload type: %T", body.Payload)
	}
	if payload["status"] != "not_running" {
		t.Fatalf("unexpected payload status: got %v want %q", payload["status"], "not_running")
	}
}

func TestBusAgentSendRegression(t *testing.T) {
	handler := newTestHandler(t, func(_ context.Context, message string, sessionID string, traceID string, _ *ConfigStore, _ *session.Store) (string, string, error) {
		if message != "hello" {
			t.Fatalf("unexpected message: got %q want %q", message, "hello")
		}
		if sessionID != "" {
			t.Fatalf("unexpected session_id: got %q want empty", sessionID)
		}
		if traceID != "trace-agent" {
			t.Fatalf("unexpected trace_id: got %q want %q", traceID, "trace-agent")
		}
		return "ok", "session-1", nil
	})

	recorder := serveRequest(handler, http.MethodPost, "/api/bus", `{"action":"AGENT_SEND","params":{"message":"hello"},"trace_id":"trace-agent"}`, nil)
	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status: got %d want %d", recorder.Code, http.StatusOK)
	}
	if recorder.Header().Get("X-Trace-ID") != "trace-agent" {
		t.Fatalf("unexpected trace header: got %q want %q", recorder.Header().Get("X-Trace-ID"), "trace-agent")
	}
	body := decodeResponseBody(t, recorder)
	if body.Status != "success" {
		t.Fatalf("unexpected body status: got %q want %q", body.Status, "success")
	}
	payload, ok := body.Payload.(map[string]any)
	if !ok {
		t.Fatalf("unexpected payload type: %T", body.Payload)
	}
	if payload["session_id"] != "session-1" {
		t.Fatalf("unexpected session_id: got %v want %q", payload["session_id"], "session-1")
	}
	if payload["session_ended"] != false {
		t.Fatalf("unexpected session_ended: got %v want %v", payload["session_ended"], false)
	}
	if _, exists := payload["session_end"]; exists {
		t.Fatalf("session_end should be absent for normal response: %+v", payload["session_end"])
	}
}

func TestBusAgentSendRejectsEmptyMessageEvenWithSessionID(t *testing.T) {
	handler := newTestHandler(t, func(_ context.Context, _ string, _ string, _ string, _ *ConfigStore, _ *session.Store) (string, string, error) {
		t.Fatal("executor should not run when message is empty")
		return "", "", nil
	})

	recorder := serveRequest(
		handler,
		http.MethodPost,
		"/api/bus",
		`{"action":"AGENT_SEND","params":{"message":"","session_id":"session-resume-like"},"trace_id":"trace-agent-empty"}`,
		map[string]string{"Content-Type": "application/json"},
	)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status: got %d want %d", recorder.Code, http.StatusBadRequest)
	}
	body := decodeResponseBody(t, recorder)
	if body.Error != "message is required" {
		t.Fatalf("unexpected error: got %q want %q", body.Error, "message is required")
	}
}
