package agent

import (
	"context"
	"testing"

	"ghost-os/bridge/llm"
)

func TestCompletionRunnerCompleteDoesNotMutateHistory(t *testing.T) {
	completer := newFakeCompleter(&llm.CompletionResponse{
		Message: llm.Message{
			Text: "follow-up",
		},
		FinishReason: llm.FinishStop,
		ConversationState: llm.ConversationState{
			Provider:           llm.ProviderCodex,
			BaseURL:            "https://api.openai.com/v1",
			Model:              "codex-mini-latest",
			PreviousResponseID: "resp_next",
		},
	})
	history := NewHistoryFromMessages([]llm.Message{
		{Role: llm.RoleSystem, Text: "system prompt"},
		{Role: llm.RoleUser, Text: "hello"},
	})
	history.SetConversationState(llm.ConversationState{
		Provider:           llm.ProviderCodex,
		BaseURL:            "https://api.openai.com/v1",
		Model:              "codex-mini-latest",
		PreviousResponseID: "resp_prev",
	})

	runner := newCompletionRunner(completer, newFakeToolCatalog(), history)
	resp, err := runner.complete(context.Background(), nil, "trace-runner", "", 0)
	if err != nil {
		t.Fatalf("complete returned error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected response but got nil")
	}
	if resp.Message.Role != llm.RoleAssistant {
		t.Fatalf("expected assistant role normalization, got %q", resp.Message.Role)
	}
	if len(history.Messages()) != 2 {
		t.Fatalf("complete should not append assistant message to history, got %d messages", len(history.Messages()))
	}
	if got := history.ConversationState().PreviousResponseID; got != "resp_prev" {
		t.Fatalf("complete should not update conversation state: got %q want %q", got, "resp_prev")
	}
}
