package memoryaug

import (
	"strings"
	"testing"

	"ghost-os/bridge/memorystore"
)

func TestFormatPromptBlockUsesEventSections(t *testing.T) {
	block := FormatPromptBlock(RecallOutput{
		PrimaryEvent: &RecallEventHit{
			Event: memorystore.EventNode{
				ID:      "event-1",
				Title:   "Android runtime settings",
				Summary: "Fix the runtime settings screen and GraphQL toggle wiring.",
			},
		},
		AdjacentEvents: []RecallEventHit{{
			Event: memorystore.EventNode{ID: "event-2", Title: "CLI envelope sync"},
		}},
		PrimaryMemories: []RecallMemoryHit{{
			Event: memorystore.EventNode{ID: "event-1", Title: "Android runtime settings"},
			Entry: memorystore.EventMemory{
				ID:         "mem-1",
				EventID:    "event-1",
				MemoryType: memorystore.MemoryTypeWorkflow,
				Summary:    "Run Android config tests after changing runtime flags.",
			},
		}},
		GlobalPreferences: []memorystore.MemoryEntry{{
			ID:         "pref-1",
			SourceKind: memorystore.SourceKindLearned,
			MemoryKey:  "reply_language",
			Summary:    "reply language",
			Metadata:   buildSlotMetadata("zh-CN", "test"),
		}},
	})

	if !strings.Contains(block, "Active event:\n- primary: Android runtime settings") {
		t.Fatalf("expected active event block, got %q", block)
	}
	if !strings.Contains(block, "Relevant event memory:\n- [primary/workflow] Android runtime settings: Run Android config tests after changing runtime flags.") {
		t.Fatalf("expected event memory block, got %q", block)
	}
	if !strings.Contains(block, "Global preferences:\n- reply_language=zh-CN") {
		t.Fatalf("expected global preference block, got %q", block)
	}
}

func TestFormatPromptBlockTrimsOverflowSummary(t *testing.T) {
	block := FormatPromptBlock(RecallOutput{
		PrimaryEvent: &RecallEventHit{
			Event: memorystore.EventNode{
				ID:      "event-1",
				Title:   "Android runtime settings",
				Summary: strings.Repeat("summary ", 30),
			},
		},
		PrimaryMemories: []RecallMemoryHit{{
			Event: memorystore.EventNode{ID: "event-1", Title: "Android runtime settings"},
			Entry: memorystore.EventMemory{
				ID:         "mem-1",
				EventID:    "event-1",
				MemoryType: memorystore.MemoryTypeFact,
				Summary:    strings.Repeat("constraint ", 30),
			},
		}},
	})

	if !strings.Contains(block, "...") {
		t.Fatalf("expected trimmed summary, got %q", block)
	}
}
