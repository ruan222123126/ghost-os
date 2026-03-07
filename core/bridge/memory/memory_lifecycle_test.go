package memory

import (
	"errors"
	"path/filepath"
	"testing"

	"ghost-os/bridge/llm"
	"ghost-os/bridge/session"
)

type loadOnlySessionStore struct {
	sessions map[string]*session.Session
}

func newLoadOnlySessionStore(sessions ...*session.Session) *loadOnlySessionStore {
	store := &loadOnlySessionStore{sessions: make(map[string]*session.Session, len(sessions))}
	for _, sess := range sessions {
		clone := *sess
		clone.Messages = llm.CloneMessages(sess.Messages)
		store.sessions[sess.ID] = &clone
	}
	return store
}

func (s *loadOnlySessionStore) Load(sessionID string) (*session.Session, error) {
	sess, ok := s.sessions[sessionID]
	if !ok {
		return nil, session.ErrSessionNotFound
	}
	clone := *sess
	clone.Messages = llm.CloneMessages(sess.Messages)
	return &clone, nil
}

func TestMemoryManagerArchiveToColdUsesLoadOnlySessionPort(t *testing.T) {
	baseDir := t.TempDir()
	sess := session.NewSession("system")
	sess.ID = "session-archive-metadata"
	sess.AddMessage(llm.Message{Role: llm.RoleUser, Text: "archive me"})

	manager := NewMemoryManager(MemoryConfig{
		WarmCapacity: 10,
		WarmPath:     filepath.Join(baseDir, "warm.json"),
		ColdBaseDir:  filepath.Join(baseDir, "cold"),
		SessionStore: newLoadOnlySessionStore(sess),
	})

	if err := manager.ArchiveToCold(sess.ID); err != nil {
		t.Fatalf("archive to cold: %v", err)
	}

	entries, err := manager.cold.Retrieve(MemoryQuery{Metadata: map[string]any{"session_id": sess.ID}})
	if err != nil {
		t.Fatalf("retrieve archived entries: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("expected archived entries in cold layer")
	}
}

func TestMemoryManagerArchiveToColdReturnsLoadError(t *testing.T) {
	baseDir := t.TempDir()
	manager := NewMemoryManager(MemoryConfig{
		WarmCapacity: 10,
		WarmPath:     filepath.Join(baseDir, "warm.json"),
		ColdBaseDir:  filepath.Join(baseDir, "cold"),
		SessionStore: &loadOnlySessionStore{sessions: map[string]*session.Session{}},
	})

	err := manager.ArchiveToCold("session-missing")
	if !errors.Is(err, session.ErrSessionNotFound) {
		t.Fatalf("unexpected archive error: got %v want %v", err, session.ErrSessionNotFound)
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
