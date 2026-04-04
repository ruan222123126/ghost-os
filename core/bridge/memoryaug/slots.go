package memoryaug

import (
	"strings"

	"ghost-os/bridge/memorystore"
)

const slotVersion = 1

type SlotSpec struct {
	Key            string   `json:"key"`
	MemoryType     string   `json:"memory_type"`
	DefaultScope   string   `json:"default_scope"`
	Summary        string   `json:"summary"`
	Description    string   `json:"description"`
	AllowedValues  []string `json:"allowed_values,omitempty"`
	PromptPriority int      `json:"prompt_priority"`
	QueryHints     []string `json:"query_hints,omitempty"`
}

var knownSlots = []SlotSpec{
	{
		Key:            "reply_language",
		MemoryType:     memorystore.MemoryTypePreference,
		DefaultScope:   memorystore.ScopeTypeUser,
		Summary:        "reply language",
		Description:    "Default language to use in replies.",
		AllowedValues:  []string{"zh-CN", "en-US", "bilingual"},
		PromptPriority: 10,
		QueryHints:     []string{"reply language", "respond in", "reply in", "language", "中文", "英文", "english", "chinese"},
	},
	{
		Key:            "response_style",
		MemoryType:     memorystore.MemoryTypePreference,
		DefaultScope:   memorystore.ScopeTypeUser,
		Summary:        "response style",
		Description:    "Preferred response style for the assistant.",
		AllowedValues:  []string{"concise", "detailed", "step_by_step"},
		PromptPriority: 20,
		QueryHints:     []string{"response style", "concise", "detailed", "step by step", "简洁", "详细", "分步"},
	},
	{
		Key:            "approval_style",
		MemoryType:     memorystore.MemoryTypePreference,
		DefaultScope:   memorystore.ScopeTypeUser,
		Summary:        "approval style",
		Description:    "How to handle edits and approvals.",
		AllowedValues:  []string{"ask_before_destructive", "auto_apply_safe_changes"},
		PromptPriority: 30,
		QueryHints:     []string{"approval", "destructive", "auto apply", "safe changes", "批准", "危险操作", "自动应用"},
	},
}

var knownSlotByKey = buildSlotIndex(knownSlots)

func buildSlotIndex(specs []SlotSpec) map[string]SlotSpec {
	index := make(map[string]SlotSpec, len(specs))
	for _, spec := range specs {
		index[spec.Key] = spec
	}
	return index
}

func slotSpecForKey(raw string) (SlotSpec, bool) {
	spec, ok := knownSlotByKey[memorystore.NormalizeMemoryKey(raw)]
	return spec, ok
}

func extractorKnownSlots() []SlotSpec {
	out := make([]SlotSpec, len(knownSlots))
	copy(out, knownSlots)
	return out
}

func globalPreferenceKeys() []string {
	keys := make([]string, 0, len(knownSlots))
	for _, spec := range knownSlots {
		keys = append(keys, spec.Key)
	}
	return keys
}

func containsNormalized(normalizedText string, raw string) bool {
	needle := normalizeText(raw)
	if needle == "" {
		return false
	}
	return strings.Contains(normalizedText, needle)
}
