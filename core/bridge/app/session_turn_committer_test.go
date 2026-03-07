package app

import (
	"path/filepath"
	"testing"

	"ghost-os/bridge/llm"
	"ghost-os/bridge/memory"
	"ghost-os/bridge/session"
)

func TestSessionTurnCommitterPersistSessionMessagesDualWritesLedger(t *testing.T) {
	baseDir := t.TempDir()
	sessionStore, err := session.NewStore(filepath.Join(baseDir, "sessions"))
	if err != nil {
		t.Fatalf("new session store: %v", err)
	}
	manager := memory.NewMemoryManager(memory.MemoryConfig{
		WarmCapacity:    8,
		WarmPath:        filepath.Join(baseDir, "warm.json"),
		ColdBaseDir:     filepath.Join(baseDir, "cold"),
		LedgerDualWrite: true,
	})
	committer := newSessionTurnCommitter(sessionStore, manager, "trace-turn-commit")
	sess := session.NewSession("system")
	sess.ID = "session-turn-commit"
	if err := sessionStore.Save(sess); err != nil {
		t.Fatalf("save seed session: %v", err)
	}
	messages := []llm.Message{{Role: llm.RoleUser, Text: "hello ledger"}, {Role: llm.RoleAssistant, Text: "ack"}}
	if err := committer.PersistSessionMessages(sess, messages); err != nil {
		t.Fatalf("persist session messages: %v", err)
	}
	replay, err := manager.ReplayLedgerSession(sess.ID)
	if err != nil {
		t.Fatalf("replay ledger session: %v", err)
	}
	if len(replay.Messages) != 2 {
		t.Fatalf("unexpected ledger message count: got %d want 2", len(replay.Messages))
	}
	if replay.Messages[0].Text != "hello ledger" || replay.Messages[1].Text != "ack" {
		t.Fatalf("unexpected ledger messages: %+v", replay.Messages)
	}
}
