package session

import (
	"errors"
	"fmt"
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

func TestStoreLoadExtendsHotWindowToUserBoundary(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	sess := NewSession("")
	sess.ID = "session-hot-window-boundary"
	sess.AddMessage(llm.Message{Role: llm.RoleUser, Text: "turn-start"})
	for i := 0; i < hotWindowMaxMessages+10; i++ {
		sess.AddMessage(llm.Message{
			Role: llm.RoleAssistant,
			Text: fmt.Sprintf("assistant-%03d", i),
		})
	}
	if err := store.Save(sess); err != nil {
		t.Fatalf("save session: %v", err)
	}

	loaded, err := store.Load(sess.ID)
	if err != nil {
		t.Fatalf("load session: %v", err)
	}
	if len(loaded.Messages) <= hotWindowMaxMessages {
		t.Fatalf("expected load to expand past hot-window size, got %d", len(loaded.Messages))
	}
	if loaded.WindowStart != 0 {
		t.Fatalf("expected window start to be rewound to user boundary, got %d", loaded.WindowStart)
	}
	if loaded.Messages[0].Role != llm.RoleUser || loaded.Messages[0].Text != "turn-start" {
		t.Fatalf("expected first loaded message to be the user boundary, got %+v", loaded.Messages[0])
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
