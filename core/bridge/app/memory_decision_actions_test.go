package app

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"ghost-os/bridge/agent"
	"ghost-os/bridge/llm"
	"ghost-os/bridge/memory"
	"ghost-os/bridge/session"
)

func TestBusMemoryDecisionStats(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("GHOST_MEMORY_DECISION_ENABLED", "true")
	t.Setenv("GHOST_MEMORY_DECISION_PATH", tempDir+"/decision")
	t.Setenv("GHOST_MEMORY_DECISION_CAPTURE_ON_TURN", "true")
	t.Setenv("GHOST_MEMORY_DECISION_RECIPE_ENABLED", "true")
	t.Setenv("GHOST_MEMORY_DECISION_RECIPE_MIN_SUPPORT", "2")

	handler, service, _ := newTestHandlerWithService(t, nil, nil)
	captureDecisionTestTurn(t, service, "workspace:test", "fix config migration", "turn-a")
	captureDecisionTestTurn(t, service, "workspace:test", "fix config migration", "turn-b")
	if _, err := service.memoryManager.DistillDecisionRecipes("workspace:test"); err != nil {
		t.Fatalf("distill decision recipes: %v", err)
	}

	resp := serveRequest(
		handler,
		http.MethodPost,
		"/api/bus",
		`{"action":"MEMORY_DECISION_STATS","params":{"namespace":"workspace:test"},"trace_id":"trace-decision-stats"}`,
		nil,
	)
	if resp.Code != http.StatusOK {
		t.Fatalf("unexpected status: got %d want %d body=%s", resp.Code, http.StatusOK, resp.Body.String())
	}
	body := decodeResponseBody(t, resp)
	payload, ok := body.Payload.(map[string]any)
	if !ok {
		t.Fatalf("unexpected payload type: %T", body.Payload)
	}
	if got := int(payload["memo_count"].(float64)); got < 2 {
		t.Fatalf("expected memo_count >= 2, got %d payload=%#v", got, payload)
	}
	if got := int(payload["recipe_count"].(float64)); got < 1 {
		t.Fatalf("expected recipe_count >= 1, got %d payload=%#v", got, payload)
	}
}

func TestBusMemoryDecisionQuery(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("GHOST_MEMORY_DECISION_ENABLED", "true")
	t.Setenv("GHOST_MEMORY_DECISION_PATH", tempDir+"/decision")
	t.Setenv("GHOST_MEMORY_DECISION_CAPTURE_ON_TURN", "true")

	handler, service, _ := newTestHandlerWithService(t, nil, nil)
	captureDecisionTestTurn(t, service, "workspace:test", "fix config migration", "turn-query")

	resp := serveRequest(
		handler,
		http.MethodPost,
		"/api/bus",
		`{"action":"MEMORY_DECISION_QUERY","params":{"namespace":"workspace:test","semantic_query":"fix config migration"},"trace_id":"trace-decision-query"}`,
		nil,
	)
	if resp.Code != http.StatusOK {
		t.Fatalf("unexpected status: got %d want %d body=%s", resp.Code, http.StatusOK, resp.Body.String())
	}
	body := decodeResponseBody(t, resp)
	payload, ok := body.Payload.(map[string]any)
	if !ok {
		t.Fatalf("unexpected payload type: %T", body.Payload)
	}
	decisionHits, ok := payload["decision_hits"].([]any)
	if !ok || len(decisionHits) == 0 {
		t.Fatalf("expected decision hits, got %#v", payload["decision_hits"])
	}
	entries, ok := payload["entries"].([]any)
	if !ok || len(entries) == 0 {
		t.Fatalf("expected decision entries, got %#v", payload["entries"])
	}
}

func TestBusMemoryDecisionRebuildDryRun(t *testing.T) {
	tempDir := t.TempDir()
	decisionDir := filepath.Join(tempDir, "decision")
	t.Setenv("GHOST_MEMORY_COLD_PATH", tempDir+"/cold")
	t.Setenv("GHOST_MEMORY_DECISION_ENABLED", "true")
	t.Setenv("GHOST_MEMORY_DECISION_PATH", decisionDir)
	t.Setenv("GHOST_MEMORY_DECISION_RECIPE_ENABLED", "true")
	t.Setenv("GHOST_MEMORY_DECISION_RECIPE_MIN_SUPPORT", "2")

	handler, service, sessionStore := newTestHandlerWithService(t, nil, nil)
	seedArchivedDecisionSession(t, handler, sessionStore, "session-decision-dry-run", "fix config migration")

	resp := serveRequest(
		handler,
		http.MethodPost,
		"/api/bus",
		`{"action":"MEMORY_DECISION_REBUILD","params":{"namespace":"workspace:test","dry_run":true,"rebuild_memos":true,"include_recipes":true},"trace_id":"trace-decision-rebuild-dry"}`,
		nil,
	)
	if resp.Code != http.StatusOK {
		t.Fatalf("unexpected status: got %d want %d body=%s", resp.Code, http.StatusOK, resp.Body.String())
	}
	body := decodeResponseBody(t, resp)
	payload, ok := body.Payload.(map[string]any)
	if !ok {
		t.Fatalf("unexpected payload type: %T", body.Payload)
	}
	if got := int(payload["memos_captured"].(float64)); got == 0 {
		t.Fatalf("expected dry run to report captured memos, got %#v", payload)
	}
	if got := service.memoryManager.DecisionStats("workspace:test").MemoCount; got != 0 {
		t.Fatalf("expected dry run not to persist memos, got %d", got)
	}
	if _, err := os.Stat(filepath.Join(decisionDir, "memos.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected no memos snapshot after dry run, got err=%v", err)
	}
}

func TestBusMemoryDecisionRebuildPersistsMemosAndRecipes(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("GHOST_MEMORY_COLD_PATH", tempDir+"/cold")
	t.Setenv("GHOST_MEMORY_DECISION_ENABLED", "true")
	t.Setenv("GHOST_MEMORY_DECISION_PATH", tempDir+"/decision")
	t.Setenv("GHOST_MEMORY_DECISION_RECIPE_ENABLED", "true")
	t.Setenv("GHOST_MEMORY_DECISION_RECIPE_MIN_SUPPORT", "2")

	handler, service, sessionStore := newTestHandlerWithService(t, nil, nil)
	seedArchivedDecisionSession(t, handler, sessionStore, "session-decision-rebuild-a", "fix config migration")
	seedArchivedDecisionSession(t, handler, sessionStore, "session-decision-rebuild-b", "fix config migration")
	seedArchivedDecisionSession(t, handler, sessionStore, "session-decision-rebuild-c", "fix config migration")

	resp := serveRequest(
		handler,
		http.MethodPost,
		"/api/bus",
		`{"action":"MEMORY_DECISION_REBUILD","params":{"namespace":"workspace:test","rebuild_memos":true,"include_recipes":true},"trace_id":"trace-decision-rebuild"}`,
		nil,
	)
	if resp.Code != http.StatusOK {
		t.Fatalf("unexpected status: got %d want %d body=%s", resp.Code, http.StatusOK, resp.Body.String())
	}
	body := decodeResponseBody(t, resp)
	payload, ok := body.Payload.(map[string]any)
	if !ok {
		t.Fatalf("unexpected payload type: %T", body.Payload)
	}
	if got := int(payload["recipes_created"].(float64)); got == 0 {
		t.Fatalf("expected recipes to be created, got %#v", payload)
	}
	stats := service.memoryManager.DecisionStats("workspace:test")
	if stats.MemoCount < 3 || stats.RecipeCount == 0 {
		t.Fatalf("expected persisted decision memos and recipes, got %+v", stats)
	}

	queryResp := serveRequest(
		handler,
		http.MethodPost,
		"/api/bus",
		`{"action":"MEMORY_DECISION_QUERY","params":{"namespace":"workspace:test","semantic_query":"fix config migration","decision_types":["recipe"]},"trace_id":"trace-decision-query-after-rebuild"}`,
		nil,
	)
	if queryResp.Code != http.StatusOK {
		t.Fatalf("unexpected query status: got %d want %d body=%s", queryResp.Code, http.StatusOK, queryResp.Body.String())
	}
	queryBody := decodeResponseBody(t, queryResp)
	queryPayload, ok := queryBody.Payload.(map[string]any)
	if !ok {
		t.Fatalf("unexpected query payload type: %T", queryBody.Payload)
	}
	decisionHits, ok := queryPayload["decision_hits"].([]any)
	if !ok || len(decisionHits) == 0 {
		t.Fatalf("expected recipe hit after rebuild, got %#v", queryPayload["decision_hits"])
	}
}

func captureDecisionTestTurn(t *testing.T, service *bridgeService, namespace string, userMessage string, turnID string) {
	t.Helper()
	startedAt := time.Now().UTC().Add(-2 * time.Minute)
	if err := service.memoryManager.CaptureDecisionTurn(memory.DecisionCaptureInput{
		Namespace:      namespace,
		SessionID:      "session-" + turnID,
		TraceID:        "trace-" + turnID,
		TurnID:         turnID,
		UserMessage:    userMessage,
		Outcome:        memory.DecisionOutcomeSuccess,
		TurnStartedAt:  startedAt,
		TurnFinishedAt: startedAt.Add(time.Minute),
		Environment: memory.DecisionEnvFingerprint{
			WorkspaceRoot:    "/workspace/ghost-os",
			Platform:         "linux/amd64",
			GraphNamespace:   namespace,
			Domain:           "coding",
			ToolNames:        []string{"read_file", "apply_diff", "bash_exec"},
			ToolsetSignature: "apply_diff,bash_exec,read_file",
		},
		NewMessages: archiveDecisionSessionMessages(userMessage),
	}); err != nil {
		t.Fatalf("capture decision turn: %v", err)
	}
}

func seedArchivedDecisionSession(t *testing.T, handler http.Handler, store *session.Store, sessionID string, userMessage string) {
	t.Helper()
	sess := session.NewSession("system")
	sess.ID = sessionID
	for _, msg := range archiveDecisionSessionMessages(userMessage) {
		sess.AddMessage(msg)
	}
	if err := store.Save(sess); err != nil {
		t.Fatalf("save session %s: %v", sessionID, err)
	}
	resp := serveRequest(
		handler,
		http.MethodPost,
		"/api/bus",
		`{"action":"MEMORY_ARCHIVE","params":{"session_id":"`+sessionID+`"},"trace_id":"trace-archive-`+sessionID+`"}`,
		nil,
	)
	if resp.Code != http.StatusOK {
		t.Fatalf("archive session %s: got %d body=%s", sessionID, resp.Code, resp.Body.String())
	}
}

func archiveDecisionSessionMessages(userMessage string) []llm.Message {
	return []llm.Message{
		{Role: llm.RoleUser, Text: userMessage},
		{Role: llm.RoleAssistant, ToolCalls: []llm.ToolCall{{ID: "call-read", Name: "read_file", Arguments: json.RawMessage(`{"path":"config.toml"}`)}}},
		{Role: llm.RoleTool, ToolCallID: "call-read", Text: agent.FormatToolResult("read_file", "trace-app", "loaded config.toml", nil)},
		{Role: llm.RoleAssistant, ToolCalls: []llm.ToolCall{{ID: "call-edit", Name: "apply_diff", Arguments: json.RawMessage(`{"path":"config.toml","diff":"patch"}`)}}},
		{Role: llm.RoleTool, ToolCallID: "call-edit", Text: agent.FormatToolResult("apply_diff", "trace-app", "patched config.toml", nil)},
		{Role: llm.RoleAssistant, ToolCalls: []llm.ToolCall{{ID: "call-check", Name: "bash_exec", Arguments: json.RawMessage(`{"command":"go test ./..."}`)}}},
		{Role: llm.RoleTool, ToolCallID: "call-check", Text: agent.FormatToolResult("bash_exec", "trace-app", "migration check passed", nil)},
		{Role: llm.RoleAssistant, Text: `{"signal":"END_SESSION","message":"Config migration fixed successfully."}`},
	}
}
