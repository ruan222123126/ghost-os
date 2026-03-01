package memory

import (
	"path/filepath"
	"testing"
	"time"
)

func TestWarmMemoryStoreUsesLRUEviction(t *testing.T) {
	warm := NewWarmMemory(2, filepath.Join(t.TempDir(), "warm.json"))

	if err := warm.Store(MemoryEntry{ID: "a", Content: "first", Timestamp: time.Now().UTC()}); err != nil {
		t.Fatalf("store a: %v", err)
	}
	if err := warm.Store(MemoryEntry{ID: "b", Content: "second", Timestamp: time.Now().UTC()}); err != nil {
		t.Fatalf("store b: %v", err)
	}

	// 访问 a，让 a 成为最近使用，后续应淘汰 b。
	if _, err := warm.Retrieve(MemoryQuery{Keywords: []string{"first"}}); err != nil {
		t.Fatalf("retrieve a: %v", err)
	}

	if err := warm.Store(MemoryEntry{ID: "c", Content: "third", Timestamp: time.Now().UTC()}); err != nil {
		t.Fatalf("store c: %v", err)
	}

	entries, err := warm.Retrieve(MemoryQuery{})
	if err != nil {
		t.Fatalf("retrieve all: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("unexpected size: got %d want %d", len(entries), 2)
	}

	foundA := false
	foundB := false
	foundC := false
	for _, entry := range entries {
		switch entry.ID {
		case "a":
			foundA = true
		case "b":
			foundB = true
		case "c":
			foundC = true
		}
	}
	if !foundA || !foundC || foundB {
		t.Fatalf("unexpected ids after eviction: a=%v b=%v c=%v", foundA, foundB, foundC)
	}
}

func TestWarmMemoryPersistAndLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "warm.json")
	now := time.Now().UTC()

	warm := NewWarmMemory(3, path)
	if err := warm.Store(MemoryEntry{ID: "fresh", Content: "fresh data", Timestamp: now}); err != nil {
		t.Fatalf("store fresh: %v", err)
	}
	if err := warm.Store(MemoryEntry{ID: "expired", Content: "old data", Timestamp: now.Add(-25 * time.Hour)}); err != nil {
		t.Fatalf("store expired: %v", err)
	}

	loaded := NewWarmMemory(3, path)
	if err := loaded.Load(); err != nil {
		t.Fatalf("load warm memory: %v", err)
	}

	entries, err := loaded.Retrieve(MemoryQuery{})
	if err != nil {
		t.Fatalf("retrieve loaded entries: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("unexpected entry count after prune: got %d want %d", len(entries), 1)
	}
	if entries[0].ID != "fresh" {
		t.Fatalf("unexpected entry id: got %q want %q", entries[0].ID, "fresh")
	}
}
