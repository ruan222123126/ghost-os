package app

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"ghost-os/bridge/agent"
	"ghost-os/bridge/llm"
	"ghost-os/bridge/session"
)

func decodeResponseBody(t *testing.T, recorder *httptest.ResponseRecorder) apiResponse {
	t.Helper()

	var body apiResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v, body=%s", err, recorder.Body.String())
	}
	return body
}

func newTestHandler(t *testing.T, executor agentExecutorFunc) http.Handler {
	t.Helper()
	handler, _ := newTestHandlerWithStore(t, executor)
	return handler
}

func newTestHandlerWithStore(t *testing.T, executor agentExecutorFunc) (http.Handler, *session.Store) {
	t.Helper()
	if executor == nil {
		executor = func(_ context.Context, _ string, _ string, _ string, _ *ConfigStore, _ *session.Store) (string, string, error) {
			return "ok", "session-test", nil
		}
	}

	store, err := NewConfigStoreFromEnv()
	if err != nil {
		t.Fatalf("new config store: %v", err)
	}

	sessionStore, err := session.NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("new session store: %v", err)
	}

	service := newBridgeService(store, sessionStore, executor)
	options := newServerOptionsFromEnv(8080)
	options.maxBodyBytes = defaultMaxRequestBodyBytes
	return newHTTPHandler(service, options), sessionStore
}

func serveRequest(handler http.Handler, method string, path string, body string, headers map[string]string) *httptest.ResponseRecorder {
	var reader *strings.Reader
	if body == "" {
		reader = strings.NewReader("")
	} else {
		reader = strings.NewReader(body)
	}

	request := httptest.NewRequest(method, path, reader)
	for key, value := range headers {
		request.Header.Set(key, value)
	}

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	return recorder
}

func TestResolveBindAddrDefaultsToLocalhost(t *testing.T) {
	t.Setenv("GHOST_BIND_ADDR", "")
	if got, want := resolveBindAddr(8080), "127.0.0.1:8080"; got != want {
		t.Fatalf("unexpected bind addr: got %q want %q", got, want)
	}
}

func TestResolveBindAddrUsesOverride(t *testing.T) {
	t.Setenv("GHOST_BIND_ADDR", "0.0.0.0:9090")
	if got, want := resolveBindAddr(8080), "0.0.0.0:9090"; got != want {
		t.Fatalf("unexpected bind addr: got %q want %q", got, want)
	}
}

func TestHandleBusMethodNotAllowed(t *testing.T) {
	handler := newTestHandler(t, nil)
	recorder := serveRequest(handler, http.MethodGet, "/api/bus", "", nil)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("unexpected status: got %d want %d", recorder.Code, http.StatusMethodNotAllowed)
	}
	body := decodeResponseBody(t, recorder)
	if body.Status != "error" {
		t.Fatalf("unexpected status field: got %q want %q", body.Status, "error")
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

func TestHandleConfigMethodNotAllowed(t *testing.T) {
	handler := newTestHandler(t, nil)
	recorder := serveRequest(handler, http.MethodPut, "/api/config", "", nil)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("unexpected status: got %d want %d", recorder.Code, http.StatusMethodNotAllowed)
	}
}

func TestHandleBusMissingFields(t *testing.T) {
	handler := newTestHandler(t, nil)
	recorder := serveRequest(handler, http.MethodPost, "/api/bus", `{"params":{},"trace_id":"trace-1"}`, nil)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status: got %d want %d", recorder.Code, http.StatusBadRequest)
	}
	body := decodeResponseBody(t, recorder)
	if body.Error != "action is required" {
		t.Fatalf("unexpected error: got %q want %q", body.Error, "action is required")
	}
}

func TestHandleBusInvalidAction(t *testing.T) {
	handler := newTestHandler(t, nil)
	recorder := serveRequest(handler, http.MethodPost, "/api/bus", `{"action":"UNKNOWN","params":{},"trace_id":"trace-unknown"}`, nil)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("unexpected status: got %d want %d", recorder.Code, http.StatusBadRequest)
	}
	if recorder.Header().Get("X-Trace-ID") != "trace-unknown" {
		t.Fatalf("missing trace header: got %q want %q", recorder.Header().Get("X-Trace-ID"), "trace-unknown")
	}
	body := decodeResponseBody(t, recorder)
	if !strings.Contains(body.Error, "unsupported action") {
		t.Fatalf("unexpected error: %q", body.Error)
	}
}

func TestAuthMissingOrWrongTokenReturnsUnauthorized(t *testing.T) {
	t.Setenv("GHOST_API_TOKEN", "secret-token")
	handler := newTestHandler(t, nil)

	missing := serveRequest(handler, http.MethodGet, "/api/config", "", nil)
	if missing.Code != http.StatusUnauthorized {
		t.Fatalf("unexpected status without token: got %d want %d", missing.Code, http.StatusUnauthorized)
	}

	wrong := serveRequest(handler, http.MethodGet, "/api/config", "", map[string]string{"X-API-Token": "wrong"})
	if wrong.Code != http.StatusUnauthorized {
		t.Fatalf("unexpected status with wrong token: got %d want %d", wrong.Code, http.StatusUnauthorized)
	}
}

func TestAuthAcceptsXAPITokenAndBearer(t *testing.T) {
	t.Setenv("GHOST_API_TOKEN", "secret-token")
	handler := newTestHandler(t, nil)

	xToken := serveRequest(handler, http.MethodGet, "/api/config", "", map[string]string{"X-API-Token": "secret-token"})
	if xToken.Code != http.StatusOK {
		t.Fatalf("unexpected status with X-API-Token: got %d want %d", xToken.Code, http.StatusOK)
	}

	bearer := serveRequest(handler, http.MethodGet, "/api/config", "", map[string]string{"Authorization": "Bearer secret-token"})
	if bearer.Code != http.StatusOK {
		t.Fatalf("unexpected status with bearer token: got %d want %d", bearer.Code, http.StatusOK)
	}
}

func TestHandleAgentBodyTooLarge(t *testing.T) {
	handler := newTestHandler(t, nil)
	oversized := strings.Repeat("a", int(defaultMaxRequestBodyBytes)+32)
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

func TestWithCORSAllowlist(t *testing.T) {
	t.Setenv("GHOST_CORS_ORIGINS", "https://console.ghost.local,http://localhost:5173")
	handler := newTestHandler(t, nil)

	allowed := serveRequest(handler, http.MethodOptions, "/api/config", "", map[string]string{"Origin": "http://localhost:5173"})
	if allowed.Code != http.StatusNoContent {
		t.Fatalf("unexpected status for allowed origin: got %d want %d", allowed.Code, http.StatusNoContent)
	}
	if got := allowed.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:5173" {
		t.Fatalf("unexpected allow origin: got %q want %q", got, "http://localhost:5173")
	}

	forbidden := serveRequest(handler, http.MethodOptions, "/api/config", "", map[string]string{"Origin": "https://evil.example"})
	if forbidden.Code != http.StatusForbidden {
		t.Fatalf("unexpected status for disallowed origin: got %d want %d", forbidden.Code, http.StatusForbidden)
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
}

func TestAgentEndpointAliasesBusDispatch(t *testing.T) {
	handler := newTestHandler(t, func(_ context.Context, message string, sessionID string, _ string, _ *ConfigStore, _ *session.Store) (string, string, error) {
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
	handler := newTestHandler(t, func(_ context.Context, message string, requestSessionID string, _ string, _ *ConfigStore, _ *session.Store) (string, string, error) {
		if message != "hello" {
			t.Fatalf("unexpected message: got %q want %q", message, "hello")
		}
		if requestSessionID != sessionID {
			t.Fatalf("unexpected session_id: got %q want %q", requestSessionID, sessionID)
		}
		return "ok", requestSessionID, nil
	})

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

func TestAgentEndpointAllowsEmptyMessageWhenSessionIDIsPresent(t *testing.T) {
	const sessionID = "session-continue-1"
	handler := newTestHandler(t, func(_ context.Context, message string, requestSessionID string, _ string, _ *ConfigStore, _ *session.Store) (string, string, error) {
		if message != "" {
			t.Fatalf("unexpected message: got %q want empty", message)
		}
		if requestSessionID != sessionID {
			t.Fatalf("unexpected session_id: got %q want %q", requestSessionID, sessionID)
		}
		return "continued", requestSessionID, nil
	})

	recorder := serveRequest(
		handler,
		http.MethodPost,
		"/api/agent",
		fmt.Sprintf(`{"message":"","session_id":"%s"}`, sessionID),
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
	if payload["message"] != "continued" {
		t.Fatalf("unexpected message: got %v want %q", payload["message"], "continued")
	}
	if payload["session_id"] != sessionID {
		t.Fatalf("unexpected session_id: got %v want %q", payload["session_id"], sessionID)
	}
}

func TestAgentEndpointReturnsBadRequestOnInvalidSessionID(t *testing.T) {
	handler := newTestHandler(t, func(_ context.Context, _ string, _ string, _ string, _ *ConfigStore, _ *session.Store) (string, string, error) {
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
	handler := newTestHandler(t, func(_ context.Context, _ string, _ string, _ string, _ *ConfigStore, _ *session.Store) (string, string, error) {
		return "", "session-awaiting-1", &agent.ErrAwaitingHuman{
			QuestionID: "q-awaiting-1",
			Prompt:     "Which database should we use?",
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
}

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

func TestBusConfigGetAndUpdateRegression(t *testing.T) {
	t.Setenv("GHOST_PROVIDER", "custom")
	t.Setenv("GHOST_BASE_URL", "http://127.0.0.1:9999")
	t.Setenv("GHOST_MODEL", "test-model")
	t.Setenv("GHOST_CHAT_PATH", "/v1/chat")
	t.Setenv("GHOST_API_KEY", "secret")

	handler := newTestHandler(t, nil)

	getResp := serveRequest(handler, http.MethodPost, "/api/bus", `{"action":"CONFIG_GET","params":{},"trace_id":"trace-config-get"}`, nil)
	if getResp.Code != http.StatusOK {
		t.Fatalf("unexpected status for config get: got %d want %d", getResp.Code, http.StatusOK)
	}
	getBody := decodeResponseBody(t, getResp)
	payload, ok := getBody.Payload.(map[string]any)
	if !ok {
		t.Fatalf("unexpected payload type: %T", getBody.Payload)
	}
	if payload["provider"] != "custom" {
		t.Fatalf("unexpected provider: got %v want %q", payload["provider"], "custom")
	}
	if payload["api_key_set"] != true {
		t.Fatalf("unexpected api_key_set: got %v want true", payload["api_key_set"])
	}

	updateResp := serveRequest(handler, http.MethodPost, "/api/bus", `{"action":"CONFIG_UPDATE","params":{"provider":"custom","model":"local"},"trace_id":"trace-config-update"}`, nil)
	if updateResp.Code != http.StatusOK {
		t.Fatalf("unexpected status for config update: got %d want %d", updateResp.Code, http.StatusOK)
	}
	updateBody := decodeResponseBody(t, updateResp)
	updatePayload, ok := updateBody.Payload.(map[string]any)
	if !ok {
		t.Fatalf("unexpected update payload type: %T", updateBody.Payload)
	}
	if updatePayload["model"] != "local" {
		t.Fatalf("unexpected model: got %v want %q", updatePayload["model"], "local")
	}
}

func TestConfigUpdateThenGetUsesStore(t *testing.T) {
	handler := newTestHandler(t, nil)

	update := serveRequest(handler, http.MethodPost, "/api/config", `{"provider":"custom","api_key":"new-key","base_url":"http://localhost:1234","model":"local-model","chat_path":"/v1/messages"}`, nil)
	if update.Code != http.StatusOK {
		t.Fatalf("unexpected update status: got %d want %d", update.Code, http.StatusOK)
	}

	get := serveRequest(handler, http.MethodGet, "/api/config", "", nil)
	if get.Code != http.StatusOK {
		t.Fatalf("unexpected get status: got %d want %d", get.Code, http.StatusOK)
	}
	body := decodeResponseBody(t, get)
	payload, ok := body.Payload.(map[string]any)
	if !ok {
		t.Fatalf("unexpected payload type: %T", body.Payload)
	}
	if payload["provider"] != "custom" {
		t.Fatalf("unexpected provider: got %v want %q", payload["provider"], "custom")
	}
	if payload["base_url"] != "http://localhost:1234" {
		t.Fatalf("unexpected base_url: got %v want %q", payload["base_url"], "http://localhost:1234")
	}
	if payload["model"] != "local-model" {
		t.Fatalf("unexpected model: got %v want %q", payload["model"], "local-model")
	}
	if payload["chat_path"] != "/v1/messages" {
		t.Fatalf("unexpected chat_path: got %v want %q", payload["chat_path"], "/v1/messages")
	}
	if payload["api_key_set"] != true {
		t.Fatalf("unexpected api_key_set: got %v want true", payload["api_key_set"])
	}
}

func TestConfigUpdateEmptyBaseURLAndModelResetDefaults(t *testing.T) {
	handler := newTestHandler(t, nil)

	update := serveRequest(handler, http.MethodPost, "/api/config", `{"base_url":"","model":""}`, nil)
	if update.Code != http.StatusOK {
		t.Fatalf("unexpected update status: got %d want %d", update.Code, http.StatusOK)
	}

	get := serveRequest(handler, http.MethodGet, "/api/config", "", nil)
	if get.Code != http.StatusOK {
		t.Fatalf("unexpected get status: got %d want %d", get.Code, http.StatusOK)
	}
	body := decodeResponseBody(t, get)
	payload, ok := body.Payload.(map[string]any)
	if !ok {
		t.Fatalf("unexpected payload type: %T", body.Payload)
	}
	if payload["base_url"] != defaultBaseURL {
		t.Fatalf("unexpected base_url: got %v want %q", payload["base_url"], defaultBaseURL)
	}
	if payload["model"] != defaultModel {
		t.Fatalf("unexpected model: got %v want %q", payload["model"], defaultModel)
	}
}

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
}

func TestHandleSessionGetNotFound(t *testing.T) {
	handler := newTestHandler(t, nil)
	recorder := serveRequest(handler, http.MethodGet, "/api/sessions/session-missing", "", nil)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("unexpected status: got %d want %d", recorder.Code, http.StatusNotFound)
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

func TestBusMemoryArchiveAndQuery(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("GHOST_MEMORY_WARM_PATH", tempDir+"/warm.json")
	t.Setenv("GHOST_MEMORY_COLD_PATH", tempDir+"/cold")

	handler, sessionStore := newTestHandlerWithStore(t, nil)
	sess := session.NewSession("system")
	sess.ID = "session-memory-test"
	sess.AddMessage(llm.Message{Role: llm.RoleUser, Text: "remember this fact"})
	if err := sessionStore.Save(sess); err != nil {
		t.Fatalf("save session: %v", err)
	}

	archiveResp := serveRequest(
		handler,
		http.MethodPost,
		"/api/bus",
		`{"action":"MEMORY_ARCHIVE","params":{"session_id":"session-memory-test"},"trace_id":"trace-memory-archive"}`,
		nil,
	)
	if archiveResp.Code != http.StatusOK {
		t.Fatalf("unexpected archive status: got %d want %d body=%s", archiveResp.Code, http.StatusOK, archiveResp.Body.String())
	}

	queryResp := serveRequest(
		handler,
		http.MethodPost,
		"/api/bus",
		`{"action":"MEMORY_QUERY","params":{"session_id":"session-memory-test","keywords":["remember"]},"trace_id":"trace-memory-query"}`,
		nil,
	)
	if queryResp.Code != http.StatusOK {
		t.Fatalf("unexpected query status: got %d want %d body=%s", queryResp.Code, http.StatusOK, queryResp.Body.String())
	}

	body := decodeResponseBody(t, queryResp)
	payload, ok := body.Payload.(map[string]any)
	if !ok {
		t.Fatalf("unexpected payload type: %T", body.Payload)
	}
	entries, ok := payload["entries"].([]any)
	if !ok {
		t.Fatalf("unexpected entries type: %T", payload["entries"])
	}
	if len(entries) == 0 {
		t.Fatal("expected at least one memory entry")
	}
}
