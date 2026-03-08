package memory

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"ghost-os/bridge/llm"
)

type testSummarizer struct {
	summary string
}

func (s testSummarizer) Summarize(_ []llm.Message) (string, error) {
	return s.summary, nil
}

func TestMemoryManagerEvolveCreatesMarkdownAndRemovesWarm(t *testing.T) {
	baseDir := t.TempDir()
	manager := NewMemoryManager(MemoryConfig{
		Warm:    WarmConfig{Capacity: 10, Path: filepath.Join(baseDir, "warm.json"), TTL: 24 * time.Hour, EvolutionEnabled: false},
		Cold:    ColdConfig{BaseDir: filepath.Join(baseDir, "cold")},
		Runtime: RuntimeConfig{Summarizer: testSummarizer{summary: "Summarized memory node content"}},
	})

	oldTS := time.Now().UTC().Add(-7 * time.Hour)
	storeEntry := func(id, content string) {
		if err := manager.warm.Store(MemoryEntry{
			ID:         id,
			Content:    content,
			Timestamp:  oldTS,
			Importance: 0.2,
			Metadata:   map[string]any{"session_id": "session-evolve", "role": "assistant"},
		}); err != nil {
			t.Fatalf("store warm evolve entry %s: %v", id, err)
		}
	}
	storeEntry("evolve-1", "Ghost-OS memory work item one")
	storeEntry("evolve-2", "Ghost-OS memory work item two")

	stats, err := manager.Evolve()
	if err != nil {
		t.Fatalf("evolve: %v", err)
	}
	if stats.NodesCreated != 1 {
		t.Fatalf("unexpected nodes created: got %d want 1", stats.NodesCreated)
	}
	if stats.ProcessedEntries != 2 {
		t.Fatalf("unexpected processed entries: got %d want 2", stats.ProcessedEntries)
	}

	remaining, err := manager.warm.Retrieve(MemoryQuery{Metadata: map[string]any{"session_id": "session-evolve"}})
	if err != nil {
		t.Fatalf("retrieve warm after evolve: %v", err)
	}
	if len(remaining) != 0 {
		t.Fatalf("expected warm entries to be evolved out, got %d", len(remaining))
	}

	nodes, err := manager.ListMarkdownNodes()
	if err != nil {
		t.Fatalf("list markdown nodes: %v", err)
	}
	if len(nodes) != 1 {
		t.Fatalf("expected one markdown node, got %d", len(nodes))
	}
	node, err := manager.LoadMarkdownNode(nodes[0])
	if err != nil {
		t.Fatalf("load markdown node: %v", err)
	}
	if node.SessionID != "session-evolve" {
		t.Fatalf("unexpected node session id: %q", node.SessionID)
	}
	if len(node.RelatedTo) != 2 {
		t.Fatalf("unexpected related links count: got %d want 2", len(node.RelatedTo))
	}
	if node.Summary != "Summarized memory node content" {
		t.Fatalf("unexpected node summary: %q", node.Summary)
	}
	if len(node.SourceIDs) != 2 {
		t.Fatalf("unexpected source ids count: got %d want 2", len(node.SourceIDs))
	}
	if !strings.Contains(node.Content, "## Summary") || !strings.Contains(node.Content, "Summarized") {
		t.Fatalf("unexpected node content: %q", node.Content)
	}
}

func TestMemoryManagerEvolveExtractsStructuredAnchors(t *testing.T) {
	baseDir := t.TempDir()
	manager := NewMemoryManager(MemoryConfig{
		Warm: WarmConfig{
			Capacity:         10,
			Path:             filepath.Join(baseDir, "warm.json"),
			TTL:              24 * time.Hour,
			EvolutionEnabled: false,
			AnchorEnabled:    true,
			AnchorMinWeight:  0.65,
		},
		Cold: ColdConfig{BaseDir: filepath.Join(baseDir, "cold")},
	})

	oldTS := time.Now().UTC().Add(-7 * time.Hour)
	entries := []MemoryEntry{
		{
			ID:         "anchor-1",
			Content:    "I prefer Go for backend services.",
			Timestamp:  oldTS,
			Importance: 0.2,
			Metadata:   map[string]any{"session_id": "session-anchor", "role": "user", "layer": "warm"},
		},
		{
			ID:         "anchor-2",
			Content:    "不要动生产库。",
			Timestamp:  oldTS.Add(time.Minute),
			Importance: 0.2,
			Metadata:   map[string]any{"session_id": "session-anchor", "role": "user", "layer": "warm"},
		},
	}
	for _, entry := range entries {
		if err := manager.warm.Store(entry); err != nil {
			t.Fatalf("store anchor entry %s: %v", entry.ID, err)
		}
	}

	stats, err := manager.Evolve()
	if err != nil {
		t.Fatalf("evolve anchors: %v", err)
	}
	if stats.NodesCreated != 1 {
		t.Fatalf("unexpected nodes created: got %d want 1", stats.NodesCreated)
	}

	nodes, err := manager.ListMarkdownNodes()
	if err != nil {
		t.Fatalf("list markdown nodes: %v", err)
	}
	if len(nodes) != 1 {
		t.Fatalf("unexpected markdown node count: got %d want 1", len(nodes))
	}
	node, err := manager.LoadMarkdownNode(nodes[0])
	if err != nil {
		t.Fatalf("load markdown node: %v", err)
	}
	if len(node.Anchors) < 2 {
		t.Fatalf("expected structured anchors, got %+v", node.Anchors)
	}
	if len(node.SourceIDs) != 2 {
		t.Fatalf("expected source ids to be preserved, got %+v", node.SourceIDs)
	}
	if node.Summary == "" {
		t.Fatal("expected summary to be populated")
	}
}

func TestMemoryManagerStopDreamingStopsBackgroundEvolution(t *testing.T) {
	baseDir := t.TempDir()
	manager := NewMemoryManager(MemoryConfig{
		Warm: WarmConfig{
			Capacity:          10,
			Path:              filepath.Join(baseDir, "warm.json"),
			EvolutionEnabled:  true,
			EvolutionInterval: 10 * time.Millisecond,
		},
		Cold: ColdConfig{BaseDir: filepath.Join(baseDir, "cold")},
	})

	deadline := time.Now().Add(500 * time.Millisecond)
	for manager.Metrics().EvolutionRuns == 0 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if manager.Metrics().EvolutionRuns == 0 {
		t.Fatal("expected background evolution to run at least once")
	}

	manager.StopDreaming()
	stoppedAt := manager.Metrics().EvolutionRuns
	time.Sleep(30 * time.Millisecond)
	if got := manager.Metrics().EvolutionRuns; got != stoppedAt {
		t.Fatalf("expected dreaming to stop after shutdown: got %d want %d", got, stoppedAt)
	}

	manager.StopDreaming()
}
