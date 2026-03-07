package memory

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestVectorSidecarRebuildAndQueryReturnsTruthObjects(t *testing.T) {
	manager := newVectorTestManager(t, "", false)
	seedVectorTruthObjects(t, manager)
	if err := manager.vector.RebuildFromTruthSnapshot(); err != nil {
		t.Fatalf("rebuild vector sidecar: %v", err)
	}
	hits, err := manager.vector.Query(MemoryQuery{
		SemanticQuery: "fix config migration",
		IncludeVector: true,
		VectorDebug:   true,
	}, nil)
	if err != nil {
		t.Fatalf("vector query: %v", err)
	}
	if len(hits) < 2 {
		t.Fatalf("expected semantic note and procedure memo hits, got %#v", hits)
	}
	if !vectorHitsContainType(hits, truthObjectTypeSemanticNote) || !vectorHitsContainType(hits, truthObjectTypeProcedureMemo) {
		t.Fatalf("expected semantic.note and procedure.memo hits, got %#v", hits)
	}
	if manager.vector.DocumentCount() < 2 {
		t.Fatalf("expected vector docs to be indexed, got %d", manager.vector.DocumentCount())
	}
}

func TestVectorSidecarIncrementalSyncAndFilters(t *testing.T) {
	manager := newVectorTestManager(t, "", false)
	if _, err := manager.truth.UpsertObject(buildVectorTestProcedureMemo()); err != nil {
		t.Fatalf("upsert procedure memo: %v", err)
	}
	if _, err := manager.truth.UpsertObject(buildVectorTestSemanticNote()); err != nil {
		t.Fatalf("upsert semantic note: %v", err)
	}
	if !manager.truth.waitForSidecar(2 * time.Second) {
		t.Fatal("timed out waiting for vector sidecar incremental sync")
	}
	if manager.vector.DocumentCount() != 2 {
		t.Fatalf("expected incremental sync to index 2 docs, got %d", manager.vector.DocumentCount())
	}
	hits := manager.vector.Search("config migration", VectorQueryOptions{
		TopK:               4,
		MinScore:           0.2,
		MinConfidence:      0.9,
		AllowedObjectTypes: []string{truthObjectTypeProcedureMemo},
	})
	if len(hits) != 1 {
		t.Fatalf("expected filtered procedure memo hit, got %#v", hits)
	}
	if hits[0].ObjectType != truthObjectTypeProcedureMemo {
		t.Fatalf("expected procedure memo hit, got %#v", hits[0])
	}
	if hits[0].Distance <= 0 || hits[0].Score <= 0 {
		t.Fatalf("expected score/distance debug fields, got %#v", hits[0])
	}
}

func TestVectorSidecarSyncFailOpenDoesNotBlockTruthWriter(t *testing.T) {
	blockedBaseDir := t.TempDir()
	blockedVectorPath := filepath.Join(blockedBaseDir, "vector-blocked")
	if err := os.WriteFile(blockedVectorPath, []byte("not-a-dir"), 0o600); err != nil {
		t.Fatalf("create blocked vector path: %v", err)
	}
	manager := newVectorTestManager(t, blockedVectorPath, false)
	object := buildVectorTestProcedureMemo()
	if _, err := manager.truth.UpsertObject(object); err != nil {
		t.Fatalf("expected truth upsert to stay fail-open, got %v", err)
	}
	if !manager.truth.waitForSidecar(2 * time.Second) {
		t.Fatal("timed out waiting for blocked vector sidecar sync")
	}
	objects, err := readTruthObjectSnapshot(manager.truth.objectsPath)
	if err != nil {
		t.Fatalf("read truth objects: %v", err)
	}
	indexed := findTruthObjectByID(objects, object.ObjectID)
	if indexed.ObjectID == "" {
		t.Fatalf("expected truth object to persist despite vector failure, got %#v", objects)
	}
	if !truthObjectHasEmbeddingStatus(indexed, vectorEmbeddingStatusError) {
		t.Fatalf("expected vector embedding ref to record error status, got %#v", indexed.EmbeddingRefs)
	}
}

func newVectorTestManager(t *testing.T, vectorPath string, shadow bool) *MemoryManager {
	t.Helper()
	baseDir := t.TempDir()
	if vectorPath == "" {
		vectorPath = filepath.Join(baseDir, "vector")
	}
	return NewMemoryManager(MemoryConfig{
		WarmCapacity:         16,
		WarmPath:             filepath.Join(baseDir, "warm.json"),
		ColdBaseDir:          filepath.Join(baseDir, "cold"),
		TruthEnabled:         true,
		TruthDualWrite:       true,
		TruthBaseDir:         filepath.Join(baseDir, "truth"),
		TruthShadowFailOpen:  true,
		IntentPlannerEnabled: true,
		VectorEnabled:        true,
		VectorPath:           vectorPath,
		VectorTopK:           4,
		VectorMinScore:       0.2,
		ShadowRecallEnabled:  shadow,
		AutoRecallEnabled:    true,
		AutoRecallLimit:      4,
	})
}

func seedVectorTruthObjects(t *testing.T, manager *MemoryManager) {
	t.Helper()
	if _, err := manager.truth.UpsertObject(buildVectorTestSemanticNote()); err != nil {
		t.Fatalf("upsert semantic note: %v", err)
	}
	if _, err := manager.truth.UpsertObject(buildVectorTestProcedureMemo()); err != nil {
		t.Fatalf("upsert procedure memo: %v", err)
	}
	if !manager.truth.waitForSidecar(2 * time.Second) {
		t.Fatal("timed out waiting for vector sidecar sync")
	}
}

func buildVectorTestSemanticNote() MemoryObject {
	now := time.Date(2026, 3, 7, 15, 0, 0, 0, time.UTC)
	return MemoryObject{
		ObjectID:   "obj-vector-note",
		ObjectType: truthObjectTypeSemanticNote,
		Summary:    "Config migration checklist",
		RawEvidence: []MemoryEvidence{{
			Kind:       "markdown.note",
			Summary:    "Remember config migration steps and validation checks",
			Timestamp:  now,
			Confidence: 0.84,
		}},
		Claims: []MemoryClaim{{
			Type:           truthClaimTypeConstraint,
			ConstraintType: "validation_check",
			Value:          "run migration check",
			Confidence:     0.84,
			CreatedAt:      now,
		}},
		SourceRefs: []SourceRef{{SourceKind: truthSourceKindMarkdownNode, SourceID: "node-vector-note"}},
		CreatedAt:  now,
		UpdatedAt:  now,
		Confidence: 0.84,
	}
}

func buildVectorTestProcedureMemo() MemoryObject {
	now := time.Date(2026, 3, 7, 15, 5, 0, 0, time.UTC)
	return MemoryObject{
		ObjectID:   "obj-vector-memo",
		ObjectType: truthObjectTypeProcedureMemo,
		Summary:    "Patch config.toml then run migration check",
		RawEvidence: []MemoryEvidence{{
			Kind:       "decision.memo",
			Summary:    "Patch config.toml and verify config migration succeeds",
			Timestamp:  now,
			Confidence: 0.93,
		}},
		Claims: []MemoryClaim{{
			Type:       truthClaimTypeIntent,
			IntentKey:  "intent.fix_config_migration",
			Value:      "fix config migration",
			Confidence: 0.95,
			CreatedAt:  now,
		}},
		SourceRefs: []SourceRef{{SourceKind: truthSourceKindDecisionMemo, SourceID: "memo-vector"}},
		CreatedAt:  now,
		UpdatedAt:  now,
		Confidence: 0.93,
	}
}

func vectorHitsContainType(hits []VectorHit, objectType string) bool {
	for _, hit := range hits {
		if hit.ObjectType == objectType {
			return true
		}
	}
	return false
}

func findTruthObjectByID(objects []MemoryObject, objectID string) MemoryObject {
	for _, object := range objects {
		if object.ObjectID == objectID {
			return object
		}
	}
	return MemoryObject{}
}

func truthObjectHasEmbeddingStatus(object MemoryObject, status string) bool {
	for _, ref := range object.EmbeddingRefs {
		if ref.Source == vectorEmbeddingSource && ref.Status == status {
			return true
		}
	}
	return false
}
