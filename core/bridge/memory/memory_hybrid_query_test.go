package memory

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func newHybridTruthManager(t *testing.T, hybrid bool) *MemoryManager {
	t.Helper()
	baseDir := t.TempDir()
	return NewMemoryManager(MemoryConfig{
		Warm:   WarmConfig{Capacity: 24, Path: filepath.Join(baseDir, "warm.json"), AutoRecallEnabled: true, AutoRecallLimit: 4},
		Cold:   ColdConfig{BaseDir: filepath.Join(baseDir, "cold")},
		Truth:  TruthConfig{Enabled: true, DualWrite: true, BaseDir: filepath.Join(baseDir, "truth"), ShadowFailOpen: true, ReadEnabled: true, TopK: 6, MinSupportRefs: 2},
		Recall: RecallConfig{IntentPlannerEnabled: true, HybridRerankEnabled: hybrid, RerankDebugEnabled: true, RecallInjectMinConfidence: 0.7, ConflictPenalty: 0.18},
		Vector: VectorConfig{Enabled: true, Path: filepath.Join(baseDir, "vector"), TopK: 4, MinScore: 0.2},
	})
}

func TestTruthReaderDebugHitsMatchFacets(t *testing.T) {
	manager := newHybridTruthManager(t, false)
	now := time.Date(2026, 3, 7, 18, 0, 0, 0, time.UTC)
	object := MemoryObject{
		ObjectID:   "obj-truth-facet",
		ObjectType: truthObjectTypeSemanticNote,
		Summary:    "Respect deploy freeze for service.api in prod.",
		Claims: []MemoryClaim{
			{Type: truthClaimTypeIntent, IntentKey: "intent.fix_deploy", Value: "deploy safely", Confidence: 0.86, CreatedAt: now},
			{Type: truthClaimTypeEntity, EntityID: "service.api", Value: "service.api", Confidence: 0.86, CreatedAt: now},
			{Type: truthClaimTypeConstraint, ConstraintType: "change_freeze", Value: "change freeze", Confidence: 0.84, CreatedAt: now},
			{Type: truthClaimTypeRisk, RiskType: "prod", Value: "prod", Confidence: 0.82, CreatedAt: now},
		},
		RawEvidence: []MemoryEvidence{{Kind: "markdown.note", Summary: "deploy checklist for service.api in prod", Timestamp: now, Confidence: 0.84}},
		SourceRefs: []SourceRef{
			{SourceKind: truthSourceKindMarkdownNode, SourceID: "node-truth-facet"},
			{SourceKind: truthSourceKindMarkdownSource, SourceID: "session-facet:000000"},
		},
		CreatedAt:  now,
		UpdatedAt:  now,
		Confidence: 0.84,
	}
	if _, err := manager.truth.UpsertObject(object); err != nil {
		t.Fatalf("upsert truth facet object: %v", err)
	}
	if !manager.truth.waitForSidecar(2 * time.Second) {
		t.Fatal("timed out waiting for truth sidecars")
	}

	result, err := manager.QueryResult(MemoryQuery{
		IncludeTruth:  true,
		TruthDebug:    true,
		SemanticQuery: "service.api prod deploy freeze",
		Metadata: map[string]any{
			"intent_key":      "intent.fix_deploy",
			"entity_id":       "service.api",
			"constraint_type": "change_freeze",
			"risk_type":       "prod",
		},
	})
	if err != nil {
		t.Fatalf("truth debug query: %v", err)
	}
	if len(result.TruthHits) == 0 {
		t.Fatal("expected truth hits")
	}
	if result.TruthHits[0].ObjectID != "obj-truth-facet" {
		t.Fatalf("unexpected truth hit: %+v", result.TruthHits[0])
	}
	if result.TruthHits[0].Status != truthStatusVerified {
		t.Fatalf("expected verified truth hit, got %+v", result.TruthHits[0])
	}
}

func TestHybridQueryPromotesVectorHitsThroughTruth(t *testing.T) {
	manager := newHybridTruthManager(t, true)
	seedVectorTruthObjects(t, manager)

	result, err := manager.QueryResult(MemoryQuery{
		SemanticQuery: "fix config migration",
		IncludeVector: true,
		VectorDebug:   true,
		TruthDebug:    true,
		RerankDebug:   true,
	})
	if err != nil {
		t.Fatalf("hybrid query: %v", err)
	}
	if len(result.VectorHits) == 0 {
		t.Fatal("expected vector debug hits")
	}
	if result.RerankReport == nil || len(result.RerankReport.Candidates) == 0 {
		t.Fatalf("expected rerank report, got %+v", result.RerankReport)
	}
	if len(result.Entries) == 0 {
		t.Fatal("expected hybrid entries")
	}
	foundPromoted := false
	for _, entry := range result.Entries {
		if entry.Confidence <= 0 || entry.Freshness <= 0 || entry.EvidenceCount <= 0 || len(entry.SourceRefs) == 0 || entry.TruthStatus == "" || entry.RerankScore <= 0 {
			t.Fatalf("expected calibrated entry contract, got %+v", entry)
		}
		if objectID, _ := entry.Metadata["object_id"].(string); objectID == "obj-vector-memo" {
			foundPromoted = true
			if entry.TruthStatus == truthStatusCandidate {
				t.Fatalf("expected vector promoted entry to be truth-backed, got %+v", entry)
			}
		}
	}
	if !foundPromoted {
		t.Fatalf("expected obj-vector-memo to enter live entries, got %+v", result.Entries)
	}
	metrics := manager.Metrics()
	if metrics.HybridRerankRuns == 0 || metrics.VectorPromotedHits == 0 {
		t.Fatalf("expected hybrid/vector metrics, got %+v", metrics)
	}
}

func TestHybridBuildContextWindowSkipsCandidateEntries(t *testing.T) {
	manager := newHybridTruthManager(t, true)
	now := time.Date(2026, 3, 7, 19, 0, 0, 0, time.UTC)
	if err := manager.warm.Store(MemoryEntry{
		ID:         "warm-candidate-only",
		Content:    "Maybe delete config.toml before migration.",
		Summary:    "Maybe delete config.toml before migration.",
		Timestamp:  now,
		Importance: 0.92,
		Metadata:   map[string]any{"layer": "warm", "session_id": "session-hybrid-context"},
	}); err != nil {
		t.Fatalf("store warm candidate: %v", err)
	}
	verified := MemoryObject{
		ObjectID:    "obj-context-verified",
		ObjectType:  truthObjectTypeProcedureMemo,
		Summary:     "Patch config.toml then run the migration check.",
		Claims:      []MemoryClaim{{Type: truthClaimTypeIntent, IntentKey: "intent.fix_config_migration", Value: "fix config migration", Confidence: 0.92, CreatedAt: now}},
		RawEvidence: []MemoryEvidence{{Kind: "decision.memo", Summary: "Patch config.toml and validate migration success.", Timestamp: now, Confidence: 0.93}},
		SourceRefs: []SourceRef{
			{SourceKind: truthSourceKindDecisionMemo, SourceID: "memo-context-verified"},
			{SourceKind: truthSourceKindDecisionInput, SourceID: "turn-context-verified"},
		},
		CreatedAt:  now,
		UpdatedAt:  now,
		Confidence: 0.93,
	}
	if _, err := manager.truth.UpsertObject(verified); err != nil {
		t.Fatalf("upsert verified object: %v", err)
	}
	if !manager.truth.waitForSidecar(2 * time.Second) {
		t.Fatal("timed out waiting for truth sidecars")
	}

	window, err := manager.BuildContextWindow("session-hybrid-context", "fix config migration")
	if err != nil {
		t.Fatalf("build hybrid context window: %v", err)
	}
	if len(window) != 1 {
		t.Fatalf("unexpected context window size: got %d want 1", len(window))
	}
	if !strings.Contains(window[0].Text, "Patch config.toml") {
		t.Fatalf("expected verified truth-backed recall, got %q", window[0].Text)
	}
	if strings.Contains(window[0].Text, "delete config.toml") {
		t.Fatalf("candidate-only content should not be injected, got %q", window[0].Text)
	}
}
