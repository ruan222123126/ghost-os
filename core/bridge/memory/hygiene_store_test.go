package memory

import (
	"path/filepath"
	"testing"
	"time"
)

func TestHygieneStoreUpsertAndLoad(t *testing.T) {
	store := NewHygieneStore(filepath.Join(t.TempDir(), "hygiene"))
	scoredAt := time.Date(2026, time.March, 8, 10, 0, 0, 0, time.UTC)
	record := HygieneRecord{
		Key:             "object:obj-1",
		ObjectID:        "obj-1",
		GarbageVotes:    3,
		GarbageScore:    0.6,
		QuarantineLevel: QuarantineLevelWeak,
		LastScoredAt:    scoredAt,
		Reasons: []HygieneReason{{
			Code:     HygieneReasonToolNoise,
			Label:    "Tool noise",
			Weight:   0.7,
			Evidence: "stream contains repeated tool chatter",
		}},
		LastTraceID: "trace-hygiene-1",
		CreatedAt:   scoredAt,
		UpdatedAt:   scoredAt,
	}
	if err := store.Upsert(record); err != nil {
		t.Fatalf("upsert hygiene record: %v", err)
	}

	reloaded := NewHygieneStore(store.BaseDir())
	if err := reloaded.Load(); err != nil {
		t.Fatalf("load hygiene store: %v", err)
	}
	got, ok := reloaded.Get(HygieneTarget{ObjectID: "obj-1"})
	if !ok {
		t.Fatal("expected reloaded hygiene record")
	}
	if got.Key != "object:obj-1" {
		t.Fatalf("expected object key, got %q", got.Key)
	}
	if got.GarbageVotes != 3 || got.GarbageScore != 0.6 {
		t.Fatalf("unexpected garbage metrics: %#v", got)
	}
	if got.QuarantineLevel != QuarantineLevelWeak {
		t.Fatalf("expected weak quarantine, got %q", got.QuarantineLevel)
	}
	if len(got.Reasons) != 1 || got.Reasons[0].Code != HygieneReasonToolNoise {
		t.Fatalf("expected persisted reasons, got %#v", got.Reasons)
	}
}
