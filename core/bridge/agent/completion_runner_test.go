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

func TestCompletionRunnerProjectsInternalMessagesForProvider(t *testing.T) {
	completer := newFakeCompleter(&llm.CompletionResponse{
		Message:      llm.Message{Role: llm.RoleAssistant, Text: "done"},
		FinishReason: llm.FinishStop,
	})
	history := NewHistoryFromMessages([]llm.Message{
		{Role: llm.RoleSystem, Text: "system prompt"},
		{Role: llm.RoleUser, Text: "hello"},
		{Role: llm.RoleInternal, Text: "[GRAPHQL_EXECUTION_RESULT]\n{\"data\":{\"viewer\":{\"id\":\"1\"}}}"},
	})

	runner := newCompletionRunner(completer, newFakeToolCatalog(), history)
	if _, err := runner.complete(context.Background(), nil, "trace-runner", "", 0); err != nil {
		t.Fatalf("complete returned error: %v", err)
	}

	if len(completer.requests) != 1 {
		t.Fatalf("expected one completion request, got %d", len(completer.requests))
	}
	request := completer.requests[0]
	last := request.Messages[len(request.Messages)-1]
	if last.Role != llm.RoleAssistant {
		t.Fatalf("expected projected assistant role for provider request, got %+v", last)
	}
	if history.Messages()[2].Role != llm.RoleInternal {
		t.Fatalf("expected persisted history role to stay internal, got %+v", history.Messages()[2])
	}
}
