package memoryaug

import (
	"context"
	"testing"

	"ghost-os/bridge/memorystore"
)

func TestLearningServiceSupersedesOlderLearnedMemory(t *testing.T) {
	store := newTestStore(t)
	if _, err := store.CreateLearned(context.Background(), memorystore.LearnedMemoryInput{
		ScopeType:  memorystore.ScopeTypeUser,
		ScopeID:    memorystore.DefaultUserScopeID,
		MemoryType: memorystore.MemoryTypePreference,
		Summary:    "preferred response style",
		Content:    "The user wants concise responses.",
		Confidence: 0.9,
	}, nil); err != nil {
		t.Fatalf("seed legacy learned memory: %v", err)
	}
	extractor := &scriptedExtractor{
		outputs: []ExtractOutput{{
			Items: []Candidate{{
				MemoryType: "preference",
				MemoryKey:  "response_style",
				Value:      "concise",
				Summary:    "response style",
				Content:    "Keep responses concise.",
				Confidence: 0.96,
			}},
		}},
	}
	service := NewLearningService(newTestSettings(), store, extractor)

	err := service.LearnFromTurn(context.Background(), LearnFromTurnInput{
		SessionID: "session-1",
		Messages:  []TurnMessage{{Role: "user", Text: "Keep responses concise."}},
	})
	if err != nil {
		t.Fatalf("first learn: %v", err)
	}
	active := mustListLearned(t, store, memorystore.LearnedListFilter{
		Statuses: []string{memorystore.MemoryStatusActive},
		Limit:    10,
	})
	if len(active) != 1 {
		t.Fatalf("expected one active learned memory, got %+v", active)
	}
	all := mustListLearned(t, store, memorystore.LearnedListFilter{
		Statuses: []string{memorystore.MemoryStatusActive, memorystore.MemoryStatusSuperseded},
		Limit:    10,
	})
	if len(all) != 2 {
		t.Fatalf("expected two learned memories total, got %+v", all)
	}
	if !containsStatus(all, memorystore.MemoryStatusSuperseded) {
		t.Fatalf("expected older memory to be superseded, got %+v", all)
	}
}

func TestLearningServiceSkipsUnknownSlotKey(t *testing.T) {
	store := newTestStore(t)
	extractor := &scriptedExtractor{
		outputs: []ExtractOutput{{
			Items: []Candidate{{
				MemoryType: "preference",
				MemoryKey:  "preferred_response_style",
				Value:      "concise",
				Summary:    "preferred response style",
				Content:    "Keep responses concise.",
				Confidence: 0.9,
			}},
		}},
	}
	service := NewLearningService(newTestSettings(), store, extractor)

	err := service.LearnFromTurn(context.Background(), LearnFromTurnInput{
		SessionID: "session-1",
		Messages:  []TurnMessage{{Role: "user", Text: "Keep responses concise."}},
	})
	if err != nil {
		t.Fatalf("learn from turn: %v", err)
	}
	items := mustListLearned(t, store, memorystore.LearnedListFilter{
		Statuses: []string{memorystore.MemoryStatusActive},
		Limit:    10,
	})
	if len(items) != 0 {
		t.Fatalf("expected unknown slot to be skipped, got %+v", items)
	}
}

func containsStatus(items []memorystore.MemoryEntry, status string) bool {
	for _, item := range items {
		if item.Status == status {
			return true
		}
	}
	return false
}
