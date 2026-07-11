package session

import (
	"testing"

	"ghost-os/bridge/llm"
)

func TestStoreUpdateTitlePreservesMessagesAndSaveMerge(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	sess := NewSession("system")
	sess.ID = "session-title"
	sess.AddMessage(llm.Message{Role: llm.RoleUser, Text: "hello"})
	if err := store.Save(sess); err != nil {
		t.Fatalf("save session: %v", err)
	}
	stale := loadTitleTestSession(t, store, sess.ID)
	if err := store.UpdateTitle(sess.ID, "Generated Title"); err != nil {
		t.Fatalf("update title: %v", err)
	}
	stale.AddMessage(llm.Message{Role: llm.RoleAssistant, Text: "hi"})
	if err := store.Save(stale); err != nil {
		t.Fatalf("save stale session: %v", err)
	}

	loaded := loadTitleTestSession(t, store, sess.ID)
	if loaded.Title != "Generated Title" {
		t.Fatalf("unexpected title: got %q", loaded.Title)
	}
	if loaded.MessageCount != 3 {
		t.Fatalf("unexpected message count: got %d", loaded.MessageCount)
	}
	if loaded.Messages[1].Text != "hello" || loaded.Messages[2].Text != "hi" {
		t.Fatalf("unexpected message order: %+v", loaded.Messages)
	}
}

func TestStorePersistsTitleInLoadAndListMetadata(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	sess := NewSession("system")
	sess.ID = "session-title-roundtrip"
	sess.Title = "Roundtrip"
	if err := store.Save(sess); err != nil {
		t.Fatalf("save session: %v", err)
	}

	loaded := loadTitleTestSession(t, store, sess.ID)
	if loaded.Title != "Roundtrip" {
		t.Fatalf("unexpected loaded title: %q", loaded.Title)
	}
	metadata, err := store.ListMetadata()
	if err != nil {
		t.Fatalf("list metadata: %v", err)
	}
	if len(metadata) != 1 || metadata[0].Title != "Roundtrip" {
		t.Fatalf("unexpected metadata title: %+v", metadata)
	}
}

func loadTitleTestSession(t *testing.T, store *Store, sessionID string) *Session {
	t.Helper()
	loaded, err := store.Load(sessionID)
	if err != nil {
		t.Fatalf("load session: %v", err)
	}
	return loaded
}
