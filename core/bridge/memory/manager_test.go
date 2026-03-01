package memory

import (
	"path/filepath"
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
		HotHistory:   hot,
		HotSessionID: "session-hot",
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

	hotEntries, err := manager.Query(MemoryQuery{Keywords: []string{"hot-layer"}})
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

	hot := agent.NewHistory("")
	for _, msg := range sess.Messages {
		hot.Append(msg)
	}

	manager := NewMemoryManager(MemoryConfig{
		WarmCapacity: 10,
		WarmPath:     filepath.Join(t.TempDir(), "warm.json"),
		ColdBaseDir:  filepath.Join(t.TempDir(), "cold"),
		HotHistory:   hot,
		HotSessionID: sess.ID,
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
