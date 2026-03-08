package memory

import (
	"path/filepath"
	"testing"
	"time"

	"ghost-os/bridge/llm"
)

func TestLedgerBucketManifestCapturesSessionStatsAndTopTerms(t *testing.T) {
	store := NewLedgerStore(filepath.Join(t.TempDir(), "ledger"), "default", "workspace-a", nil)
	march := time.Date(2026, time.March, 2, 10, 0, 0, 0, time.UTC)
	if _, err := store.AppendTurn("session-a", "turn-a", "trace-a", 0, []llm.Message{{Role: llm.RoleUser, Text: "deploy checklist update"}, {Role: llm.RoleAssistant, Text: "deployment checklist updated"}}, march); err != nil {
		t.Fatalf("append first turn: %v", err)
	}
	if _, err := store.AppendTurn("session-b", "turn-b", "trace-b", 0, []llm.Message{{Role: llm.RoleUser, Text: "database rollback plan"}}, march.Add(2*time.Hour)); err != nil {
		t.Fatalf("append second turn: %v", err)
	}
	manifests, err := store.ListBucketManifests("")
	if err != nil {
		t.Fatalf("list bucket manifests: %v", err)
	}
	if len(manifests) != 1 {
		t.Fatalf("expected exactly one month bucket, got %d", len(manifests))
	}
	manifest := manifests[0]
	if manifest.TraceCount != 2 {
		t.Fatalf("expected trace count 2, got %+v", manifest)
	}
	if len(manifest.Sessions) != 2 {
		t.Fatalf("expected two session manifests, got %+v", manifest.Sessions)
	}
	if !containsString(manifest.TopTerms, "checklist") {
		t.Fatalf("expected top terms to include checklist, got %+v", manifest.TopTerms)
	}
	if !containsString(manifest.Kinds, ledgerEventKindMessageAppended) {
		t.Fatalf("expected kinds to include message_appended, got %+v", manifest.Kinds)
	}
	if manifest.UserTurns == 0 || manifest.AssistantTurns == 0 {
		t.Fatalf("expected coarse role counters, got %+v", manifest)
	}
}

func TestBucketPlannerPrefersSessionAndRespectsBudget(t *testing.T) {
	baseDir := t.TempDir()
	manager := NewMemoryManager(MemoryConfig{
		Cold:   ColdConfig{BaseDir: filepath.Join(baseDir, "cold")},
		Ledger: LedgerConfig{DualWrite: true, WorkspaceID: "workspace-a"},
		Recall: RecallConfig{BucketReadEnabled: true, MaxBuckets: 2, MaxRecentBuckets: 1, MaxHistoricalBuckets: 1, BucketSessionBudget: 1},
	})
	store := manager.cold.ledger
	if store == nil {
		t.Fatal("expected ledger store to be configured")
	}
	if _, err := store.AppendTurn("session-current", "turn-1", "trace-1", 0, []llm.Message{{Role: llm.RoleUser, Text: "recent workspace task"}}, time.Date(2026, time.March, 5, 9, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("append current month: %v", err)
	}
	if _, err := store.AppendTurn("session-old", "turn-2", "trace-2", 0, []llm.Message{{Role: llm.RoleUser, Text: "historical workspace task"}}, time.Date(2026, time.January, 15, 9, 0, 0, 0, time.UTC)); err != nil {
		t.Fatalf("append historical month: %v", err)
	}
	plan := manager.query.buckets.Plan(MemoryQuery{
		WorkspaceID:  "workspace-a",
		SessionHints: []string{"session-current"},
		SemanticQuery: "recent workspace task",
		MaxBuckets:   2,
	}, SessionScope{SessionID: "session-current"}, &QueryIntentPlan{Terms: []string{"recent", "workspace", "task"}})
	if plan == nil {
		t.Fatal("expected bucket plan")
	}
	if len(plan.SelectedBuckets) == 0 {
		t.Fatalf("expected selected buckets, got %+v", plan)
	}
	if plan.SelectedBuckets[0].Bucket.Key.Month != "2026-03" {
		t.Fatalf("expected current session bucket first, got %+v", plan.SelectedBuckets)
	}
	if len(plan.SelectedBuckets) > 2 {
		t.Fatalf("expected planner to honor max bucket budget, got %+v", plan.SelectedBuckets)
	}
}

func TestHybridQueryBucketShadowProducesPlanAndReport(t *testing.T) {
	baseDir := t.TempDir()
	manager := NewMemoryManager(MemoryConfig{
		Warm:   WarmConfig{Capacity: 8, Path: filepath.Join(baseDir, "warm.json")},
		Cold:   ColdConfig{BaseDir: filepath.Join(baseDir, "cold")},
		Ledger: LedgerConfig{DualWrite: true, WorkspaceID: "workspace-shadow"},
		Truth:  TruthConfig{Enabled: true, DualWrite: true, BaseDir: filepath.Join(baseDir, "truth"), ShadowFailOpen: true, ReadEnabled: true, TopK: 4, MinSupportRefs: 2},
		Recall: RecallConfig{IntentPlannerEnabled: true, HybridRerankEnabled: true, BucketReadEnabled: true, BucketShadowCompare: true, MaxBuckets: 2},
		Vector: VectorConfig{Enabled: true, Path: filepath.Join(baseDir, "vector"), TopK: 4, MinScore: 0.1},
	})
	store := manager.cold.ledger
	turnAt := time.Date(2026, time.March, 8, 10, 0, 0, 0, time.UTC)
	if _, err := store.AppendTurn("session-shadow", "turn-shadow", "trace-shadow", 0, []llm.Message{{Role: llm.RoleUser, Text: "shadow bucket recall"}, {Role: llm.RoleAssistant, Text: "shadow bucket response"}}, turnAt); err != nil {
		t.Fatalf("append ledger turn: %v", err)
	}
	result, err := manager.QueryResultWithScope(MemoryQuery{
		SemanticQuery:   "shadow bucket recall",
		Keywords:        []string{"shadow", "bucket"},
		IncludeMarkdown: true,
		IncludeVector:   true,
		BucketDebug:     true,
	}, SessionScope{SessionID: "session-shadow"})
	if err != nil {
		t.Fatalf("query result with scope: %v", err)
	}
	if result.BucketPlan == nil || len(result.BucketPlan.SelectedBuckets) == 0 {
		t.Fatalf("expected bucket plan in result, got %+v", result.BucketPlan)
	}
	if result.ShadowRead == nil {
		t.Fatalf("expected bucket shadow debug report, got %+v", result)
	}
}
