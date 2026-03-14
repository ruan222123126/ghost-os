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
}
