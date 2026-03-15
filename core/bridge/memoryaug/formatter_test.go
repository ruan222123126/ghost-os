package memoryaug

import (
	"strings"
	"testing"

	"ghost-os/bridge/memorystore"
)

func TestFormatPromptBlockSeparatesSlotsAndOtherContext(t *testing.T) {
	block := FormatPromptBlock([]RecallItem{
		{
			Entry: memorystore.MemoryEntry{
				ID:         "mem-slot",
				ScopeType:  memorystore.ScopeTypeUser,
				ScopeID:    memorystore.DefaultUserScopeID,
				SourceKind: memorystore.SourceKindLearned,
				MemoryType: memorystore.MemoryTypePreference,
				MemoryKey:  "reply_language",
				Summary:    "reply language",
				Content:    "Reply in Chinese by default.",
				Metadata:   buildSlotMetadata("zh-CN", "test"),
				Confidence: 0.95,
				Status:     memorystore.MemoryStatusActive,
			},
			SlotMatch: true,
		},
		{
			Entry: memorystore.MemoryEntry{
				ID:         "mem-other",
				ScopeType:  memorystore.ScopeTypeSession,
				ScopeID:    "session-1",
				SourceKind: memorystore.SourceKindLearned,
				MemoryType: memorystore.MemoryTypeWorkflow,
				Summary:    "current focus",
				Content:    "Android runtime settings screen is the current focus.",
				Confidence: 0.8,
				Status:     memorystore.MemoryStatusActive,
			},
		},
	})

	if !strings.Contains(block, "Memory slots:\n- reply_language=zh-CN") {
		t.Fatalf("expected slot block, got %q", block)
	}
	if !strings.Contains(block, "Other memory context:\n- [session/workflow] current focus") {
		t.Fatalf("expected other memory block, got %q", block)
	}
	if strings.Contains(block, "content=") {
		t.Fatalf("expected compact prompt block without content, got %q", block)
	}
}

func TestFormatPromptBlockLimitsOtherMemoryContext(t *testing.T) {
	longSummary := "This Android runtime settings workflow summary is intentionally long so the formatter must trim it before injecting the prompt."
	block := FormatPromptBlock([]RecallItem{
		{
			Entry: memorystore.MemoryEntry{
				ID:         "mem-1",
				ScopeType:  memorystore.ScopeTypeSession,
				ScopeID:    "session-1",
				SourceKind: memorystore.SourceKindLearned,
				MemoryType: memorystore.MemoryTypeWorkflow,
				Summary:    longSummary,
				Confidence: 0.9,
				Status:     memorystore.MemoryStatusActive,
			},
		},
		{
			Entry: memorystore.MemoryEntry{
				ID:         "mem-2",
				ScopeType:  memorystore.ScopeTypeUser,
				ScopeID:    memorystore.DefaultUserScopeID,
				SourceKind: memorystore.SourceKindExplicit,
				MemoryType: memorystore.MemoryTypePreference,
				Summary:    "pnpm is the package manager",
				Confidence: 1,
				Status:     memorystore.MemoryStatusActive,
			},
		},
		{
			Entry: memorystore.MemoryEntry{
				ID:         "mem-3",
				ScopeType:  memorystore.ScopeTypeSession,
				ScopeID:    "session-1",
				SourceKind: memorystore.SourceKindLearned,
				MemoryType: memorystore.MemoryTypeWorkflow,
				Summary:    "should be dropped by prompt limit",
				Confidence: 0.7,
				Status:     memorystore.MemoryStatusActive,
			},
		},
	})

	if strings.Count(block, "\n- [") != 2 {
		t.Fatalf("expected two non-slot prompt lines, got %q", block)
	}
	if strings.Contains(block, "should be dropped by prompt limit") {
		t.Fatalf("expected prompt to drop overflow memory lines, got %q", block)
	}
	if !strings.Contains(block, "...") {
		t.Fatalf("expected long summary to be trimmed, got %q", block)
	}
}
