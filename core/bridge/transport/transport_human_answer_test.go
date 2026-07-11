package transport

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"ghost-os/bridge/agent"
	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/session"
	"ghost-os/bridge/streaming"
)

func TestQuestionAnswerEndpointReturnsFinalReply(t *testing.T) {
	const sessionID = "session-question-answer-final"
	handler, sessionStore := newTestHandlerWithStore(t, func(_ context.Context, message string, requestSessionID string, _ string, _ bridgeconfig.Store, store *session.Store) (string, string, error) {
		if message != "" {
			t.Fatalf("unexpected resume message: got %q want empty", message)
		}
		if requestSessionID != sessionID {
			t.Fatalf("unexpected session_id: got %q want %q", requestSessionID, sessionID)
		}

		loaded, err := store.Load(sessionID)
		if err != nil {
			t.Fatalf("load session during resume: %v", err)
		}
		if loaded.HumanAnswers["q-1"] != "PostgreSQL" {
			t.Fatalf("unexpected stored answer before resume: %+v", loaded.HumanAnswers)
		}

		return "Use PostgreSQL.", requestSessionID, nil
	})

	sess := session.NewSession("system")
	sess.ID = sessionID
	sess.AddPendingQuestion("q-1", session.PendingHumanQuestion{
		Prompt:     "Which database should we use?",
		ToolCallID: "call-ask-1",
		TraceID:    "trace-ask-final",
	})
	if err := sessionStore.Save(sess); err != nil {
		t.Fatalf("save session: %v", err)
	}

	recorder := serveRequest(
		handler,
		http.MethodPost,
		"/api/questions/answer",
		`{"session_id":"session-question-answer-final","question_id":"q-1","answer":"PostgreSQL"}`,
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
	if payload["message"] != "Use PostgreSQL." {
		t.Fatalf("unexpected message: got %v want %q", payload["message"], "Use PostgreSQL.")
	}
	if payload["session_id"] != sessionID {
		t.Fatalf("unexpected session_id: got %v want %q", payload["session_id"], sessionID)
	}
}

func TestQuestionAnswerEndpointCanReturnAwaitingHumanAgain(t *testing.T) {
	const sessionID = "session-question-answer-awaiting"
	handler, sessionStore := newTestHandlerWithStore(t, func(_ context.Context, message string, requestSessionID string, _ string, _ bridgeconfig.Store, _ *session.Store) (string, string, error) {
		if message != "" {
			t.Fatalf("unexpected resume message: got %q want empty", message)
		}
		if requestSessionID != sessionID {
			t.Fatalf("unexpected session_id: got %q want %q", requestSessionID, sessionID)
		}
		return "", requestSessionID, &agent.ErrAwaitingHuman{
			QuestionID: "q-2",
			Prompt:     "Need another approval.",
		}
	})

	sess := session.NewSession("system")
	sess.ID = sessionID
	sess.AddPendingQuestion("q-1", session.PendingHumanQuestion{
		Prompt:     "Ship now?",
		ToolCallID: "call-ask-2",
		TraceID:    "trace-ask-awaiting",
	})
	if err := sessionStore.Save(sess); err != nil {
		t.Fatalf("save session: %v", err)
	}

	recorder := serveRequest(
		handler,
		http.MethodPost,
		"/api/questions/answer",
		`{"session_id":"session-question-answer-awaiting","question_id":"q-1","answer":"Yes"}`,
		map[string]string{"Content-Type": "application/json"},
	)
	if recorder.Code != http.StatusAccepted {
		t.Fatalf("unexpected status: got %d want %d body=%s", recorder.Code, http.StatusAccepted, recorder.Body.String())
	}

	body := decodeResponseBody(t, recorder)
	payload, ok := body.Payload.(map[string]any)
	if !ok {
		t.Fatalf("unexpected payload type: %T", body.Payload)
	}
	if payload["status"] != "awaiting_human" {
		t.Fatalf("unexpected awaiting status: got %v want %q", payload["status"], "awaiting_human")
	}
	if payload["question_id"] != "q-2" {
		t.Fatalf("unexpected question_id: got %v want %q", payload["question_id"], "q-2")
	}
	if payload["prompt"] != "Need another approval." {
		t.Fatalf("unexpected prompt: got %v want %q", payload["prompt"], "Need another approval.")
	}
}

func TestQuestionAnswerEndpointCanCancelPendingQuestion(t *testing.T) {
	const sessionID = "session-question-answer-cancel"
	handler, sessionStore := newTestHandlerWithStore(t, func(_ context.Context, _ string, _ string, _ string, _ bridgeconfig.Store, _ *session.Store) (string, string, error) {
		t.Fatal("executor should not run when question is cancelled")
		return "", "", nil
	})

	sess := session.NewSession("system")
	sess.ID = sessionID
	sess.AddPendingQuestion("q-1", session.PendingHumanQuestion{
		Prompt:     "Ship now?",
		ToolCallID: "call-ask-cancel",
		TraceID:    "trace-ask-cancel",
	})
	if err := sessionStore.Save(sess); err != nil {
		t.Fatalf("save session: %v", err)
	}

	recorder := serveRequest(
		handler,
		http.MethodPost,
		"/api/questions/answer",
		`{"session_id":"session-question-answer-cancel","question_id":"q-1","answer":"","cancelled":true}`,
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
	if payload["message"] != cancelledHumanDialogueMessage {
		t.Fatalf("unexpected message: got %v want %q", payload["message"], cancelledHumanDialogueMessage)
	}
	if payload["session_ended"] != true {
		t.Fatalf("unexpected session_ended: got %v want true", payload["session_ended"])
	}

	loaded, err := sessionStore.Load(sessionID)
	if err != nil {
		t.Fatalf("load session: %v", err)
	}
	if !loaded.IsEnded() {
		t.Fatal("cancelled session should be marked ended")
	}
	if _, ok := loaded.PendingQuestions["q-1"]; ok {
		t.Fatal("pending question should be removed after cancellation")
	}
}

func TestQuestionAnswerEndpointRejectsMissingQuestion(t *testing.T) {
	const sessionID = "session-question-answer-missing"
	handler, sessionStore := newTestHandlerWithStore(t, nil)

	sess := session.NewSession("system")
	sess.ID = sessionID
	if err := sessionStore.Save(sess); err != nil {
		t.Fatalf("save session: %v", err)
	}

	recorder := serveRequest(
		handler,
		http.MethodPost,
		"/api/questions/answer",
		`{"session_id":"session-question-answer-missing","question_id":"q-missing","answer":"PostgreSQL"}`,
		map[string]string{"Content-Type": "application/json"},
	)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("unexpected status: got %d want %d body=%s", recorder.Code, http.StatusNotFound, recorder.Body.String())
	}

	body := decodeResponseBody(t, recorder)
	if !strings.Contains(body.Error, "question not found") {
		t.Fatalf("unexpected error: %q", body.Error)
	}
}

func TestQuestionAnswerEndpointRejectsEndedSession(t *testing.T) {
	const sessionID = "session-question-answer-ended"
	handler, sessionStore := newTestHandlerWithStore(t, func(_ context.Context, _ string, _ string, _ string, _ bridgeconfig.Store, _ *session.Store) (string, string, error) {
		t.Fatal("executor should not be called for ended session")
		return "", "", nil
	})

	sess := session.NewSession("system")
	sess.ID = sessionID
	sess.AddPendingQuestion("q-1", session.PendingHumanQuestion{
		Prompt:     "Approve deploy?",
		ToolCallID: "call-ended",
		TraceID:    "trace-ended",
	})
	sess.MarkEnded(time.Now().UTC())
	if err := sessionStore.Save(sess); err != nil {
		t.Fatalf("save session: %v", err)
	}

	recorder := serveRequest(
		handler,
		http.MethodPost,
		"/api/questions/answer",
		`{"session_id":"session-question-answer-ended","question_id":"q-1","answer":"Yes"}`,
		map[string]string{"Content-Type": "application/json"},
	)
	if recorder.Code != http.StatusConflict {
		t.Fatalf("unexpected status: got %d want %d body=%s", recorder.Code, http.StatusConflict, recorder.Body.String())
	}

	body := decodeResponseBody(t, recorder)
	if !strings.Contains(body.Error, "already ended") {
		t.Fatalf("unexpected error: %q", body.Error)
	}

	loaded, err := sessionStore.Load(sessionID)
	if err != nil {
		t.Fatalf("load session: %v", err)
	}
	if len(loaded.HumanAnswers) != 0 {
		t.Fatalf("ended session should not store answers, got %+v", loaded.HumanAnswers)
	}
}

func TestQuestionAnswerEndpointRejectsInflightSession(t *testing.T) {
	const sessionID = "session-question-answer-busy"
	handler, service, sessionStore := newTestHandlerWithService(t, func(_ context.Context, _ string, _ string, _ string, _ bridgeconfig.Store, _ *session.Store) (string, string, error) {
		t.Fatal("executor should not be called for inflight session")
		return "", "", nil
	}, nil)

	sess := session.NewSession("system")
	sess.ID = sessionID
	sess.AddPendingQuestion("q-1", session.PendingHumanQuestion{
		Prompt:     "Approve deploy?",
		ToolCallID: "call-busy",
		TraceID:    "trace-busy",
	})
	if err := sessionStore.Save(sess); err != nil {
		t.Fatalf("save session: %v", err)
	}
	if err := service.RunRegistry().Register(sessionID, "trace-busy-answer", func() {}); err != nil {
		t.Fatalf("register inflight session: %v", err)
	}
	defer service.RunRegistry().Unregister(sessionID)

	recorder := serveRequest(
		handler,
		http.MethodPost,
		"/api/questions/answer",
		`{"session_id":"session-question-answer-busy","question_id":"q-1","answer":"Yes"}`,
		map[string]string{"Content-Type": "application/json"},
	)
	if recorder.Code != http.StatusConflict {
		t.Fatalf("unexpected status: got %d want %d body=%s", recorder.Code, http.StatusConflict, recorder.Body.String())
	}

	body := decodeResponseBody(t, recorder)
	if !strings.Contains(body.Error, "already running") {
		t.Fatalf("unexpected error: %q", body.Error)
	}

	loaded, err := sessionStore.Load(sessionID)
	if err != nil {
		t.Fatalf("load session: %v", err)
	}
	if len(loaded.HumanAnswers) != 0 {
		t.Fatalf("inflight session should not store answers, got %+v", loaded.HumanAnswers)
	}
}

func TestQuestionAnswerStreamEndpointReturnsSSEEvents(t *testing.T) {
	const sessionID = "session-question-answer-stream"
	handler, sessionStore := newTestHandlerWithStreamExecutor(t, nil, func(
		ctx context.Context,
		message string,
		incomingSessionID string,
		traceID string,
		_ bridgeconfig.Store,
		_ *session.Store,
		sink streaming.Sink,
	) (string, string, error) {
		if message != "" {
			t.Fatalf("unexpected message: got %q want empty", message)
		}
		if incomingSessionID != sessionID {
			t.Fatalf("unexpected session id: got %q want %q", incomingSessionID, sessionID)
		}
		if _, err := sink.Emit(ctx, mustAppEvent(t, traceID, sessionID, 0, mustAppToolStepID(t, 0, 0), streaming.EventToolCallStarted, map[string]any{
			"tool":         "exec",
			"tool_call_id": "call-answer-1",
		})); err != nil {
			return "", "", err
		}
		if _, err := sink.Emit(ctx, mustAppEvent(t, traceID, sessionID, 1, mustAppAssistantStepID(t, 1), streaming.EventMessage, map[string]any{
			"text":       "继续完成",
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
		return "继续完成", sessionID, nil
	})

	sess := session.NewSession("system")
	sess.ID = sessionID
	sess.AddPendingQuestion("q-1", session.PendingHumanQuestion{
		Prompt:     "继续吗？",
		ToolCallID: "call-ask-stream",
		TraceID:    "trace-ask-stream",
	})
	if err := sessionStore.Save(sess); err != nil {
		t.Fatalf("save session: %v", err)
	}

	recorder := serveRequest(
		handler,
		http.MethodPost,
		"/api/questions/answer/stream",
		`{"session_id":"session-question-answer-stream","question_id":"q-1","answer":"继续"}`,
		map[string]string{"Content-Type": "application/json"},
	)
	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status: got %d want %d body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	events := decodeSSEEvents(t, recorder)
	if len(events) != 3 {
		t.Fatalf("unexpected event count: got %d want %d", len(events), 3)
	}
	if events[0].Type != streaming.EventToolCallStarted {
		t.Fatalf("unexpected first event: got %q want %q", events[0].Type, streaming.EventToolCallStarted)
	}
	if events[1].Type != streaming.EventMessage {
		t.Fatalf("unexpected second event: got %q want %q", events[1].Type, streaming.EventMessage)
	}
	if events[2].Type != streaming.EventDone {
		t.Fatalf("unexpected third event: got %q want %q", events[2].Type, streaming.EventDone)
	}
}
