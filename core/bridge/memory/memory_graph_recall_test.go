package memory

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func newGraphRecallManager(t *testing.T) *MemoryManager {
	t.Helper()
	baseDir := t.TempDir()
	return NewMemoryManager(MemoryConfig{
		WarmCapacity:         8,
		WarmPath:             filepath.Join(baseDir, "warm.json"),
		ColdBaseDir:          filepath.Join(baseDir, "cold"),
		AutoRecallEnabled:    true,
		AutoRecallLimit:      3,
		GraphEnabled:         true,
		GraphPath:            filepath.Join(baseDir, "graph"),
		GraphExtractOnEvolve: true,
		GraphNamespace:       "workspace:test",
		GraphDebugEnabled:    true,
	})
}

func TestQueryResultWithScopeIncludesGraphRecallHits(t *testing.T) {
	manager := newGraphRecallManager(t)
	createdAt := time.Date(2026, 3, 8, 10, 0, 0, 0, time.UTC)
	node := MarkdownNode{
		ID:         "node-graph-recall",
		SessionID:  "session-graph-recall",
		CreatedAt:  createdAt,
		Summary:    "migration-project depends on config.toml",
		Content:    "migration-project depends on config.toml. Alice owns migration-project.",
		SourceIDs:  []string{"session-graph-recall:000000"},
		Confidence: 0.86,
	}
	if err := manager.SaveMarkdownNode(node); err != nil {
		t.Fatalf("save markdown node: %v", err)
	}
	if err := manager.graph.IngestMarkdownNode(node); err != nil {
		t.Fatalf("ingest markdown node into graph: %v", err)
	}

	result, err := manager.QueryResultWithScope(MemoryQuery{
		IncludeGraph:    true,
		GraphDebug:      true,
		GraphPredicates: []string{GraphPredicateDependsOn},
		SemanticQuery:   "migration-project config.toml dependency",
	}, SessionScope{SessionID: "session-graph-recall"})
	if err != nil {
		t.Fatalf("query result with graph recall: %v", err)
	}
	if len(result.GraphHits) == 0 {
		t.Fatalf("expected graph debug hits, got %+v", result)
	}
	foundGraphEntry := false
	for _, entry := range result.Entries {
		if entryLayer(entry) != "graph" {
			continue
		}
		foundGraphEntry = true
		if !strings.Contains(entry.Content, "migration-project") || !strings.Contains(entry.Content, "config.toml") {
			t.Fatalf("expected graph recall entry to describe dependency edge, got %+v", entry)
		}
		break
	}
	if !foundGraphEntry {
		t.Fatalf("expected graph entry in query results, got %+v", result.Entries)
	}
	matchedPredicate := false
	for _, hit := range result.GraphHits {
		if hit.Predicate == GraphPredicateDependsOn && strings.Contains(hit.Content, "config.toml") {
			matchedPredicate = true
			break
		}
	}
	if !matchedPredicate {
		t.Fatalf("expected depends_on graph hit, got %+v", result.GraphHits)
	}
	stats := manager.GraphStats("workspace:test")
	if stats.NodeCount == 0 || stats.EdgeCount == 0 {
		t.Fatalf("expected graph stats to reflect ingested recall data, got %+v", stats)
	}
}
