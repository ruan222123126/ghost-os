package agent

import (
	"context"
	"errors"
	"testing"

	"ghost-os/bridge/llm"
)

func TestRunFailedTurnDoesNotCommitHistory(t *testing.T) {
	completer := newFakeCompleter()
	catalog := newFakeToolCatalog()
	preloaded := NewHistoryFromMessages([]llm.Message{
		{Role: llm.RoleSystem, Text: "system prompt"},
		{Role: llm.RoleUser, Text: "old user message"},
		{Role: llm.RoleAssistant, Text: "old assistant message"},
	})

	agent := NewAgentWithHistory(completer, catalog, preloaded, 1)
	_, err := agent.Run(context.Background(), "new user message")
	if err == nil {
		t.Fatal("expected error but got nil")
	}

	if got := agent.GetNewMessages(); len(got) != 0 {
		t.Fatalf("failed turn should not expose committed messages: %+v", got)
	}

	historyMessages := agent.history.Messages()
	if len(historyMessages) != 3 {
		t.Fatalf("unexpected history length after failed turn: got %d want %d", len(historyMessages), 3)
	}
	if historyMessages[len(historyMessages)-1].Text != "old assistant message" {
		t.Fatalf("failed turn should keep original history intact: %+v", historyMessages)
	}
}

func TestRunFailedTurnDoesNotCommitConversationState(t *testing.T) {
	completer := newFakeCompleter(&llm.CompletionResponse{
		Message:      llm.Message{Role: llm.RoleAssistant, Text: "partial"},
		FinishReason: llm.FinishReason("unsupported"),
		ConversationState: llm.ConversationState{
			Provider:           llm.ProviderCodex,
			BaseURL:            "https://api.openai.com/v1",
			Model:              "codex-mini-latest",
			PreviousResponseID: "resp_failed",
		},
	})
	catalog := newFakeToolCatalog()
	preloaded := NewHistoryFromMessages([]llm.Message{
		{Role: llm.RoleSystem, Text: "system prompt"},
		{Role: llm.RoleUser, Text: "old user message"},
		{Role: llm.RoleAssistant, Text: "old assistant message"},
	})
	preloaded.SetConversationState(llm.ConversationState{
		Provider:           llm.ProviderCodex,
		BaseURL:            "https://api.openai.com/v1",
		Model:              "codex-mini-latest",
		PreviousResponseID: "resp_prev",
	})

	agent := NewAgentWithHistory(completer, catalog, preloaded, 1)
	_, err := agent.Run(context.Background(), "new user message")
	if err == nil {
		t.Fatal("expected error but got nil")
	}

	if got := agent.GetConversationState().PreviousResponseID; got != "resp_prev" {
		t.Fatalf("failed turn should keep conversation state unchanged: got %q want %q", got, "resp_prev")
	}
	if got := agent.GetNewMessages(); len(got) != 0 {
		t.Fatalf("failed turn should not expose partial assistant messages: %+v", got)
	}
}

func TestRunAwaitingHumanCommitsTurnState(t *testing.T) {
	tool := newAwaitingHumanTool("approval_gate", "q-123", "Which database?")
	completer := newFakeCompleter(
		newToolCallsResponse(newToolCall("call-ask-1", "approval_gate", `{"prompt":"Which database?"}`)),
	)
	catalog := newFakeToolCatalog(tool)

	agent := newTestAgent(completer, catalog, 3)
	_, err := agent.Run(context.Background(), "pick db")
	if err == nil {
		t.Fatal("expected awaiting-human error")
	}

	var awaitingErr *ErrAwaitingHuman
	if !errors.As(err, &awaitingErr) {
		t.Fatalf("expected ErrAwaitingHuman, got %v", err)
	}

	newMessages := agent.GetNewMessages()
	if len(newMessages) != 2 {
		t.Fatalf("awaiting-human turn should commit user and assistant messages: got %d want %d", len(newMessages), 2)
	}
	if newMessages[0].Role != llm.RoleUser || newMessages[0].Text != "pick db" {
		t.Fatalf("unexpected committed user message: %+v", newMessages[0])
	}
	if newMessages[1].Role != llm.RoleAssistant || len(newMessages[1].ToolCalls) != 1 {
		t.Fatalf("unexpected committed assistant message: %+v", newMessages[1])
	}
}

func TestResetNewMessagesAdvancesBaseline(t *testing.T) {
	history := NewHistory("system prompt")
	agent := NewAgentWithHistory(nil, nil, history, 1)

	history.Append(llm.Message{Role: llm.RoleUser, Text: "hello"})
	if got := agent.GetNewMessages(); len(got) != 1 {
		t.Fatalf("expected 1 new message, got %d", len(got))
	}

	agent.ResetNewMessages()
	if got := agent.GetNewMessages(); len(got) != 0 {
		t.Fatalf("expected baseline reset to consume new messages, got %d", len(got))
	}

	history.Append(llm.Message{Role: llm.RoleAssistant, Text: "world"})
	if got := agent.GetNewMessages(); len(got) != 1 {
		t.Fatalf("expected 1 new message after reset, got %d", len(got))
	}
}
