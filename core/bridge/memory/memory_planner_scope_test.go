package memory

import (
	"path/filepath"
	"testing"
	"time"
)

func newPlannerScopeManager(t *testing.T) *MemoryManager {
	t.Helper()
	baseDir := t.TempDir()
	return NewMemoryManager(MemoryConfig{
		Warm:     WarmConfig{Capacity: 16, Path: filepath.Join(baseDir, "warm.json")},
		Cold:     ColdConfig{BaseDir: filepath.Join(baseDir, "cold")},
		Decision: DecisionConfig{Enabled: true, Path: filepath.Join(baseDir, "decision")},
		Recall:   RecallConfig{IntentPlannerEnabled: true, HybridRerankEnabled: true},
	})
}

func seedPlannerScopeFixtures(t *testing.T, manager *MemoryManager) {
	t.Helper()
	now := time.Date(2026, 3, 8, 10, 0, 0, 0, time.UTC)
	if err := manager.cold.SaveMarkdownNode(MarkdownNode{
		ID:         "node-config-migration",
		Summary:    "Fix config migration without breaking API.",
		Content:    "Markdown note: fix config migration without breaking API.",
		Importance: 0.84,
		Confidence: 0.88,
		CreatedAt:  now.Add(-2 * time.Hour),
		LastSeenAt: now.Add(-time.Hour),
	}); err != nil {
		t.Fatalf("save markdown node: %v", err)
	}
	env := DecisionEnvFingerprint{WorkspaceRoot: "/workspace/ghost-os", Platform: "linux/amd64", GraphNamespace: "workspace:test"}
	if _, err := manager.decision.debugUpsertMemo(DecisionMemo{
		ID:              "memo-config-migration",
		Namespace:       "workspace:test",
		SessionID:       "session-planner-scope",
		IntentKey:       "intent.fix_config_migration",
		IntentSummary:   "fix config migration without breaking API",
		StrategySummary: "Fix config migration without breaking API.",
		Outcome:         DecisionOutcomeSuccess,
		Confidence:      0.93,
		ReuseScore:      0.91,
		CreatedAt:       now.Add(-90 * time.Minute),
		LastUsedAt:      now.Add(-30 * time.Minute),
		Environment:     env,
	}); err != nil {
		t.Fatalf("upsert decision memo: %v", err)
	}
}

func TestPlannerHydrationScopeNarrowsEnabledLayers(t *testing.T) {
	manager := newPlannerScopeManager(t)
	seedPlannerScopeFixtures(t, manager)
	env := &DecisionEnvFingerprint{WorkspaceRoot: "/workspace/ghost-os", Platform: "linux/amd64", GraphNamespace: "workspace:test"}

	procedural, err := manager.QueryResultWithScope(MemoryQuery{
		SemanticQuery:   "fix config migration without breaking API",
		IncludeMarkdown: true,
		IncludeDecision: true,
		DecisionDebug:   true,
		IntentDebug:     true,
		Environment:     env,
	}, SessionScope{Environment: env})
	if err != nil {
		t.Fatalf("procedural query result: %v", err)
	}
	proceduralDebug := procedural.Debug()
	if proceduralDebug == nil || proceduralDebug.IntentPlan == nil {
		t.Fatalf("expected procedural debug intent plan, got %+v", proceduralDebug)
	}
	if proceduralDebug.IntentPlan.RecallMode != plannerRecallModeProceduralFirst {
		t.Fatalf("expected procedural-first plan, got %+v", proceduralDebug.IntentPlan)
	}
	foundDecision := false
	foundMarkdown := false
	for _, entry := range procedural.Entries {
		switch entryLayer(entry) {
		case "decision":
			foundDecision = true
		case "markdown":
			foundMarkdown = true
		}
	}
	if !foundDecision || foundMarkdown {
		t.Fatalf("expected only decision hydration for procedural query, got %+v", procedural.Entries)
	}

	semantic, err := manager.QueryResultWithScope(MemoryQuery{
		SemanticQuery:   "explain markdown note about API compatibility and config changes",
		IncludeMarkdown: true,
		IncludeDecision: true,
		DecisionDebug:   true,
		IntentDebug:     true,
		Environment:     env,
	}, SessionScope{Environment: env})
	if err != nil {
		t.Fatalf("semantic query result: %v", err)
	}
	semanticDebug := semantic.Debug()
	if semanticDebug == nil || semanticDebug.IntentPlan == nil {
		t.Fatalf("expected semantic debug intent plan, got %+v", semanticDebug)
	}
	if semanticDebug.IntentPlan.RecallMode != plannerRecallModeSemanticFirst {
		t.Fatalf("expected semantic-first plan, got %+v", semanticDebug.IntentPlan)
	}
	foundDecision = false
	foundMarkdown = false
	for _, entry := range semantic.Entries {
		switch entryLayer(entry) {
		case "decision":
			foundDecision = true
		case "markdown":
			foundMarkdown = true
		}
	}
	if !foundMarkdown || foundDecision {
		t.Fatalf("expected only markdown hydration for semantic query, got %+v", semantic.Entries)
	}
}
