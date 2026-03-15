package orchestration

import (
	"context"
	"strings"
	"testing"

	"ghost-os/bridge/llm"
	"ghost-os/bridge/memoryaug"
	"ghost-os/bridge/memorystore"
)

const compactRecallLongSummary = "This Android runtime settings workflow summary is intentionally long so the formatter must trim it before injecting the prompt."

func TestSessionRunnerInjectsCompactRecallPromptBlock(t *testing.T) {
	sessionStore := newTempSessionStore(t)
	completer := newCompactRecallCompleter()
	runner := NewSessionAgentRunner(proTestRuntimeFactory{
		deps: buildMemoryTestDeps(completer, fixedRecallService{items: compactRecallItems()}),
	}, nil, sessionStore, nil)

	if _, _, err := runner.RunTurn(context.Background(), "continue", "", "trace-memory-compact"); err != nil {
		t.Fatalf("run turn: %v", err)
	}
	assertCompactRecallPrompt(t, completer.requests[0].Messages[0].Text)
}

func newCompactRecallCompleter() *proTestCompleter {
	return &proTestCompleter{
		responses: []*llm.CompletionResponse{{
			Message:      llm.Message{Role: llm.RoleAssistant, Text: "noted"},
			FinishReason: llm.FinishStop,
		}},
	}
}

func compactRecallItems() []memoryaug.RecallItem {
	return []memoryaug.RecallItem{
		compactRecallSlotItem(),
		compactSessionWorkflowItem("mem-1", compactRecallLongSummary, 0.9, "This content should never appear in the system prompt."),
		compactUserPreferenceItem("mem-2", "pnpm is the package manager"),
		compactSessionWorkflowItem("mem-3", "should be dropped by prompt limit", 0.7, ""),
	}
}

func compactRecallSlotItem() memoryaug.RecallItem {
	return memoryaug.RecallItem{
		Entry: memorystore.MemoryEntry{
			ID:         "mem-slot",
			ScopeType:  memorystore.ScopeTypeUser,
			ScopeID:    memorystore.DefaultUserScopeID,
			SourceKind: memorystore.SourceKindLearned,
			MemoryType: memorystore.MemoryTypePreference,
			MemoryKey:  "reply_language",
			Summary:    "reply language",
			Content:    "Reply in Chinese by default.",
			Metadata:   compactRecallSlotMetadata("zh-CN"),
			Confidence: 0.95,
			Status:     memorystore.MemoryStatusActive,
		},
		SlotMatch: true,
	}
}

func compactSessionWorkflowItem(id string, summary string, confidence float64, content string) memoryaug.RecallItem {
	return memoryaug.RecallItem{
		Entry: memorystore.MemoryEntry{
			ID:         id,
			ScopeType:  memorystore.ScopeTypeSession,
			ScopeID:    "session-1",
			SourceKind: memorystore.SourceKindLearned,
			MemoryType: memorystore.MemoryTypeWorkflow,
			Summary:    summary,
			Content:    content,
			Confidence: confidence,
			Status:     memorystore.MemoryStatusActive,
		},
	}
}

func compactUserPreferenceItem(id string, summary string) memoryaug.RecallItem {
	return memoryaug.RecallItem{
		Entry: memorystore.MemoryEntry{
			ID:         id,
			ScopeType:  memorystore.ScopeTypeUser,
			ScopeID:    memorystore.DefaultUserScopeID,
			SourceKind: memorystore.SourceKindExplicit,
			MemoryType: memorystore.MemoryTypePreference,
			Summary:    summary,
			Confidence: 1,
			Status:     memorystore.MemoryStatusActive,
		},
	}
}

func assertCompactRecallPrompt(t *testing.T, prompt string) {
	t.Helper()
	if strings.Contains(prompt, "content=") {
		t.Fatalf("expected compact memory prompt without content field, got %q", prompt)
	}
	if strings.Count(prompt, "[session/workflow]")+strings.Count(prompt, "[user/preference]") != 2 {
		t.Fatalf("expected prompt to keep only two non-slot memory lines, got %q", prompt)
	}
	if strings.Contains(prompt, "should be dropped by prompt limit") {
		t.Fatalf("expected prompt to omit overflow memory line, got %q", prompt)
	}
	if !strings.Contains(prompt, "Memory slots:\n- reply_language=zh-CN") {
		t.Fatalf("expected slot line to remain in prompt, got %q", prompt)
	}
	if !strings.Contains(prompt, "...") {
		t.Fatalf("expected long summary to be trimmed, got %q", prompt)
	}
}

func compactRecallSlotMetadata(value string) map[string]any {
	return map[string]any{
		"value":        value,
		"slot_version": "test",
	}
}
