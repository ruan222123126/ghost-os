package transport

import (
	"net/http"
	"net/http/httptest"
	"net/url"
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

func TestBusSessionsListReturnsMetadata(t *testing.T) {
	handler, sessionStore := newTestHandlerWithStore(t, nil)

	sess := session.NewSession("system")
	sess.AddMessage(llm.Message{Role: llm.RoleUser, Text: "hello from bus"})
	if err := sessionStore.Save(sess); err != nil {
		t.Fatalf("save session: %v", err)
	}

	recorder := serveRequest(
		handler,
		http.MethodPost,
		"/api/bus",
		`{"action":"SESSIONS_LIST","params":{},"trace_id":"trace-sessions-list"}`,
		map[string]string{"Content-Type": "application/json"},
	)
	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status: got %d want %d body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	body := decodeResponseBody(t, recorder)
	payload, ok := body.Payload.([]any)
	if !ok {
		t.Fatalf("unexpected payload type: %T", body.Payload)
	}
	if len(payload) != 1 {
		t.Fatalf("unexpected session count: got %d want 1", len(payload))
	}
	entry, ok := payload[0].(map[string]any)
	if !ok {
		t.Fatalf("unexpected metadata entry type: %T", payload[0])
	}
	if entry["id"] != sess.ID {
		t.Fatalf("unexpected session id: got %v want %q", entry["id"], sess.ID)
	}
}

func TestBusSessionGetReturnsDetail(t *testing.T) {
	handler, sessionStore := newTestHandlerWithStore(t, nil)

	sess := session.NewSession("system")
	sess.ID = "session-bus-get"
	for i := 0; i < 3; i++ {
		sess.AddMessage(llm.Message{Role: llm.RoleUser, Text: strings.Repeat("x", i+1)})
	}
	if err := sessionStore.Save(sess); err != nil {
		t.Fatalf("save session: %v", err)
	}

	recorder := serveRequest(
		handler,
		http.MethodPost,
		"/api/bus",
		`{"action":"SESSION_GET","params":{"id":"session-bus-get","limit":2},"trace_id":"trace-session-get"}`,
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
	if payload["id"] != sess.ID {
		t.Fatalf("unexpected session id: got %v want %q", payload["id"], sess.ID)
	}
	messages, ok := payload["messages"].([]any)
	if !ok {
		t.Fatalf("unexpected messages type: %T", payload["messages"])
	}
	if len(messages) != 2 {
		t.Fatalf("unexpected message page size: got %d want 2", len(messages))
	}
	page, ok := payload["page"].(map[string]any)
	if !ok {
		t.Fatalf("unexpected page type: %T", payload["page"])
	}
	if page["limit"] != float64(2) {
		t.Fatalf("unexpected page limit: got %v want 2", page["limit"])
	}
}

func TestBusSessionGetRejectsInvalidParams(t *testing.T) {
	handler := newTestHandler(t, nil)

	emptyID := serveRequest(
		handler,
		http.MethodPost,
		"/api/bus",
		`{"action":"SESSION_GET","params":{"id":"","limit":100},"trace_id":"trace-session-empty"}`,
		map[string]string{"Content-Type": "application/json"},
	)
	if emptyID.Code != http.StatusBadRequest {
		t.Fatalf("unexpected empty id status: got %d want %d body=%s", emptyID.Code, http.StatusBadRequest, emptyID.Body.String())
	}

	unknownField := serveRequest(
		handler,
		http.MethodPost,
		"/api/bus",
		`{"action":"SESSION_GET","params":{"id":"session-1","limit":100,"extra":true},"trace_id":"trace-session-extra"}`,
		map[string]string{"Content-Type": "application/json"},
	)
	if unknownField.Code != http.StatusBadRequest {
		t.Fatalf(
			"unexpected unknown field status: got %d want %d body=%s",
			unknownField.Code,
			http.StatusBadRequest,
			unknownField.Body.String(),
		)
	}
	if !strings.Contains(unknownField.Body.String(), `unknown field \"extra\"`) {
		t.Fatalf("unexpected unknown field error: %s", unknownField.Body.String())
	}
}

func TestHandleSessionsSearchMatchesTitle(t *testing.T) {
	handler, sessionStore := newTestHandlerWithStore(t, nil)

	painting := session.NewSession("system")
	painting.ID = "session-painting"
	painting.Title = "水彩绘画目录"
	painting.AddMessage(llm.Message{Role: llm.RoleUser, Text: "paint"})
	if err := sessionStore.Save(painting); err != nil {
		t.Fatalf("save painting session: %v", err)
	}

	other := session.NewSession("system")
	other.ID = "session-notes"
	other.Title = "会议记录"
	other.AddMessage(llm.Message{Role: llm.RoleUser, Text: "notes"})
	if err := sessionStore.Save(other); err != nil {
		t.Fatalf("save other session: %v", err)
	}

	recorder := serveRequest(handler, http.MethodGet, "/api/sessions/search?q="+url.QueryEscape("绘画"), "", nil)
	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status: got %d want %d body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	body := decodeResponseBody(t, recorder)
	payload, ok := body.Payload.([]any)
	if !ok {
		t.Fatalf("unexpected payload type: %T", body.Payload)
	}
	if len(payload) != 1 {
		t.Fatalf("unexpected search result count: got %d want 1 payload=%v", len(payload), payload)
	}
	result, ok := payload[0].(map[string]any)
	if !ok {
		t.Fatalf("unexpected result type: %T", payload[0])
	}
	if result["id"] != painting.ID || result["title"] != painting.Title {
		t.Fatalf("unexpected result: %+v", result)
	}
}

func TestHandleSessionsSearchMatchesManualPartitionName(t *testing.T) {
	handler, sessionStore := newTestHandlerWithStore(t, nil)

	project := session.NewSession("system")
	project.ID = "session-project"
	project.Title = "Alpha discussion"
	project.AddMessage(llm.Message{Role: llm.RoleUser, Text: "alpha"})
	if err := sessionStore.Save(project); err != nil {
		t.Fatalf("save project session: %v", err)
	}

	review := session.NewSession("system")
	review.ID = "session-review"
	review.Title = "Review notes"
	review.AddMessage(llm.Message{Role: llm.RoleUser, Text: "review"})
	if err := sessionStore.Save(review); err != nil {
		t.Fatalf("save review session: %v", err)
	}

	other := session.NewSession("system")
	other.ID = "session-other"
	other.Title = "Unrelated"
	other.AddMessage(llm.Message{Role: llm.RoleUser, Text: "other"})
	if err := sessionStore.Save(other); err != nil {
		t.Fatalf("save other session: %v", err)
	}

	if _, err := sessionStore.SaveSidebarPartitionState(session.SessionSidebarPartitionState{
		Version: 1,
		Partitions: []session.SessionSidebarPartition{
			{ID: "partition-client", Name: "Client Projects"},
		},
		Assignments: map[string]string{
			project.ID: "partition-client",
			review.ID:  "partition-client",
		},
	}); err != nil {
		t.Fatalf("save sidebar partition state: %v", err)
	}

	recorder := serveRequest(handler, http.MethodGet, "/api/sessions/search?q="+url.QueryEscape("client projects"), "", nil)
	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status: got %d want %d body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	ids := decodeSessionSearchResultIDs(t, recorder)
	if len(ids) != 2 || !ids[project.ID] || !ids[review.ID] {
		t.Fatalf("unexpected partition search ids: %v", ids)
	}
	if ids[other.ID] {
		t.Fatalf("partition search should not include unassigned session: %v", ids)
	}
}

func TestHandleSessionsSearchKeepsIDSearchLimit(t *testing.T) {
	handler, sessionStore := newTestHandlerWithStore(t, nil)

	first := session.NewSession("system")
	first.ID = "session-first"
	first.AddMessage(llm.Message{Role: llm.RoleUser, Text: "first"})
	if err := sessionStore.Save(first); err != nil {
		t.Fatalf("save first session: %v", err)
	}

	second := session.NewSession("system")
	second.ID = "session-second"
	second.AddMessage(llm.Message{Role: llm.RoleUser, Text: "second"})
	if err := sessionStore.Save(second); err != nil {
		t.Fatalf("save second session: %v", err)
	}

	recorder := serveRequest(handler, http.MethodGet, "/api/sessions/search?q=session&limit=1", "", nil)
	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status: got %d want %d body=%s", recorder.Code, http.StatusOK, recorder.Body.String())
	}

	ids := decodeSessionSearchResultIDs(t, recorder)
	if len(ids) != 1 {
		t.Fatalf("unexpected limited search result count: got %d ids=%v", len(ids), ids)
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

func TestHandleSessionGetIncludesAssistantDraftForLatestWindow(t *testing.T) {
	handler, sessionStore := newTestHandlerWithStore(t, nil)
	sess := session.NewSession("system")
	sess.ID = "session-with-draft"
	sess.AddMessage(llm.Message{Role: llm.RoleUser, Text: "hello"})
	sess.AssistantDraft = &session.AssistantDraft{
		Text:    "partial answer",
		TraceID: "trace-draft",
		Turn:    1,
	}
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
	messages, ok := payload["messages"].([]any)
	if !ok {
		t.Fatalf("unexpected messages type: %T", payload["messages"])
	}
	if len(messages) != len(sess.Messages)+1 {
		t.Fatalf("expected committed messages + draft, got %d", len(messages))
	}
	draft, ok := messages[len(messages)-1].(map[string]any)
	if !ok {
		t.Fatalf("unexpected draft message type: %T", messages[len(messages)-1])
	}
	if draft["role"] != string(llm.RoleAssistant) {
		t.Fatalf("unexpected draft role: %v", draft["role"])
	}
	if draft["text"] != "partial answer" {
		t.Fatalf("unexpected draft text: %v", draft["text"])
	}
	if draft["in_progress"] != true {
		t.Fatalf("unexpected draft in_progress: %v", draft["in_progress"])
	}
	if payload["message_count"] != float64(len(sess.Messages)) {
		t.Fatalf("unexpected message_count: got %v want %d", payload["message_count"], len(sess.Messages))
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

func decodeSessionSearchResultIDs(t *testing.T, recorder *httptest.ResponseRecorder) map[string]bool {
	t.Helper()
	body := decodeResponseBody(t, recorder)
	payload, ok := body.Payload.([]any)
	if !ok {
		t.Fatalf("unexpected payload type: %T", body.Payload)
	}
	ids := make(map[string]bool, len(payload))
	for _, item := range payload {
		entry, ok := item.(map[string]any)
		if !ok {
			t.Fatalf("unexpected metadata entry type: %T", item)
		}
		id, _ := entry["id"].(string)
		if strings.TrimSpace(id) == "" {
			t.Fatalf("metadata id should not be empty: %v", entry)
		}
		ids[id] = true
	}
	return ids
}
