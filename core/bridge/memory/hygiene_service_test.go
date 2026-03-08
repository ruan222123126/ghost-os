package memory

import (
	"math"
	"os"
	"path/filepath"
	"testing"
)

func TestHygieneServiceAccumulatesVotes(t *testing.T) {
	service := newHygieneTestService(t)
	if err := service.UpsertAssessment(HygieneAssessmentInput{
		Target:    HygieneTarget{ObjectID: "obj-accumulate"},
		VoteDelta: 1,
		Reasons:   []HygieneReason{{Code: HygieneReasonDuplicateSummary, Evidence: "same summary as newer note"}},
		TraceID:   "trace-1",
	}); err != nil {
		t.Fatalf("first hygiene upsert: %v", err)
	}
	if err := service.UpsertAssessment(HygieneAssessmentInput{
		Target:    HygieneTarget{ObjectID: "obj-accumulate"},
		VoteDelta: 2,
		Reasons:   []HygieneReason{{Code: HygieneReasonToolNoise, Evidence: "tool echo"}},
		TraceID:   "trace-2",
	}); err != nil {
		t.Fatalf("second hygiene upsert: %v", err)
	}
	got, ok := service.Get(HygieneTarget{ObjectID: "obj-accumulate"})
	if !ok {
		t.Fatal("expected hygiene record")
	}
	if got.GarbageVotes != 3 {
		t.Fatalf("expected 3 votes, got %d", got.GarbageVotes)
	}
	if math.Abs(got.GarbageScore-0.6) > 1e-9 {
		t.Fatalf("expected garbage score 0.6, got %.2f", got.GarbageScore)
	}
	if got.LastTraceID != "trace-2" {
		t.Fatalf("expected last trace id to update, got %q", got.LastTraceID)
	}
}

func TestHygieneServicePromotesQuarantineLevel(t *testing.T) {
	service := newHygieneTestService(t)
	steps := []struct {
		delta int
		want  string
	}{
		{delta: 1, want: QuarantineLevelNone},
		{delta: 1, want: QuarantineLevelWeak},
		{delta: 2, want: QuarantineLevelStrong},
		{delta: 2, want: QuarantineLevelHard},
	}
	for _, step := range steps {
		if err := service.UpsertAssessment(HygieneAssessmentInput{
			Target:    HygieneTarget{ObjectID: "obj-promote"},
			VoteDelta: step.delta,
			Reasons:   []HygieneReason{{Code: HygieneReasonStalePlan}},
		}); err != nil {
			t.Fatalf("upsert hygiene assessment: %v", err)
		}
		got, ok := service.Get(HygieneTarget{ObjectID: "obj-promote"})
		if !ok {
			t.Fatal("expected hygiene record")
		}
		if got.QuarantineLevel != step.want {
			t.Fatalf("expected quarantine %q, got %q after delta %d", step.want, got.QuarantineLevel, step.delta)
		}
	}
}

func TestHygieneServiceUsesObjectIDAsPrimaryKey(t *testing.T) {
	service := newHygieneTestService(t)
	if err := service.UpsertAssessment(HygieneAssessmentInput{
		Target:    HygieneTarget{ObjectID: "shared-id", EntryID: "shared-id"},
		VoteDelta: 1,
		Reasons:   []HygieneReason{{Code: HygieneReasonDuplicateSummary}},
	}); err != nil {
		t.Fatalf("upsert object-target hygiene assessment: %v", err)
	}
	if err := service.UpsertAssessment(HygieneAssessmentInput{
		Target:    HygieneTarget{EntryID: "shared-id"},
		VoteDelta: 1,
		Reasons:   []HygieneReason{{Code: HygieneReasonToolNoise}},
	}); err != nil {
		t.Fatalf("upsert entry-target hygiene assessment: %v", err)
	}
	results := service.BatchGet([]HygieneTarget{{ObjectID: "shared-id"}, {EntryID: "shared-id"}})
	if len(results) != 2 {
		t.Fatalf("expected distinct object and entry records, got %#v", results)
	}
	if record, ok := results["object:shared-id"]; !ok || record.Key != "object:shared-id" {
		t.Fatalf("expected object key to be primary, got %#v", results)
	}
}

func TestHygieneServiceFallsBackToEntryID(t *testing.T) {
	service := newHygieneTestService(t)
	if err := service.UpsertAssessment(HygieneAssessmentInput{
		Target:    HygieneTarget{EntryID: "entry-42"},
		VoteDelta: 2,
		Reasons:   []HygieneReason{{Code: HygieneReasonTransientState}},
	}); err != nil {
		t.Fatalf("upsert entry hygiene assessment: %v", err)
	}
	got, ok := service.Get(HygieneTarget{EntryID: "entry-42"})
	if !ok {
		t.Fatal("expected entry-backed hygiene record")
	}
	if got.Key != "entry:entry-42" {
		t.Fatalf("expected entry key, got %q", got.Key)
	}
	if got.ObjectID != "" {
		t.Fatalf("expected empty object id for entry-backed record, got %q", got.ObjectID)
	}
}

func TestHygieneReasonsDedupedOrMerged(t *testing.T) {
	service := newHygieneTestService(t)
	if err := service.UpsertAssessment(HygieneAssessmentInput{
		Target:    HygieneTarget{ObjectID: "obj-reasons"},
		VoteDelta: 1,
		Reasons: []HygieneReason{
			{Code: HygieneReasonLowSignalSummary, Label: "Low signal", Weight: 0.2, Evidence: "too generic"},
			{Code: HygieneReasonToolNoise, Label: "Tool noise", Weight: 0.4, Evidence: "noisy trace"},
		},
	}); err != nil {
		t.Fatalf("first reason upsert: %v", err)
	}
	if err := service.UpsertAssessment(HygieneAssessmentInput{
		Target:    HygieneTarget{ObjectID: "obj-reasons"},
		VoteDelta: 1,
		Reasons: []HygieneReason{
			{Code: HygieneReasonLowSignalSummary, Label: "Low signal summary", Weight: 0.9, Evidence: "superseded by better note"},
		},
	}); err != nil {
		t.Fatalf("second reason upsert: %v", err)
	}
	got, ok := service.Get(HygieneTarget{ObjectID: "obj-reasons"})
	if !ok {
		t.Fatal("expected hygiene record")
	}
	if len(got.Reasons) != 2 {
		t.Fatalf("expected 2 merged reasons, got %#v", got.Reasons)
	}
	if got.Reasons[0].Code != HygieneReasonLowSignalSummary || got.Reasons[0].Weight != 0.9 || got.Reasons[0].Evidence != "superseded by better note" {
		t.Fatalf("expected low-signal reason to merge, got %#v", got.Reasons[0])
	}
}

func TestHygieneSidecarDoesNotRequireTruthLedger(t *testing.T) {
	baseDir := t.TempDir()
	manager := NewMemoryManager(MemoryConfig{
		Warm:    WarmConfig{Capacity: 8, Path: filepath.Join(baseDir, "warm.json")},
		Cold:    ColdConfig{BaseDir: filepath.Join(baseDir, "cold")},
		Hygiene: HygieneConfig{Enabled: true},
	})
	if err := manager.ScoreGarbage(HygieneAssessmentInput{
		Target:    HygieneTarget{ObjectID: "obj-manager"},
		VoteDelta: 2,
		Reasons:   []HygieneReason{{Code: HygieneReasonRedundantDecisionMemo}},
		TraceID:   "trace-manager",
	}); err != nil {
		t.Fatalf("score garbage via manager: %v", err)
	}
	got, ok := manager.GetHygiene(HygieneTarget{ObjectID: "obj-manager"})
	if !ok {
		t.Fatal("expected manager hygiene record")
	}
	if got.QuarantineLevel != QuarantineLevelWeak {
		t.Fatalf("expected weak quarantine, got %q", got.QuarantineLevel)
	}
	if _, err := os.Stat(filepath.Join(baseDir, "cold", "ledger")); !os.IsNotExist(err) {
		t.Fatalf("expected ledger to stay untouched, err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(baseDir, "cold-truth-shadow")); !os.IsNotExist(err) {
		t.Fatalf("expected truth shadow to stay untouched, err=%v", err)
	}
}

func newHygieneTestService(t *testing.T) *HygieneService {
	t.Helper()
	return NewHygieneService(HygieneConfig{
		Enabled: true,
		Path:    filepath.Join(t.TempDir(), "hygiene"),
	})
}
