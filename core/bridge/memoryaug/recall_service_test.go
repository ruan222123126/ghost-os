package memoryaug

import (
	"context"
	"testing"

	"ghost-os/bridge/memorystore"
)

func TestRecallServicePrefersExplicitAndUpdatesLastUsed(t *testing.T) {
	store := newTestStore(t)
	settings := newTestSettings()
	recall := NewRecallService(settings, store)

	mustCreateExplicit(t, store, "user://language", "Reply in Chinese by default.", nil)
	if _, err := store.CreateLearned(context.Background(), memorystore.LearnedMemoryInput{
		ScopeType:  memorystore.ScopeTypeUser,
		ScopeID:    memorystore.DefaultUserScopeID,
		MemoryType: memorystore.MemoryTypePreference,
		Summary:    "reply in Chinese",
		Content:    "The user prefers Chinese replies by default.",
		Confidence: 0.9,
	}, nil); err != nil {
		t.Fatalf("create learned: %v", err)
	}

	items, err := recall.Recall(context.Background(), RecallInput{
		SessionID: "session-1",
		Query:     "reply in Chinese",
	})
	if err != nil {
		t.Fatalf("recall: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected duplicate learned memory to be suppressed, got %+v", items)
	}
	if items[0].Entry.SourceKind != memorystore.SourceKindExplicit {
		t.Fatalf("expected explicit memory first, got %+v", items[0])
	}
	record, err := store.Read(context.Background(), "user://language")
	if err != nil {
		t.Fatalf("read explicit: %v", err)
	}
	if record.LastUsedAt.IsZero() {
		t.Fatalf("expected explicit last_used_at to update")
	}
}

func TestRecallServiceKeepsSessionScopeIsolatedAndUserScopeShared(t *testing.T) {
	store := newTestStore(t)
	settings := newTestSettings()
	recall := NewRecallService(settings, store)

	if _, err := store.CreateLearned(context.Background(), memorystore.LearnedMemoryInput{
		ScopeType:  memorystore.ScopeTypeSession,
		ScopeID:    "session-a",
		MemoryType: memorystore.MemoryTypeWorkflow,
		Summary:    "current phase",
		Content:    "The current session is focused on memory refactoring.",
		Confidence: 0.88,
	}, nil); err != nil {
		t.Fatalf("create session learned: %v", err)
	}
	if _, err := store.CreateLearned(context.Background(), memorystore.LearnedMemoryInput{
		ScopeType:  memorystore.ScopeTypeUser,
		ScopeID:    memorystore.DefaultUserScopeID,
		MemoryType: memorystore.MemoryTypePreference,
		Summary:    "reply in Chinese",
		Content:    "The user prefers Chinese replies.",
		Confidence: 0.95,
	}, nil); err != nil {
		t.Fatalf("create user learned: %v", err)
	}

	items, err := recall.Recall(context.Background(), RecallInput{
		SessionID: "session-b",
		Query:     "Chinese replies and memory refactoring",
	})
	if err != nil {
		t.Fatalf("recall: %v", err)
	}
	if len(items) != 1 || items[0].Entry.ScopeType != memorystore.ScopeTypeUser {
		t.Fatalf("expected only user-scoped memory across sessions, got %+v", items)
	}

	items, err = recall.Recall(context.Background(), RecallInput{
		SessionID: "session-a",
		Query:     "Chinese replies and memory refactoring",
	})
	if err != nil {
		t.Fatalf("recall same session: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected both session and user memories, got %+v", items)
	}
	if items[0].Entry.ScopeType != memorystore.ScopeTypeSession {
		t.Fatalf("expected session memory to rank first, got %+v", items)
	}
}

func TestRecallServiceHonorsBudgetWithStableOrdering(t *testing.T) {
	store := newTestStore(t)
	settings := newTestSettings()
	settings.MaxRecallItems = 2
	recall := NewRecallService(settings, store)

	candidates := []memorystore.LearnedMemoryInput{
		{ScopeType: "user", ScopeID: memorystore.DefaultUserScopeID, MemoryType: "fact", Summary: "alpha", Content: "alpha project constraint", Confidence: 0.8},
		{ScopeType: "user", ScopeID: memorystore.DefaultUserScopeID, MemoryType: "fact", Summary: "beta", Content: "beta project constraint", Confidence: 0.9},
		{ScopeType: "user", ScopeID: memorystore.DefaultUserScopeID, MemoryType: "fact", Summary: "gamma", Content: "gamma project constraint", Confidence: 0.85},
	}
	for _, candidate := range candidates {
		if _, err := store.CreateLearned(context.Background(), candidate, nil); err != nil {
			t.Fatalf("create learned: %v", err)
		}
	}

	items, err := recall.Recall(context.Background(), RecallInput{
		SessionID: "session-1",
		Query:     "project constraint",
	})
	if err != nil {
		t.Fatalf("recall: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected recall budget of 2, got %+v", items)
	}
	if items[0].Entry.Summary != "beta" || items[1].Entry.Summary != "gamma" {
		t.Fatalf("expected stable confidence ordering under truncation, got %+v", items)
	}
}

func TestRecallServiceDirectSlotLookupRanksBeforeFreeText(t *testing.T) {
	store := newTestStore(t)
	settings := newTestSettings()
	recall := NewRecallService(settings, store)

	if _, err := store.CreateLearned(context.Background(), memorystore.LearnedMemoryInput{
		ScopeType:  memorystore.ScopeTypeSession,
		ScopeID:    "session-1",
		MemoryType: memorystore.MemoryTypeWorkflow,
		MemoryKey:  "test_command",
		Summary:    "test command",
		Content:    "Use `pnpm --dir apps/web test` as the test command.",
		Metadata:   buildSlotMetadata("pnpm --dir apps/web test", "seed"),
		Confidence: 0.82,
	}, nil); err != nil {
		t.Fatalf("create slot learned: %v", err)
	}
	if _, err := store.CreateLearned(context.Background(), memorystore.LearnedMemoryInput{
		ScopeType:  memorystore.ScopeTypeSession,
		ScopeID:    "session-1",
		MemoryType: memorystore.MemoryTypeFact,
		Summary:    "test setup",
		Content:    "The test setup uses Vitest and contract fixtures.",
		Confidence: 0.99,
	}, nil); err != nil {
		t.Fatalf("create free text learned: %v", err)
	}

	items, err := recall.Recall(context.Background(), RecallInput{
		SessionID: "session-1",
		Query:     "Which test command should I run?",
	})
	if err != nil {
		t.Fatalf("recall: %v", err)
	}
	if len(items) == 0 {
		t.Fatal("expected recalled items")
	}
	if items[0].Entry.MemoryKey != "test_command" || !items[0].SlotMatch {
		t.Fatalf("expected slot recall to rank first, got %+v", items)
	}
}
