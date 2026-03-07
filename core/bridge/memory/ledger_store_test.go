package memory

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"ghost-os/bridge/llm"
)

func TestLedgerStoreAppendReplaySessionStableOrder(t *testing.T) {
	store := NewLedgerStore(filepath.Join(t.TempDir(), "ledger"), "default", "", nil)
	firstTurnAt := time.Date(2026, time.March, 7, 10, 0, 0, 0, time.UTC)
	secondTurnAt := firstTurnAt.Add(5 * time.Minute)
	if _, err := store.AppendTurn("session-ledger-order", "", "trace-1", 0, []llm.Message{{Role: llm.RoleUser, Text: "hello"}, {Role: llm.RoleAssistant, Text: "hi there"}}, firstTurnAt); err != nil {
		t.Fatalf("append first turn: %v", err)
	}
	if _, err := store.AppendTurn("session-ledger-order", "", "trace-2", 2, []llm.Message{{Role: llm.RoleUser, Text: "follow up"}}, secondTurnAt); err != nil {
		t.Fatalf("append second turn: %v", err)
	}

	replay, err := store.ReplaySession("session-ledger-order")
	if err != nil {
		t.Fatalf("replay session: %v", err)
	}
	if len(replay.Messages) != 3 {
		t.Fatalf("unexpected message count: got %d want 3", len(replay.Messages))
	}
	if replay.Messages[0].Text != "hello" || replay.Messages[1].Text != "hi there" || replay.Messages[2].Text != "follow up" {
		t.Fatalf("unexpected replay order: %+v", replay.Messages)
	}
	if len(replay.Events) != 5 {
		t.Fatalf("unexpected event count: got %d want 5", len(replay.Events))
	}
	if !replay.ArchivedAt.Equal(secondTurnAt.Add(time.Millisecond)) {
		t.Fatalf("unexpected archived_at: got %s want %s", replay.ArchivedAt, secondTurnAt.Add(time.Millisecond))
	}
}

func TestLedgerStoreAppendDedupe(t *testing.T) {
	store := NewLedgerStore(filepath.Join(t.TempDir(), "ledger"), "default", "", nil)
	turnAt := time.Date(2026, time.March, 7, 11, 0, 0, 0, time.UTC)
	messages := []llm.Message{{Role: llm.RoleUser, Text: "same batch"}}
	if _, err := store.AppendTurn("session-ledger-dedupe", "", "trace-dedupe", 0, messages, turnAt); err != nil {
		t.Fatalf("append first batch: %v", err)
	}
	result, err := store.AppendTurn("session-ledger-dedupe", "", "trace-dedupe", 0, messages, turnAt)
	if err != nil {
		t.Fatalf("append duplicate batch: %v", err)
	}
	if result.Written != 0 || result.Duplicates == 0 {
		t.Fatalf("expected duplicate batch to be skipped, got %+v", result)
	}
	replay, err := store.ReplaySession("session-ledger-dedupe")
	if err != nil {
		t.Fatalf("replay dedupe session: %v", err)
	}
	if len(replay.Messages) != 1 {
		t.Fatalf("unexpected replay messages: got %d want 1", len(replay.Messages))
	}
	if len(replay.Events) != 2 {
		t.Fatalf("unexpected replay events: got %d want 2", len(replay.Events))
	}
}

func TestLedgerStoreRepairTailCorruption(t *testing.T) {
	store := NewLedgerStore(filepath.Join(t.TempDir(), "ledger"), "default", "", nil)
	turnAt := time.Date(2026, time.March, 7, 12, 0, 0, 0, time.UTC)
	if _, err := store.AppendTurn("session-ledger-tail", "", "trace-tail", 0, []llm.Message{{Role: llm.RoleUser, Text: "repair me"}}, turnAt); err != nil {
		t.Fatalf("append initial turn: %v", err)
	}
	segmentPath := filepath.Join(store.monthDir(store.Namespace(), store.WorkspaceID(), turnAt), defaultLedgerSegmentFileName)
	file, err := os.OpenFile(segmentPath, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatalf("open segment: %v", err)
	}
	if _, err := file.WriteString("{\"broken\":"); err != nil {
		_ = file.Close()
		t.Fatalf("write broken tail: %v", err)
	}
	_ = file.Close()

	replay, err := store.ReplaySession("session-ledger-tail")
	if err != nil {
		t.Fatalf("replay repaired session: %v", err)
	}
	if len(replay.Messages) != 1 || replay.Messages[0].Text != "repair me" {
		t.Fatalf("unexpected repaired replay: %+v", replay.Messages)
	}
	data, err := os.ReadFile(segmentPath)
	if err != nil {
		t.Fatalf("read repaired segment: %v", err)
	}
	if string(data) == "" || string(data[len(data)-1]) != "\n" {
		t.Fatalf("expected repaired segment to end with newline, got %q", string(data))
	}
}

func TestColdMemoryLedgerBackfillCompat(t *testing.T) {
	baseDir := t.TempDir()
	legacyCold := NewColdMemory(baseDir)
	sessionID := "session-ledger-backfill"
	messages := []llm.Message{{Role: llm.RoleUser, Text: "backfill this"}, {Role: llm.RoleAssistant, Text: "done"}}
	if err := legacyCold.Archive(sessionID, messages); err != nil {
		t.Fatalf("archive legacy cold: %v", err)
	}
	legacyEntries, err := legacyCold.Retrieve(MemoryQuery{Metadata: map[string]any{"session_id": sessionID}})
	if err != nil {
		t.Fatalf("retrieve legacy entries: %v", err)
	}
	legacyArchives, err := legacyCold.ListArchives(nil)
	if err != nil {
		t.Fatalf("list legacy archives: %v", err)
	}
	stats, err := legacyCold.BackfillLedger(LedgerBackfillOptions{})
	if err != nil {
		t.Fatalf("backfill legacy to ledger: %v", err)
	}
	if stats.EventsWritten == 0 || stats.CheckpointsWritten == 0 {
		t.Fatalf("expected backfill to write events and checkpoints, got %+v", stats)
	}
	repeatStats, err := legacyCold.BackfillLedger(LedgerBackfillOptions{})
	if err != nil {
		t.Fatalf("repeat backfill: %v", err)
	}
	if repeatStats.MonthsCompleted != 0 {
		t.Fatalf("expected checkpoint to skip completed month, got %+v", repeatStats)
	}
	ledgerCold := NewColdMemoryWithConfig(baseDir, ColdMemoryConfig{LedgerReadEnabled: true})
	ledgerEntries, err := ledgerCold.Retrieve(MemoryQuery{Metadata: map[string]any{"session_id": sessionID}})
	if err != nil {
		t.Fatalf("retrieve ledger entries: %v", err)
	}
	if len(ledgerEntries) != len(legacyEntries) {
		t.Fatalf("unexpected ledger entry count: got %d want %d", len(ledgerEntries), len(legacyEntries))
	}
	for index := range legacyEntries {
		if legacyEntries[index].ID != ledgerEntries[index].ID || legacyEntries[index].Content != ledgerEntries[index].Content {
			t.Fatalf("entry mismatch at %d: legacy=%+v ledger=%+v", index, legacyEntries[index], ledgerEntries[index])
		}
	}
	ledgerArchives, err := ledgerCold.ListArchives(nil)
	if err != nil {
		t.Fatalf("list ledger archives: %v", err)
	}
	if len(ledgerArchives) != len(legacyArchives) {
		t.Fatalf("unexpected ledger archive count: got %d want %d", len(ledgerArchives), len(legacyArchives))
	}
	if ledgerArchives[0].SessionID != legacyArchives[0].SessionID || len(ledgerArchives[0].Messages) != len(legacyArchives[0].Messages) {
		t.Fatalf("archive mismatch: legacy=%+v ledger=%+v", legacyArchives[0], ledgerArchives[0])
	}
}

func TestColdMemoryShadowCompareMismatchMetrics(t *testing.T) {
	baseDir := t.TempDir()
	manager := NewMemoryManager(MemoryConfig{
		WarmCapacity:        8,
		WarmPath:            filepath.Join(baseDir, "warm.json"),
		ColdBaseDir:         filepath.Join(baseDir, "cold"),
		LedgerDualWrite:     true,
		LedgerShadowCompare: true,
	})
	if err := manager.AppendLedgerTurn("session-shadow-mismatch", "trace-shadow", 0, []llm.Message{{Role: llm.RoleUser, Text: "only ledger copy"}}); err != nil {
		t.Fatalf("append ledger-only turn: %v", err)
	}
	entries, err := manager.cold.Retrieve(MemoryQuery{Metadata: map[string]any{"session_id": "session-shadow-mismatch"}})
	if err != nil {
		t.Fatalf("retrieve with shadow compare: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected legacy primary read to stay empty, got %+v", entries)
	}
	if manager.Metrics().LedgerShadowMismatchTotal == 0 {
		t.Fatal("expected shadow mismatch metric to increase")
	}
}
