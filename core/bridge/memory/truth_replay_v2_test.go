package memory

import (
	"path/filepath"
	"testing"
	"time"
)

func TestTruthReplayV2RebuildsClaimStatus(t *testing.T) {
	writer := newTruthTestWriter(t, filepath.Join(t.TempDir(), "truth-v2"))
	first := normalizeMemoryObject(MemoryObject{
		ObjectID:   "obj-v2-a",
		ObjectType: truthObjectTypeSemanticNote,
		Summary:    "first preference",
		SourceRefs: []SourceRef{{SessionID: "session-v2", SourceKind: truthSourceKindMarkdownNode, SourceID: "node-a"}},
		Claims: []MemoryClaim{{
			Subject:    ClaimTerm{Kind: "session", ID: "session-v2"},
			Predicate:  "preference.language",
			Value:      "Go",
			Confidence: 0.82,
			AssertedAt: time.Date(2026, 3, 8, 10, 0, 0, 0, time.UTC),
			CreatedAt:  time.Date(2026, 3, 8, 10, 0, 0, 0, time.UTC),
			SourceRefs: []SourceRef{{SessionID: "session-v2", SourceKind: truthSourceKindMarkdownNode, SourceID: "node-a"}},
		}},
		RawEvidence: []MemoryEvidence{{
			Kind:       "markdown.note",
			Text:       "prefers Go",
			Timestamp:  time.Date(2026, 3, 8, 10, 0, 0, 0, time.UTC),
			Confidence: 0.82,
			SourceRefs: []SourceRef{{SessionID: "session-v2", SourceKind: truthSourceKindMarkdownNode, SourceID: "node-a"}},
		}},
		CreatedAt:  time.Date(2026, 3, 8, 10, 0, 0, 0, time.UTC),
		UpdatedAt:  time.Date(2026, 3, 8, 10, 0, 0, 0, time.UTC),
		Confidence: 0.82,
	})
	second := normalizeMemoryObject(MemoryObject{
		ObjectID:   "obj-v2-b",
		ObjectType: truthObjectTypeProcedureMemo,
		Summary:    "newer preference",
		SourceRefs: []SourceRef{{SessionID: "session-v2", SourceKind: truthSourceKindDecisionMemo, SourceID: "memo-b"}, {SessionID: "session-v2", SourceKind: truthSourceKindDecisionInput, SourceID: "turn-b"}},
		Claims: []MemoryClaim{{
			Subject:    ClaimTerm{Kind: "session", ID: "session-v2"},
			Predicate:  "preference.language",
			Value:      "Rust",
			Confidence: 0.93,
			AssertedAt: time.Date(2026, 3, 10, 10, 0, 0, 0, time.UTC),
			CreatedAt:  time.Date(2026, 3, 10, 10, 0, 0, 0, time.UTC),
			SourceRefs: []SourceRef{{SessionID: "session-v2", SourceKind: truthSourceKindDecisionMemo, SourceID: "memo-b"}, {SessionID: "session-v2", SourceKind: truthSourceKindDecisionInput, SourceID: "turn-b"}},
		}},
		RawEvidence: []MemoryEvidence{{
			Kind:       "decision.memo",
			Text:       "prefers Rust now",
			Timestamp:  time.Date(2026, 3, 10, 10, 0, 0, 0, time.UTC),
			Confidence: 0.93,
			SourceRefs: []SourceRef{{SessionID: "session-v2", SourceKind: truthSourceKindDecisionMemo, SourceID: "memo-b"}, {SessionID: "session-v2", SourceKind: truthSourceKindDecisionInput, SourceID: "turn-b"}},
		}},
		CreatedAt:  time.Date(2026, 3, 10, 10, 0, 0, 0, time.UTC),
		UpdatedAt:  time.Date(2026, 3, 10, 10, 0, 0, 0, time.UTC),
		Confidence: 0.93,
	})
	if err := writeTruthObjectShadow(writer, truthEventTypeMarkdownNode, first, "trace-a"); err != nil {
		t.Fatalf("write first object: %v", err)
	}
	if err := writeTruthObjectShadow(writer, truthEventTypeDecisionMemo, second, "trace-b"); err != nil {
		t.Fatalf("write second object: %v", err)
	}
	verify, err := NewTruthVerifier(writer).Verify()
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if !verify.Match {
		t.Fatalf("expected replay to match live snapshots, got %+v", verify)
	}
	claims, err := readTruthClaimSnapshot(writer.claimsPath)
	if err != nil {
		t.Fatalf("read claims: %v", err)
	}
	if len(claims) != 2 {
		t.Fatalf("expected two claims in claim center, got %+v", claims)
	}
	statusByValue := map[string]string{}
	for _, claim := range claims {
		statusByValue[claim.Value] = claim.Status
	}
	if statusByValue["Go"] != truthClaimStatusSuperseded {
		t.Fatalf("expected old preference superseded, got %+v", statusByValue)
	}
	if statusByValue["Rust"] != truthClaimStatusActive {
		t.Fatalf("expected new preference active, got %+v", statusByValue)
	}
}
