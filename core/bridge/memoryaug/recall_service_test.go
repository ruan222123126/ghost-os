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

func TestRecallServiceSkipsSessionScopeWhenDisabled(t *testing.T) {
	store := newTestStore(t)
	settings := newTestSettings()
	settings.SessionScopeEnabled = false
	recall := NewRecallService(settings, store)

	primary := mustCreateEventNode(t, store, "session-1", "Android runtime settings")
	mustCreateEventMemory(t, store, primary.ID, memorystore.MemoryTypeWorkflow, "run Android tests after runtime flag changes")
	mustCreateLearnedPreference(t, store, "reply_language", "zh-CN")

	output, err := recall.Recall(context.Background(), RecallInput{
		SessionID:      "session-1",
		PrimaryEventID: primary.ID,
		ActiveEventIDs: []string{primary.ID},
		FocusText:      "continue runtime config work",
		RecallPlan: RecallPlan{
			EventIDs:           []string{primary.ID},
			IncludeNodeSummary: true,
			IncludeWorkflow:    true,
			IncludePreference:  true,
			AllowLearning:      true,
		},
	})
	if err != nil {
		t.Fatalf("recall: %v", err)
	}
	if output.PrimaryEvent != nil || len(output.PrimaryMemories) != 0 || len(output.AdjacentMemories) != 0 {
		t.Fatalf("expected session-scoped recall to be disabled, got %+v", output)
	}
	if len(output.GlobalPreferences) != 1 || output.GlobalPreferences[0].MemoryKey != "reply_language" {
		t.Fatalf("expected user-scoped preferences to remain enabled, got %+v", output.GlobalPreferences)
	}
	if strings.Contains(output.PromptBlock, "Active event:") || strings.Contains(output.PromptBlock, "Relevant event memory:") {
		t.Fatalf("prompt block should not contain session-scoped sections when disabled: %q", output.PromptBlock)
	}
}

func TestRecallServiceSkipsUserScopeWhenDisabled(t *testing.T) {
	store := newTestStore(t)
	settings := newTestSettings()
	settings.UserScopeEnabled = false
	recall := NewRecallService(settings, store)

	primary := mustCreateEventNode(t, store, "session-1", "Android runtime settings")
	mustCreateEventMemory(t, store, primary.ID, memorystore.MemoryTypeWorkflow, "run Android tests after runtime flag changes")
	mustCreateLearnedPreference(t, store, "reply_language", "zh-CN")

	output, err := recall.Recall(context.Background(), RecallInput{
		SessionID:      "session-1",
		PrimaryEventID: primary.ID,
		ActiveEventIDs: []string{primary.ID},
		FocusText:      "continue runtime config work",
		RecallPlan: RecallPlan{
			EventIDs:           []string{primary.ID},
			IncludeNodeSummary: true,
			IncludeWorkflow:    true,
			IncludePreference:  true,
			AllowLearning:      true,
		},
	})
	if err != nil {
		t.Fatalf("recall: %v", err)
	}
	if len(output.GlobalPreferences) != 0 {
		t.Fatalf("expected user-scoped preferences to be disabled, got %+v", output.GlobalPreferences)
	}
	if output.PrimaryEvent == nil || len(output.PrimaryMemories) != 1 {
		t.Fatalf("expected session-scoped recall to remain enabled, got %+v", output)
	}
	if strings.Contains(output.PromptBlock, "Global preferences:") {
		t.Fatalf("prompt block should not contain global preferences when disabled: %q", output.PromptBlock)
	}
}

func TestRecallServiceHonorsMaxRecallItems(t *testing.T) {
	store := newTestStore(t)
	settings := newTestSettings()
	settings.MaxRecallItems = 6
	recall := NewRecallService(settings, store)

	primary := mustCreateEventNode(t, store, "session-1", "Android runtime settings")
	for idx := 0; idx < 8; idx++ {
		mustCreateEventMemory(
			t,
			store,
			primary.ID,
			memorystore.MemoryTypeFact,
			"runtime constraint memory "+string(rune('A'+idx)),
		)
	}

	output, err := recall.Recall(context.Background(), RecallInput{
		SessionID:      "session-1",
		PrimaryEventID: primary.ID,
		ActiveEventIDs: []string{primary.ID},
		FocusText:      "runtime constraint",
		RecallPlan: RecallPlan{
			EventIDs:           []string{primary.ID},
			IncludeNodeSummary: true,
			IncludeFact:        true,
			AllowLearning:      true,
		},
	})
	if err != nil {
		t.Fatalf("recall: %v", err)
	}
	if total := countRecalledMemories(output); total != 6 {
		t.Fatalf("expected max_recall_items=6 to cap recalled memories, got %d (%+v)", total, output)
	}
	if strings.Count(output.PromptBlock, "[primary/fact]") != 6 {
		t.Fatalf("prompt block should include all recalled items without extra hard cap, got %q", output.PromptBlock)
	}
}

func TestRecallServiceNormalizesUnknownRecallPlanEventIDs(t *testing.T) {
	store := newTestStore(t)
	recall := NewRecallService(newTestSettings(), store)

	primary := mustCreateEventNode(t, store, "session-1", "搜索当前AI发展情况")

	output, err := recall.Recall(context.Background(), RecallInput{
		SessionID:      "session-1",
		PrimaryEventID: primary.ID,
		ActiveEventIDs: []string{primary.ID},
		FocusText:      "帮我搜索现在ai的情况",
		RecallPlan: RecallPlan{
			EventIDs:           []string{"evt_ai_current_landscape_search"},
			IncludeNodeSummary: true,
			AllowLearning:      true,
		},
	})
	if err != nil {
		t.Fatalf("recall: %v", err)
	}
	if output.PrimaryEvent == nil || output.PrimaryEvent.Event.ID != primary.ID {
		t.Fatalf("expected primary event recall to fall back to the resolved active id, got %+v", output)
	}
	if len(output.AdjacentEvents) != 0 {
		t.Fatalf("expected unknown recall_plan.event_ids to be dropped instead of treated as adjacent events, got %+v", output.AdjacentEvents)
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

func countRecalledMemories(output RecallOutput) int {
	return len(output.PrimaryMemories) + len(output.AdjacentMemories)
}
