package memory

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"ghost-os/bridge/agent"
	"ghost-os/bridge/llm"
)

func TestMemoryManagerQueryCascadesHotWarmCold(t *testing.T) {
	baseDir := t.TempDir()
	warmPath := filepath.Join(baseDir, "warm.json")
	coldDir := filepath.Join(baseDir, "cold")

	hot := agent.NewHistory("")
	hot.Append(llm.Message{Role: llm.RoleUser, Text: "hot-layer message"})

	manager := NewMemoryManager(MemoryConfig{
		WarmCapacity: 100,
		WarmPath:     warmPath,
		ColdBaseDir:  coldDir,
	})

	if err := manager.warm.Store(MemoryEntry{
		ID:        "warm-1",
		Content:   "warm-layer message",
		Type:      MemoryTypeSummary,
		Timestamp: time.Now().UTC(),
		Metadata:  map[string]any{"session_id": "session-warm", "layer": "warm"},
	}); err != nil {
		t.Fatalf("store warm entry: %v", err)
	}
	if err := manager.cold.Archive("session-cold", []llm.Message{{Role: llm.RoleAssistant, Text: "cold-layer message"}}); err != nil {
		t.Fatalf("archive cold session: %v", err)
	}

	hotEntries, err := manager.QueryWithScope(MemoryQuery{Keywords: []string{"hot-layer"}}, SessionScope{
		SessionID: "session-hot",
		History:   hot,
	})
	if err != nil {
		t.Fatalf("query hot: %v", err)
	}
	if len(hotEntries) == 0 {
		t.Fatal("expected hot layer result")
	}
	if hotEntries[0].Metadata["layer"] != "hot" {
		t.Fatalf("expected hot layer, got %+v", hotEntries[0].Metadata)
	}

	warmEntries, err := manager.Query(MemoryQuery{Keywords: []string{"warm-layer"}})
	if err != nil {
		t.Fatalf("query warm: %v", err)
	}
	if len(warmEntries) == 0 {
		t.Fatal("expected warm layer result")
	}

	coldEntries, err := manager.Query(MemoryQuery{Keywords: []string{"cold-layer"}})
	if err != nil {
		t.Fatalf("query cold: %v", err)
	}
	if len(coldEntries) == 0 {
		t.Fatal("expected cold layer result")
	}
}

func TestMemoryManagerBuildContextWindowAutoRecall(t *testing.T) {
	baseDir := t.TempDir()
	manager := NewMemoryManager(MemoryConfig{
		WarmCapacity:      10,
		WarmPath:          filepath.Join(baseDir, "warm.json"),
		ColdBaseDir:       filepath.Join(baseDir, "cold"),
		AutoRecallEnabled: true,
		AutoRecallLimit:   2,
	})

	now := time.Now().UTC()
	if err := manager.warm.Store(MemoryEntry{
		ID:         "warm-recall-1",
		Content:    "Ghost-OS memory architecture uses L1 L2 L3 layers",
		Timestamp:  now,
		Importance: 0.8,
		Metadata:   map[string]any{"session_id": "session-recall", "role": "assistant"},
	}); err != nil {
		t.Fatalf("store warm recall entry: %v", err)
	}

	window, err := manager.BuildContextWindow("session-recall", "Explain memory architecture in Ghost-OS")
	if err != nil {
		t.Fatalf("build context window: %v", err)
	}
	if len(window) != 1 {
		t.Fatalf("unexpected context window size: got %d want 1", len(window))
	}
	if window[0].Role != llm.RoleSystem {
		t.Fatalf("expected system role, got %s", window[0].Role)
	}
	if !strings.Contains(window[0].Text, "memory architecture") {
		t.Fatalf("unexpected context text: %q", window[0].Text)
	}
	if strings.Contains(window[0].Text, "importance=") {
		t.Fatalf("auto-recall output should stay compact, got %q", window[0].Text)
	}
}

func TestMemoryManagerBuildContextWindowSkipsVisibleMessages(t *testing.T) {
	baseDir := t.TempDir()
	manager := NewMemoryManager(MemoryConfig{
		WarmCapacity:      10,
		WarmPath:          filepath.Join(baseDir, "warm.json"),
		ColdBaseDir:       filepath.Join(baseDir, "cold"),
		AutoRecallEnabled: true,
		AutoRecallLimit:   2,
	})

	content := "Ghost-OS memory architecture uses L1 L2 L3 layers"
	if err := manager.warm.Store(MemoryEntry{
		ID:         "warm-visible-1",
		Content:    content,
		Timestamp:  time.Now().UTC(),
		Importance: 0.8,
		Metadata:   map[string]any{"session_id": "session-recall", "role": "assistant"},
	}); err != nil {
		t.Fatalf("store warm recall entry: %v", err)
	}

	history := agent.NewHistory("")
	history.Append(llm.Message{Role: llm.RoleAssistant, Text: content})

	window, err := manager.BuildContextWindowWithScope(SessionScope{
		SessionID: "session-recall",
		History:   history,
	}, "Explain memory architecture in Ghost-OS")
	if err != nil {
		t.Fatalf("build scoped context window: %v", err)
	}
	if len(window) != 0 {
		t.Fatalf("expected visible history to suppress duplicate recall, got %+v", window)
	}
}

func TestMemoryManagerBuildContextWindowIsolatesSessionRecall(t *testing.T) {
	baseDir := t.TempDir()
	manager := NewMemoryManager(MemoryConfig{
		WarmCapacity:      10,
		WarmPath:          filepath.Join(baseDir, "warm.json"),
		ColdBaseDir:       filepath.Join(baseDir, "cold"),
		AutoRecallEnabled: true,
		AutoRecallLimit:   4,
	})

	now := time.Now().UTC()
	store := func(id string, sessionID string, content string) {
		if err := manager.warm.Store(MemoryEntry{
			ID:         id,
			Content:    content,
			Timestamp:  now,
			Importance: 0.8,
			Metadata:   map[string]any{"session_id": sessionID, "role": "assistant"},
		}); err != nil {
			t.Fatalf("store warm entry %s: %v", id, err)
		}
	}
	store("warm-a", "session-a", "Alpha memory architecture details for Ghost-OS")
	store("warm-b", "session-b", "Beta memory architecture details for Ghost-OS")

	alphaWindow, err := manager.BuildContextWindow("session-a", "Explain memory architecture in Ghost-OS")
	if err != nil {
		t.Fatalf("build alpha context window: %v", err)
	}
	if len(alphaWindow) != 1 {
		t.Fatalf("unexpected alpha context window size: got %d want 1", len(alphaWindow))
	}
	if !strings.Contains(alphaWindow[0].Text, "Alpha") {
		t.Fatalf("expected alpha recall, got %q", alphaWindow[0].Text)
	}
	if strings.Contains(alphaWindow[0].Text, "Beta") {
		t.Fatalf("unexpected cross-session recall: %q", alphaWindow[0].Text)
	}

	betaWindow, err := manager.BuildContextWindow("session-b", "Explain memory architecture in Ghost-OS")
	if err != nil {
		t.Fatalf("build beta context window: %v", err)
	}
	if len(betaWindow) != 1 {
		t.Fatalf("unexpected beta context window size: got %d want 1", len(betaWindow))
	}
	if !strings.Contains(betaWindow[0].Text, "Beta") {
		t.Fatalf("expected beta recall, got %q", betaWindow[0].Text)
	}
	if strings.Contains(betaWindow[0].Text, "Alpha") {
		t.Fatalf("unexpected cross-session recall: %q", betaWindow[0].Text)
	}
}

func TestMemoryManagerQueryIncludesMarkdownNodes(t *testing.T) {
	baseDir := t.TempDir()
	manager := NewMemoryManager(MemoryConfig{
		WarmCapacity: 10,
		WarmPath:     filepath.Join(baseDir, "warm.json"),
		ColdBaseDir:  filepath.Join(baseDir, "cold"),
	})

	if err := manager.SaveMarkdownNode(MarkdownNode{
		ID:         "node_markdown_query",
		Importance: 0.9,
		CreatedAt:  time.Now().UTC(),
		Tags:       []string{"ghost-os"},
		Content:    "Ghost-OS memory node for unified query",
	}); err != nil {
		t.Fatalf("save markdown node: %v", err)
	}

	entries, err := manager.Query(MemoryQuery{
		Keywords:        []string{"unified"},
		IncludeMarkdown: true,
	})
	if err != nil {
		t.Fatalf("query with markdown: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("expected markdown query results")
	}

	foundMarkdown := false
	for _, entry := range entries {
		if layer, ok := entry.Metadata["layer"].(string); ok && layer == "markdown" {
			foundMarkdown = true
			break
		}
	}
	if !foundMarkdown {
		t.Fatalf("expected markdown layer entry, got: %+v", entries)
	}
}
