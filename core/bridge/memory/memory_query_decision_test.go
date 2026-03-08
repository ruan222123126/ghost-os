package memory

import (
	"strings"
	"testing"
	"time"
)

func TestMemoryManagerQueryResultIncludesDecisionHits(t *testing.T) {
	manager := newDecisionEnabledManager(t)
	now := time.Now().UTC()
	env := DecisionEnvFingerprint{
		WorkspaceRoot:    "/workspace/ghost-os",
		Platform:         "linux/amd64",
		GraphNamespace:   "workspace:test",
		ToolsetSignature: "bash_exec",
		ToolNames:        []string{"bash_exec"},
	}
	if _, err := manager.decision.debugUpsertMemo(DecisionMemo{
		ID:              "memo-query-result",
		Namespace:       "workspace:test",
		SessionID:       "session-query-result",
		IntentSummary:   "fix config migration",
		StrategySummary: "Patch config.toml, then run the migration check.",
		Outcome:         DecisionOutcomeSuccess,
		Confidence:      0.92,
		ReuseScore:      0.89,
		CreatedAt:       now.Add(-24 * time.Hour),
		LastUsedAt:      now.Add(-time.Hour),
		Environment:     env,
	}); err != nil {
		t.Fatalf("upsert decision memo: %v", err)
	}

	result, err := manager.QueryResultWithScope(MemoryQuery{
		IncludeDecision: true,
		DecisionDebug:   true,
		SemanticQuery:   "fix config migration",
		Environment:     &env,
	}, SessionScope{Environment: &env})
	if err != nil {
		t.Fatalf("query result with decision: %v", err)
	}
	if len(result.DecisionHits) == 0 {
		t.Fatal("expected decision hits in query result")
	}
	foundDecisionEntry := false
	for _, entry := range result.Entries {
		if entryLayer(entry) == "decision" {
			foundDecisionEntry = true
			break
		}
	}
	if !foundDecisionEntry {
		t.Fatalf("expected decision-backed entries, got %+v", result.Entries)
	}
}

func TestBuildContextWindowInjectsConciseDecisionRecall(t *testing.T) {
	manager := newDecisionEnabledManager(t)
	now := time.Now().UTC()
	env := DecisionEnvFingerprint{
		WorkspaceRoot:    "/workspace/ghost-os",
		Platform:         "linux/amd64",
		GraphNamespace:   "workspace:test",
		ToolsetSignature: "bash_exec",
		ToolNames:        []string{"bash_exec"},
	}
	if _, err := manager.decision.debugUpsertMemo(DecisionMemo{
		ID:              "memo-context-window",
		Namespace:       "workspace:test",
		SessionID:       "session-context-window",
		IntentSummary:   "fix config migration",
		StrategySummary: "Patch config.toml, then run the migration check.",
		Outcome:         DecisionOutcomeSuccess,
		Confidence:      0.92,
		ReuseScore:      0.88,
		CreatedAt:       now.Add(-24 * time.Hour),
		LastUsedAt:      now.Add(-90 * time.Minute),
		Environment:     env,
	}); err != nil {
		t.Fatalf("upsert context memo: %v", err)
	}

	window, err := manager.BuildContextWindowWithScope(SessionScope{
		SessionID:   "session-current",
		Environment: &env,
	}, "fix config migration")
	if err != nil {
		t.Fatalf("build context window: %v", err)
	}
	if len(window) != 1 {
		t.Fatalf("unexpected context window size: got %d want 1", len(window))
	}
	if !strings.Contains(window[0].Text, "Prior similar experience:") {
		t.Fatalf("expected concise decision recall line, got %q", window[0].Text)
	}
	if strings.Contains(window[0].Text, "avoid_patterns") || strings.Contains(window[0].Text, "reuse_score") {
		t.Fatalf("expected compact decision recall only, got %q", window[0].Text)
	}
}

func TestBuildContextWindowDecisionRecallSkipsDuplicateSummary(t *testing.T) {
	manager := newDecisionEnabledManager(t)
	now := time.Now().UTC()
	env := DecisionEnvFingerprint{
		WorkspaceRoot:    "/workspace/ghost-os",
		Platform:         "linux/amd64",
		GraphNamespace:   "workspace:test",
		ToolsetSignature: "bash_exec",
		ToolNames:        []string{"bash_exec"},
	}
	memo := DecisionMemo{
		ID:              "memo-duplicate",
		Namespace:       "workspace:test",
		SessionID:       "session-duplicate-source",
		IntentSummary:   "fix config migration",
		StrategySummary: "Patch config.toml, then run the migration check.",
		Outcome:         DecisionOutcomeSuccess,
		Confidence:      0.92,
		ReuseScore:      0.9,
		CreatedAt:       now.Add(-24 * time.Hour),
		LastUsedAt:      now.Add(-time.Hour),
		Environment:     env,
	}
	line := memoryEntryFromDecisionHit(decisionHitFromMemo(memo, 0.92, "matched intent", "")).Summary
	if err := manager.warm.Store(MemoryEntry{
		ID:         "warm-duplicate",
		Content:    line,
		Summary:    line,
		Timestamp:  now,
		Importance: 0.8,
		Metadata:   map[string]any{"layer": "warm", "session_id": "session-current", "role": "assistant"},
	}); err != nil {
		t.Fatalf("store warm duplicate entry: %v", err)
	}
	if _, err := manager.decision.debugUpsertMemo(memo); err != nil {
		t.Fatalf("upsert duplicate memo: %v", err)
	}

	window, err := manager.BuildContextWindowWithScope(SessionScope{
		SessionID:   "session-current",
		Environment: &env,
	}, "fix config migration")
	if err != nil {
		t.Fatalf("build context window with duplicate summary: %v", err)
	}
	if len(window) != 1 {
		t.Fatalf("unexpected context window size: got %d want 1", len(window))
	}
	if strings.Count(window[0].Text, line) != 1 {
		t.Fatalf("expected duplicate summary to be deduped, got %q", window[0].Text)
	}
}
