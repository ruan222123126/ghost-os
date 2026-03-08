package memory

import (
	"testing"
	"time"
)

func TestPrimaryCandidateFromTruthMatch(t *testing.T) {
	match := truthQueryMatch{
		Object: MemoryObject{
			ObjectID: "obj-1",
			Claims: []MemoryClaim{
				{ClaimID: "claim-active", Status: truthClaimStatusActive, EvidenceRefs: []EvidenceRef{{EvidenceID: "ev-1"}}},
				{ClaimID: "claim-conflict", Status: truthClaimStatusConflicted, EvidenceRefs: []EvidenceRef{{EvidenceID: "ev-2"}}},
				{ClaimID: "claim-old", Status: truthClaimStatusSuperseded, EvidenceRefs: []EvidenceRef{{EvidenceID: "ev-3"}}},
			},
			RawEvidence: []MemoryEvidence{{EvidenceID: "ev-1"}, {EvidenceID: "ev-4"}},
		},
		Score:      0.88,
		WhyMatched: "predicate overlap",
	}

	candidate := primaryCandidateFromTruthMatch(match)
	if candidate.ObjectID != "obj-1" {
		t.Fatalf("expected object id, got %#v", candidate)
	}
	if got := candidate.MatchedClaimIDs; len(got) != 2 || got[0] != "claim-active" || got[1] != "claim-conflict" {
		t.Fatalf("unexpected matched claim ids: %#v", got)
	}
	if got := candidate.ConflictedClaimIDs; len(got) != 1 || got[0] != "claim-conflict" {
		t.Fatalf("unexpected conflicted claim ids: %#v", got)
	}
	if got := candidate.MatchedEvidenceIDs; len(got) != 3 || got[0] != "ev-1" || got[1] != "ev-4" || got[2] != "ev-2" {
		t.Fatalf("unexpected matched evidence ids: %#v", got)
	}
	if candidate.Explain.WhyMatched != "predicate overlap" {
		t.Fatalf("expected explain why_matched to propagate, got %#v", candidate.Explain)
	}
}

func TestProjectionHitFromEntry(t *testing.T) {
	entry := normalizeEntry(MemoryEntry{
		ID:        "markdown:node-1",
		Summary:   "remembered note",
		Source:    "markdown",
		Timestamp: time.Now().UTC(),
		Explain: map[string]any{
			"source_claim_ids":    []string{"claim-1"},
			"source_evidence_ids": []string{"ev-1"},
		},
		Metadata: map[string]any{
			"layer":               "markdown",
			"node_id":             "node-1",
			"object_id":           "obj-1",
			"source_claim_ids":    []string{"claim-1", "claim-2"},
			"source_evidence_ids": []string{"ev-1", "ev-2"},
			"source_ids":          []string{"legacy-1"},
		},
	})

	hit, ok := projectionHitFromEntry(entry)
	if !ok {
		t.Fatal("expected markdown entry to hydrate into projection hit")
	}
	if hit.ProjectionType != "markdown" || hit.ProjectionID != "node-1" || hit.ObjectID != "obj-1" {
		t.Fatalf("unexpected projection identity: %#v", hit)
	}
	if got := hit.SourceClaimIDs; len(got) != 2 || got[0] != "claim-1" || got[1] != "claim-2" {
		t.Fatalf("unexpected source claim ids: %#v", got)
	}
	if got := hit.SourceEvidenceIDs; len(got) != 2 || got[0] != "ev-1" || got[1] != "ev-2" {
		t.Fatalf("unexpected source evidence ids: %#v", got)
	}
	if got := metadataStrings(hit.Explain, "legacy_source_ids"); len(got) != 1 || got[0] != "legacy-1" {
		t.Fatalf("expected legacy source ids in explain, got %#v", hit.Explain)
	}
}
