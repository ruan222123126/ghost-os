package session

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"ghost-os/bridge/llm"
)

func TestStoreSaveAndLoadSession(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	s := NewSession("system")
	s.ID = "session-roundtrip"
	s.AddMessage(llm.Message{Role: llm.RoleUser, Text: "hello"})
	s.AddMessage(llm.Message{Role: llm.RoleAssistant, Text: "hi"})

	if err := store.Save(s); err != nil {
		t.Fatalf("save session: %v", err)
	}

	loaded, err := store.Load(s.ID)
	if err != nil {
		t.Fatalf("load session: %v", err)
	}
	if loaded.ID != s.ID {
		t.Fatalf("unexpected id: got %q want %q", loaded.ID, s.ID)
	}
	if !reflect.DeepEqual(loaded.Messages, s.Messages) {
		t.Fatalf("messages mismatch: got=%+v want=%+v", loaded.Messages, s.Messages)
	}
	if loaded.TokenCount <= 0 {
		t.Fatalf("unexpected token count: got %d want > 0", loaded.TokenCount)
	}
}

func TestStoreLoadMissingSession(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	_, err = store.Load("missing-session")
	if !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("expected ErrSessionNotFound, got: %v", err)
	}
}

func TestStoreLoadCorruptedSession(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	path := filepath.Join(dir, "broken-session.json")
	if err := os.WriteFile(path, []byte("{invalid json"), 0o600); err != nil {
		t.Fatalf("write corrupted session: %v", err)
	}

	_, err = store.Load("broken-session")
	if !errors.Is(err, ErrSessionCorrupted) {
		t.Fatalf("expected ErrSessionCorrupted, got: %v", err)
	}
}

func TestStoreListSessions(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	s1 := NewSession("system")
	s1.ID = "session-a"
	s2 := NewSession("system")
	s2.ID = "session-b"

	if err := store.Save(s2); err != nil {
		t.Fatalf("save s2: %v", err)
	}
	if err := store.Save(s1); err != nil {
		t.Fatalf("save s1: %v", err)
	}

	got, err := store.List()
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}
	want := []string{"session-a", "session-b"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected list result: got=%v want=%v", got, want)
	}
}

func TestStoreRejectsInvalidSessionID(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	_, err = store.Load("../etc/passwd")
	if !errors.Is(err, ErrInvalidSessionID) {
		t.Fatalf("expected ErrInvalidSessionID, got: %v", err)
	}
}
