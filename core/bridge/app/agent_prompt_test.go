package app

import (
	"testing"

	"ghost-os/bridge/llm"
)

func TestMessagesWithSystemPromptReplacesFirstSystemMessage(t *testing.T) {
	input := []llm.Message{
		{
			Role: llm.RoleSystem,
			Text: "old prompt",
		},
		{
			Role: llm.RoleUser,
			Text: "hello",
		},
	}

	got := messagesWithSystemPrompt(input, "new prompt")

	if len(got) != 2 {
		t.Fatalf("unexpected message count: got %d want %d", len(got), 2)
	}
	if got[0].Role != llm.RoleSystem || got[0].Text != "new prompt" {
		t.Fatalf("unexpected first message: %+v", got[0])
	}
	if got[1].Role != llm.RoleUser || got[1].Text != "hello" {
		t.Fatalf("unexpected second message: %+v", got[1])
	}
	if input[0].Text != "old prompt" {
		t.Fatalf("input should stay unchanged: got %q want %q", input[0].Text, "old prompt")
	}
}

func TestMessagesWithSystemPromptPrependsWhenMissing(t *testing.T) {
	input := []llm.Message{
		{
			Role: llm.RoleUser,
			Text: "hello",
		},
	}

	got := messagesWithSystemPrompt(input, "system prompt")

	if len(got) != 2 {
		t.Fatalf("unexpected message count: got %d want %d", len(got), 2)
	}
	if got[0].Role != llm.RoleSystem || got[0].Text != "system prompt" {
		t.Fatalf("unexpected first message: %+v", got[0])
	}
	if got[1].Role != llm.RoleUser || got[1].Text != "hello" {
		t.Fatalf("unexpected second message: %+v", got[1])
	}
	if len(input) != 1 {
		t.Fatalf("input should stay unchanged: got len=%d want %d", len(input), 1)
	}
}
