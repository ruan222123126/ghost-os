package memory

import "testing"

func TestRecallCandidateKeyPrefersObjectAndProjectionRules(t *testing.T) {
	primary := normalizeRecallCandidate(RecallCandidate{
		ObjectID: "obj-primary",
		Entry: MemoryEntry{
			ID:     "truth:obj-primary",
			Source: "truth",
			Metadata: map[string]any{
				"memo_id":    "memo-legacy",
				"node_id":    "node-legacy",
				"source_ids": []string{"legacy-1", "legacy-2"},
			},
		},
	})
	if got := recallCandidateKey(primary); got != "object:obj-primary" {
		t.Fatalf("expected object key, got %q", got)
	}

	projection := normalizeRecallCandidate(RecallCandidate{
		ObjectID: "obj-primary",
		Entry: MemoryEntry{
			ID:     "markdown:node-1",
			Source: "markdown",
			Metadata: map[string]any{
				"layer":      "markdown",
				"node_id":    "node-1",
				"source_ids": []string{"legacy-1"},
			},
		},
	})
	if got := recallCandidateKey(projection); got != "projection:markdown:node-1" {
		t.Fatalf("expected projection key, got %q", got)
	}

	legacy := normalizeRecallCandidate(RecallCandidate{
		Entry: MemoryEntry{
			ID:      "warm-1",
			Summary: "legacy only",
			Metadata: map[string]any{
				"session_id": "session-1",
				"source_ids": []string{"legacy-1", "legacy-2"},
			},
		},
	})
	if got := recallCandidateKey(legacy); got != "session:session-1:warm-1" {
		t.Fatalf("expected session fallback key, got %q", got)
	}
}

func TestEntrySourceRefsForHydrationKeepsLegacyFallbackOnlyForArchive(t *testing.T) {
	service := &QueryService{}
	entry := normalizeEntry(MemoryEntry{
		ID:     "markdown:node-1",
		Source: "markdown",
		Metadata: map[string]any{
			"session_id": "session-1",
			"memo_id":    "memo-1",
			"node_id":    "node-1",
			"source_ids": []string{"legacy-1"},
		},
		SourceRefs: []SourceRef{{SessionID: "session-1", SourceKind: truthSourceKindMarkdownNode, SourceID: "node-explicit"}},
	})
	refs := service.entrySourceRefsForHydration(entry)
	if len(refs) != 1 || refs[0].SourceID != "node-explicit" {
		t.Fatalf("expected only explicit source refs, got %#v", refs)
	}

	archive := normalizeEntry(MemoryEntry{
		ID:       "archive-msg-1",
		Source:   "archive",
		Metadata: map[string]any{"session_id": "session-1", "source_ids": []string{"legacy-1"}},
	})
	refs = service.entrySourceRefsForHydration(archive)
	if len(refs) != 1 || refs[0].SourceKind != truthSourceKindArchiveMessage || refs[0].SourceID != "archive-msg-1" {
		t.Fatalf("expected archive fallback ref, got %#v", refs)
	}
}

func TestObjectIDFromEntryPrefersClaimAndEvidenceLineageBeforeFallback(t *testing.T) {
	reader := newTruthReaderForPrimaryQueryTests()
	service := &QueryService{truth: reader}

	claimEntry := normalizeEntry(MemoryEntry{
		ID:     "decision:memo-1",
		Source: "decision",
		Metadata: map[string]any{
			"source_claim_ids": []string{"claim-rust-active"},
			"memo_id":          "memo-legacy",
		},
	})
	if got := service.objectIDFromEntry(claimEntry); got != "obj-alice-language" {
		t.Fatalf("expected claim lineage to resolve object, got %q", got)
	}

	evidenceEntry := normalizeEntry(MemoryEntry{
		ID:     "graph:edge-1",
		Source: "graph",
		Metadata: map[string]any{
			"matched_evidence_ids": []string{"ev-rust"},
			"node_id":              "node-legacy",
		},
	})
	if got := service.objectIDFromEntry(evidenceEntry); got != "obj-alice-language" {
		t.Fatalf("expected evidence lineage to resolve object, got %q", got)
	}
}
