package memory

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"ghost-os/bridge/llm"
)

func TestShadowRecallKeepsLiveOutputStable(t *testing.T) {
	baseline := newShadowRecallTestManager(t, false)
	shadow := newShadowRecallTestManager(t, true)
	seedShadowRecallFixtures(t, baseline)
	seedShadowRecallFixtures(t, shadow)
	if !shadow.truth.waitForSidecar(2 * time.Second) {
		t.Fatal("timed out waiting for shadow sidecar sync")
	}

	baseWindow, err := baseline.BuildContextWindow("session-shadow-recall", "fix config migration")
	if err != nil {
		t.Fatalf("baseline build context: %v", err)
	}
	shadowWindow, err := shadow.BuildContextWindow("session-shadow-recall", "fix config migration")
	if err != nil {
		t.Fatalf("shadow build context: %v", err)
	}
	if !reflect.DeepEqual(baseWindow, shadowWindow) {
		t.Fatalf("expected BuildContextWindow to stay unchanged, base=%+v shadow=%+v", baseWindow, shadowWindow)
	}

	query := MemoryQuery{
		Keywords:        []string{"config", "migration"},
		SemanticQuery:   "fix config migration",
		IncludeMarkdown: true,
		IncludeDecision: true,
		IncludeVector:   true,
		IntentDebug:     true,
		VectorDebug:     true,
		ShadowDebug:     true,
	}
	baseResult, err := baseline.QueryResultWithScope(query, SessionScope{SessionID: "session-shadow-recall"})
	if err != nil {
		t.Fatalf("baseline query result: %v", err)
	}
	shadowResult, err := shadow.QueryResultWithScope(query, SessionScope{SessionID: "session-shadow-recall"})
	if err != nil {
		t.Fatalf("shadow query result: %v", err)
	}
	if !reflect.DeepEqual(stripEntryTimes(baseResult.Entries), stripEntryTimes(shadowResult.Entries)) {
		t.Fatalf("expected shadow recall to keep live entries unchanged, base=%+v shadow=%+v", baseResult.Entries, shadowResult.Entries)
	}
	if !reflect.DeepEqual(baseResult.DecisionHits, shadowResult.DecisionHits) {
		t.Fatalf("expected decision hits to stay unchanged, base=%+v shadow=%+v", baseResult.DecisionHits, shadowResult.DecisionHits)
	}
	if shadowResult.IntentPlan == nil || len(shadowResult.VectorHits) == 0 || shadowResult.ShadowReport == nil {
		t.Fatalf("expected shadow debug output, got %+v", shadowResult)
	}
	if shadowResult.ShadowReport.VectorHits != len(shadowResult.VectorHits) {
		t.Fatalf("expected shadow report to track vector hit count, got %+v", shadowResult.ShadowReport)
	}
	metrics := shadow.Metrics()
	if metrics.PlannerRuns == 0 || metrics.VectorShadowHits == 0 {
		t.Fatalf("expected shadow metrics to increment, got %+v", metrics)
	}
}

func TestShadowRecallBuildContextDoesNotInjectVectorContent(t *testing.T) {
	manager := newShadowRecallTestManager(t, true)
	seedShadowRecallFixtures(t, manager)
	if !manager.truth.waitForSidecar(2 * time.Second) {
		t.Fatal("timed out waiting for shadow sidecar sync")
	}
	window, err := manager.BuildContextWindow("session-shadow-recall", "fix config migration")
	if err != nil {
		t.Fatalf("build context window: %v", err)
	}
	if len(window) == 0 {
		t.Fatal("expected auto recall window to remain available")
	}
	if strings.Contains(strings.ToLower(window[0].Text), "vector_hits") || strings.Contains(strings.ToLower(window[0].Text), "shadow_only_candidates") {
		t.Fatalf("expected BuildContextWindow to skip vector debug content, got %q", window[0].Text)
	}
}

func newShadowRecallTestManager(t *testing.T, shadow bool) *MemoryManager {
	t.Helper()
	baseDir := t.TempDir()
	config := MemoryConfig{
		WarmCapacity:         32,
		WarmPath:             baseDir + "/warm.json",
		ColdBaseDir:          baseDir + "/cold",
		AutoRecallEnabled:    true,
		AutoRecallLimit:      4,
		DecisionEnabled:      true,
		DecisionPath:         baseDir + "/decision",
		IntentPlannerEnabled: shadow,
		VectorEnabled:        shadow,
		VectorPath:           baseDir + "/vector",
		ShadowRecallEnabled:  shadow,
	}
	if shadow {
		config.TruthEnabled = true
		config.TruthDualWrite = true
		config.TruthBaseDir = baseDir + "/truth"
		config.TruthShadowFailOpen = true
	}
	return NewMemoryManager(config)
}

func seedShadowRecallFixtures(t *testing.T, manager *MemoryManager) {
	t.Helper()
	storedAt := time.Date(2026, 3, 7, 16, 0, 0, 0, time.UTC)
	if err := manager.warm.Store(MemoryEntry{
		ID:         "warm-shadow-recall",
		Content:    "Patch config.toml before running the config migration check.",
		Summary:    "Patch config.toml before running the config migration check.",
		Timestamp:  storedAt,
		Importance: 0.82,
		Metadata:   map[string]any{"session_id": "session-shadow-recall", "role": "assistant"},
	}); err != nil {
		t.Fatalf("store warm entry: %v", err)
	}
	if err := manager.SaveMarkdownNode(MarkdownNode{
		ID:         "node-shadow-recall",
		SessionID:  "session-shadow-recall",
		CreatedAt:  storedAt,
		Summary:    "Config migration checklist",
		Content:    "Remember config migration validation checks before editing config.toml.",
		SourceIDs:  []string{"session-shadow-recall:000000"},
		Confidence: 0.84,
	}); err != nil {
		t.Fatalf("save markdown node: %v", err)
	}
	input := decisionCaptureTestInput(DecisionOutcomeSuccess)
	input.SessionID = "session-shadow-recall"
	input.TraceID = "trace-shadow-recall"
	input.TurnID = "turn-shadow-recall"
	input.UserMessage = "fix config migration"
	input.TurnStartedAt = storedAt
	input.TurnFinishedAt = storedAt.Add(2 * time.Minute)
	input.NewMessages = []llm.Message{{Role: llm.RoleAssistant, Text: "Patch config.toml, then run the migration check."}}
	if err := manager.CaptureDecisionTurn(input); err != nil {
		t.Fatalf("capture decision turn: %v", err)
	}
}
