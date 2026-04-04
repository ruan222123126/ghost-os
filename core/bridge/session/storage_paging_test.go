package session

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"ghost-os/bridge/llm"
)

func TestStoreLoadPageSupportsBeforeCursor(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	sess := NewSession("")
	sess.ID = "session-page"
	for i := 0; i < 6; i++ {
		sess.AddMessage(llm.Message{Role: llm.RoleUser, Text: fmt.Sprintf("message-%d", i)})
	}
	if err := store.Save(sess); err != nil {
		t.Fatalf("save session: %v", err)
	}

	loaded, latest, err := store.LoadPage(sess.ID, PageParams{Limit: 2})
	if err != nil {
		t.Fatalf("load latest page: %v", err)
	}
	if loaded.MessageCount != 6 {
		t.Fatalf("unexpected message count: got=%d want=6", loaded.MessageCount)
	}
	if len(latest.Messages) != 2 {
		t.Fatalf("unexpected latest page size: got=%d want=2", len(latest.Messages))
	}
	if latest.StartIndex == nil || *latest.StartIndex != 4 {
		t.Fatalf("unexpected latest start index: %+v", latest.StartIndex)
	}
	if latest.EndIndex == nil || *latest.EndIndex != 5 {
		t.Fatalf("unexpected latest end index: %+v", latest.EndIndex)
	}
	if !latest.HasMoreBefore {
		t.Fatal("expected older history before latest page")
	}
	if latest.NextBefore == nil || *latest.NextBefore != 4 {
		t.Fatalf("unexpected latest next_before: %+v", latest.NextBefore)
	}

	_, older, err := store.LoadPage(sess.ID, PageParams{
		Limit:  2,
		Before: latest.NextBefore,
	})
	if err != nil {
		t.Fatalf("load older page: %v", err)
	}
	if len(older.Messages) != 2 {
		t.Fatalf("unexpected older page size: got=%d want=2", len(older.Messages))
	}
	if older.Messages[0].Index != 2 || older.Messages[1].Index != 3 {
		t.Fatalf("unexpected older page indexes: %+v", older.Messages)
	}
	if older.NextBefore == nil || *older.NextBefore != 2 {
		t.Fatalf("unexpected older next_before: %+v", older.NextBefore)
	}
}

func TestStoreLoadKeepsOnlyHotWindowInMemory(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	sess := NewSession("")
	sess.ID = "session-hot-window"
	totalMessages := hotWindowMaxMessages + 25
	for i := 0; i < totalMessages; i++ {
		sess.AddMessage(llm.Message{Role: llm.RoleUser, Text: fmt.Sprintf("m-%03d", i)})
	}
	if err := store.Save(sess); err != nil {
		t.Fatalf("save session: %v", err)
	}

	loaded, err := store.Load(sess.ID)
	if err != nil {
		t.Fatalf("load session: %v", err)
	}
	if loaded.MessageCount != totalMessages {
		t.Fatalf("unexpected total message count: got=%d want=%d", loaded.MessageCount, totalMessages)
	}
	if len(loaded.Messages) != hotWindowMaxMessages {
		t.Fatalf("unexpected hot window size: got=%d want=%d", len(loaded.Messages), hotWindowMaxMessages)
	}
	if loaded.WindowStart != totalMessages-hotWindowMaxMessages {
		t.Fatalf("unexpected window start: got=%d want=%d", loaded.WindowStart, totalMessages-hotWindowMaxMessages)
	}
	if loaded.Messages[0].Text != "m-025" {
		t.Fatalf("unexpected first hot-window message: %q", loaded.Messages[0].Text)
	}
	if loaded.Messages[len(loaded.Messages)-1].Text != fmt.Sprintf("m-%03d", totalMessages-1) {
		t.Fatalf("unexpected last hot-window message: %q", loaded.Messages[len(loaded.Messages)-1].Text)
	}
}

func TestStoreSaveRejectsNonAppendOnlyMutation(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	sess := NewSession("")
	sess.ID = "session-append-only"
	sess.AddMessage(llm.Message{Role: llm.RoleUser, Text: "hello"})
	sess.AddMessage(llm.Message{Role: llm.RoleAssistant, Text: "hi"})
	if err := store.Save(sess); err != nil {
		t.Fatalf("save session: %v", err)
	}

	sess.Messages[0].Text = "mutated"
	err = store.Save(sess)
	if !errors.Is(err, ErrSessionNotAppendOnly) {
		t.Fatalf("expected ErrSessionNotAppendOnly, got: %v", err)
	}
}

func TestStoreLoadImportsLegacySessionFile(t *testing.T) {
	dir := t.TempDir()
	legacy := NewSession("")
	legacy.ID = "legacy-session"
	legacy.AddMessage(llm.Message{Role: llm.RoleUser, Text: "hello from legacy"})
	encoded, err := json.MarshalIndent(legacy, "", "  ")
	if err != nil {
		t.Fatalf("encode legacy session: %v", err)
	}
	encoded = append(encoded, '\n')
	legacyPath := filepath.Join(dir, legacy.ID+".json")
	if err := os.WriteFile(legacyPath, encoded, 0o600); err != nil {
		t.Fatalf("write legacy session: %v", err)
	}

	store, err := NewStore(dir)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	loaded, err := store.Load(legacy.ID)
	if err != nil {
		t.Fatalf("load imported legacy session: %v", err)
	}
	if loaded.ID != legacy.ID {
		t.Fatalf("unexpected loaded id: got=%q want=%q", loaded.ID, legacy.ID)
	}
	if loaded.MessageCount != len(legacy.Messages) {
		t.Fatalf("unexpected loaded message count: got=%d want=%d", loaded.MessageCount, len(legacy.Messages))
	}
	if _, err := os.Stat(legacyPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected legacy file removal, got stat err=%v", err)
	}
}
