package transport

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"ghost-os/bridge/agent"
	bridgeconfig "ghost-os/bridge/config"
	bridgeorchestration "ghost-os/bridge/orchestration"
	"ghost-os/bridge/session"
	"ghost-os/bridge/tools"
)

func TestHandleAgentBodyTooLarge(t *testing.T) {
	handler := newTestHandler(t, nil)
	oversized := strings.Repeat("a", int(bridgeorchestration.DefaultMaxRequestBodyBytes)+32)
	requestBody := fmt.Sprintf(`{"message":"%s"}`, oversized)

	recorder := serveRequest(handler, http.MethodPost, "/api/agent", requestBody, map[string]string{"Content-Type": "application/json"})
	if recorder.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("unexpected status: got %d want %d", recorder.Code, http.StatusRequestEntityTooLarge)
	}

	body := decodeResponseBody(t, recorder)
	if body.Status != "error" {
		t.Fatalf("unexpected status field: got %q want %q", body.Status, "error")
	}
	if body.Error != "request body too large" {
		t.Fatalf("unexpected error: got %q want %q", body.Error, "request body too large")
	}
}

func TestHandleAgentMethodNotAllowed(t *testing.T) {
	handler := newTestHandler(t, nil)
	recorder := serveRequest(handler, http.MethodGet, "/api/agent", "", nil)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("unexpected status: got %d want %d", recorder.Code, http.StatusMethodNotAllowed)
	}
	body := decodeResponseBody(t, recorder)
	if body.Status != "error" {
		t.Fatalf("unexpected status field: got %q want %q", body.Status, "error")
	}
}

func TestAgentEndpointAliasesBusDispatch(t *testing.T) {
	handler := newTestHandler(t, func(_ context.Context, message string, sessionID string, _ string, _ bridgeconfig.Store, _ *session.Store) (string, string, error) {
		if message != "hello" {
			t.Fatalf("unexpected message: got %q want %q", message, "hello")
		}
		if sessionID != "" {
			t.Fatalf("unexpected session_id: got %q want empty", sessionID)
		}
		return "ok", "session-1", nil
	})

	traceID := "trace-alias"
	agentResp := serveRequest(
		handler,
		http.MethodPost,
		"/api/agent",
		fmt.Sprintf(`{"message":"hello","trace_id":"%s"}`, traceID),
		map[string]string{"Content-Type": "application/json"},
	)
	busResp := serveRequest(
		handler,
		http.MethodPost,
		"/api/bus",
		fmt.Sprintf(`{"action":"AGENT_SEND","params":{"message":"hello"},"trace_id":"%s"}`, traceID),
		map[string]string{"Content-Type": "application/json"},
	)

	if agentResp.Code != busResp.Code {
		t.Fatalf("agent/bus status mismatch: agent=%d bus=%d", agentResp.Code, busResp.Code)
	}
	if got := agentResp.Header().Get("X-Trace-ID"); got != traceID {
		t.Fatalf("unexpected agent trace header: got %q want %q", got, traceID)
	}
	if got := busResp.Header().Get("X-Trace-ID"); got != traceID {
		t.Fatalf("unexpected bus trace header: got %q want %q", got, traceID)
	}

	agentBody := decodeResponseBody(t, agentResp)
	busBody := decodeResponseBody(t, busResp)
	if !reflect.DeepEqual(agentBody, busBody) {
		t.Fatalf("agent/bus body mismatch: agent=%+v bus=%+v", agentBody, busBody)
	}
}

func TestAgentEndpointUsesHeaderTraceID(t *testing.T) {
	handler := newTestHandler(t, nil)
	traceID := "trace-from-header"
	recorder := serveRequest(
		handler,
		http.MethodPost,
		"/api/agent",
		`{"message":"hello"}`,
		map[string]string{
			"Content-Type": "application/json",
			"X-Trace-ID":   traceID,
		},
	)

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status: got %d want %d", recorder.Code, http.StatusOK)
	}
	if got := recorder.Header().Get("X-Trace-ID"); got != traceID {
		t.Fatalf("unexpected trace header: got %q want %q", got, traceID)
	}
}

func TestAgentEndpointPassesSessionIDAndReturnsIt(t *testing.T) {
	const sessionID = "session-from-client"
	handler, sessionStore := newTestHandlerWithStore(t, func(_ context.Context, message string, requestSessionID string, _ string, _ bridgeconfig.Store, _ *session.Store) (string, string, error) {
		if message != "hello" {
			t.Fatalf("unexpected message: got %q want %q", message, "hello")
		}
		if requestSessionID != sessionID {
			t.Fatalf("unexpected session_id: got %q want %q", requestSessionID, sessionID)
		}
		return "ok", requestSessionID, nil
	})
	sess := session.NewSession("system")
	sess.ID = sessionID
	if err := sessionStore.Save(sess); err != nil {
		t.Fatalf("save session: %v", err)
	}

	recorder := serveRequest(
		handler,
		http.MethodPost,
		"/api/agent",
		fmt.Sprintf(`{"message":"hello","session_id":"%s"}`, sessionID),
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
	if payload["session_id"] != sessionID {
		t.Fatalf("unexpected session_id in payload: got %v want %q", payload["session_id"], sessionID)
	}
}

func TestAgentEndpointPassesRuntimeOverrides(t *testing.T) {
	var capturedStore bridgeconfig.Store
	handler := newTestHandler(t, func(
		_ context.Context,
		_ string,
		_ string,
		_ string,
		store bridgeconfig.Store,
		_ *session.Store,
	) (string, string, error) {
		capturedStore = store
		return "ok", "session-runtime", nil
	})

	recorder := serveRequest(
		handler,
		http.MethodPost,
		"/api/agent",
		`{"message":"hello","runtime_overrides":{"provider_name":"openai","model":"gpt-5.4"}}`,
		map[string]string{"Content-Type": "application/json"},
	)
	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status: got %d want %d body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if capturedStore == nil {
		t.Fatal("expected runtime override request to provide a config store")
	}
	cfg, err := capturedStore.Config()
	if err != nil {
		t.Fatalf("Config(): %v", err)
	}
	if cfg.Provider.Model != "gpt-5.4" {
		t.Fatalf("unexpected runtime override model: got %q want %q", cfg.Provider.Model, "gpt-5.4")
	}
}

func TestAgentEndpointRejectsEmptyMessageWhenSessionIDIsPresent(t *testing.T) {
	const sessionID = "session-continue-1"
	handler := newTestHandler(t, func(_ context.Context, message string, requestSessionID string, _ string, _ bridgeconfig.Store, _ *session.Store) (string, string, error) {
		t.Fatal("executor should not run when message is empty")
		return "", "", nil
	})

	recorder := serveRequest(
		handler,
		http.MethodPost,
		"/api/agent",
		fmt.Sprintf(`{"message":"","session_id":"%s"}`, sessionID),
		map[string]string{"Content-Type": "application/json"},
	)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status: got %d want %d", recorder.Code, http.StatusBadRequest)
	}

	body := decodeResponseBody(t, recorder)
	if body.Error != "message or images is required" {
		t.Fatalf("unexpected error: got %q want %q", body.Error, "message or images is required")
	}
}

func TestAgentEndpointRejectsUnsupportedMode(t *testing.T) {
	handler := newTestHandler(t, func(_ context.Context, _ string, _ string, _ string, _ bridgeconfig.Store, _ *session.Store) (string, string, error) {
		t.Fatal("executor should not run when mode is invalid")
		return "", "", nil
	})

	recorder := serveRequest(
		handler,
		http.MethodPost,
		"/api/agent",
		`{"mode":"execute","message":"hello"}`,
		map[string]string{"Content-Type": "application/json"},
	)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status: got %d want %d", recorder.Code, http.StatusBadRequest)
	}

	body := decodeResponseBody(t, recorder)
	if body.Error != `unsupported agent mode: "execute"` {
		t.Fatalf("unexpected error: got %q want %q", body.Error, `unsupported agent mode: "execute"`)
	}
}

func TestAgentEndpointRejectsMissingSessionID(t *testing.T) {
	handler := newTestHandler(t, func(_ context.Context, _ string, _ string, _ string, _ bridgeconfig.Store, _ *session.Store) (string, string, error) {
		t.Fatal("executor should not run when session is missing")
		return "", "", nil
	})

	recorder := serveRequest(
		handler,
		http.MethodPost,
		"/api/agent",
		`{"message":"hello","session_id":"missing-session"}`,
		map[string]string{"Content-Type": "application/json"},
	)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("unexpected status: got %d want %d body=%s", recorder.Code, http.StatusNotFound, recorder.Body.String())
	}
	body := decodeResponseBody(t, recorder)
	if body.Status != "error" {
		t.Fatalf("unexpected status field: got %q want %q", body.Status, "error")
	}
	if !strings.Contains(body.Error, "omit session_id") {
		t.Fatalf("unexpected error message: %q", body.Error)
	}
}

func TestAgentEndpointStructuredSessionEndSignalMarksSessionEnded(t *testing.T) {
	const sessionID = "session-end-1"
	handler, sessionStore := newTestHandlerWithStore(t, func(_ context.Context, _ string, requestSessionID string, _ string, _ bridgeconfig.Store, _ *session.Store) (string, string, error) {
		if requestSessionID != sessionID {
			t.Fatalf("unexpected session_id: got %q want %q", requestSessionID, sessionID)
		}
		return `{"signal":"END_SESSION","message":"bye"}`, requestSessionID, nil
	})

	sess := session.NewSession("system")
	sess.ID = sessionID
	if err := sessionStore.Save(sess); err != nil {
		t.Fatalf("save session: %v", err)
	}

	recorder := serveRequest(
		handler,
		http.MethodPost,
		"/api/agent",
		fmt.Sprintf(`{"message":"finish","session_id":"%s"}`, sessionID),
		map[string]string{"Content-Type": "application/json"},
	)
	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status: got %d want %d body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	body := decodeResponseBody(t, recorder)
	payload, ok := body.Payload.(map[string]any)
	if !ok {
		t.Fatalf("unexpected payload type: %T", body.Payload)
	}
	if payload["message"] != "bye" {
		t.Fatalf("unexpected message: got %v want %q", payload["message"], "bye")
	}
	if payload["session_ended"] != true {
		t.Fatalf("unexpected session_ended: got %v want %v", payload["session_ended"], true)
	}
	sessionEnd, ok := payload["session_end"].(map[string]any)
	if !ok {
		t.Fatalf("unexpected session_end type: %T", payload["session_end"])
	}
	if sessionEnd["signal"] != bridgeorchestration.BusAssistantSessionEndSignal {
		t.Fatalf("unexpected session_end.signal: got %v want %q", sessionEnd["signal"], bridgeorchestration.BusAssistantSessionEndSignal)
	}
	if sessionEnd["message"] != "bye" {
		t.Fatalf("unexpected session_end.message: got %v want %q", sessionEnd["message"], "bye")
	}

	updated, err := sessionStore.Load(sessionID)
	if err != nil {
		t.Fatalf("load session after end signal: %v", err)
	}
	if updated.EndedAt.IsZero() {
		t.Fatal("session should be marked ended")
	}
}

func TestAgentEndpointRejectsAlreadyEndedSession(t *testing.T) {
	const sessionID = "session-ended-1"
	handler, sessionStore := newTestHandlerWithStore(t, func(_ context.Context, _ string, _ string, _ string, _ bridgeconfig.Store, _ *session.Store) (string, string, error) {
		t.Fatal("executor should not be called for ended session")
		return "", "", nil
	})

	sess := session.NewSession("system")
	sess.ID = sessionID
	sess.MarkEnded(time.Now().UTC())
	if err := sessionStore.Save(sess); err != nil {
		t.Fatalf("save ended session: %v", err)
	}

	recorder := serveRequest(
		handler,
		http.MethodPost,
		"/api/agent",
		fmt.Sprintf(`{"message":"hello","session_id":"%s"}`, sessionID),
		map[string]string{"Content-Type": "application/json"},
	)
	if recorder.Code != http.StatusConflict {
		t.Fatalf("unexpected status: got %d want %d body=%s", recorder.Code, http.StatusConflict, recorder.Body.String())
	}
	body := decodeResponseBody(t, recorder)
	if !strings.Contains(body.Error, "already ended") {
		t.Fatalf("unexpected error: %q", body.Error)
	}
}

func TestAgentEndpointReturnsBadRequestOnInvalidSessionID(t *testing.T) {
	handler := newTestHandler(t, func(_ context.Context, _ string, _ string, _ string, _ bridgeconfig.Store, _ *session.Store) (string, string, error) {
		return "", "", fmt.Errorf("%w: invalid characters", session.ErrInvalidSessionID)
	})

	recorder := serveRequest(
		handler,
		http.MethodPost,
		"/api/agent",
		`{"message":"hello","session_id":"../bad"}`,
		map[string]string{"Content-Type": "application/json"},
	)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status: got %d want %d", recorder.Code, http.StatusBadRequest)
	}
}

func TestAgentEndpointReturnsAcceptedWhenAwaitingHuman(t *testing.T) {
	handler := newTestHandler(t, func(_ context.Context, _ string, _ string, _ string, _ bridgeconfig.Store, _ *session.Store) (string, string, error) {
		return "", "session-awaiting-1", &agent.ErrAwaitingHuman{
			QuestionID:    "q-awaiting-1",
			Prompt:        "Which database should we use?",
			SelectionMode: session.HumanQuestionSelectionSingle,
			Options: []tools.AskHumanOption{
				{Label: "PostgreSQL"},
				{Label: "Other", AllowCustom: true},
			},
		}
	})

	recorder := serveRequest(
		handler,
		http.MethodPost,
		"/api/agent",
		`{"message":"Choose DB"}`,
		map[string]string{"Content-Type": "application/json"},
	)
	if recorder.Code != http.StatusAccepted {
		t.Fatalf("unexpected status: got %d want %d", recorder.Code, http.StatusAccepted)
	}

	body := decodeResponseBody(t, recorder)
	if body.Status != "success" {
		t.Fatalf("unexpected status field: got %q want %q", body.Status, "success")
	}
	payload, ok := body.Payload.(map[string]any)
	if !ok {
		t.Fatalf("unexpected payload type: %T", body.Payload)
	}
	if payload["status"] != "awaiting_human" {
		t.Fatalf("unexpected awaiting status: got %v want %q", payload["status"], "awaiting_human")
	}
	if payload["question_id"] != "q-awaiting-1" {
		t.Fatalf("unexpected question_id: got %v want %q", payload["question_id"], "q-awaiting-1")
	}
	if payload["selection_mode"] != session.HumanQuestionSelectionSingle {
		t.Fatalf("unexpected selection_mode: got %v want %q", payload["selection_mode"], session.HumanQuestionSelectionSingle)
	}
	options, ok := payload["options"].([]any)
	if !ok || len(options) != 2 {
		t.Fatalf("unexpected options payload: %#v", payload["options"])
	}
}
