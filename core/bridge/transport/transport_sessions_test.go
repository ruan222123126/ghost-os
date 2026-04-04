package transport

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ghost-os/bridge/artifacts"
	"ghost-os/bridge/llm"
	"ghost-os/bridge/session"
)

func TestHandleSessionsListReturnsMetadata(t *testing.T) {
	handler, sessionStore := newTestHandlerWithStore(t, nil)

	first := session.NewSession("system")
	first.AddMessage(llm.Message{Role: llm.RoleUser, Text: "hello"})
	if err := sessionStore.Save(first); err != nil {
		t.Fatalf("save first session: %v", err)
	}

	second := session.NewSession("system")
	second.AddMessage(llm.Message{Role: llm.RoleUser, Text: "task"})
	second.AddMessage(llm.Message{Role: llm.RoleAssistant, Text: "done"})
	if err := sessionStore.Save(second); err != nil {
		t.Fatalf("save second session: %v", err)
	}

	recorder := serveRequest(handler, http.MethodGet, "/api/sessions", "", nil)
	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status: got %d want %d", recorder.Code, http.StatusOK)
	}

	body := decodeResponseBody(t, recorder)
	payload, ok := body.Payload.([]any)
	if !ok {
		t.Fatalf("unexpected payload type: %T", body.Payload)
	}
	if len(payload) != 2 {
		t.Fatalf("unexpected session count: got %d want %d", len(payload), 2)
	}

	seen := make(map[string]bool, len(payload))
	for _, item := range payload {
		entry, ok := item.(map[string]any)
		if !ok {
			t.Fatalf("unexpected metadata entry type: %T", item)
		}

		id, _ := entry["id"].(string)
		if strings.TrimSpace(id) == "" {
			t.Fatalf("metadata id should not be empty")
		}
		seen[id] = true

		if strings.TrimSpace(entry["created_at"].(string)) == "" {
			t.Fatalf("created_at should not be empty")
		}
		if strings.TrimSpace(entry["updated_at"].(string)) == "" {
			t.Fatalf("updated_at should not be empty")
		}
		if entry["token_count"].(float64) <= 0 {
			t.Fatalf("token_count should be positive, got %v", entry["token_count"])
		}
	}

	if !seen[first.ID] || !seen[second.ID] {
		t.Fatalf("missing sessions in list: seen=%v first=%q second=%q", seen, first.ID, second.ID)
	}
}

func TestHandleSessionGetReturnsDetails(t *testing.T) {
	handler, sessionStore := newTestHandlerWithStore(t, nil)
	sess := session.NewSession("system")
	sess.AddMessage(llm.Message{Role: llm.RoleUser, Text: "what is status"})
	sess.AddMessage(llm.Message{Role: llm.RoleAssistant, Text: "all good"})
	if err := sessionStore.Save(sess); err != nil {
		t.Fatalf("save session: %v", err)
	}

	recorder := serveRequest(handler, http.MethodGet, "/api/sessions/"+sess.ID, "", nil)
	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status: got %d want %d", recorder.Code, http.StatusOK)
	}

	body := decodeResponseBody(t, recorder)
	payload, ok := body.Payload.(map[string]any)
	if !ok {
		t.Fatalf("unexpected payload type: %T", body.Payload)
	}
	if payload["id"] != sess.ID {
		t.Fatalf("unexpected id: got %v want %q", payload["id"], sess.ID)
	}

	messages, ok := payload["messages"].([]any)
	if !ok {
		t.Fatalf("unexpected messages type: %T", payload["messages"])
	}
	if len(messages) != len(sess.Messages) {
		t.Fatalf("unexpected message count: got %d want %d", len(messages), len(sess.Messages))
	}
	firstMessage, ok := messages[0].(map[string]any)
	if !ok {
		t.Fatalf("unexpected first message type: %T", messages[0])
	}
	if firstMessage["role"] != string(llm.RoleSystem) {
		t.Fatalf("unexpected first message role: got %v want %q", firstMessage["role"], llm.RoleSystem)
	}
	if _, hasLegacy := firstMessage["Role"]; hasLegacy {
		t.Fatal("session detail should not expose legacy Role field")
	}
	secondMessage, ok := messages[1].(map[string]any)
	if !ok {
		t.Fatalf("unexpected second message type: %T", messages[1])
	}
	if secondMessage["role"] != string(llm.RoleUser) {
		t.Fatalf("unexpected second message role: got %v want %q", secondMessage["role"], llm.RoleUser)
	}
	if secondMessage["text"] != "what is status" {
		t.Fatalf("unexpected second message text: got %v want %q", secondMessage["text"], "what is status")
	}
}

func TestHandleSessionGetNotFound(t *testing.T) {
	handler := newTestHandler(t, nil)
	recorder := serveRequest(handler, http.MethodGet, "/api/sessions/session-missing", "", nil)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("unexpected status: got %d want %d", recorder.Code, http.StatusNotFound)
	}
}

func TestHandleSessionGetSupportsWindowQueries(t *testing.T) {
	handler, sessionStore := newTestHandlerWithStore(t, nil)
	sess := session.NewSession("")
	sess.ID = "session-window"
	for i := 0; i < 5; i++ {
		sess.AddMessage(llm.Message{Role: llm.RoleUser, Text: strings.Repeat("x", i+1)})
	}
	if err := sessionStore.Save(sess); err != nil {
		t.Fatalf("save session: %v", err)
	}

	recorder := serveRequest(handler, http.MethodGet, "/api/sessions/"+sess.ID+"?limit=2&before=4", "", nil)
	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status: got %d want %d body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	body := decodeResponseBody(t, recorder)
	payload, ok := body.Payload.(map[string]any)
	if !ok {
		t.Fatalf("unexpected payload type: %T", body.Payload)
	}
	if payload["message_count"] != float64(5) {
		t.Fatalf("unexpected message_count: %v", payload["message_count"])
	}

	messages, ok := payload["messages"].([]any)
	if !ok {
		t.Fatalf("unexpected messages type: %T", payload["messages"])
	}
	if len(messages) != 2 {
		t.Fatalf("unexpected page size: got %d want 2", len(messages))
	}
	firstMessage := messages[0].(map[string]any)
	secondMessage := messages[1].(map[string]any)
	if firstMessage["index"] != float64(2) || secondMessage["index"] != float64(3) {
		t.Fatalf("unexpected message indexes: first=%v second=%v", firstMessage["index"], secondMessage["index"])
	}

	page, ok := payload["page"].(map[string]any)
	if !ok {
		t.Fatalf("unexpected page type: %T", payload["page"])
	}
	if page["limit"] != float64(2) {
		t.Fatalf("unexpected limit: %v", page["limit"])
	}
	if page["before"] != float64(4) {
		t.Fatalf("unexpected before: %v", page["before"])
	}
	if page["next_before"] != float64(2) {
		t.Fatalf("unexpected next_before: %v", page["next_before"])
	}
}

func TestHandleSessionGetRejectsInvalidWindowQueries(t *testing.T) {
	handler := newTestHandler(t, nil)

	limitResp := serveRequest(handler, http.MethodGet, "/api/sessions/session-1?limit=0", "", nil)
	if limitResp.Code != http.StatusBadRequest {
		t.Fatalf("unexpected limit status: got %d want %d", limitResp.Code, http.StatusBadRequest)
	}
	if !strings.Contains(limitResp.Body.String(), "limit must be an integer between 1 and 200") {
		t.Fatalf("unexpected limit error: %s", limitResp.Body.String())
	}

	beforeResp := serveRequest(handler, http.MethodGet, "/api/sessions/session-1?before=-1", "", nil)
	if beforeResp.Code != http.StatusBadRequest {
		t.Fatalf("unexpected before status: got %d want %d", beforeResp.Code, http.StatusBadRequest)
	}
	if !strings.Contains(beforeResp.Body.String(), "before must be a non-negative integer") {
		t.Fatalf("unexpected before error: %s", beforeResp.Body.String())
	}
}

func TestHandleSessionDeleteRemovesSession(t *testing.T) {
	handler, sessionStore := newTestHandlerWithStore(t, nil)
	sess := session.NewSession("system")
	sess.AddMessage(llm.Message{Role: llm.RoleUser, Text: "cleanup"})
	if err := sessionStore.Save(sess); err != nil {
		t.Fatalf("save session: %v", err)
	}

	deleteResp := serveRequest(handler, http.MethodDelete, "/api/sessions/"+sess.ID, "", nil)
	if deleteResp.Code != http.StatusOK {
		t.Fatalf("unexpected delete status: got %d want %d", deleteResp.Code, http.StatusOK)
	}

	getResp := serveRequest(handler, http.MethodGet, "/api/sessions/"+sess.ID, "", nil)
	if getResp.Code != http.StatusNotFound {
		t.Fatalf("unexpected get status after delete: got %d want %d", getResp.Code, http.StatusNotFound)
	}
}

func TestHandleSessionDeleteNotFound(t *testing.T) {
	handler := newTestHandler(t, nil)
	recorder := serveRequest(handler, http.MethodDelete, "/api/sessions/session-missing", "", nil)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("unexpected status: got %d want %d", recorder.Code, http.StatusNotFound)
	}
}

func TestHandleSessionArtifactDownloadReturnsBinaryFile(t *testing.T) {
	artifactDir := t.TempDir()
	t.Setenv("GHOST_ARTIFACTS_PATH", artifactDir)
	handler := newTestHandler(t, nil)

	store, err := artifacts.NewSessionArtifactStoreFromEnv()
	if err != nil {
		t.Fatalf("new artifact store: %v", err)
	}
	sessionDir, err := store.SessionDir("session-artifact")
	if err != nil {
		t.Fatalf("session dir: %v", err)
	}
	if err := os.MkdirAll(sessionDir, 0o700); err != nil {
		t.Fatalf("mkdir artifact dir: %v", err)
	}
	filePath := filepath.Join(sessionDir, "artifact-1.txt")
	if err := os.WriteFile(filePath, []byte("hello attachment"), 0o600); err != nil {
		t.Fatalf("write artifact file: %v", err)
	}
	if err := store.WriteMetadata(artifacts.SessionFileArtifact{
		ArtifactID:  "artifact-1",
		SessionID:   "session-artifact",
		Name:        "report.txt",
		MimeType:    "text/plain",
		Bytes:       int64(len("hello attachment")),
		SHA256:      "abc123",
		DownloadURL: "/api/sessions/session-artifact/artifacts/artifact-1",
		SourcePath:  "/tmp/report.txt",
		StoredPath:  filePath,
	}); err != nil {
		t.Fatalf("write artifact metadata: %v", err)
	}

	recorder := serveRequest(handler, http.MethodGet, "/api/sessions/session-artifact/artifacts/artifact-1", "", nil)
	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status: got %d want %d body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if got := recorder.Header().Get("Content-Type"); got != "text/plain" {
		t.Fatalf("unexpected content-type: got %q want %q", got, "text/plain")
	}
	if got := recorder.Header().Get("Content-Disposition"); !strings.Contains(got, `filename="report.txt"`) {
		t.Fatalf("unexpected content-disposition: %q", got)
	}
	if recorder.Body.String() != "hello attachment" {
		t.Fatalf("unexpected body: %q", recorder.Body.String())
	}
}
