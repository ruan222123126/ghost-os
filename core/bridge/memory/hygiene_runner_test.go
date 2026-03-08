package memory

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRunHygieneDryRunDoesNotPersist(t *testing.T) {
	manager := newHygieneRunTestManager(t)
	seedWarmHygieneCandidate(t, manager, "warm-dry-run")

	result, err := manager.RunHygiene(HygieneRunOptions{
		Scope:          HygieneScopeWarm,
		Limit:          10,
		DryRun:         true,
		MinConfidence:  0.5,
		MaxVotesPerRun: 1,
		TraceID:        "trace-hygiene-dry-run",
	})
	if err != nil {
		t.Fatalf("run hygiene dry-run: %v", err)
	}
	if result.Scanned != 1 || result.Scored != 1 {
		t.Fatalf("unexpected dry-run result: %+v", result)
	}
	if _, ok := manager.GetHygiene(HygieneTarget{EntryID: "warm-dry-run"}); ok {
		t.Fatal("expected dry-run to avoid persisting hygiene record")
	}
}

func TestRunHygieneWritesRecordAndScoreLog(t *testing.T) {
	manager := newHygieneRunTestManager(t)
	seedWarmHygieneCandidate(t, manager, "warm-write")

	result, err := manager.RunHygiene(HygieneRunOptions{
		Scope:          HygieneScopeWarm,
		Limit:          10,
		DryRun:         false,
		MinConfidence:  0.5,
		MaxVotesPerRun: 1,
		TraceID:        "trace-hygiene-write",
		TaskID:         "task-hygiene-write",
	})
	if err != nil {
		t.Fatalf("run hygiene write: %v", err)
	}
	if result.Scored != 1 {
		t.Fatalf("expected one scored candidate, got %+v", result)
	}
	record, ok := manager.GetHygiene(HygieneTarget{EntryID: "warm-write"})
	if !ok {
		t.Fatal("expected persisted hygiene record")
	}
	if record.GarbageVotes != 1 {
		t.Fatalf("expected one garbage vote, got %+v", record)
	}
	entries, err := os.ReadDir(manager.hygiene.store.scoresDir)
	if err != nil {
		t.Fatalf("read hygiene scores dir: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("expected hygiene score log file to be written")
	}
}

func TestRunHygieneScopes(t *testing.T) {
	t.Run("warm", func(t *testing.T) {
		manager := newHygieneRunTestManager(t)
		seedWarmHygieneCandidate(t, manager, "warm-scope")
		result, err := manager.RunHygiene(HygieneRunOptions{Scope: HygieneScopeWarm, Limit: 10, DryRun: true, MinConfidence: 0.5, MaxVotesPerRun: 1})
		if err != nil {
			t.Fatalf("run warm hygiene: %v", err)
		}
		if result.Scanned != 1 {
			t.Fatalf("expected one warm candidate, got %+v", result)
		}
	})

	t.Run("projection", func(t *testing.T) {
		manager := newHygieneRunTestManager(t)
		seedProjectionHygieneCandidate(t, manager, "projection-scope")
		result, err := manager.RunHygiene(HygieneRunOptions{Scope: HygieneScopeProjection, Limit: 10, DryRun: true, MinConfidence: 0.5, MaxVotesPerRun: 1})
		if err != nil {
			t.Fatalf("run projection hygiene: %v", err)
		}
		if result.Scanned != 1 {
			t.Fatalf("expected one projection candidate, got %+v", result)
		}
	})

	t.Run("report", func(t *testing.T) {
		manager := newHygieneRunTestManager(t)
		seedWarmHygieneCandidate(t, manager, "warm-report")
		seedProjectionHygieneCandidate(t, manager, "projection-report")
		result, err := manager.RunHygiene(HygieneRunOptions{Scope: HygieneScopeReport, Limit: 2, DryRun: false, MinConfidence: 0.5, MaxVotesPerRun: 1})
		if err != nil {
			t.Fatalf("run report hygiene: %v", err)
		}
		if !result.DryRun {
			t.Fatalf("expected report scope to stay dry-run, got %+v", result)
		}
		if _, ok := manager.GetHygiene(HygieneTarget{EntryID: "warm-report"}); ok {
			t.Fatal("expected report scope to avoid persisting hygiene record")
		}
	})
}

func newHygieneRunTestManager(t *testing.T) *MemoryManager {
	t.Helper()
	baseDir := t.TempDir()
	return NewMemoryManager(MemoryConfig{
		Warm:     WarmConfig{Capacity: 16, Path: filepath.Join(baseDir, "warm.json")},
		Cold:     ColdConfig{BaseDir: filepath.Join(baseDir, "cold")},
		Decision: DecisionConfig{Enabled: true, Path: filepath.Join(baseDir, "decision")},
		Hygiene:  HygieneConfig{Enabled: true, Path: filepath.Join(baseDir, "hygiene")},
	})
}

func seedWarmHygieneCandidate(t *testing.T, manager *MemoryManager, id string) {
	t.Helper()
	if err := manager.warm.Store(MemoryEntry{
		ID:        id,
		Content:   "stderr: tool trace output for a temporary maintenance step",
		Summary:   "tool trace stderr output",
		Type:      MemoryTypeMessage,
		Timestamp: time.Now().UTC().Add(-72 * time.Hour),
		Metadata:  map[string]any{"role": "tool", "source": "conversation"},
		Source:    "conversation",
	}); err != nil {
		t.Fatalf("store warm hygiene candidate: %v", err)
	}
}

func seedProjectionHygieneCandidate(t *testing.T, manager *MemoryManager, id string) {
	t.Helper()
	createdAt := time.Now().UTC().Add(-7 * 24 * time.Hour)
	if err := manager.SaveMarkdownNode(MarkdownNode{
		ID:         id,
		CreatedAt:  createdAt,
		LastSeenAt: createdAt,
		Summary:    "Plan next step later",
		Content:    "TODO: follow up later with the old migration plan.",
		Confidence: 0.2,
	}); err != nil {
		t.Fatalf("save projection hygiene candidate: %v", err)
	}
}
