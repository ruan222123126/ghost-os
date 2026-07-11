package transport

import (
	"context"
	"errors"
	"net/http"
	"testing"

	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/session"
	"ghost-os/bridge/streaming"
)

func TestHandleAgentStreamEmitsFallbackErrorEventWhenServiceReturnsError(t *testing.T) {
	const sessionID = "session-stream-error"

	handler, _ := newTestHandlerWithStreamExecutor(t, nil, func(
		_ context.Context,
		message string,
		incomingSessionID string,
		_ string,
		_ bridgeconfig.Store,
		_ *session.Store,
		_ streaming.Sink,
	) (string, string, error) {
		if message != "hello" {
			t.Fatalf("unexpected message: got %q want %q", message, "hello")
		}
		if incomingSessionID != "" {
			t.Fatalf("unexpected session_id: got %q want empty", incomingSessionID)
		}
		return "", sessionID, errors.New("provider failed")
	})

	recorder := serveRequest(
		handler,
		http.MethodPost,
		"/api/agent/stream",
		`{"message":"hello","trace_id":"trace-stream-error"}`,
		map[string]string{"Content-Type": "application/json"},
	)
	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status: got %d want %d", recorder.Code, http.StatusOK)
	}

	assertFallbackErrorEvent(t, decodeSSEEvents(t, recorder), "provider failed", sessionID)
}

func TestHandleAgentStreamSuppressesFallbackErrorEventWhenRunCancelled(t *testing.T) {
	const sessionID = "session-stream-cancelled"

	handler, _ := newTestHandlerWithStreamExecutor(t, nil, func(
		_ context.Context,
		_ string,
		_ string,
		_ string,
		_ bridgeconfig.Store,
		_ *session.Store,
		_ streaming.Sink,
	) (string, string, error) {
		return "", sessionID, context.Canceled
	})

	recorder := serveRequest(
		handler,
		http.MethodPost,
		"/api/agent/stream",
		`{"message":"hello","trace_id":"trace-stream-cancelled"}`,
		map[string]string{"Content-Type": "application/json"},
	)
	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status: got %d want %d", recorder.Code, http.StatusOK)
	}

	assertNoSSEEvents(t, decodeSSEEvents(t, recorder))
}

func TestHandleQuestionAnswerStreamEmitsFallbackErrorEventWhenResumeReturnsError(t *testing.T) {
	const (
		sessionID  = "session-question-answer-error"
		questionID = "q-error"
	)

	handler, sessionStore := newTestHandlerWithStreamExecutor(t, nil, func(
		_ context.Context,
		message string,
		incomingSessionID string,
		_ string,
		_ bridgeconfig.Store,
		_ *session.Store,
		_ streaming.Sink,
	) (string, string, error) {
		if message != "" {
			t.Fatalf("unexpected message: got %q want empty", message)
		}
		if incomingSessionID != sessionID {
			t.Fatalf("unexpected session_id: got %q want %q", incomingSessionID, sessionID)
		}
		return "", sessionID, errors.New("resume failed")
	})

	sess := session.NewSession("system")
	sess.ID = sessionID
	sess.AddPendingQuestion(questionID, session.PendingHumanQuestion{
		Prompt:     "继续执行？",
		ToolCallID: "call-error",
		TraceID:    "trace-error",
	})
	if err := sessionStore.Save(sess); err != nil {
		t.Fatalf("save session: %v", err)
	}

	recorder := serveRequest(
		handler,
		http.MethodPost,
		"/api/questions/answer/stream",
		`{"session_id":"session-question-answer-error","question_id":"q-error","answer":"继续"}`,
		map[string]string{"Content-Type": "application/json"},
	)
	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status: got %d want %d", recorder.Code, http.StatusOK)
	}

	assertFallbackErrorEvent(t, decodeSSEEvents(t, recorder), "resume failed", sessionID)
}

func TestHandleQuestionAnswerStreamSuppressesFallbackErrorEventWhenRunCancelled(t *testing.T) {
	const (
		sessionID  = "session-question-answer-cancelled"
		questionID = "q-cancel"
	)

	handler, sessionStore := newTestHandlerWithStreamExecutor(t, nil, func(
		_ context.Context,
		message string,
		incomingSessionID string,
		_ string,
		_ bridgeconfig.Store,
		_ *session.Store,
		_ streaming.Sink,
	) (string, string, error) {
		if message != "" {
			t.Fatalf("unexpected message: got %q want empty", message)
		}
		if incomingSessionID != sessionID {
			t.Fatalf("unexpected session_id: got %q want %q", incomingSessionID, sessionID)
		}
		return "", sessionID, context.Canceled
	})

	sess := session.NewSession("system")
	sess.ID = sessionID
	sess.AddPendingQuestion(questionID, session.PendingHumanQuestion{
		Prompt:     "继续执行？",
		ToolCallID: "call-cancel",
		TraceID:    "trace-cancel",
	})
	if err := sessionStore.Save(sess); err != nil {
		t.Fatalf("save session: %v", err)
	}

	recorder := serveRequest(
		handler,
		http.MethodPost,
		"/api/questions/answer/stream",
		`{"session_id":"session-question-answer-cancelled","question_id":"q-cancel","answer":"继续"}`,
		map[string]string{"Content-Type": "application/json"},
	)
	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status: got %d want %d", recorder.Code, http.StatusOK)
	}

	assertNoSSEEvents(t, decodeSSEEvents(t, recorder))
}

func assertNoSSEEvents(t *testing.T, events []streaming.Event) {
	t.Helper()
	if len(events) != 0 {
		t.Fatalf("unexpected event count: got %d want 0", len(events))
	}
}

func assertFallbackErrorEvent(t *testing.T, events []streaming.Event, wantMessage string, wantSessionID string) {
	t.Helper()
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
	if payload["message"] != wantMessage {
		t.Fatalf("unexpected error message: got %v want %q", payload["message"], wantMessage)
	}
	if payload["session_id"] != wantSessionID {
		t.Fatalf("unexpected payload session_id: got %v want %q", payload["session_id"], wantSessionID)
	}
	if events[0].SessionID != wantSessionID {
		t.Fatalf("unexpected event session_id: got %q want %q", events[0].SessionID, wantSessionID)
	}
}
