package memory

import (
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func TestDecisionStoreLoadMissingFilesReturnsEmpty(t *testing.T) {
	store := NewDecisionStore(filepath.Join(t.TempDir(), "decision"))
	if err := store.Load(); err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if len(store.ListMemos("")) != 0 {
		t.Fatalf("expected empty memo list")
	}
	if len(store.ListRecipes("")) != 0 {
		t.Fatalf("expected empty recipe list")
	}
	if len(store.ListClusters("")) != 0 {
		t.Fatalf("expected empty cluster list")
	}
	stats := store.Stats(defaultDecisionNamespace)
	if stats.MemoCount != 0 || stats.RecipeCount != 0 || stats.ClusterCount != 0 {
		t.Fatalf("expected zero stats, got %+v", stats)
	}
}

func TestDecisionStoreUpsertMemoCreatesAndReplacesByID(t *testing.T) {
	store := NewDecisionStore(filepath.Join(t.TempDir(), "decision"))
	localTime := time.Date(2026, 3, 7, 10, 30, 0, 0, time.FixedZone("CST", 8*3600))

	created, err := store.UpsertMemo(DecisionMemo{
		ID:            " memo-1 ",
		IntentKey:     " intent.fix ",
		IntentSummary: "  fix config parse  ",
		Constraints:   []string{" shell-first ", "", "shell-first"},
		GraphNodeRefs: []string{" node-1 ", "", "node-1"},
		AnchorKeys:    []string{" anchor-1 ", "anchor-1", ""},
		ToolsUsed:     []DecisionToolUse{{Name: " bash_exec "}},
		Confidence:    1.4,
		ReuseScore:    -1,
		CreatedAt:     localTime,
		LastUsedAt:    localTime,
		Environment:   DecisionEnvFingerprint{ToolNames: []string{" bash_exec ", "bash_exec"}},
		AccessCount:   -2,
	})
	if err != nil {
		t.Fatalf("UpsertMemo returned error: %v", err)
	}
	if !created {
		t.Fatalf("expected memo create path")
	}

	memo, ok := store.Memo("memo-1")
	if !ok {
		t.Fatalf("expected memo to exist")
	}
	if memo.Namespace != defaultDecisionNamespace {
		t.Fatalf("unexpected namespace: got %q want %q", memo.Namespace, defaultDecisionNamespace)
	}
	if memo.IntentKey != "intent.fix" {
		t.Fatalf("unexpected intent key: %q", memo.IntentKey)
	}
	if !reflect.DeepEqual(memo.Constraints, []string{"shell-first"}) {
		t.Fatalf("unexpected constraints: %#v", memo.Constraints)
	}
	if !reflect.DeepEqual(memo.GraphNodeRefs, []string{"node-1"}) {
		t.Fatalf("unexpected graph refs: %#v", memo.GraphNodeRefs)
	}
	if memo.Confidence != 1 || memo.ReuseScore != 0 {
		t.Fatalf("unexpected confidence/reuse: %v/%v", memo.Confidence, memo.ReuseScore)
	}
	if memo.AccessCount != 0 {
		t.Fatalf("unexpected access count: %d", memo.AccessCount)
	}
	if memo.CreatedAt.Location() != time.UTC || memo.LastUsedAt.Location() != time.UTC {
		t.Fatalf("expected UTC timestamps, got created=%s last=%s", memo.CreatedAt.Location(), memo.LastUsedAt.Location())
	}
	if !reflect.DeepEqual(memo.Environment.ToolNames, []string{"bash_exec"}) {
		t.Fatalf("unexpected env tools: %#v", memo.Environment.ToolNames)
	}

	updated := memo
	updated.ContextSummary = "updated summary"
	updated.Outcome = DecisionOutcomeSuccess
	updated.AccessCount = 3
	created, err = store.UpsertMemo(updated)
	if err != nil {
		t.Fatalf("UpsertMemo update returned error: %v", err)
	}
	if created {
		t.Fatalf("expected memo update path")
	}

	memo, ok = store.Memo("memo-1")
	if !ok || memo.ContextSummary != "updated summary" || memo.Outcome != DecisionOutcomeSuccess || memo.AccessCount != 3 {
		t.Fatalf("unexpected updated memo: %+v ok=%v", memo, ok)
	}

	memo.ContextSummary = "mutated"
	reloaded, ok := store.Memo("memo-1")
	if !ok || reloaded.ContextSummary != "updated summary" {
		t.Fatalf("expected clone-on-read, got %+v ok=%v", reloaded, ok)
	}
}

func TestDecisionStoreUpsertRecipeAndCluster(t *testing.T) {
	store := NewDecisionStore(filepath.Join(t.TempDir(), "decision"))
	localTime := time.Date(2026, 3, 7, 11, 0, 0, 0, time.FixedZone("PST", -8*3600))

	created, err := store.UpsertRecipe(DecisionRecipe{
		ID:                  " recipe-1 ",
		Namespace:           " workspace:a ",
		IntentKey:           " intent.fix ",
		TriggerPhrases:      []string{" open config ", "open config"},
		RecommendedTools:    []string{" bash_exec ", "bash_exec"},
		ValidationChecklist: []string{" run config test ", "run config test"},
		SuccessRate:         1.8,
		Confidence:          -0.4,
		SourceMemoIDs:       []string{" memo-1 ", "memo-1"},
		GraphRefs:           []string{" node:1 ", "node:1"},
		AnchorKeys:          []string{" anchor-1 ", "anchor-1"},
		CreatedAt:           localTime,
		UpdatedAt:           localTime,
	})
	if err != nil {
		t.Fatalf("UpsertRecipe returned error: %v", err)
	}
	if !created {
		t.Fatalf("expected recipe create path")
	}

	recipe, ok := store.Recipe("recipe-1")
	if !ok {
		t.Fatalf("expected recipe to exist")
	}
	if recipe.Namespace != "workspace:a" {
		t.Fatalf("unexpected recipe namespace: %q", recipe.Namespace)
	}
	if recipe.SuccessRate != 1 || recipe.Confidence != 0 {
		t.Fatalf("unexpected recipe scores: %v/%v", recipe.SuccessRate, recipe.Confidence)
	}
	if recipe.UpdatedAt.Location() != time.UTC {
		t.Fatalf("expected recipe UTC timestamp, got %s", recipe.UpdatedAt.Location())
	}

	created, err = store.UpsertCluster(DecisionCluster{
		ID:             " cluster-1 ",
		Namespace:      " workspace:a ",
		IntentKey:      " intent.fix ",
		EnvironmentKey: " env:local ",
		MemoIDs:        []string{" memo-1 ", "memo-1"},
		SuccessCount:   -1,
		FailureCount:   2,
		RecipeID:       " recipe-1 ",
		UpdatedAt:      localTime,
	})
	if err != nil {
		t.Fatalf("UpsertCluster returned error: %v", err)
	}
	if !created {
		t.Fatalf("expected cluster create path")
	}

	cluster, ok := store.Cluster("cluster-1")
	if !ok {
		t.Fatalf("expected cluster to exist")
	}
	if cluster.SuccessCount != 0 || cluster.FailureCount != 2 {
		t.Fatalf("unexpected cluster counts: %+v", cluster)
	}
	if !reflect.DeepEqual(cluster.MemoIDs, []string{"memo-1"}) {
		t.Fatalf("unexpected cluster memo ids: %#v", cluster.MemoIDs)
	}
	if cluster.UpdatedAt.Location() != time.UTC {
		t.Fatalf("expected cluster UTC timestamp, got %s", cluster.UpdatedAt.Location())
	}
}

func TestDecisionStorePersistAndReloadRebuildsIndexes(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "decision")
	store := NewDecisionStore(dir)
	base := time.Date(2026, 3, 7, 9, 0, 0, 0, time.UTC)

	memoA := DecisionMemo{
		ID:            "memo-a",
		Namespace:     "workspace:a",
		IntentKey:     "intent.fix",
		GraphNodeRefs: []string{"node-1"},
		AnchorKeys:    []string{"anchor-1"},
		ToolsUsed:     []DecisionToolUse{{Name: "bash_exec"}},
		Outcome:       DecisionOutcomeSuccess,
		CreatedAt:     base,
		LastUsedAt:    base.Add(time.Minute),
	}
	memoB := DecisionMemo{
		ID:            "memo-b",
		Namespace:     "workspace:a",
		IntentKey:     "intent.fix",
		GraphNodeRefs: []string{"node-2"},
		AnchorKeys:    []string{"anchor-1", "anchor-2"},
		ToolsUsed:     []DecisionToolUse{{Name: "browser_query"}},
		Outcome:       DecisionOutcomePartial,
		CreatedAt:     base.Add(time.Hour),
		LastUsedAt:    base.Add(2 * time.Hour),
	}
	recipe := DecisionRecipe{
		ID:        "recipe-a",
		Namespace: "workspace:a",
		IntentKey: "intent.fix",
		CreatedAt: base,
		UpdatedAt: base.Add(3 * time.Hour),
	}
	cluster := DecisionCluster{
		ID:        "cluster-a",
		Namespace: "workspace:a",
		IntentKey: "intent.fix",
		MemoIDs:   []string{"memo-a", "memo-b"},
		UpdatedAt: base.Add(4 * time.Hour),
	}

	if _, err := store.UpsertMemo(memoA); err != nil {
		t.Fatalf("upsert memoA: %v", err)
	}
	if _, err := store.UpsertMemo(memoB); err != nil {
		t.Fatalf("upsert memoB: %v", err)
	}
	if _, err := store.UpsertRecipe(recipe); err != nil {
		t.Fatalf("upsert recipe: %v", err)
	}
	if _, err := store.UpsertCluster(cluster); err != nil {
		t.Fatalf("upsert cluster: %v", err)
	}
	if err := store.Persist(); err != nil {
		t.Fatalf("Persist returned error: %v", err)
	}

	reloaded := NewDecisionStore(dir)
	if err := reloaded.Load(); err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if !reflect.DeepEqual(reloaded.ListMemos("workspace:a"), store.ListMemos("workspace:a")) {
		t.Fatalf("memo reload mismatch: got %#v want %#v", reloaded.ListMemos("workspace:a"), store.ListMemos("workspace:a"))
	}
	if !reflect.DeepEqual(reloaded.ListRecipes("workspace:a"), store.ListRecipes("workspace:a")) {
		t.Fatalf("recipe reload mismatch")
	}
	if !reflect.DeepEqual(reloaded.ListClusters("workspace:a"), store.ListClusters("workspace:a")) {
		t.Fatalf("cluster reload mismatch")
	}
	if !reflect.DeepEqual(reloaded.memoIDsByIntentKey["intent.fix"], []string{"memo-a", "memo-b"}) {
		t.Fatalf("unexpected memoIDsByIntentKey: %#v", reloaded.memoIDsByIntentKey)
	}
	if !reflect.DeepEqual(reloaded.memosByGraphNode["node-1"], []string{"memo-a"}) {
		t.Fatalf("unexpected memosByGraphNode: %#v", reloaded.memosByGraphNode)
	}
	if !reflect.DeepEqual(reloaded.memosByAnchorKey["anchor-1"], []string{"memo-a", "memo-b"}) {
		t.Fatalf("unexpected memosByAnchorKey: %#v", reloaded.memosByAnchorKey)
	}
	if !reflect.DeepEqual(reloaded.memosByToolName["bash_exec"], []string{"memo-a"}) {
		t.Fatalf("unexpected memosByToolName: %#v", reloaded.memosByToolName)
	}
	if !reflect.DeepEqual(reloaded.recipeIDsByIntentKey["intent.fix"], []string{"recipe-a"}) {
		t.Fatalf("unexpected recipeIDsByIntentKey: %#v", reloaded.recipeIDsByIntentKey)
	}
}

func TestDecisionStoreResetNamespaceOnlyClearsTarget(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "decision")
	store := NewDecisionStore(dir)
	now := time.Date(2026, 3, 7, 12, 0, 0, 0, time.UTC)

	entries := []DecisionMemo{
		{ID: "memo-a", Namespace: "workspace:a", Outcome: DecisionOutcomeSuccess, CreatedAt: now, LastUsedAt: now},
		{ID: "memo-b", Namespace: "workspace:b", Outcome: DecisionOutcomeFailure, CreatedAt: now, LastUsedAt: now},
	}
	for _, memo := range entries {
		if _, err := store.UpsertMemo(memo); err != nil {
			t.Fatalf("upsert memo %s: %v", memo.ID, err)
		}
	}
	if _, err := store.UpsertRecipe(DecisionRecipe{ID: "recipe-a", Namespace: "workspace:a", CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatalf("upsert recipe-a: %v", err)
	}
	if _, err := store.UpsertRecipe(DecisionRecipe{ID: "recipe-b", Namespace: "workspace:b", CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatalf("upsert recipe-b: %v", err)
	}
	if _, err := store.UpsertCluster(DecisionCluster{ID: "cluster-a", Namespace: "workspace:a", UpdatedAt: now}); err != nil {
		t.Fatalf("upsert cluster-a: %v", err)
	}
	if _, err := store.UpsertCluster(DecisionCluster{ID: "cluster-b", Namespace: "workspace:b", UpdatedAt: now}); err != nil {
		t.Fatalf("upsert cluster-b: %v", err)
	}
	if err := store.Persist(); err != nil {
		t.Fatalf("persist before reset: %v", err)
	}

	if err := store.ResetNamespace("workspace:a"); err != nil {
		t.Fatalf("ResetNamespace returned error: %v", err)
	}
	if len(store.ListMemos("workspace:a")) != 0 || len(store.ListRecipes("workspace:a")) != 0 || len(store.ListClusters("workspace:a")) != 0 {
		t.Fatalf("expected namespace workspace:a to be cleared")
	}
	if len(store.ListMemos("workspace:b")) != 1 || len(store.ListRecipes("workspace:b")) != 1 || len(store.ListClusters("workspace:b")) != 1 {
		t.Fatalf("expected namespace workspace:b to remain intact")
	}

	reloaded := NewDecisionStore(dir)
	if err := reloaded.Load(); err != nil {
		t.Fatalf("reload after reset: %v", err)
	}
	if len(reloaded.ListMemos("workspace:a")) != 0 || len(reloaded.ListMemos("workspace:b")) != 1 {
		t.Fatalf("unexpected reload state after reset")
	}
}

func TestDecisionStoreStatsCountsNamespace(t *testing.T) {
	store := NewDecisionStore(filepath.Join(t.TempDir(), "decision"))
	now := time.Date(2026, 3, 7, 13, 0, 0, 0, time.UTC)
	memos := []DecisionMemo{
		{ID: "m1", Namespace: "workspace:a", Outcome: DecisionOutcomeSuccess, CreatedAt: now, LastUsedAt: now},
		{ID: "m2", Namespace: "workspace:a", Outcome: DecisionOutcomePartial, CreatedAt: now, LastUsedAt: now},
		{ID: "m3", Namespace: "workspace:a", Outcome: DecisionOutcomeFailure, HumanBlocked: true, CreatedAt: now, LastUsedAt: now},
		{ID: "m4", Namespace: "workspace:a", Outcome: DecisionOutcomeAwaitingHuman, HumanBlocked: true, CreatedAt: now, LastUsedAt: now},
		{ID: "m5", Namespace: "workspace:a", Outcome: DecisionOutcomeCancelled, CreatedAt: now, LastUsedAt: now},
		{ID: "m6", Namespace: "workspace:b", Outcome: DecisionOutcomeSuccess, CreatedAt: now, LastUsedAt: now},
	}
	for _, memo := range memos {
		if _, err := store.UpsertMemo(memo); err != nil {
			t.Fatalf("upsert memo %s: %v", memo.ID, err)
		}
	}
	if _, err := store.UpsertRecipe(DecisionRecipe{ID: "r1", Namespace: "workspace:a", CreatedAt: now, UpdatedAt: now}); err != nil {
		t.Fatalf("upsert recipe: %v", err)
	}
	if _, err := store.UpsertCluster(DecisionCluster{ID: "c1", Namespace: "workspace:a", UpdatedAt: now}); err != nil {
		t.Fatalf("upsert cluster: %v", err)
	}

	stats := store.Stats("workspace:a")
	if stats.MemoCount != 5 || stats.RecipeCount != 1 || stats.ClusterCount != 1 {
		t.Fatalf("unexpected counts: %+v", stats)
	}
	if stats.SuccessCount != 1 || stats.PartialCount != 1 || stats.FailureCount != 1 || stats.AwaitingHumanCount != 1 || stats.CancelledCount != 1 {
		t.Fatalf("unexpected outcome stats: %+v", stats)
	}
	if stats.HumanBlockedCount != 2 {
		t.Fatalf("unexpected human blocked count: %+v", stats)
	}
}

func TestNormalizeDecisionStructures(t *testing.T) {
	localTime := time.Date(2026, 3, 7, 8, 45, 0, 0, time.FixedZone("JST", 9*3600))
	memo := normalizeDecisionMemo(DecisionMemo{
		ID:            " memo-1 ",
		Constraints:   []string{" a ", "", "a"},
		Assumptions:   []string{" b ", "b"},
		GraphNodeRefs: []string{" node-1 ", "node-1"},
		GraphEdgeRefs: []string{" edge-1 ", "edge-1"},
		AnchorKeys:    []string{" anchor-1 ", "anchor-1"},
		NeedsHumanFor: []string{" approval ", "approval"},
		Confidence:    -0.4,
		ReuseScore:    1.4,
		CreatedAt:     localTime,
		LastUsedAt:    localTime,
		Environment:   DecisionEnvFingerprint{GraphNamespace: "", ToolNames: []string{" bash_exec ", "bash_exec"}},
	})
	if memo.Namespace != defaultDecisionNamespace {
		t.Fatalf("expected default namespace, got %q", memo.Namespace)
	}
	if !reflect.DeepEqual(memo.Constraints, []string{"a"}) || !reflect.DeepEqual(memo.Assumptions, []string{"b"}) {
		t.Fatalf("unexpected normalized memo slices: %+v", memo)
	}
	if !reflect.DeepEqual(memo.GraphNodeRefs, []string{"node-1"}) || !reflect.DeepEqual(memo.GraphEdgeRefs, []string{"edge-1"}) {
		t.Fatalf("unexpected graph refs: %+v", memo)
	}
	if !reflect.DeepEqual(memo.Environment.ToolNames, []string{"bash_exec"}) {
		t.Fatalf("unexpected env tools: %#v", memo.Environment.ToolNames)
	}
	if memo.CreatedAt.Location() != time.UTC || memo.LastUsedAt.Location() != time.UTC {
		t.Fatalf("expected UTC memo timestamps")
	}
	if memo.Confidence != 0 || memo.ReuseScore != 1 {
		t.Fatalf("unexpected memo confidence scores: %+v", memo)
	}

	recipe := normalizeDecisionRecipe(DecisionRecipe{
		ID:                  " recipe-1 ",
		TriggerPhrases:      []string{" start ", "start"},
		ValidationChecklist: []string{" test ", "test"},
		SuccessRate:         1.2,
		Confidence:          -1,
		CreatedAt:           localTime,
		UpdatedAt:           localTime,
	})
	if !reflect.DeepEqual(recipe.TriggerPhrases, []string{"start"}) || !reflect.DeepEqual(recipe.ValidationChecklist, []string{"test"}) {
		t.Fatalf("unexpected recipe slices: %+v", recipe)
	}
	if recipe.SuccessRate != 1 || recipe.Confidence != 0 {
		t.Fatalf("unexpected recipe scores: %+v", recipe)
	}
	if recipe.CreatedAt.Location() != time.UTC || recipe.UpdatedAt.Location() != time.UTC {
		t.Fatalf("expected UTC recipe timestamps")
	}

	cluster := normalizeDecisionCluster(DecisionCluster{ID: " cluster-1 ", SuccessCount: -1, FailureCount: 2, UpdatedAt: localTime})
	if cluster.SuccessCount != 0 || cluster.FailureCount != 2 || cluster.UpdatedAt.Location() != time.UTC {
		t.Fatalf("unexpected normalized cluster: %+v", cluster)
	}

	hit := normalizeDecisionHit(DecisionHit{Type: " memo ", Namespace: "", Score: 1.7, Confidence: -1, ReuseScore: 2, Outcome: " success ", GraphRefs: []string{" node-1 ", "node-1"}})
	if hit.Type != DecisionHitTypeMemo || hit.Namespace != defaultDecisionNamespace || hit.Score != 1 || hit.Confidence != 0 || hit.ReuseScore != 1 || hit.Outcome != DecisionOutcomeSuccess {
		t.Fatalf("unexpected normalized hit: %+v", hit)
	}
	if !reflect.DeepEqual(hit.GraphRefs, []string{"node-1"}) {
		t.Fatalf("unexpected hit graph refs: %#v", hit.GraphRefs)
	}
}
