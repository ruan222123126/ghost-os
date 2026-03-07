package memory

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDecisionDistiller_CreatesRecipeWhenSupportThresholdMet(t *testing.T) {
	service, distiller, _ := newDecisionRecipeTestService(t, 3)
	namespace := "workspace:test"
	env := decisionRecipeTestEnv("/workspace/ghost-os")
	for i := 0; i < 3; i++ {
		upsertDecisionRecipeTestMemo(t, service, decisionRecipeTestMemo(decisionRecipeTestMemoInput{
			ID:               "memo-success-" + string(rune('a'+i)),
			Namespace:        namespace,
			IntentKey:        "intent.fix_config",
			IntentSummary:    "fix config migration",
			StrategySummary:  "Inspect config.toml, patch it, then run migration check.",
			Outcome:          DecisionOutcomeSuccess,
			ValidationChecks: []string{"run migration check"},
			GraphNodeRefs:    []string{"config.toml"},
			AnchorKeys:       []string{"config.toml", "migration"},
			Tools:            []string{"read_file", "apply_diff", "bash_exec"},
			Environment:      env,
			CreatedAt:        time.Date(2026, 3, 7, 10+i, 0, 0, 0, time.UTC),
		}))
	}
	upsertDecisionRecipeTestMemo(t, service, decisionRecipeTestMemo(decisionRecipeTestMemoInput{
		ID:             "memo-warning",
		Namespace:      namespace,
		IntentKey:      "intent.fix_config",
		IntentSummary:  "fix config migration",
		Outcome:        DecisionOutcomeFailure,
		AvoidPatterns:  []string{"skip the migration check"},
		FailureReasons: []string{"Skipping validation broke startup."},
		Environment:    env,
		CreatedAt:      time.Date(2026, 3, 7, 14, 0, 0, 0, time.UTC),
	}))

	stats, err := distiller.DistillAll(namespace)
	if err != nil {
		t.Fatalf("distill recipes: %v", err)
	}
	if stats.RecipesCreated != 1 {
		t.Fatalf("expected one created recipe, got %+v", stats)
	}
	recipes := service.store.ListRecipes(namespace)
	if len(recipes) != 1 {
		t.Fatalf("expected one recipe, got %d", len(recipes))
	}
	if recipes[0].SupportCount != 3 {
		t.Fatalf("unexpected support count: %+v", recipes[0])
	}
	if recipes[0].SuccessRate < 0.74 {
		t.Fatalf("unexpected success rate: %+v", recipes[0])
	}
}

func TestDecisionDistiller_DoesNotCreateRecipeBelowMinSupport(t *testing.T) {
	service, distiller, _ := newDecisionRecipeTestService(t, 3)
	namespace := "workspace:test"
	env := decisionRecipeTestEnv("/workspace/ghost-os")
	for i := 0; i < 2; i++ {
		upsertDecisionRecipeTestMemo(t, service, decisionRecipeTestMemo(decisionRecipeTestMemoInput{
			ID:              "memo-below-" + string(rune('a'+i)),
			Namespace:       namespace,
			IntentKey:       "intent.fix_config",
			IntentSummary:   "fix config migration",
			StrategySummary: "Inspect config.toml then patch it.",
			Outcome:         DecisionOutcomeSuccess,
			Tools:           []string{"read_file", "apply_diff"},
			Environment:     env,
			CreatedAt:       time.Date(2026, 3, 7, 10+i, 0, 0, 0, time.UTC),
		}))
	}

	stats, err := distiller.DistillAll(namespace)
	if err != nil {
		t.Fatalf("distill recipes: %v", err)
	}
	if stats.RecipesCreated != 0 || len(service.store.ListRecipes(namespace)) != 0 {
		t.Fatalf("expected no recipes below support threshold, got %+v", stats)
	}
}

func TestDecisionDistiller_FailureMemosOnlyContributeCaution(t *testing.T) {
	service, distiller, _ := newDecisionRecipeTestService(t, 2)
	namespace := "workspace:test"
	env := decisionRecipeTestEnv("/workspace/ghost-os")
	for i := 0; i < 2; i++ {
		upsertDecisionRecipeTestMemo(t, service, decisionRecipeTestMemo(decisionRecipeTestMemoInput{
			ID:               "memo-success-caution-" + string(rune('a'+i)),
			Namespace:        namespace,
			IntentKey:        "intent.fix_config",
			IntentSummary:    "fix config migration",
			StrategySummary:  "Inspect config.toml, patch carefully, then validate.",
			Outcome:          DecisionOutcomeSuccess,
			ValidationChecks: []string{"run migration check"},
			Tools:            []string{"read_file", "apply_diff", "bash_exec"},
			Environment:      env,
			CreatedAt:        time.Date(2026, 3, 7, 10+i, 0, 0, 0, time.UTC),
		}))
	}
	upsertDecisionRecipeTestMemo(t, service, decisionRecipeTestMemo(decisionRecipeTestMemoInput{
		ID:             "memo-failure-caution",
		Namespace:      namespace,
		IntentKey:      "intent.fix_config",
		IntentSummary:  "fix config migration",
		Outcome:        DecisionOutcomeAwaitingHuman,
		NeedsHumanFor:  []string{"confirm production database target"},
		AvoidPatterns:  []string{"skip the migration check"},
		FailureReasons: []string{"Patching the wrong config file broke boot."},
		Environment:    env,
		CreatedAt:      time.Date(2026, 3, 7, 15, 0, 0, 0, time.UTC),
	}))

	if _, err := distiller.DistillAll(namespace); err != nil {
		t.Fatalf("distill recipes: %v", err)
	}
	recipe := service.store.ListRecipes(namespace)[0]
	if !containsString(recipe.AvoidPatterns, "skip the migration check") {
		t.Fatalf("expected caution from failure memo, got %+v", recipe)
	}
	if !containsSubstring(recipe.AvoidPatterns, "confirm production database target") {
		t.Fatalf("expected ask-human caution, got %+v", recipe.AvoidPatterns)
	}
	if strings.Contains(strings.ToLower(recipe.StrategySummary), "wrong config file") {
		t.Fatalf("failure details should stay out of strategy summary: %+v", recipe)
	}
}

func TestDecisionDistiller_SeparatesDifferentEnvironmentClusters(t *testing.T) {
	service, distiller, _ := newDecisionRecipeTestService(t, 2)
	namespace := "workspace:test"
	for i := 0; i < 2; i++ {
		upsertDecisionRecipeTestMemo(t, service, decisionRecipeTestMemo(decisionRecipeTestMemoInput{
			ID:              "memo-env-a-" + string(rune('a'+i)),
			Namespace:       namespace,
			IntentKey:       "intent.fix_config",
			IntentSummary:   "fix config migration",
			StrategySummary: "Patch repo A config and verify.",
			Outcome:         DecisionOutcomeSuccess,
			Tools:           []string{"read_file", "apply_diff"},
			Environment:     decisionRecipeTestEnv("/workspace/repo-a"),
			CreatedAt:       time.Date(2026, 3, 7, 10+i, 0, 0, 0, time.UTC),
		}))
		upsertDecisionRecipeTestMemo(t, service, decisionRecipeTestMemo(decisionRecipeTestMemoInput{
			ID:              "memo-env-b-" + string(rune('a'+i)),
			Namespace:       namespace,
			IntentKey:       "intent.fix_config",
			IntentSummary:   "fix config migration",
			StrategySummary: "Patch repo B config and verify.",
			Outcome:         DecisionOutcomeSuccess,
			Tools:           []string{"read_file", "apply_diff"},
			Environment:     decisionRecipeTestEnv("/workspace/repo-b"),
			CreatedAt:       time.Date(2026, 3, 7, 12+i, 0, 0, 0, time.UTC),
		}))
	}

	recipes, clusters, stats, err := distiller.DistillClusters(namespace, service.store.ListMemos(namespace))
	if err != nil {
		t.Fatalf("distill clusters: %v", err)
	}
	if len(recipes) != 2 || len(clusters) != 2 || stats.ClustersBuilt != 2 {
		t.Fatalf("expected two clusters and recipes, got recipes=%d clusters=%d stats=%+v", len(recipes), len(clusters), stats)
	}
	if clusters[0].EnvironmentKey == clusters[1].EnvironmentKey {
		t.Fatalf("expected distinct environment clusters, got %+v", clusters)
	}
}

func TestDecisionDistiller_PrefersStableToolSequenceAcrossMemos(t *testing.T) {
	service, distiller, _ := newDecisionRecipeTestService(t, 3)
	namespace := "workspace:test"
	env := decisionRecipeTestEnv("/workspace/ghost-os")
	toolSets := [][]string{
		{"read_file", "apply_diff", "bash_exec"},
		{"read_file", "apply_diff"},
		{"read_file", "apply_diff", "search_files"},
	}
	for index, tools := range toolSets {
		upsertDecisionRecipeTestMemo(t, service, decisionRecipeTestMemo(decisionRecipeTestMemoInput{
			ID:              "memo-tools-" + string(rune('a'+index)),
			Namespace:       namespace,
			IntentKey:       "intent.fix_config",
			IntentSummary:   "fix config migration",
			StrategySummary: "Inspect config, patch it, then validate.",
			Outcome:         DecisionOutcomeSuccess,
			Tools:           tools,
			Environment:     env,
			CreatedAt:       time.Date(2026, 3, 7, 10+index, 0, 0, 0, time.UTC),
		}))
	}

	if _, err := distiller.DistillAll(namespace); err != nil {
		t.Fatalf("distill recipes: %v", err)
	}
	recipe := service.store.ListRecipes(namespace)[0]
	if len(recipe.RecommendedTools) < 2 {
		t.Fatalf("expected stable tool prefix, got %+v", recipe.RecommendedTools)
	}
	if recipe.RecommendedTools[0] != "read_file" || recipe.RecommendedTools[1] != "apply_diff" {
		t.Fatalf("unexpected stable tool prefix: %+v", recipe.RecommendedTools)
	}
}

func TestDecisionDistiller_CollectsValidationChecklist(t *testing.T) {
	service, distiller, _ := newDecisionRecipeTestService(t, 3)
	namespace := "workspace:test"
	env := decisionRecipeTestEnv("/workspace/ghost-os")
	validations := [][]string{
		{"run migration check", "verify config loads"},
		{"run migration check"},
		{"run migration check", "build"},
	}
	for index, checks := range validations {
		upsertDecisionRecipeTestMemo(t, service, decisionRecipeTestMemo(decisionRecipeTestMemoInput{
			ID:               "memo-validation-" + string(rune('a'+index)),
			Namespace:        namespace,
			IntentKey:        "intent.fix_config",
			IntentSummary:    "fix config migration",
			StrategySummary:  "Patch config.toml and verify promptly.",
			Outcome:          DecisionOutcomeSuccess,
			ValidationChecks: checks,
			Tools:            []string{"read_file", "apply_diff", "bash_exec"},
			Environment:      env,
			CreatedAt:        time.Date(2026, 3, 7, 10+index, 0, 0, 0, time.UTC),
		}))
	}

	if _, err := distiller.DistillAll(namespace); err != nil {
		t.Fatalf("distill recipes: %v", err)
	}
	recipe := service.store.ListRecipes(namespace)[0]
	if !containsString(recipe.ValidationChecklist, "run migration check") {
		t.Fatalf("expected shared validation check, got %+v", recipe.ValidationChecklist)
	}
}

func TestDecisionDistiller_UpsertsClustersAndRecipes(t *testing.T) {
	service, distiller, dir := newDecisionRecipeTestService(t, 2)
	namespace := "workspace:test"
	env := decisionRecipeTestEnv("/workspace/ghost-os")
	for i := 0; i < 2; i++ {
		upsertDecisionRecipeTestMemo(t, service, decisionRecipeTestMemo(decisionRecipeTestMemoInput{
			ID:               "memo-upsert-" + string(rune('a'+i)),
			Namespace:        namespace,
			IntentKey:        "intent.fix_config",
			IntentSummary:    "fix config migration",
			StrategySummary:  "Inspect config.toml, patch it, then run migration check.",
			Outcome:          DecisionOutcomeSuccess,
			ValidationChecks: []string{"run migration check"},
			Tools:            []string{"read_file", "apply_diff"},
			Environment:      env,
			CreatedAt:        time.Date(2026, 3, 7, 10+i, 0, 0, 0, time.UTC),
		}))
	}

	stats, err := distiller.DistillAll(namespace)
	if err != nil {
		t.Fatalf("initial distill: %v", err)
	}
	if stats.RecipesCreated != 1 {
		t.Fatalf("expected recipe creation, got %+v", stats)
	}
	if len(service.store.ListClusters(namespace)) != 1 || len(service.store.ListRecipes(namespace)) != 1 {
		t.Fatalf("expected persisted cluster and recipe")
	}
	if _, err := os.Stat(filepath.Join(dir, defaultDecisionRecipesPathName)); err != nil {
		t.Fatalf("expected recipe snapshot to be persisted: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, defaultDecisionClustersPathName)); err != nil {
		t.Fatalf("expected cluster snapshot to be persisted: %v", err)
	}

	upsertDecisionRecipeTestMemo(t, service, decisionRecipeTestMemo(decisionRecipeTestMemoInput{
		ID:               "memo-upsert-b",
		Namespace:        namespace,
		IntentKey:        "intent.fix_config",
		IntentSummary:    "fix config migration",
		StrategySummary:  "Inspect config.toml carefully before patching.",
		Outcome:          DecisionOutcomeSuccess,
		ValidationChecks: []string{"run migration check", "verify config loads"},
		Tools:            []string{"read_file", "apply_diff"},
		Environment:      env,
		CreatedAt:        time.Date(2026, 3, 7, 16, 0, 0, 0, time.UTC),
	}))

	stats, err = distiller.DistillAll(namespace)
	if err != nil {
		t.Fatalf("second distill: %v", err)
	}
	if stats.RecipesUpdated != 1 {
		t.Fatalf("expected recipe update on second distill, got %+v", stats)
	}
}

type decisionRecipeTestMemoInput struct {
	ID               string
	Namespace        string
	IntentKey        string
	IntentSummary    string
	StrategySummary  string
	Outcome          string
	ValidationChecks []string
	AvoidPatterns    []string
	NeedsHumanFor    []string
	FailureReasons   []string
	GraphNodeRefs    []string
	AnchorKeys       []string
	Tools            []string
	Environment      DecisionEnvFingerprint
	CreatedAt        time.Time
}

func newDecisionRecipeTestService(t *testing.T, minSupport int) (*DecisionService, *DecisionDistiller, string) {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "decision")
	service := NewDecisionService(MemoryConfig{
		DecisionEnabled:          true,
		DecisionCaptureOnTurn:    true,
		DecisionCaptureOnTurnSet: true,
		DecisionPath:             dir,
		DecisionRecipeEnabled:    true,
		DecisionRecipeMinSupport: minSupport,
		DecisionRecipeInterval:   time.Hour,
	}, nil)
	if service.distiller == nil {
		service.distiller = NewDecisionDistiller(service, time.Hour, minSupport)
	}
	return service, service.distiller, dir
}

func decisionRecipeTestEnv(workspaceRoot string) DecisionEnvFingerprint {
	return DecisionEnvFingerprint{
		WorkspaceRoot:    workspaceRoot,
		Platform:         "linux/amd64",
		GraphNamespace:   "workspace:test",
		Domain:           "coding",
		ToolNames:        []string{"read_file", "apply_diff", "bash_exec"},
		ToolsetSignature: "apply_diff,bash_exec,read_file",
		PathHints:        []string{"config.toml"},
	}
}

func decisionRecipeTestMemo(input decisionRecipeTestMemoInput) DecisionMemo {
	tools := make([]DecisionToolUse, 0, len(input.Tools))
	for _, name := range input.Tools {
		tools = append(tools, DecisionToolUse{Name: name, Purpose: decisionToolPurpose(name)})
	}
	createdAt := input.CreatedAt.UTC()
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	return DecisionMemo{
		ID:               input.ID,
		Namespace:        input.Namespace,
		SessionID:        "session-" + input.ID,
		TraceID:          "trace-" + input.ID,
		TurnID:           "turn-" + input.ID,
		IntentKey:        input.IntentKey,
		IntentSummary:    input.IntentSummary,
		StrategySummary:  input.StrategySummary,
		Outcome:          input.Outcome,
		ValidationChecks: input.ValidationChecks,
		AvoidPatterns:    input.AvoidPatterns,
		NeedsHumanFor:    input.NeedsHumanFor,
		FailureReasons:   input.FailureReasons,
		GraphNodeRefs:    input.GraphNodeRefs,
		AnchorKeys:       input.AnchorKeys,
		ToolsUsed:        tools,
		Confidence:       0.92,
		ReuseScore:       0.88,
		CreatedAt:        createdAt,
		LastUsedAt:       createdAt.Add(30 * time.Minute),
		Environment:      input.Environment,
	}
}

func upsertDecisionRecipeTestMemo(t *testing.T, service *DecisionService, memo DecisionMemo) {
	t.Helper()
	if _, err := service.store.UpsertMemo(memo); err != nil {
		t.Fatalf("upsert memo %s: %v", memo.ID, err)
	}
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) == target {
			return true
		}
	}
	return false
}

func containsSubstring(values []string, needle string) bool {
	for _, value := range values {
		if strings.Contains(value, needle) {
			return true
		}
	}
	return false
}
