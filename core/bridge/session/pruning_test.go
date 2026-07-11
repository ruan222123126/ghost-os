package session

import (
	"encoding/json"
	"strings"
	"testing"

	"ghost-os/bridge/llm"
)

func TestEstimateTokensStructuredContentCostsMore(t *testing.T) {
	plain := llm.Message{
		Role: llm.RoleUser,
		Text: strings.Repeat("abcd", 30),
	}
	structured := llm.Message{
		Role: llm.RoleUser,
		Text: strings.Repeat(`{"k":1}`, 15),
	}

	plainTokens := EstimateTokens(plain)
	structuredTokens := EstimateTokens(structured)
	if structuredTokens <= plainTokens {
		t.Fatalf("expected structured message to consume more tokens: plain=%d structured=%d", plainTokens, structuredTokens)
	}
}

func TestPruneMessagesKeepsSystemPrompt(t *testing.T) {
	messages := []llm.Message{
		{Role: llm.RoleSystem, Text: "system"},
	}
	for i := 0; i < 25; i++ {
		messages = append(messages, llm.Message{
			Role: llm.RoleUser,
			Text: strings.Repeat("payload ", 20),
		})
	}

	pruned := PruneMessages(messages, 120)
	if len(pruned) == 0 {
		t.Fatal("pruned messages should not be empty")
	}
	if pruned[0].Role != llm.RoleSystem {
		t.Fatalf("unexpected first role: got %q want %q", pruned[0].Role, llm.RoleSystem)
	}
}

func TestPruneMessagesKeepsRecentMessages(t *testing.T) {
	messages := []llm.Message{
		{Role: llm.RoleSystem, Text: "system"},
	}
	for i := 0; i < 20; i++ {
		messages = append(messages, llm.Message{
			Role: llm.RoleUser,
			Text: strings.Repeat("message-"+string(rune('a'+(i%26)))+" ", 6),
		})
	}

	recent := messages[len(messages)-defaultRecentMessagesToKeep:]
	recentTokens := 0
	for _, msg := range recent {
		recentTokens += EstimateTokens(msg)
	}
	pruned := PruneMessages(messages, recentTokens+EstimateTokens(messages[0])+10)
	if len(pruned) == 0 {
		t.Fatal("pruned messages should not be empty")
	}

	// 最后 10 条应被完整保留。
	tail := messages[len(messages)-defaultRecentMessagesToKeep:]
	for _, want := range tail {
		found := false
		for _, got := range pruned {
			if got.Role == want.Role && got.Text == want.Text {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("recent message missing after prune: role=%q text=%q", want.Role, want.Text)
		}
	}
}

func TestPruneMessagesDropsRecentWhenStillOverLimit(t *testing.T) {
	messages := []llm.Message{
		{Role: llm.RoleSystem, Text: "system"},
	}
	for i := 0; i < 18; i++ {
		messages = append(messages, llm.Message{
			Role: llm.RoleUser,
			Text: strings.Repeat("oversize ", 40),
		})
	}

	pruned := PruneMessages(messages, 180)
	if len(pruned) == 0 {
		t.Fatal("pruned messages should not be empty")
	}

	total := 0
	for _, msg := range pruned {
		total += EstimateTokens(msg)
	}
	if total > 180 {
		t.Fatalf("expected pruned messages to fit budget: got %d", total)
	}
	if len(pruned) >= defaultRecentMessagesToKeep+1 {
		t.Fatalf("expected recent messages to be trimmed when oversized: got %d", len(pruned))
	}
}

func TestPruneMessagesTruncatesOversizedSpan(t *testing.T) {
	huge := strings.Repeat("payload ", 200)
	messages := []llm.Message{
		{Role: llm.RoleSystem, Text: "system"},
		{Role: llm.RoleUser, Text: huge},
	}

	pruned := PruneMessages(messages, 120)
	if len(pruned) < 2 {
		t.Fatalf("expected pruned messages to keep recent span, got %d", len(pruned))
	}

	total := 0
	for _, msg := range pruned {
		total += EstimateTokens(msg)
	}
	if total > 120 {
		t.Fatalf("expected pruned messages to fit budget: got %d", total)
	}
	if len(pruned[1].Text) >= len(huge) {
		t.Fatalf("expected recent message to be truncated")
	}
}

func TestPruneMessagesKeepsToolCallAndToolResultTogether(t *testing.T) {
	messages := []llm.Message{
		{Role: llm.RoleSystem, Text: "system"},
		{
			Role: llm.RoleAssistant,
			ToolCalls: []llm.ToolCall{
				{
					ID:        "call-1",
					Name:      "echo",
					Arguments: json.RawMessage(`{"input":"x"}`),
				},
			},
		},
		{
			Role:       llm.RoleTool,
			ToolCallID: "call-1",
			Text:       `{"status":"success"}`,
		},
	}

	for i := 0; i < 16; i++ {
		messages = append(messages, llm.Message{
			Role: llm.RoleUser,
			Text: strings.Repeat("later message ", 20),
		})
	}

	pruned := PruneMessages(messages, 200)
	if len(pruned) >= len(messages) {
		t.Fatalf("expected prune to remove old messages: before=%d after=%d", len(messages), len(pruned))
	}

	hasAssistantCall := false
	hasToolResult := false
	for _, msg := range pruned {
		if msg.Role == llm.RoleAssistant && len(msg.ToolCalls) > 0 && msg.ToolCalls[0].ID == "call-1" {
			hasAssistantCall = true
		}
		if msg.Role == llm.RoleTool && msg.ToolCallID == "call-1" {
			hasToolResult = true
		}
	}
	if hasAssistantCall != hasToolResult {
		t.Fatalf("tool call/result pair should stay together, got assistant=%v tool=%v", hasAssistantCall, hasToolResult)
	}
}

func TestPruneMessagesKeepsWholeLatestToolTurn(t *testing.T) {
	messages := []llm.Message{
		{Role: llm.RoleSystem, Text: "system"},
		{Role: llm.RoleUser, Text: "older"},
		{Role: llm.RoleAssistant, Text: strings.Repeat("older answer ", 80)},
		{Role: llm.RoleUser, Text: "browser demo"},
		{
			Role:             llm.RoleAssistant,
			ReasoningContent: json.RawMessage(`"step 1"`),
			ToolCalls: []llm.ToolCall{{
				ID:        "call-1",
				Name:      "script_exec",
				Arguments: json.RawMessage(`{"script":"one"}`),
			}},
		},
		{Role: llm.RoleTool, ToolCallID: "call-1", Text: strings.Repeat("tool-one ", 60)},
		{
			Role:             llm.RoleAssistant,
			ReasoningContent: json.RawMessage(`"step 2"`),
			ToolCalls: []llm.ToolCall{{
				ID:        "call-2",
				Name:      "script_exec",
				Arguments: json.RawMessage(`{"script":"two"}`),
			}},
		},
		{Role: llm.RoleTool, ToolCallID: "call-2", Text: strings.Repeat("tool-two ", 60)},
		{Role: llm.RoleAssistant, Text: "done"},
	}

	latestTurnTokens := 0
	for _, msg := range messages[3:] {
		latestTurnTokens += EstimateTokens(msg)
	}
	pruned := PruneMessages(messages, EstimateTokens(messages[0])+latestTurnTokens+8)

	if len(pruned) != len(messages[3:])+1 {
		t.Fatalf("expected system + full latest turn, got %d messages", len(pruned))
	}
	if pruned[1].Role != llm.RoleUser || pruned[1].Text != "browser demo" {
		t.Fatalf("expected latest user turn boundary to be preserved, got %+v", pruned[1])
	}
	if pruned[len(pruned)-1].Role != llm.RoleAssistant || pruned[len(pruned)-1].Text != "done" {
		t.Fatalf("expected final assistant message to remain, got %+v", pruned[len(pruned)-1])
	}
}

func TestPruneMessagesDoesNotTruncateOversizedToolTurn(t *testing.T) {
	messages := []llm.Message{
		{Role: llm.RoleSystem, Text: "system"},
		{Role: llm.RoleUser, Text: "run browser flow"},
		{
			Role:             llm.RoleAssistant,
			ReasoningContent: json.RawMessage(`"thinking"`),
			ToolCalls: []llm.ToolCall{{
				ID:        "call-1",
				Name:      "script_exec",
				Arguments: json.RawMessage(`{"script":"print(1)"}`),
			}},
		},
		{Role: llm.RoleTool, ToolCallID: "call-1", Text: strings.Repeat("very long tool output ", 120)},
		{Role: llm.RoleAssistant, Text: "all done"},
	}

	pruned := PruneMessages(messages, 120)
	if len(pruned) != len(messages) {
		t.Fatalf("expected oversized tool turn to stay intact, got %d want %d", len(pruned), len(messages))
	}
	if pruned[3].Text != messages[3].Text {
		t.Fatalf("expected tool output to remain untruncated")
	}
}

func TestGetContextLimit(t *testing.T) {
	if got := GetContextLimit(llm.ProviderOpenAI, "gpt-4o", ContextLimitConfig{}); got != openAIModernContextTokens-openAIResponseReserve {
		t.Fatalf("unexpected openai limit: got %d", got)
	}
	if got := GetContextLimit(llm.ProviderAnthropic, "claude-3-opus", ContextLimitConfig{}); got != anthropicModernContextTokens-anthropicResponseReserve {
		t.Fatalf("unexpected anthropic limit: got %d", got)
	}
	if got := GetContextLimit(llm.ProviderCodex, "codex-mini-latest", ContextLimitConfig{}); got != openAIModernContextTokens-openAIResponseReserve {
		t.Fatalf("unexpected codex limit: got %d", got)
	}
	if got := GetContextLimit(llm.ProviderCustom, "local-model", ContextLimitConfig{}); got != customContextTokens {
		t.Fatalf("unexpected custom limit: got %d", got)
	}
	override := ContextLimitConfig{
		ContextWindowTokens:   9000,
		ResponseReserveTokens: 500,
	}
	if got := GetContextLimit(llm.ProviderOpenAI, "gpt-4o", override); got != 8500 {
		t.Fatalf("unexpected override limit: got %d", got)
	}
}

func TestGetContextLimitModelOverrides(t *testing.T) {
	cfg := ContextLimitConfig{
		ModelContextWindowTokens: map[string]int{
			"gpt-*":    32000,
			"gpt-4o":   64000,
			"claude-*": 120000,
		},
		ModelResponseReserveTokens: map[string]int{
			"gpt-*":    2000,
			"gpt-4o":   1000,
			"claude-*": 3000,
		},
	}

	if got := GetContextLimit(llm.ProviderOpenAI, "gpt-4o", cfg); got != 63000 {
		t.Fatalf("unexpected exact override limit: got %d", got)
	}
	if got := GetContextLimit(llm.ProviderAnthropic, "claude-3-haiku", cfg); got != 117000 {
		t.Fatalf("unexpected prefix override limit: got %d", got)
	}
}

func TestMessagePrunerUsesInjectedEstimator(t *testing.T) {
	calls := 0
	pruner := newMessagePruner(1, func(msg llm.Message) int {
		calls++
		if msg.Role == llm.RoleSystem {
			return 1
		}
		return len(msg.Text)
	})

	messages := []llm.Message{
		{Role: llm.RoleSystem, Text: "system"},
		{Role: llm.RoleUser, Text: "1111"},
		{Role: llm.RoleUser, Text: "2222"},
		{Role: llm.RoleUser, Text: "3333"},
	}

	pruned := pruner.Prune(messages, 6)
	if calls == 0 {
		t.Fatal("expected custom estimator to be used")
	}
	if len(pruned) != 2 {
		t.Fatalf("unexpected pruned size: got %d", len(pruned))
	}
	if pruned[0].Role != llm.RoleSystem || pruned[1].Text != "3333" {
		t.Fatalf("unexpected pruned messages: %+v", pruned)
	}
}
