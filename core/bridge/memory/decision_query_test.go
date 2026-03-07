package memory

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDecisionServiceRetrieveReturnsSuccessMemo(t *testing.T) {
	service, _ := newDecisionCaptureTestService(t, nil)
	now := time.Now().UTC()
	env := DecisionEnvFingerprint{
		WorkspaceRoot:    "/workspace/ghost-os",
		Platform:         "linux/amd64",
		GraphNamespace:   "workspace:test",
		ToolsetSignature: "bash_exec,read_file",
		ToolNames:        []string{"bash_exec", "read_file"},
		PathHints:        []string{"config.toml"},
	}
	if _, err := service.store.UpsertMemo(DecisionMemo{
		ID:               "memo-success",
		Namespace:        "workspace:test",
		SessionID:        "session-success",
		IntentKey:        "intent.fix_config",
		IntentSummary:    "fix config migration",
		StrategySummary:  "Patch config.toml, then run the migration check.",
		ValidationChecks: []string{"run migration check"},
		Outcome:          DecisionOutcomeSuccess,
		Confidence:       0.93,
		ReuseScore:       0.91,
		CreatedAt:        now.Add(-48 * time.Hour),
		LastUsedAt:       now.Add(-2 * time.Hour),
		AnchorKeys:       []string{"config.toml", "migration"},
		ToolsUsed:        []DecisionToolUse{{Name: "bash_exec"}},
		Environment:      env,
	}); err != nil {
		t.Fatalf("upsert success memo: %v", err)
	}

	entries, hits, err := service.Retrieve(MemoryQuery{
		IncludeDecision: true,
		SemanticQuery:   "fix config migration",
		DecisionTypes:   []string{DecisionHitTypeMemo, DecisionHitTypeRecipe, DecisionHitTypeWarning},
		Environment:     &env,
	}, SessionScope{Environment: &env})
	if err != nil {
		t.Fatalf("retrieve decision memo: %v", err)
	}
	if len(entries) == 0 || len(hits) == 0 {
		t.Fatalf("expected decision hits, got entries=%d hits=%d", len(entries), len(hits))
	}
	if hits[0].Type != DecisionHitTypeMemo {
		t.Fatalf("expected memo hit, got %+v", hits[0])
	}
	if !strings.HasPrefix(entries[0].Summary, "Prior similar experience:") {
		t.Fatalf("expected concise decision recall line, got %q", entries[0].Summary)
	}
	if strings.Contains(entries[0].Summary, "{\"") || strings.Contains(entries[0].Summary, "reuse_score") {
		t.Fatalf("decision recall should stay compact, got %q", entries[0].Summary)
	}
}

func TestDecisionServiceRetrieveReturnsFailureAsWarning(t *testing.T) {
	service, _ := newDecisionCaptureTestService(t, nil)
	now := time.Now().UTC()
	env := DecisionEnvFingerprint{
		WorkspaceRoot:    "/workspace/ghost-os",
		Platform:         "linux/amd64",
		GraphNamespace:   "workspace:test",
		ToolsetSignature: "bash_exec",
		ToolNames:        []string{"bash_exec"},
	}
	if _, err := service.store.UpsertMemo(DecisionMemo{
		ID:              "memo-failure",
		Namespace:       "workspace:test",
		SessionID:       "session-failure",
		IntentSummary:   "fix config migration",
		StrategySummary: "Patch config.toml directly.",
		Outcome:         DecisionOutcomeFailure,
		FailureReasons:  []string{"Skipping the config migration check broke startup."},
		AvoidPatterns:   []string{"skip the migration check"},
		Confidence:      0.9,
		ReuseScore:      0.22,
		CreatedAt:       now.Add(-24 * time.Hour),
		LastUsedAt:      now.Add(-time.Hour),
		ToolsUsed:       []DecisionToolUse{{Name: "bash_exec"}},
		Environment:     env,
	}); err != nil {
		t.Fatalf("upsert failure memo: %v", err)
	}

	entries, hits, err := service.Retrieve(MemoryQuery{
		IncludeDecision: true,
		SemanticQuery:   "fix config migration",
		Environment:     &env,
	}, SessionScope{Environment: &env})
	if err != nil {
		t.Fatalf("retrieve failure decision: %v", err)
	}
	if len(entries) == 0 || len(hits) == 0 {
		t.Fatalf("expected warning hit, got entries=%d hits=%d", len(entries), len(hits))
	}
	if hits[0].Type != DecisionHitTypeWarning {
		t.Fatalf("expected warning hit, got %+v", hits[0])
	}
	if !strings.HasPrefix(entries[0].Summary, "Caution:") {
		t.Fatalf("expected caution line, got %q", entries[0].Summary)
	}
}

func TestDecisionServiceRetrieveFiltersByEnvironmentStrict(t *testing.T) {
	service, _ := newDecisionCaptureTestService(t, nil)
	now := time.Now().UTC()
	if _, err := service.store.UpsertMemo(DecisionMemo{
		ID:              "memo-mismatch",
		Namespace:       "workspace:test",
		IntentSummary:   "fix config migration",
		StrategySummary: "Patch config.toml, then run checks.",
		Outcome:         DecisionOutcomeSuccess,
		Confidence:      0.92,
		ReuseScore:      0.9,
		CreatedAt:       now.Add(-12 * time.Hour),
		LastUsedAt:      now.Add(-time.Hour),
		Environment: DecisionEnvFingerprint{
			WorkspaceRoot:    "/workspace/other-repo",
			Platform:         "linux/amd64",
			GraphNamespace:   "workspace:test",
			ToolsetSignature: "bash_exec",
			ToolNames:        []string{"bash_exec"},
		},
	}); err != nil {
		t.Fatalf("upsert strict memo: %v", err)
	}

	currentEnv := DecisionEnvFingerprint{
		WorkspaceRoot:    "/workspace/ghost-os",
		Platform:         "linux/amd64",
		GraphNamespace:   "workspace:test",
		ToolsetSignature: "bash_exec",
		ToolNames:        []string{"bash_exec"},
	}
	entries, hits, err := service.Retrieve(MemoryQuery{
		IncludeDecision:   true,
		SemanticQuery:     "fix config migration",
		Environment:       &currentEnv,
		EnvironmentStrict: true,
	}, SessionScope{Environment: &currentEnv})
	if err != nil {
		t.Fatalf("retrieve strict decision: %v", err)
	}
	if len(entries) != 0 || len(hits) != 0 {
		t.Fatalf("expected strict environment filter to drop hits, got entries=%d hits=%d", len(entries), len(hits))
	}
}

func TestDecisionServiceRetrieveHonorsDecisionTypesAndReuseScore(t *testing.T) {
	service, _ := newDecisionCaptureTestService(t, nil)
	now := time.Now().UTC()
	env := DecisionEnvFingerprint{
		WorkspaceRoot:    "/workspace/ghost-os",
		Platform:         "linux/amd64",
		GraphNamespace:   "workspace:test",
		ToolsetSignature: "bash_exec",
		ToolNames:        []string{"bash_exec"},
	}
	for _, memo := range []DecisionMemo{
		{
			ID:              "memo-low-reuse",
			Namespace:       "workspace:test",
			IntentSummary:   "fix config migration",
			StrategySummary: "Patch config.toml and run checks.",
			Outcome:         DecisionOutcomeSuccess,
			Confidence:      0.9,
			ReuseScore:      0.61,
			CreatedAt:       now.Add(-8 * time.Hour),
			LastUsedAt:      now.Add(-time.Hour),
			Environment:     env,
		},
		{
			ID:             "memo-warning",
			Namespace:      "workspace:test",
			IntentSummary:  "fix config migration",
			Outcome:        DecisionOutcomeFailure,
			FailureReasons: []string{"Patch without verification broke startup."},
			Confidence:     0.88,
			ReuseScore:     0.1,
			CreatedAt:      now.Add(-7 * time.Hour),
			LastUsedAt:     now.Add(-30 * time.Minute),
			Environment:    env,
		},
	} {
		if _, err := service.store.UpsertMemo(memo); err != nil {
			t.Fatalf("upsert memo %s: %v", memo.ID, err)
		}
	}

	entries, hits, err := service.Retrieve(MemoryQuery{
		IncludeDecision:   true,
		SemanticQuery:     "fix config migration",
		DecisionReuseOnly: true,
		DecisionTypes:     []string{DecisionHitTypeMemo, DecisionHitTypeWarning},
		MinReuseScore:     0.9,
		Environment:       &env,
	}, SessionScope{Environment: &env})
	if err != nil {
		t.Fatalf("retrieve filtered decision hits: %v", err)
	}
	if len(entries) != 1 || len(hits) != 1 {
		t.Fatalf("expected only warning hit to survive filters, got entries=%d hits=%d", len(entries), len(hits))
	}
	if hits[0].Type != DecisionHitTypeWarning {
		t.Fatalf("expected warning hit after reuse filter, got %+v", hits[0])
	}
}

func TestDecisionServiceRetrievePrefersRecipeWhenSupportIsHigher(t *testing.T) {
	service, _ := newDecisionCaptureTestService(t, nil)
	now := time.Now().UTC()
	env := DecisionEnvFingerprint{
		WorkspaceRoot:    "/workspace/ghost-os",
		Platform:         "linux/amd64",
		GraphNamespace:   "workspace:test",
		ToolsetSignature: "bash_exec,read_file",
		ToolNames:        []string{"bash_exec", "read_file"},
	}
	if _, err := service.store.UpsertMemo(DecisionMemo{
		ID:              "memo-source",
		Namespace:       "workspace:test",
		SessionID:       "session-recipe-source",
		IntentKey:       "intent.fix_config",
		IntentSummary:   "fix config migration",
		StrategySummary: "Patch config.toml and verify with a migration check.",
		Outcome:         DecisionOutcomeSuccess,
		Confidence:      0.88,
		ReuseScore:      0.8,
		CreatedAt:       now.Add(-72 * time.Hour),
		LastUsedAt:      now.Add(-6 * time.Hour),
		Environment:     env,
		ToolsUsed:       []DecisionToolUse{{Name: "bash_exec"}},
	}); err != nil {
		t.Fatalf("upsert source memo: %v", err)
	}
	if _, err := service.store.UpsertRecipe(DecisionRecipe{
		ID:                  "recipe-config",
		Namespace:           "workspace:test",
		IntentKey:           "intent.fix_config",
		TriggerPhrases:      []string{"fix config migration"},
		StrategySummary:     "Use the proven config patch sequence and verify immediately.",
		RecommendedTools:    []string{"bash_exec", "read_file"},
		OrderedActions:      []RecipeStep{{Instruction: "Patch config.toml"}, {Instruction: "Run the migration check"}},
		ValidationChecklist: []string{"run migration check"},
		SupportCount:        6,
		SuccessRate:         0.95,
		Confidence:          0.92,
		SourceMemoIDs:       []string{"memo-source"},
		CreatedAt:           now.Add(-48 * time.Hour),
		UpdatedAt:           now.Add(-time.Hour),
	}); err != nil {
		t.Fatalf("upsert recipe: %v", err)
	}

	_, hits, err := service.Retrieve(MemoryQuery{
		IncludeDecision: true,
		SemanticQuery:   "fix config migration",
		DecisionTypes:   []string{DecisionHitTypeRecipe, DecisionHitTypeMemo},
		Environment:     &env,
	}, SessionScope{Environment: &env})
	if err != nil {
		t.Fatalf("retrieve recipe hit: %v", err)
	}
	if len(hits) == 0 {
		t.Fatal("expected recipe hit")
	}
	if hits[0].Type != DecisionHitTypeRecipe {
		t.Fatalf("expected recipe hit to rank first, got %+v", hits[0])
	}
}

func TestDecisionServiceRetrieveRecordsMemoAccess(t *testing.T) {
	service, _ := newDecisionCaptureTestService(t, nil)
	now := time.Now().UTC()
	env := DecisionEnvFingerprint{
		WorkspaceRoot:    "/workspace/ghost-os",
		Platform:         "linux/amd64",
		GraphNamespace:   "workspace:test",
		ToolsetSignature: "bash_exec",
		ToolNames:        []string{"bash_exec"},
	}
	if _, err := service.store.UpsertMemo(DecisionMemo{
		ID:              "memo-access",
		Namespace:       "workspace:test",
		IntentSummary:   "fix config migration",
		StrategySummary: "Patch config.toml and verify.",
		Outcome:         DecisionOutcomeSuccess,
		Confidence:      0.91,
		ReuseScore:      0.87,
		CreatedAt:       now.Add(-6 * time.Hour),
		LastUsedAt:      now.Add(-2 * time.Hour),
		Environment:     env,
	}); err != nil {
		t.Fatalf("upsert access memo: %v", err)
	}

	_, hits, err := service.Retrieve(MemoryQuery{
		IncludeDecision: true,
		SemanticQuery:   "fix config migration",
		Environment:     &env,
	}, SessionScope{Environment: &env})
	if err != nil {
		t.Fatalf("retrieve access memo: %v", err)
	}
	accessedAt := now.Add(time.Minute)
	memoIDs := make([]string, 0, len(hits))
	for _, hit := range hits {
		if hit.MemoID != "" {
			memoIDs = append(memoIDs, hit.MemoID)
		}
	}
	service.RecordMemoAccess(memoIDs, accessedAt)
	memo, ok := service.store.Memo("memo-access")
	if !ok {
		t.Fatal("expected memo to exist after access record")
	}
	if memo.AccessCount != 1 {
		t.Fatalf("expected memo access count to increment, got %d", memo.AccessCount)
	}
	if !memo.LastUsedAt.Equal(accessedAt.UTC()) {
		t.Fatalf("expected last_used_at update, got %s want %s", memo.LastUsedAt, accessedAt.UTC())
	}
}

func newDecisionEnabledManager(t *testing.T) *MemoryManager {
	t.Helper()
	baseDir := t.TempDir()
	return NewMemoryManager(MemoryConfig{
		WarmCapacity:      32,
		WarmPath:          filepath.Join(baseDir, "warm.json"),
		ColdBaseDir:       filepath.Join(baseDir, "cold"),
		AutoRecallEnabled: true,
		AutoRecallLimit:   3,
		DecisionEnabled:   true,
		DecisionPath:      filepath.Join(baseDir, "decision"),
	})
}
