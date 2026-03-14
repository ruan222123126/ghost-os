package transport

import (
	"net/http"
	"testing"

	"ghost-os/bridge/session"
)

func TestBusHumanResponseStoresAnswer(t *testing.T) {
	handler, sessionStore := newTestHandlerWithStore(t, nil)
	sess := session.NewSession("system")
	sess.ID = "session-human-response"
	sess.AddPendingQuestion("q-1", session.PendingHumanQuestion{
		Prompt:     "Which database should we use?",
		ToolCallID: "call-ask-1",
		TraceID:    "trace-ask",
	})
	if err := sessionStore.Save(sess); err != nil {
		t.Fatalf("save session: %v", err)
	}

	recorder := serveRequest(
		handler,
		http.MethodPost,
		"/api/bus",
		`{"action":"HUMAN_RESPONSE","params":{"session_id":"session-human-response","question_id":"q-1","answer":"PostgreSQL"},"trace_id":"trace-human-response"}`,
		map[string]string{"Content-Type": "application/json"},
	)
	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status: got %d want %d", recorder.Code, http.StatusOK)
	}

	loaded, err := sessionStore.Load(sess.ID)
	if err != nil {
		t.Fatalf("load session: %v", err)
	}
	if loaded.HumanAnswers["q-1"] != "PostgreSQL" {
		t.Fatalf("unexpected stored answer: %+v", loaded.HumanAnswers)
	}
}
