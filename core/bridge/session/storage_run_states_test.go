package session

import (
	"testing"
	"time"
)

func TestStoreListRunStatesReadsLightweightPersistedState(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore returned error: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	first := NewSession("")
	first.ID = "session-1"
	first.Title = "First"
	first.SetLastRunState(RunStatusRunning, "trace-1", time.Now().UTC().Add(-time.Minute))
	if err := store.Save(first); err != nil {
		t.Fatalf("save first session: %v", err)
	}

	second := NewSession("")
	second.ID = "session-2"
	second.Title = "Second"
	second.SetLastRunState(RunStatusSuccess, "trace-2", time.Now().UTC())
	if err := store.Save(second); err != nil {
		t.Fatalf("save second session: %v", err)
	}

	recent, err := store.ListRunStates(RunStateQuery{Limit: 1})
	if err != nil {
		t.Fatalf("ListRunStates recent returned error: %v", err)
	}
	if len(recent) != 1 || recent[0].SessionID != "session-2" || recent[0].Status != RunStatusSuccess {
		t.Fatalf("unexpected recent run states: %+v", recent)
	}

	selected, err := store.ListRunStates(RunStateQuery{SessionIDs: []string{"session-1"}})
	if err != nil {
		t.Fatalf("ListRunStates selected returned error: %v", err)
	}
	if len(selected) != 1 || selected[0].TraceID != "trace-1" || selected[0].Title != "First" {
		t.Fatalf("unexpected selected run states: %+v", selected)
	}
}

func TestStoreListRunStatesReturnsIdleForLegacySession(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore returned error: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	sess := NewSession("")
	sess.ID = "legacy-session"
	if err := store.Save(sess); err != nil {
		t.Fatalf("save legacy session: %v", err)
	}

	states, err := store.ListRunStates(RunStateQuery{SessionIDs: []string{sess.ID}})
	if err != nil {
		t.Fatalf("ListRunStates returned error: %v", err)
	}
	if len(states) != 1 || states[0].Status != RunStatusIdle || states[0].TraceID != "" {
		t.Fatalf("unexpected legacy state: %+v", states)
	}
}
