package app

import (
	"net/http"
	"testing"
	"time"

	"ghost-os/bridge/llm"
	"ghost-os/bridge/memory"
)

func TestBusMemoryQueryReturnsDecisionHits(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("GHOST_MEMORY_DECISION_ENABLED", "true")
	t.Setenv("GHOST_MEMORY_DECISION_PATH", tempDir+"/decision")
	t.Setenv("GHOST_MEMORY_DECISION_CAPTURE_ON_TURN", "true")

	handler, service, _ := newTestHandlerWithService(t, nil, nil)
	startedAt := time.Now().UTC().Add(-2 * time.Minute)
	if err := service.memoryManager.CaptureDecisionTurn(memory.DecisionCaptureInput{
		SessionID:      "session-decision-test",
		TraceID:        "trace-decision-capture",
		TurnID:         "turn-decision-test",
		UserMessage:    "fix config migration",
		Outcome:        memory.DecisionOutcomeSuccess,
		TurnStartedAt:  startedAt,
		TurnFinishedAt: startedAt.Add(time.Minute),
		Environment: memory.DecisionEnvFingerprint{
			WorkspaceRoot:    "/workspace/ghost-os",
			Platform:         "linux/amd64",
			ToolsetSignature: "bash_exec",
			ToolNames:        []string{"bash_exec"},
		},
		NewMessages: []llm.Message{{
			Role: llm.RoleAssistant,
			Text: "Patch config.toml, then run the migration check.",
		}},
	}); err != nil {
		t.Fatalf("capture decision turn: %v", err)
	}

	queryResp := serveRequest(
		handler,
		http.MethodPost,
		"/api/bus",
		`{"action":"MEMORY_QUERY","params":{"semantic_query":"fix config migration","include_decision":true},"trace_id":"trace-decision-query"}`,
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
	decisionHits, ok := payload["decision_hits"].([]any)
	if !ok || len(decisionHits) == 0 {
		t.Fatalf("expected decision hits in payload, got %#v", payload["decision_hits"])
	}
	entries, ok := payload["entries"].([]any)
	if !ok || len(entries) == 0 {
		t.Fatalf("expected decision-backed entries, got %#v", payload["entries"])
	}
}
