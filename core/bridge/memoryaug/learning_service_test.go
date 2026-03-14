package memoryaug

import (
	"context"
	"testing"

	"ghost-os/bridge/memorystore"
)

func TestLearningServiceSkipsLowSignalTurnWithoutExtractorCall(t *testing.T) {
	store := newTestStore(t)
	extractor := &scriptedExtractor{}
	service := NewLearningService(newTestSettings(), store, extractor)

	err := service.LearnFromTurn(context.Background(), LearnFromTurnInput{
		SessionID: "session-1",
		Messages: []TurnMessage{
			{Role: "user", Text: "hello"},
			{Role: "assistant", Text: "ok"},
		},
	})
	if err != nil {
		t.Fatalf("learn from turn: %v", err)
	}
	if extractor.calls != 0 {
		t.Fatalf("extractor should not be called, got %d", extractor.calls)
	}
	items := mustListLearned(t, store, memorystore.LearnedListFilter{
		Statuses: []string{memorystore.MemoryStatusActive},
		Limit:    10,
	})
	if len(items) != 0 {
		t.Fatalf("expected no learned memories, got %+v", items)
	}
}

func TestLearningServiceDropsLowConfidenceCandidates(t *testing.T) {
	store := newTestStore(t)
	extractor := &scriptedExtractor{
		outputs: []ExtractOutput{{
			RawJSON: `{"items":[{"memory_type":"preference"}]}`,
			Items: []Candidate{{
				MemoryType: "preference",
				Summary:    "reply in Chinese",
				Content:    "The user prefers Chinese replies.",
				ScopeType:  "user",
				Confidence: 0.4,
			}},
		}},
	}
	service := NewLearningService(newTestSettings(), store, extractor)

	err := service.LearnFromTurn(context.Background(), LearnFromTurnInput{
		SessionID: "session-1",
		Messages:  []TurnMessage{{Role: "user", Text: "Please reply in Chinese by default."}},
	})
	if err != nil {
		t.Fatalf("learn from turn: %v", err)
	}
	items := mustListLearned(t, store, memorystore.LearnedListFilter{
		Statuses: []string{memorystore.MemoryStatusActive},
		Limit:    10,
	})
	if len(items) != 0 {
		t.Fatalf("expected no learned memories, got %+v", items)
	}
}

func TestLearningServiceDedupesRepeatedPreference(t *testing.T) {
	store := newTestStore(t)
	extractor := &scriptedExtractor{
		outputs: []ExtractOutput{
			{Items: []Candidate{{
				MemoryType: "preference",
				Summary:    "reply in Chinese",
				Content:    "The user prefers Chinese replies by default.",
				ScopeType:  "user",
				Confidence: 0.92,
			}}},
			{Items: []Candidate{{
				MemoryType: "preference",
				Summary:    "reply in Chinese",
				Content:    "The user prefers Chinese replies by default.",
				ScopeType:  "user",
				Confidence: 0.95,
			}}},
		},
	}
	service := NewLearningService(newTestSettings(), store, extractor)

	for range 2 {
		err := service.LearnFromTurn(context.Background(), LearnFromTurnInput{
			SessionID: "session-1",
			Messages:  []TurnMessage{{Role: "user", Text: "Please reply in Chinese by default."}},
		})
		if err != nil {
			t.Fatalf("learn from turn: %v", err)
		}
	}

	items := mustListLearned(t, store, memorystore.LearnedListFilter{
		ScopeType: memorystore.ScopeTypeUser,
		ScopeID:   memorystore.DefaultUserScopeID,
		Statuses:  []string{memorystore.MemoryStatusActive},
		Limit:     10,
	})
	if len(items) != 1 {
		t.Fatalf("expected one active learned memory, got %+v", items)
	}
	if items[0].Confidence != 0.95 {
		t.Fatalf("expected refreshed confidence 0.95, got %.2f", items[0].Confidence)
	}
}

func TestLearningServiceRefreshesSameMemoryKey(t *testing.T) {
	store := newTestStore(t)
	extractor := &scriptedExtractor{
		outputs: []ExtractOutput{
			{Items: []Candidate{{
				MemoryType: "preference",
				MemoryKey:  "reply_language",
				Summary:    "reply language",
				Content:    "Reply in Chinese by default.",
				ScopeType:  "user",
				Confidence: 0.9,
			}}},
			{Items: []Candidate{{
				MemoryType: "preference",
				MemoryKey:  "reply_language",
				Summary:    "reply language",
				Content:    "Reply in Chinese by default.",
				ScopeType:  "user",
				Confidence: 0.96,
			}}},
		},
	}
	service := NewLearningService(newTestSettings(), store, extractor)

	for range 2 {
		err := service.LearnFromTurn(context.Background(), LearnFromTurnInput{
			SessionID: "session-1",
			Messages:  []TurnMessage{{Role: "user", Text: "Please reply in Chinese by default."}},
		})
		if err != nil {
			t.Fatalf("learn from turn: %v", err)
		}
	}

	items := mustListLearned(t, store, memorystore.LearnedListFilter{
		ScopeType: memorystore.ScopeTypeUser,
		ScopeID:   memorystore.DefaultUserScopeID,
		Statuses:  []string{memorystore.MemoryStatusActive},
		Limit:     10,
	})
	if len(items) != 1 {
		t.Fatalf("expected one active memory, got %+v", items)
	}
	if items[0].MemoryKey != "reply_language" {
		t.Fatalf("expected stable memory key, got %+v", items[0])
	}
	if items[0].Confidence != 0.96 {
		t.Fatalf("expected refreshed confidence 0.96, got %.2f", items[0].Confidence)
	}
}

func TestLearningServiceSupersedesByMemoryKey(t *testing.T) {
	store := newTestStore(t)
	extractor := &scriptedExtractor{
		outputs: []ExtractOutput{
			{Items: []Candidate{{
				MemoryType: "workflow",
				MemoryKey:  "preferred_response_style",
				Summary:    "preferred response style",
				Content:    "Keep responses concise.",
				ScopeType:  "user",
				Confidence: 0.9,
			}}},
			{Items: []Candidate{{
				MemoryType: "workflow",
				MemoryKey:  "preferred_response_style",
				Summary:    "preferred response style",
				Content:    "Keep responses very concise.",
				ScopeType:  "user",
				Confidence: 0.95,
			}}},
		},
	}
	service := NewLearningService(newTestSettings(), store, extractor)

	err := service.LearnFromTurn(context.Background(), LearnFromTurnInput{
		SessionID: "session-1",
		Messages:  []TurnMessage{{Role: "user", Text: "Keep responses concise."}},
	})
	if err != nil {
		t.Fatalf("first learn: %v", err)
	}
	err = service.LearnFromTurn(context.Background(), LearnFromTurnInput{
		SessionID: "session-2",
		Messages:  []TurnMessage{{Role: "user", Text: "Keep responses very concise in future responses."}},
	})
	if err != nil {
		t.Fatalf("second learn: %v", err)
	}

	all := mustListLearned(t, store, memorystore.LearnedListFilter{
		Statuses: []string{memorystore.MemoryStatusActive, memorystore.MemoryStatusSuperseded},
		Limit:    10,
	})
	if len(all) != 2 {
		t.Fatalf("expected one refresh lineage with superseded entry, got %+v", all)
	}
	if !containsStatus(all, memorystore.MemoryStatusSuperseded) {
		t.Fatalf("expected older keyed memory to be superseded, got %+v", all)
	}
}

func TestLearningServiceSupersedesOlderLearnedMemory(t *testing.T) {
	store := newTestStore(t)
	extractor := &scriptedExtractor{
		outputs: []ExtractOutput{{
			Items: []Candidate{{
				MemoryType: "workflow",
				Summary:    "preferred response style",
				Content:    "The user wants concise responses.",
				ScopeType:  "user",
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
		t.Fatalf("first learn: %v", err)
	}
	first := mustListLearned(t, store, memorystore.LearnedListFilter{
		Statuses: []string{memorystore.MemoryStatusActive},
		Limit:    10,
	})
	if len(first) != 1 {
		t.Fatalf("expected first learned memory, got %+v", first)
	}

	extractor.outputs = []ExtractOutput{{
		Items: []Candidate{{
			MemoryType:   "workflow",
			Summary:      "preferred response style",
			Content:      "The user wants very concise responses.",
			ScopeType:    "user",
			Confidence:   0.96,
			SupersedesID: []string{first[0].ID},
		}},
	}}
	err = service.LearnFromTurn(context.Background(), LearnFromTurnInput{
		SessionID: "session-2",
		Messages:  []TurnMessage{{Role: "user", Text: "Be very concise in future responses."}},
	})
	if err != nil {
		t.Fatalf("second learn: %v", err)
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

func containsStatus(items []memorystore.MemoryEntry, status string) bool {
	for _, item := range items {
		if item.Status == status {
			return true
		}
	}
	return false
}
