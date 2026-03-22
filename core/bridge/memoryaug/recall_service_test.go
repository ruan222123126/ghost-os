package memoryaug

import (
	"context"
	"strings"
	"testing"

	"ghost-os/bridge/memorystore"
)

func TestRecallServiceOnlyReturnsActiveSubgraphAndGlobalPreferences(t *testing.T) {
	store := newTestStore(t)
	recall := NewRecallService(newTestSettings(), store)

	primary := mustCreateEventNode(t, store, "session-1", "Android runtime settings")
	adjacent := mustCreateEventNode(t, store, "session-1", "CLI envelope sync")
	other := mustCreateEventNode(t, store, "session-1", "Web config parser")
	mustCreateEventMemory(t, store, primary.ID, memorystore.MemoryTypeWorkflow, "run Android tests after runtime flag changes")
	mustCreateEventMemory(t, store, adjacent.ID, memorystore.MemoryTypeFact, "CLI schema depends on config_runtime.json")
	mustCreateEventMemory(t, store, other.ID, memorystore.MemoryTypeFact, "this should never be recalled")
	mustCreateLearnedPreference(t, store, "reply_language", "zh-CN")

	output, err := recall.Recall(context.Background(), RecallInput{
		SessionID:      "session-1",
		PrimaryEventID: primary.ID,
		ActiveEventIDs: []string{primary.ID, adjacent.ID},
		FocusText:      "continue runtime config work",
		RecallPlan: RecallPlan{
			EventIDs:           []string{primary.ID, adjacent.ID},
			IncludeNodeSummary: true,
			IncludeWorkflow:    true,
			IncludePreference:  true,
			IncludeFact:        true,
			AllowLearning:      true,
		},
	})
	if err != nil {
		t.Fatalf("recall: %v", err)
	}
	if output.PrimaryEvent == nil || output.PrimaryEvent.Event.ID != primary.ID {
		t.Fatalf("unexpected primary event: %+v", output.PrimaryEvent)
	}
	if len(output.AdjacentEvents) != 1 || output.AdjacentEvents[0].Event.ID != adjacent.ID {
		t.Fatalf("unexpected adjacent events: %+v", output.AdjacentEvents)
	}
	if containsEventMemory(output, other.ID) {
		t.Fatalf("non-active event memory should not be injected: %+v", output)
	}
	if len(output.GlobalPreferences) != 1 || output.GlobalPreferences[0].MemoryKey != "reply_language" {
		t.Fatalf("expected global preferences, got %+v", output.GlobalPreferences)
	}
	if strings.Contains(output.PromptBlock, "this should never be recalled") {
		t.Fatalf("prompt block leaked non-active memory: %q", output.PromptBlock)
	}
}

func TestRecallServiceHonorsMemoryTypeFilters(t *testing.T) {
	store := newTestStore(t)
	recall := NewRecallService(newTestSettings(), store)

	primary := mustCreateEventNode(t, store, "session-1", "Android runtime settings")
	mustCreateEventMemory(t, store, primary.ID, memorystore.MemoryTypeWorkflow, "run Android tests after runtime flag changes")
	mustCreateEventMemory(t, store, primary.ID, memorystore.MemoryTypeFact, "GraphQL toggle defaults to off in config schema")

	output, err := recall.Recall(context.Background(), RecallInput{
		SessionID:      "session-1",
		PrimaryEventID: primary.ID,
		ActiveEventIDs: []string{primary.ID},
		FocusText:      "continue runtime config work",
		RecallPlan: RecallPlan{
			EventIDs:        []string{primary.ID},
			IncludeWorkflow: true,
			IncludeFact:     false,
			AllowLearning:   true,
		},
	})
	if err != nil {
		t.Fatalf("recall: %v", err)
	}
	if len(output.PrimaryMemories) != 1 || output.PrimaryMemories[0].Entry.MemoryType != memorystore.MemoryTypeWorkflow {
		t.Fatalf("expected workflow-only recall, got %+v", output.PrimaryMemories)
	}
}

func mustCreateEventNode(t *testing.T, store *memorystore.Store, sessionID string, title string) memorystore.EventNode {
	t.Helper()
	node, err := store.CreateEventNode(context.Background(), memorystore.EventNodeInput{
		SessionID: sessionID,
		Title:     title,
		Summary:   title + " summary",
		Status:    memorystore.EventStatusActive,
	})
	if err != nil {
		t.Fatalf("create event node: %v", err)
	}
	return node
}

func mustCreateEventMemory(t *testing.T, store *memorystore.Store, eventID string, memoryType string, summary string) {
	t.Helper()
	if _, err := store.CreateEventMemory(context.Background(), memorystore.EventMemoryInput{
		EventID:    eventID,
		MemoryType: memoryType,
		Summary:    summary,
		Content:    summary,
		Confidence: 0.9,
	}, nil); err != nil {
		t.Fatalf("create event memory: %v", err)
	}
}

func mustCreateLearnedPreference(t *testing.T, store *memorystore.Store, memoryKey string, value string) {
	t.Helper()
	if _, err := store.CreateLearned(context.Background(), memorystore.LearnedMemoryInput{
		ScopeType:  memorystore.ScopeTypeUser,
		ScopeID:    memorystore.DefaultUserScopeID,
		MemoryType: memorystore.MemoryTypePreference,
		MemoryKey:  memoryKey,
		Summary:    memoryKey,
		Content:    value,
		Metadata:   buildSlotMetadata(value, "test"),
		Confidence: 0.9,
	}, nil); err != nil {
		t.Fatalf("create learned preference: %v", err)
	}
}

func containsEventMemory(output RecallOutput, eventID string) bool {
	for _, item := range output.PrimaryMemories {
		if item.Event.ID == eventID {
			return true
		}
	}
	for _, item := range output.AdjacentMemories {
		if item.Event.ID == eventID {
			return true
		}
	}
	return false
}
