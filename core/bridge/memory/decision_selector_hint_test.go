package memory

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func newSelectorHintTestService(t *testing.T) *DecisionService {
	t.Helper()
	return NewDecisionService(MemoryConfig{
		DecisionEnabled: true,
		DecisionPath:    filepath.Join(t.TempDir(), "decision"),
	}, nil)
}

func selectorHintTestQuery(env DecisionEnvFingerprint) MemoryQuery {
	return MemoryQuery{
		IncludeDecision:   true,
		DecisionReuseOnly: true,
		DecisionTypes:     []string{DecisionHitTypeRecipe, DecisionHitTypeMemo, DecisionHitTypeWarning},
		EnvironmentStrict: true,
		SemanticQuery:     "fix config migration",
		Environment:       &env,
	}
}

func selectorHintTestEnv() DecisionEnvFingerprint {
	return DecisionEnvFingerprint{
		WorkspaceRoot:    "/workspace/ghost-os",
		Platform:         "linux/amd64",
		GraphNamespace:   "workspace:test",
		ToolNames:        []string{"read_file", "search_files", "apply_diff", "bash_exec"},
		ToolsetSignature: "apply_diff,bash_exec,read_file,search_files",
	}
}

func TestDecisionServiceBuildSelectorHintFromRecipe(t *testing.T) {
	service := newSelectorHintTestService(t)
	env := selectorHintTestEnv()
	now := time.Now().UTC()
	if _, err := service.store.UpsertMemo(DecisionMemo{
		ID:              "memo-source",
		Namespace:       env.GraphNamespace,
		SessionID:       "session-recipe",
		IntentSummary:   "fix config migration",
		StrategySummary: "inspect target file before patching",
		ToolsUsed:       []DecisionToolUse{{Name: "read_file"}, {Name: "apply_diff"}, {Name: "bash_exec"}},
		Outcome:         DecisionOutcomeSuccess,
		Confidence:      0.93,
		ReuseScore:      0.91,
		Environment:     env,
		CreatedAt:       now.Add(-2 * time.Hour),
		LastUsedAt:      now.Add(-time.Hour),
	}); err != nil {
		t.Fatalf("upsert source memo: %v", err)
	}
	if _, err := service.store.UpsertRecipe(DecisionRecipe{
		ID:               "recipe-config",
		Namespace:        env.GraphNamespace,
		IntentKey:        "intent.fix_config",
		TriggerPhrases:   []string{"fix config migration"},
		StrategySummary:  "inspect target file before patching",
		RecommendedTools: []string{"read_file", "apply_diff", "bash_exec"},
		OrderedActions:   []RecipeStep{{Instruction: "inspect target file before patching"}, {Instruction: "patch config.toml"}},
		SupportCount:     6,
		SuccessRate:      0.95,
		Confidence:       0.92,
		SourceMemoIDs:    []string{"memo-source"},
		CreatedAt:        now.Add(-24 * time.Hour),
		UpdatedAt:        now.Add(-30 * time.Minute),
	}); err != nil {
		t.Fatalf("upsert recipe: %v", err)
	}

	hint, hits, err := service.BuildSelectorHint(selectorHintTestQuery(env), SessionScope{Environment: &env})
	if err != nil {
		t.Fatalf("BuildSelectorHint: %v", err)
	}
	if len(hits) == 0 || hits[0].Type != DecisionHitTypeRecipe {
		t.Fatalf("expected recipe hit first, got %#v", hits)
	}
	if !strings.Contains(hint, "Similar successful cases used: read_file, apply_diff, bash_exec") {
		t.Fatalf("expected recipe tool hint, got %q", hint)
	}
	if !strings.Contains(hint, "Usually start with: inspect target file before patching") {
		t.Fatalf("expected recipe start hint, got %q", hint)
	}
}

func TestDecisionServiceBuildSelectorHintFromMemo(t *testing.T) {
	service := newSelectorHintTestService(t)
	env := selectorHintTestEnv()
	now := time.Now().UTC()
	if _, err := service.store.UpsertMemo(DecisionMemo{
		ID:              "memo-config",
		Namespace:       env.GraphNamespace,
		SessionID:       "session-memo",
		IntentSummary:   "fix config migration",
		StrategySummary: "inspect target file before patching",
		ToolsUsed:       []DecisionToolUse{{Name: "read_file"}, {Name: "search_files"}, {Name: "apply_diff"}},
		Outcome:         DecisionOutcomeSuccess,
		Confidence:      0.92,
		ReuseScore:      0.9,
		Environment:     env,
		CreatedAt:       now.Add(-4 * time.Hour),
		LastUsedAt:      now.Add(-45 * time.Minute),
	}); err != nil {
		t.Fatalf("upsert memo: %v", err)
	}

	hint, hits, err := service.BuildSelectorHint(selectorHintTestQuery(env), SessionScope{Environment: &env})
	if err != nil {
		t.Fatalf("BuildSelectorHint: %v", err)
	}
	if len(hits) == 0 || hits[0].Type != DecisionHitTypeMemo {
		t.Fatalf("expected memo hit, got %#v", hits)
	}
	if !strings.Contains(hint, "Similar successful cases used: read_file, search_files, apply_diff") {
		t.Fatalf("expected memo tools hint, got %q", hint)
	}
	if !strings.Contains(hint, "Usually start with: inspect target file before patching") {
		t.Fatalf("expected memo strategy hint, got %q", hint)
	}
}

func TestDecisionServiceBuildSelectorHintFromWarning(t *testing.T) {
	service := newSelectorHintTestService(t)
	env := selectorHintTestEnv()
	now := time.Now().UTC()
	if _, err := service.store.UpsertMemo(DecisionMemo{
		ID:            "memo-warning",
		Namespace:     env.GraphNamespace,
		SessionID:     "session-warning",
		IntentSummary: "fix config migration",
		Outcome:       DecisionOutcomeAwaitingHuman,
		Confidence:    0.9,
		ReuseScore:    0.84,
		NeedsHumanFor: []string{"provider credentials are unclear"},
		Environment:   env,
		CreatedAt:     now.Add(-90 * time.Minute),
		LastUsedAt:    now.Add(-20 * time.Minute),
	}); err != nil {
		t.Fatalf("upsert warning memo: %v", err)
	}

	hint, hits, err := service.BuildSelectorHint(selectorHintTestQuery(env), SessionScope{Environment: &env})
	if err != nil {
		t.Fatalf("BuildSelectorHint: %v", err)
	}
	if len(hits) == 0 || hits[0].Type != DecisionHitTypeWarning {
		t.Fatalf("expected warning hit, got %#v", hits)
	}
	if !strings.Contains(hint, "Ask human early if provider credentials are unclear") {
		t.Fatalf("expected ask-human warning hint, got %q", hint)
	}
}

func TestDecisionServiceBuildSelectorHintDedupesRepeatedToolsAndCautions(t *testing.T) {
	service := newSelectorHintTestService(t)
	env := selectorHintTestEnv()
	now := time.Now().UTC()
	memos := []DecisionMemo{
		{
			ID:              "memo-a",
			Namespace:       env.GraphNamespace,
			SessionID:       "session-a",
			IntentSummary:   "fix config migration",
			StrategySummary: "inspect target file before patching",
			ToolsUsed:       []DecisionToolUse{{Name: "read_file"}, {Name: "apply_diff"}},
			Outcome:         DecisionOutcomeSuccess,
			Confidence:      0.92,
			ReuseScore:      0.9,
			AvoidPatterns:   []string{"script_exec first unless atomic tools fail"},
			Environment:     env,
			CreatedAt:       now.Add(-4 * time.Hour),
			LastUsedAt:      now.Add(-time.Hour),
		},
		{
			ID:              "memo-b",
			Namespace:       env.GraphNamespace,
			SessionID:       "session-b",
			IntentSummary:   "fix config migration",
			StrategySummary: "inspect target file before patching",
			ToolsUsed:       []DecisionToolUse{{Name: "read_file"}, {Name: "bash_exec"}},
			Outcome:         DecisionOutcomeSuccess,
			Confidence:      0.91,
			ReuseScore:      0.88,
			AvoidPatterns:   []string{"script_exec first unless atomic tools fail"},
			Environment:     env,
			CreatedAt:       now.Add(-3 * time.Hour),
			LastUsedAt:      now.Add(-30 * time.Minute),
		},
	}
	for _, memo := range memos {
		if _, err := service.store.UpsertMemo(memo); err != nil {
			t.Fatalf("upsert memo %s: %v", memo.ID, err)
		}
	}

	hint, _, err := service.BuildSelectorHint(selectorHintTestQuery(env), SessionScope{Environment: &env})
	if err != nil {
		t.Fatalf("BuildSelectorHint: %v", err)
	}
	if strings.Count(hint, "read_file") != 1 {
		t.Fatalf("expected deduped read_file tool hint, got %q", hint)
	}
	if strings.Count(hint, "script_exec first unless atomic tools fail") != 1 {
		t.Fatalf("expected deduped caution hint, got %q", hint)
	}
}

func TestDecisionServiceBuildSelectorHintTruncatesLongOutput(t *testing.T) {
	service := newSelectorHintTestService(t)
	env := selectorHintTestEnv()
	now := time.Now().UTC()
	longStrategy := strings.Repeat("inspect the target file and then patch carefully with validation ", 8)
	longCaution := strings.Repeat("destructive scope is unclear and provider credentials may be missing ", 6)
	if _, err := service.store.UpsertMemo(DecisionMemo{
		ID:              "memo-long",
		Namespace:       env.GraphNamespace,
		SessionID:       "session-long",
		IntentSummary:   "fix config migration",
		StrategySummary: longStrategy,
		ToolsUsed:       []DecisionToolUse{{Name: "read_file"}, {Name: "search_files"}, {Name: "apply_diff"}, {Name: "bash_exec"}, {Name: "read_file"}},
		Outcome:         DecisionOutcomePartial,
		Confidence:      0.93,
		ReuseScore:      0.9,
		AvoidPatterns:   []string{longCaution},
		Environment:     env,
		CreatedAt:       now.Add(-2 * time.Hour),
		LastUsedAt:      now.Add(-10 * time.Minute),
	}); err != nil {
		t.Fatalf("upsert long memo: %v", err)
	}

	hint, _, err := service.BuildSelectorHint(selectorHintTestQuery(env), SessionScope{Environment: &env})
	if err != nil {
		t.Fatalf("BuildSelectorHint: %v", err)
	}
	if got := len([]rune(hint)); got > selectorHintMaxChars {
		t.Fatalf("expected hint length <= %d, got %d: %q", selectorHintMaxChars, got, hint)
	}
	if lines := strings.Count(strings.TrimSpace(hint), "\n") + 1; lines > selectorHintMaxLines {
		t.Fatalf("expected <= %d lines, got %d: %q", selectorHintMaxLines, lines, hint)
	}
}

func TestDecisionServiceBuildSelectorHintReturnsEmptyWhenNoHits(t *testing.T) {
	service := newSelectorHintTestService(t)
	env := selectorHintTestEnv()

	hint, hits, err := service.BuildSelectorHint(selectorHintTestQuery(env), SessionScope{Environment: &env})
	if err != nil {
		t.Fatalf("BuildSelectorHint: %v", err)
	}
	if hint != "" {
		t.Fatalf("expected empty hint, got %q", hint)
	}
	if len(hits) != 0 {
		t.Fatalf("expected no hits, got %#v", hits)
	}
}
