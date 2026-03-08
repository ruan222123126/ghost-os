package memory

import "testing"

func TestTruthClaimIDUsesV2ShapeDeterministically(t *testing.T) {
	claim := normalizeMemoryClaim(MemoryClaim{
		ObjectID:   "obj-1",
		Subject:    ClaimTerm{Kind: "session", ID: "session-1"},
		Predicate:  "preference.language",
		Value:      "Go",
		Datatype:   "language",
		Confidence: 0.9,
	}, "obj-1")
	again := normalizeMemoryClaim(claim, "obj-1")
	if claim.ClaimID == "" || again.ClaimID == "" {
		t.Fatal("expected claim id to be generated")
	}
	if claim.ClaimID != again.ClaimID {
		t.Fatalf("expected deterministic v2 claim ids, got %q and %q", claim.ClaimID, again.ClaimID)
	}
}
