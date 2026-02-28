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
			Text: strings.Repeat("message-"+string(rune('a'+(i%26)))+" ", 15),
		})
	}

	pruned := PruneMessages(messages, 140)
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

func TestGetContextLimit(t *testing.T) {
	if got := GetContextLimit(llm.ProviderOpenAI, "gpt-4o"); got != openAIContextTokens-openAIResponseReserve {
		t.Fatalf("unexpected openai limit: got %d", got)
	}
	if got := GetContextLimit(llm.ProviderAnthropic, "claude-3-opus"); got != anthropicContextTokens-anthropicResponseReserve {
		t.Fatalf("unexpected anthropic limit: got %d", got)
	}
	if got := GetContextLimit(llm.ProviderCustom, "local-model"); got != customContextTokens {
		t.Fatalf("unexpected custom limit: got %d", got)
	}
}
