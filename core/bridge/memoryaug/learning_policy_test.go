package memoryaug

import (
	"context"
	"testing"

	"ghost-os/bridge/memorystore"
)

func TestLearningServiceSkipsOneOffTaskWithoutExtractorCall(t *testing.T) {
	store := newTestStore(t)
	extractor := &scriptedExtractor{}
	service := NewLearningService(newTestSettings(), store, extractor)

	err := service.LearnFromTurn(context.Background(), LearnFromTurnInput{
		SessionID: "session-1",
		Messages: []TurnMessage{
			{Role: "user", Text: "Please open README.md and summarize it for this turn."},
		},
	})
	if err != nil {
		t.Fatalf("learn from turn: %v", err)
	}
	if extractor.calls != 0 {
		t.Fatalf("extractor should stay idle for one-off tasks, got %d calls", extractor.calls)
	}
}

func TestLearningServiceSkipsAutoFactsWithoutRememberSignal(t *testing.T) {
	store := newTestStore(t)
	extractor := &scriptedExtractor{
		outputs: []ExtractOutput{{
			Items: []Candidate{{
				MemoryType: "fact",
				Summary:    "repository package manager",
				Content:    "This repository uses pnpm.",
				ScopeType:  "user",
				Confidence: 0.93,
			}},
		}},
	}
	service := NewLearningService(newTestSettings(), store, extractor)

	err := service.LearnFromTurn(context.Background(), LearnFromTurnInput{
		SessionID: "session-1",
		Messages:  []TurnMessage{{Role: "user", Text: "Use pnpm for this project."}},
	})
	if err != nil {
		t.Fatalf("learn from turn: %v", err)
	}
	items := mustListLearned(t, store, memorystore.LearnedListFilter{
		Statuses: []string{memorystore.MemoryStatusActive},
		Limit:    10,
	})
	if len(items) != 0 {
		t.Fatalf("expected auto facts to stay out of learned memory, got %+v", items)
	}
}

func TestLearningServiceAllowsFactsWithRememberSignal(t *testing.T) {
	store := newTestStore(t)
	extractor := &scriptedExtractor{
		outputs: []ExtractOutput{{
			Items: []Candidate{{
				MemoryType: "fact",
				MemoryKey:  "repository_package_manager",
				Summary:    "repository package manager",
				Content:    "This repository uses pnpm.",
				ScopeType:  "user",
				Confidence: 0.93,
			}},
		}},
	}
	service := NewLearningService(newTestSettings(), store, extractor)

	err := service.LearnFromTurn(context.Background(), LearnFromTurnInput{
		SessionID: "session-1",
		Messages:  []TurnMessage{{Role: "user", Text: "Remember that this repository uses pnpm."}},
	})
	if err != nil {
		t.Fatalf("learn from turn: %v", err)
	}
	items := mustListLearned(t, store, memorystore.LearnedListFilter{
		Statuses: []string{memorystore.MemoryStatusActive},
		Limit:    10,
	})
	if len(items) != 1 || items[0].MemoryType != memorystore.MemoryTypeFact {
		t.Fatalf("expected remembered fact to be learned, got %+v", items)
	}
}
