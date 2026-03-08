package memory

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"ghost-os/bridge/agent"
	"ghost-os/bridge/llm"
)

func newRecipeReuseManager(t *testing.T) *MemoryManager {
	t.Helper()
	baseDir := t.TempDir()
	return NewMemoryManager(MemoryConfig{
		WarmCapacity:                      32,
		WarmPath:                          filepath.Join(baseDir, "warm.json"),
		ColdBaseDir:                       filepath.Join(baseDir, "cold"),
		AutoRecallEnabled:                 true,
		AutoRecallLimit:                   4,
		DecisionEnabled:                   true,
		DecisionPath:                      filepath.Join(baseDir, "decision"),
		DecisionRecipeEnabled:             true,
		DecisionRecipeMinSupport:          2,
		RecipeReuseEnabled:                true,
		RecipeReuseEnabledSet:             true,
		RecipeExecutionTrackingEnabled:    true,
		RecipeExecutionTrackingEnabledSet: true,
		RecipeDefaultEnabled:              true,
		RecipeDefaultEnabledSet:           true,
		RecipeDefaultGrayPercent:          100,
		RecipeMinSelectionConfidence:      0.60,
		RecipeMinSuccessRate:              0.60,
	})
}

func TestRecipeSelectionPrefersEnvironmentMatchedRecipe(t *testing.T) {
	manager := newRecipeReuseManager(t)
	now := time.Date(2026, 3, 7, 13, 0, 0, 0, time.UTC)
	matchedEnv := DecisionEnvFingerprint{
		WorkspaceRoot:    "/workspace/ghost-os",
		Platform:         "linux/amd64",
		GraphNamespace:   "workspace:test",
		ToolsetSignature: "apply_diff,bash_exec,read_file",
		ToolNames:        []string{"read_file", "apply_diff", "bash_exec"},
	}
	mismatchedEnv := DecisionEnvFingerprint{
		WorkspaceRoot:    "/workspace/other",
		Platform:         "linux/amd64",
		GraphNamespace:   "workspace:test",
		ToolsetSignature: "apply_diff,bash_exec,read_file",
		ToolNames:        []string{"read_file", "apply_diff", "bash_exec"},
	}
	if _, err := manager.decision.debugUpsertMemo(DecisionMemo{
		ID:            "memo-recipe-match",
		Namespace:     "workspace:test",
		SessionID:     "session-match",
		IntentKey:     "intent.fix_config_migration",
		IntentSummary: "fix config migration",
		Outcome:       DecisionOutcomeSuccess,
		Confidence:    0.92,
		ReuseScore:    0.88,
		CreatedAt:     now.Add(-2 * time.Hour),
		LastUsedAt:    now.Add(-time.Hour),
		Environment:   matchedEnv,
	}); err != nil {
		t.Fatalf("upsert matched memo: %v", err)
	}
	if _, err := manager.decision.debugUpsertMemo(DecisionMemo{
		ID:            "memo-recipe-mismatch",
		Namespace:     "workspace:test",
		SessionID:     "session-mismatch",
		IntentKey:     "intent.fix_config_migration",
		IntentSummary: "fix config migration",
		Outcome:       DecisionOutcomeSuccess,
		Confidence:    0.95,
		ReuseScore:    0.93,
		CreatedAt:     now.Add(-90 * time.Minute),
		LastUsedAt:    now.Add(-45 * time.Minute),
		Environment:   mismatchedEnv,
	}); err != nil {
		t.Fatalf("upsert mismatched memo: %v", err)
	}
	if _, err := manager.decision.debugUpsertRecipe(DecisionRecipe{
		ID:                  "recipe-match",
		Namespace:           "workspace:test",
		IntentKey:           "intent.fix_config_migration",
		EnvironmentKey:      decisionClusterEnvironmentKey(matchedEnv, nil, nil),
		StrategySummary:     "Inspect config, patch, then verify migration.",
		RecommendedTools:    []string{"read_file", "apply_diff", "bash_exec"},
		OrderedActions:      []RecipeStep{{Title: "Inspect", ToolName: "read_file", Instruction: "Read config.toml"}, {Title: "Patch", ToolName: "apply_diff", Instruction: "Apply the migration patch"}},
		ValidationChecklist: []string{"run migration check"},
		SupportCount:        4,
		SuccessRate:         0.86,
		Confidence:          0.84,
		Status:              RecipeStatusActive,
		SourceMemoIDs:       []string{"memo-recipe-match"},
		CreatedAt:           now.Add(-2 * time.Hour),
		UpdatedAt:           now.Add(-30 * time.Minute),
	}); err != nil {
		t.Fatalf("upsert matched recipe: %v", err)
	}
	if _, err := manager.decision.debugUpsertRecipe(DecisionRecipe{
		ID:               "recipe-mismatch",
		Namespace:        "workspace:test",
		IntentKey:        "intent.fix_config_migration",
		EnvironmentKey:   decisionClusterEnvironmentKey(mismatchedEnv, nil, nil),
		StrategySummary:  "Patch quickly, then verify.",
		RecommendedTools: []string{"apply_diff", "bash_exec"},
		OrderedActions:   []RecipeStep{{Title: "Patch", ToolName: "apply_diff", Instruction: "Patch config.toml"}},
		SupportCount:     3,
		SuccessRate:      0.90,
		Confidence:       0.86,
		Status:           RecipeStatusActive,
		SourceMemoIDs:    []string{"memo-recipe-mismatch"},
		CreatedAt:        now.Add(-90 * time.Minute),
		UpdatedAt:        now.Add(-20 * time.Minute),
	}); err != nil {
		t.Fatalf("upsert mismatched recipe: %v", err)
	}

	result, err := manager.QueryResultWithScope(MemoryQuery{
		IncludeDecision: true,
		DecisionTypes:   []string{DecisionHitTypeRecipe},
		SemanticQuery:   "fix config migration",
		Environment:     &matchedEnv,
	}, SessionScope{SessionID: "session-select", Environment: &matchedEnv})
	if err != nil {
		t.Fatalf("query result with recipe selection: %v", err)
	}
	if result.SelectedRecipe == nil {
		t.Fatalf("expected selected recipe, got %+v", result)
	}
	if result.SelectedRecipe.ID != "recipe-match" {
		t.Fatalf("expected matched recipe to win, got %+v", result.SelectedRecipe)
	}
	if result.RecipeAdvisory == nil || !result.RecipeAdvisory.ApplyByDefault {
		t.Fatalf("expected default-on advisory, got %+v", result.RecipeAdvisory)
	}
	if result.RecipeSelection == nil || len(result.RecipeSelection.WhySelected) == 0 {
		t.Fatalf("expected selection report reasons, got %+v", result.RecipeSelection)
	}
}

func TestRecipeExecutionTrackingWritesRunAndFeedback(t *testing.T) {
	manager := newRecipeReuseManager(t)
	now := time.Date(2026, 3, 7, 14, 0, 0, 0, time.UTC)
	env := DecisionEnvFingerprint{
		WorkspaceRoot:    "/workspace/ghost-os",
		Platform:         "linux/amd64",
		GraphNamespace:   "workspace:test",
		ToolsetSignature: "apply_diff,read_file",
		ToolNames:        []string{"read_file", "apply_diff"},
	}
	if _, err := manager.decision.debugUpsertMemo(DecisionMemo{
		ID:            "memo-track-source",
		Namespace:     "workspace:test",
		SessionID:     "session-track-source",
		IntentKey:     "intent.fix_config_migration",
		IntentSummary: "fix config migration",
		Outcome:       DecisionOutcomeSuccess,
		Confidence:    0.94,
		ReuseScore:    0.90,
		CreatedAt:     now.Add(-2 * time.Hour),
		LastUsedAt:    now.Add(-time.Hour),
		Environment:   env,
	}); err != nil {
		t.Fatalf("upsert source memo: %v", err)
	}
	if _, err := manager.decision.debugUpsertRecipe(DecisionRecipe{
		ID:               "recipe-track",
		Namespace:        "workspace:test",
		IntentKey:        "intent.fix_config_migration",
		EnvironmentKey:   decisionClusterEnvironmentKey(env, nil, nil),
		StrategySummary:  "Read first, then patch.",
		RecommendedTools: []string{"read_file", "apply_diff"},
		OrderedActions: []RecipeStep{
			{Title: "Inspect", ToolName: "read_file", Instruction: "Read config.toml", Validation: "config opened"},
			{Title: "Patch", ToolName: "apply_diff", Instruction: "Apply migration patch", Validation: "patch applied"},
		},
		ValidationChecklist: []string{"config opened", "patch applied"},
		SupportCount:        3,
		SuccessRate:         0.84,
		Confidence:          0.86,
		Status:              RecipeStatusActive,
		SourceMemoIDs:       []string{"memo-track-source"},
		CreatedAt:           now.Add(-2 * time.Hour),
		UpdatedAt:           now.Add(-20 * time.Minute),
	}); err != nil {
		t.Fatalf("upsert recipe: %v", err)
	}

	hint, hits, err := manager.BuildDecisionSelectorHintWithScope(SessionScope{SessionID: "session-track", Environment: &env}, "fix config migration")
	if err != nil {
		t.Fatalf("build selector hint: %v", err)
	}
	if hint == "" || len(hits) == 0 {
		t.Fatalf("expected staged recipe hint, got hint=%q hits=%+v", hint, hits)
	}

	input := DecisionCaptureInput{
		Namespace:      "workspace:test",
		SessionID:      "session-track",
		TraceID:        "trace-track",
		TurnID:         "turn-track",
		UserMessage:    "fix config migration",
		Outcome:        DecisionOutcomeSuccess,
		TurnStartedAt:  now,
		TurnFinishedAt: now.Add(2 * time.Minute),
		Environment:    env,
		NewMessages: []llm.Message{
			{Role: llm.RoleUser, Text: "fix config migration"},
			{Role: llm.RoleAssistant, ToolCalls: []llm.ToolCall{{ID: "call-patch", Name: "apply_diff", Arguments: json.RawMessage(`{"path":"config.toml"}`)}, {ID: "call-read", Name: "read_file", Arguments: json.RawMessage(`{"path":"config.toml"}`)}}},
			{Role: llm.RoleTool, ToolCallID: "call-patch", Text: agent.FormatToolResult("apply_diff", "trace-track", "patch applied", nil)},
			{Role: llm.RoleTool, ToolCallID: "call-read", Text: agent.FormatToolResult("read_file", "trace-track", "config opened", nil)},
		},
	}
	if err := manager.CaptureDecisionTurn(input); err != nil {
		t.Fatalf("capture decision turn with recipe feedback: %v", err)
	}
	runs := manager.decision.debugListRecipeRuns("workspace:test")
	if len(runs) != 1 {
		t.Fatalf("expected one recipe run, got %+v", runs)
	}
	run := runs[0]
	if run.RecipeID != "recipe-track" {
		t.Fatalf("unexpected recipe run: %+v", run)
	}
	if run.Feedback.Outcome != DecisionOutcomeSuccess {
		t.Fatalf("expected success feedback, got %+v", run.Feedback)
	}
	if !recipeRunHasDeviation(run) {
		t.Fatalf("expected order deviation to be recorded, got %+v", run.Steps)
	}
	recipe, ok := manager.decision.debugRecipe("recipe-track")
	if !ok {
		t.Fatal("expected recipe stats after feedback")
	}
	if recipe.SelectedCount != 1 || recipe.AppliedCount != 1 || recipe.SuccessCount != 1 {
		t.Fatalf("expected selection/applied/success counters to update, got %+v", recipe)
	}
	if recipe.DeviationCount != 1 {
		t.Fatalf("expected deviation counter to update, got %+v", recipe)
	}
	metrics := manager.Metrics()
	if metrics.RecipeSelectedCount != 1 || metrics.RecipeAppliedCount != 1 {
		t.Fatalf("expected recipe metrics to update, got %+v", metrics)
	}
	if metrics.RecipeDeviationRate <= 0 {
		t.Fatalf("expected deviation rate > 0, got %+v", metrics)
	}
}

func TestBuildContextWindowPrependsRecipeAdvisory(t *testing.T) {
	manager := newRecipeReuseManager(t)
	now := time.Date(2026, 3, 8, 9, 0, 0, 0, time.UTC)
	env := DecisionEnvFingerprint{
		WorkspaceRoot:    "/workspace/ghost-os",
		Platform:         "linux/amd64",
		GraphNamespace:   "workspace:test",
		ToolsetSignature: "apply_diff,bash_exec,read_file",
		ToolNames:        []string{"read_file", "apply_diff", "bash_exec"},
	}
	if _, err := manager.decision.debugUpsertMemo(DecisionMemo{
		ID:            "memo-recipe-window",
		Namespace:     "workspace:test",
		SessionID:     "session-recipe-window-source",
		IntentKey:     "intent.fix_config_migration",
		IntentSummary: "fix config migration",
		Outcome:       DecisionOutcomeSuccess,
		Confidence:    0.93,
		ReuseScore:    0.90,
		CreatedAt:     now.Add(-2 * time.Hour),
		LastUsedAt:    now.Add(-time.Hour),
		Environment:   env,
	}); err != nil {
		t.Fatalf("upsert recipe window memo: %v", err)
	}
	if _, err := manager.decision.debugUpsertRecipe(DecisionRecipe{
		ID:                  "recipe-window",
		Namespace:           "workspace:test",
		IntentKey:           "intent.fix_config_migration",
		EnvironmentKey:      decisionClusterEnvironmentKey(env, nil, nil),
		StrategySummary:     "Inspect config, patch, then verify migration.",
		RecommendedTools:    []string{"read_file", "apply_diff", "bash_exec"},
		OrderedActions:      []RecipeStep{{Title: "Inspect", ToolName: "read_file", Instruction: "Read config.toml"}, {Title: "Patch", ToolName: "apply_diff", Instruction: "Apply the migration patch"}},
		ValidationChecklist: []string{"run migration check"},
		AvoidPatterns:       []string{"rewrite unrelated files"},
		SupportCount:        4,
		SuccessRate:         0.88,
		Confidence:          0.86,
		Status:              RecipeStatusActive,
		SourceMemoIDs:       []string{"memo-recipe-window"},
		CreatedAt:           now.Add(-2 * time.Hour),
		UpdatedAt:           now.Add(-20 * time.Minute),
	}); err != nil {
		t.Fatalf("upsert recipe window recipe: %v", err)
	}

	window, err := manager.BuildContextWindowWithScope(SessionScope{
		SessionID:   "session-recipe-window",
		Environment: &env,
	}, "fix config migration")
	if err != nil {
		t.Fatalf("build context window with recipe advisory: %v", err)
	}
	if len(window) != 1 {
		t.Fatalf("unexpected context window size: got %d want 1", len(window))
	}
	text := window[0].Text
	selectedIdx := strings.Index(text, "Selected recipe: start with")
	recallIdx := strings.Index(text, "Prior similar experience:")
	if selectedIdx < 0 {
		t.Fatalf("expected recipe advisory line in context window, got %q", text)
	}
	if recallIdx < 0 {
		t.Fatalf("expected decision recall line in context window, got %q", text)
	}
	if selectedIdx > recallIdx {
		t.Fatalf("expected recipe advisory to stay ahead of recall summary, got %q", text)
	}
	if !strings.Contains(text, "run migration check") {
		t.Fatalf("expected recipe validation hint in context window, got %q", text)
	}
	if strings.Contains(text, "apply_by_default") || strings.Contains(text, "recommended_tools") {
		t.Fatalf("expected compact advisory only, got %q", text)
	}
}
