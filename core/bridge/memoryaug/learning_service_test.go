package memoryaug

import (
	"context"
	"testing"

	"ghost-os/bridge/memorystore"
)

func TestLearningServiceWritesEventMemoryToPrimaryEvent(t *testing.T) {
	store := newTestStore(t)
	service := NewLearningService(
		newTestSettings(),
		store,
		&scriptedExtractor{},
		&scriptedEventExtractor{outputs: []EventExtractOutput{{
			Items: []EventMemoryCandidate{{
				MemoryType: memorystore.MemoryTypeWorkflow,
				Summary:    "run Android tests after changing runtime flags",
				Content:    "run Android tests after changing runtime flags",
				Confidence: 0.91,
			}},
		}}},
	)

	primary := mustCreateEventNode(t, store, "session-1", "Android runtime settings")
	err := service.LearnFromTurn(context.Background(), LearnFromTurnInput{
		SessionID:      "session-1",
		PrimaryEventID: primary.ID,
		ActiveEventIDs: []string{primary.ID},
		AllowWrite:     true,
		Messages: []TurnMessage{{
			Role: "user",
			Text: "Keep in mind that we should run Android tests after changing runtime flags.",
		}},
	})
	if err != nil {
		t.Fatalf("learn: %v", err)
	}

	items, _, err := store.ListEventMemories(context.Background(), memorystore.EventMemoryListFilter{
		EventID:  primary.ID,
		Statuses: []string{memorystore.MemoryStatusActive},
	})
	if err != nil {
		t.Fatalf("list event memories: %v", err)
	}
	if len(items) != 1 || items[0].EventID != primary.ID {
		t.Fatalf("expected one event memory on primary event, got %+v", items)
	}
}

func TestLearningServiceKeepsGlobalPreferencesOutOfEvents(t *testing.T) {
	store := newTestStore(t)
	service := NewLearningService(
		newTestSettings(),
		store,
		&scriptedExtractor{outputs: []ExtractOutput{{
			Items: []Candidate{{
				MemoryType: memorystore.MemoryTypePreference,
				MemoryKey:  "reply_language",
				Value:      "zh-CN",
				Summary:    "reply language",
				Content:    "reply in Chinese by default",
				Confidence: 0.94,
			}},
		}}},
		&scriptedEventExtractor{},
	)

	primary := mustCreateEventNode(t, store, "session-1", "Android runtime settings")
	err := service.LearnFromTurn(context.Background(), LearnFromTurnInput{
		SessionID:      "session-1",
		PrimaryEventID: primary.ID,
		ActiveEventIDs: []string{primary.ID},
		AllowWrite:     true,
		Messages: []TurnMessage{{
			Role: "user",
			Text: "Please reply in Chinese by default.",
		}},
	})
	if err != nil {
		t.Fatalf("learn: %v", err)
	}

	global, err := store.ListGlobalPreferences(context.Background(), []string{"reply_language"})
	if err != nil {
		t.Fatalf("list global preferences: %v", err)
	}
	if len(global) != 1 || global[0].MemoryKey != "reply_language" {
		t.Fatalf("expected global reply_language preference, got %+v", global)
	}
	items, _, err := store.ListEventMemories(context.Background(), memorystore.EventMemoryListFilter{
		EventID:  primary.ID,
		Statuses: []string{memorystore.MemoryStatusActive},
	})
	if err != nil {
		t.Fatalf("list event memories: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("global preference should not be written into event memories: %+v", items)
	}
}

func TestLearningServiceSkipsWhenPlannerDisablesWrite(t *testing.T) {
	store := newTestStore(t)
	service := NewLearningService(
		newTestSettings(),
		store,
		&scriptedExtractor{},
		&scriptedEventExtractor{outputs: []EventExtractOutput{{
			Items: []EventMemoryCandidate{{
				MemoryType: memorystore.MemoryTypeFact,
				Summary:    "should be skipped",
				Content:    "should be skipped",
				Confidence: 0.9,
			}},
		}}},
	)

	primary := mustCreateEventNode(t, store, "session-1", "Android runtime settings")
	err := service.LearnFromTurn(context.Background(), LearnFromTurnInput{
		SessionID:      "session-1",
		PrimaryEventID: primary.ID,
		ActiveEventIDs: []string{primary.ID},
		AllowWrite:     false,
		Messages: []TurnMessage{{
			Role: "user",
			Text: "Remember the current runtime flag wiring.",
		}},
	})
	if err != nil {
		t.Fatalf("learn: %v", err)
	}

	items, _, err := store.ListEventMemories(context.Background(), memorystore.EventMemoryListFilter{
		EventID:  primary.ID,
		Statuses: []string{memorystore.MemoryStatusActive},
	})
	if err != nil {
		t.Fatalf("list event memories: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected no event memories when learning is disabled, got %+v", items)
	}
}
