package memory

import (
	"testing"
	"time"

	"ghost-os/bridge/llm"
)

func TestColdMemoryArchiveAndRetrieve(t *testing.T) {
	cold := NewColdMemory(t.TempDir())
	sessionID := "session-cold-retrieve"

	messages := []llm.Message{
		{Role: llm.RoleUser, Text: "hello cold memory"},
		{Role: llm.RoleAssistant, Text: "cold response"},
	}
	if err := cold.Archive(sessionID, messages); err != nil {
		t.Fatalf("archive: %v", err)
	}

	entries, err := cold.Retrieve(MemoryQuery{
		Metadata: map[string]any{"session_id": sessionID},
	})
	if err != nil {
		t.Fatalf("retrieve: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("unexpected entry count: got %d want %d", len(entries), 2)
	}
	if entries[0].Metadata["session_id"] != sessionID {
		t.Fatalf("unexpected session_id metadata: %+v", entries[0].Metadata)
	}
}

func TestColdMemoryListSessionsByTimeRange(t *testing.T) {
	cold := NewColdMemory(t.TempDir())
	sessionID := "session-time-1"

	if err := cold.Archive(sessionID, []llm.Message{{Role: llm.RoleUser, Text: "recent"}}); err != nil {
		t.Fatalf("archive: %v", err)
	}

	recentRange := TimeRange{
		Start: time.Now().UTC().Add(-24 * time.Hour),
		End:   time.Now().UTC().Add(24 * time.Hour),
	}
	ids, err := cold.ListSessions(recentRange)
	if err != nil {
		t.Fatalf("list recent sessions: %v", err)
	}
	if len(ids) != 1 || ids[0] != sessionID {
		t.Fatalf("unexpected recent sessions: %v", ids)
	}

	oldIDs, err := cold.ListSessions(TimeRange{
		Start: time.Now().UTC().AddDate(-1, 0, 0),
		End:   time.Now().UTC().AddDate(-1, 0, 0).Add(24 * time.Hour),
	})
	if err != nil {
		t.Fatalf("list old sessions: %v", err)
	}
	if len(oldIDs) != 0 {
		t.Fatalf("unexpected old sessions: %v", oldIDs)
	}
}
