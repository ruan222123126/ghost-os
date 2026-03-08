package memory

import (
	"testing"
	"time"
)

func TestTruthArbitrationMarksConflictsForCloseSingleValueClaims(t *testing.T) {
	snapshot := map[string]MemoryClaim{}
	base := time.Date(2026, 3, 8, 10, 0, 0, 0, time.UTC)
	first := normalizeMemoryClaim(MemoryClaim{
		ObjectID:   "obj-a",
		Subject:    ClaimTerm{Kind: "session", ID: "session-1"},
		Predicate:  "preference.language",
		Value:      "Go",
		Confidence: 0.82,
		AssertedAt: base,
		CreatedAt:  base,
		SourceRefs: []SourceRef{{SessionID: "session-1", SourceKind: "markdown_node", SourceID: "node-a"}},
	}, "obj-a")
	result := truthArbitrateClaims(snapshot, []MemoryClaim{first})
	if len(result.UpsertedClaims) != 1 || snapshot[first.ClaimID].Status != truthInitialClaimStatus(first) {
		t.Fatalf("expected initial insert, got %+v snapshot=%+v", result, snapshot)
	}
	second := normalizeMemoryClaim(MemoryClaim{
		ObjectID:   "obj-b",
		Subject:    ClaimTerm{Kind: "session", ID: "session-1"},
		Predicate:  "preference.language",
		Value:      "Rust",
		Confidence: 0.80,
		AssertedAt: base.Add(2 * time.Hour),
		CreatedAt:  base.Add(2 * time.Hour),
		SourceRefs: []SourceRef{{SessionID: "session-1", SourceKind: "decision_memo", SourceID: "memo-b"}},
	}, "obj-b")
	result = truthArbitrateClaims(snapshot, []MemoryClaim{second})
	if snapshot[first.ClaimID].Status != truthClaimStatusConflicted || snapshot[second.ClaimID].Status != truthClaimStatusConflicted {
		t.Fatalf("expected both claims conflicted, got first=%+v second=%+v result=%+v", snapshot[first.ClaimID], snapshot[second.ClaimID], result)
	}
}

func TestTruthArbitrationSupersedesOlderStrongerDomainClaim(t *testing.T) {
	snapshot := map[string]MemoryClaim{}
	base := time.Date(2026, 3, 8, 10, 0, 0, 0, time.UTC)
	older := normalizeMemoryClaim(MemoryClaim{
		ObjectID:   "obj-a",
		Subject:    ClaimTerm{Kind: "session", ID: "session-1"},
		Predicate:  "preference.language",
		Value:      "Go",
		Confidence: 0.70,
		AssertedAt: base,
		CreatedAt:  base,
		SourceRefs: []SourceRef{{SessionID: "session-1", SourceKind: "markdown_node", SourceID: "node-a"}, {SessionID: "session-1", SourceKind: "markdown_source", SourceID: "src-a"}},
	}, "obj-a")
	truthArbitrateClaims(snapshot, []MemoryClaim{older})
	newer := normalizeMemoryClaim(MemoryClaim{
		ObjectID:   "obj-b",
		Subject:    ClaimTerm{Kind: "session", ID: "session-1"},
		Predicate:  "preference.language",
		Value:      "Rust",
		Confidence: 0.91,
		AssertedAt: base.Add(48 * time.Hour),
		CreatedAt:  base.Add(48 * time.Hour),
		SourceRefs: []SourceRef{{SessionID: "session-1", SourceKind: "decision_memo", SourceID: "memo-b"}, {SessionID: "session-1", SourceKind: "decision_input", SourceID: "turn-b"}},
	}, "obj-b")
	truthArbitrateClaims(snapshot, []MemoryClaim{newer})
	if snapshot[older.ClaimID].Status != truthClaimStatusSuperseded {
		t.Fatalf("expected older claim superseded, got %+v", snapshot[older.ClaimID])
	}
	if snapshot[newer.ClaimID].Status != truthClaimStatusActive {
		t.Fatalf("expected newer claim active, got %+v", snapshot[newer.ClaimID])
	}
}

func TestTruthArbitrationKeepsNonOverlappingWindows(t *testing.T) {
	snapshot := map[string]MemoryClaim{}
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	first := normalizeMemoryClaim(MemoryClaim{
		ObjectID:   "obj-a",
		Subject:    ClaimTerm{Kind: "session", ID: "session-1"},
		Predicate:  "owner_of",
		Value:      "alice",
		Confidence: 0.95,
		ValidFrom:  base,
		ValidTo:    base.Add(24 * time.Hour),
		AssertedAt: base,
		CreatedAt:  base,
		SourceRefs: []SourceRef{{SessionID: "session-1", SourceKind: "decision_memo", SourceID: "memo-a"}, {SessionID: "session-1", SourceKind: "decision_input", SourceID: "turn-a"}},
	}, "obj-a")
	truthArbitrateClaims(snapshot, []MemoryClaim{first})
	second := normalizeMemoryClaim(MemoryClaim{
		ObjectID:   "obj-b",
		Subject:    ClaimTerm{Kind: "session", ID: "session-1"},
		Predicate:  "owner_of",
		Value:      "bob",
		Confidence: 0.96,
		ValidFrom:  base.Add(48 * time.Hour),
		ValidTo:    base.Add(72 * time.Hour),
		AssertedAt: base.Add(48 * time.Hour),
		CreatedAt:  base.Add(48 * time.Hour),
		SourceRefs: []SourceRef{{SessionID: "session-1", SourceKind: "decision_memo", SourceID: "memo-b"}, {SessionID: "session-1", SourceKind: "decision_input", SourceID: "turn-b"}},
	}, "obj-b")
	truthArbitrateClaims(snapshot, []MemoryClaim{second})
	if snapshot[first.ClaimID].Status != truthClaimStatusSuperseded {
		t.Fatalf("expected first window superseded by later active claim, got %+v", snapshot[first.ClaimID])
	}
	if snapshot[second.ClaimID].Status != truthClaimStatusActive {
		t.Fatalf("expected later window active, got %+v", snapshot[second.ClaimID])
	}
}

func TestTruthArbitrationLeavesSingleEvidenceClaimUnverified(t *testing.T) {
	snapshot := map[string]MemoryClaim{}
	claim := normalizeMemoryClaim(MemoryClaim{
		ObjectID:   "obj-a",
		Subject:    ClaimTerm{Kind: "session", ID: "session-1"},
		Predicate:  "preference.language",
		Value:      "Go",
		Confidence: 0.5,
		SourceRefs: []SourceRef{{SessionID: "session-1", SourceKind: "markdown_node", SourceID: "node-a"}},
	}, "obj-a")
	truthArbitrateClaims(snapshot, []MemoryClaim{claim})
	if snapshot[claim.ClaimID].Status != truthClaimStatusUnverified {
		t.Fatalf("expected single-evidence claim unverified, got %+v", snapshot[claim.ClaimID])
	}
}
