package memory

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"ghost-os/bridge/agent"
	"ghost-os/bridge/llm"
	"ghost-os/bridge/session"
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

func TestMemoryManagerArchiveUpdatesSessionMetadata(t *testing.T) {
	sessionStore, err := session.NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("new session store: %v", err)
	}

	sess := session.NewSession("system")
	sess.ID = "session-archive-metadata"
	sess.AddMessage(llm.Message{Role: llm.RoleUser, Text: "archive me"})
	if err := sessionStore.Save(sess); err != nil {
		t.Fatalf("save session: %v", err)
	}

	manager := NewMemoryManager(MemoryConfig{
		WarmCapacity: 10,
		WarmPath:     filepath.Join(t.TempDir(), "warm.json"),
		ColdBaseDir:  filepath.Join(t.TempDir(), "cold"),
		SessionStore: sessionStore,
	})

	if err := manager.ArchiveToCold(sess.ID); err != nil {
		t.Fatalf("archive to cold: %v", err)
	}

	loaded, err := sessionStore.Load(sess.ID)
	if err != nil {
		t.Fatalf("load session after archive: %v", err)
	}
	if loaded.MemoryMetadata.ArchivedAt.IsZero() {
		t.Fatal("expected archived_at to be set")
	}
}

func TestMemoryManagerPromoteToWarm(t *testing.T) {
	baseDir := t.TempDir()
	manager := NewMemoryManager(MemoryConfig{
		WarmCapacity: 10,
		WarmPath:     filepath.Join(baseDir, "warm.json"),
		ColdBaseDir:  filepath.Join(baseDir, "cold"),
	})

	if err := manager.cold.Archive("session-promote", []llm.Message{{Role: llm.RoleUser, Text: "promote from cold"}}); err != nil {
		t.Fatalf("archive cold data: %v", err)
	}
	if err := manager.PromoteToWarm("session-promote"); err != nil {
		t.Fatalf("promote to warm: %v", err)
	}

	entries, err := manager.warm.Retrieve(MemoryQuery{
		Metadata: map[string]any{"session_id": "session-promote"},
	})
	if err != nil {
		t.Fatalf("retrieve warm entries: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("expected promoted entries in warm layer")
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

type testSummarizer struct {
	summary string
}

func (s testSummarizer) Summarize(_ []llm.Message) (string, error) {
	return s.summary, nil
}

func TestMemoryManagerEvolveCreatesMarkdownAndRemovesWarm(t *testing.T) {
	baseDir := t.TempDir()
	manager := NewMemoryManager(MemoryConfig{
		WarmCapacity:     10,
		WarmPath:         filepath.Join(baseDir, "warm.json"),
		ColdBaseDir:      filepath.Join(baseDir, "cold"),
		WarmTTL:          24 * time.Hour,
		EvolutionEnabled: false,
		Summarizer:       testSummarizer{summary: "Summarized memory node content"},
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
	if !strings.Contains(node.Content, "Summarized") {
		t.Fatalf("unexpected node content: %q", node.Content)
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

func TestMemoryManagerStoreWarmMessages(t *testing.T) {
	baseDir := t.TempDir()
	manager := NewMemoryManager(MemoryConfig{
		WarmCapacity: 10,
		WarmPath:     filepath.Join(baseDir, "warm.json"),
		ColdBaseDir:  filepath.Join(baseDir, "cold"),
	})

	messages := []llm.Message{
		{Role: llm.RoleUser, Text: "remember this user preference"},
		{Role: llm.RoleAssistant, Text: "stored in warm memory"},
	}
	if err := manager.StoreWarmMessages("session-store-warm", 5, messages); err != nil {
		t.Fatalf("store warm messages: %v", err)
	}

	entries, err := manager.warm.Retrieve(MemoryQuery{
		Metadata: map[string]any{"session_id": "session-store-warm"},
		Limit:    10,
	})
	if err != nil {
		t.Fatalf("retrieve warm messages: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("unexpected warm entries count: got %d want 2", len(entries))
	}

	found := map[string]bool{
		"session-store-warm:000005": false,
		"session-store-warm:000006": false,
	}
	for _, entry := range entries {
		if _, ok := found[entry.ID]; ok {
			found[entry.ID] = true
		}
	}
	for id, ok := range found {
		if !ok {
			t.Fatalf("missing expected warm entry id: %s", id)
		}
	}
}

func TestMemoryManagerStopDreamingStopsBackgroundEvolution(t *testing.T) {
	baseDir := t.TempDir()
	manager := NewMemoryManager(MemoryConfig{
		WarmCapacity:      10,
		WarmPath:          filepath.Join(baseDir, "warm.json"),
		ColdBaseDir:       filepath.Join(baseDir, "cold"),
		EvolutionEnabled:  true,
		EvolutionInterval: 10 * time.Millisecond,
	})

	deadline := time.Now().Add(500 * time.Millisecond)
	for manager.evolutionRuns.Load() == 0 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if manager.evolutionRuns.Load() == 0 {
		t.Fatal("expected background evolution to run at least once")
	}

	manager.StopDreaming()
	stoppedAt := manager.evolutionRuns.Load()
	time.Sleep(30 * time.Millisecond)
	if got := manager.evolutionRuns.Load(); got != stoppedAt {
		t.Fatalf("expected dreaming to stop after shutdown: got %d want %d", got, stoppedAt)
	}

	manager.StopDreaming()
}
