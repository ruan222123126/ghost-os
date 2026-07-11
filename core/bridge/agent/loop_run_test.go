package agent

import (
	"context"
	"strings"
	"testing"

	"ghost-os/bridge/llm"
)

func TestGetNewMessagesWithPreloadedHistory(t *testing.T) {
	completer := newFakeCompleter(newStopResponse("follow-up"))
	catalog := newFakeToolCatalog()
	preloaded := NewHistoryFromMessages([]llm.Message{
		{Role: llm.RoleSystem, Text: "system prompt"},
		{Role: llm.RoleUser, Text: "old user message"},
		{Role: llm.RoleAssistant, Text: "old assistant message"},
	})

	agent := NewAgentWithHistory(completer, catalog, preloaded, 3)
	if _, err := agent.Run(context.Background(), "new user message"); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	newMessages := agent.GetNewMessages()
	if len(newMessages) != 2 {
		t.Fatalf("unexpected new message count: got %d want %d", len(newMessages), 2)
	}
	if newMessages[0].Role != llm.RoleUser || newMessages[0].Text != "new user message" {
		t.Fatalf("unexpected first new message: %+v", newMessages[0])
	}
	if newMessages[1].Role != llm.RoleAssistant || newMessages[1].Text != "follow-up" {
		t.Fatalf("unexpected second new message: %+v", newMessages[1])
	}
}

func TestRunWithPreloadedHistorySkipsEmptyResumeUserMessage(t *testing.T) {
	completer := newFakeCompleter(newStopResponse("resume-ok"))
	catalog := newFakeToolCatalog()
	preloaded := NewHistoryFromMessages([]llm.Message{
		{Role: llm.RoleSystem, Text: "system prompt"},
		{Role: llm.RoleAssistant, ToolCalls: []llm.ToolCall{{ID: "call-ask", Name: "ask_human", Arguments: []byte(`{"prompt":"Ship now?"}`)}}},
		{Role: llm.RoleTool, ToolCallID: "call-ask", Text: `{"status":"success","tool":"ask_human","trace_id":"trace-resume","output":"{\"question_id\":\"q-1\",\"prompt\":\"Ship now?\",\"answer\":\"Yes\"}"}`},
	})
	initialMessageCount := len(preloaded.Messages())

	agent := NewAgentWithHistory(completer, catalog, preloaded, 2)
	got, err := agent.Run(context.Background(), "   ")
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if got != "resume-ok" {
		t.Fatalf("unexpected output: got %q want %q", got, "resume-ok")
	}
	if len(completer.requests) != 1 {
		t.Fatalf("unexpected complete call count: got %d want %d", len(completer.requests), 1)
	}

	requestMessages := completer.requests[0].Messages
	if len(requestMessages) != initialMessageCount {
		t.Fatalf("unexpected message count: got %d want %d", len(requestMessages), initialMessageCount)
	}
	last := requestMessages[len(requestMessages)-1]
	if last.Role != llm.RoleTool || last.ToolCallID != "call-ask" {
		t.Fatalf("expected resume to end with existing tool message, got %+v", last)
	}

	newMessages := agent.GetNewMessages()
	if len(newMessages) != 1 {
		t.Fatalf("unexpected new message count: got %d want %d", len(newMessages), 1)
	}
	if newMessages[0].Role != llm.RoleAssistant || newMessages[0].Text != "resume-ok" {
		t.Fatalf("unexpected new message: %+v", newMessages[0])
	}
}

func TestRunWithPreloadedConversationStatePassesPreviousResponseID(t *testing.T) {
	completer := newFakeCompleter(&llm.CompletionResponse{
		Message:      llm.Message{Role: llm.RoleAssistant, Text: "follow-up"},
		FinishReason: llm.FinishStop,
		ConversationState: llm.ConversationState{
			Provider:           llm.ProviderCodex,
			BaseURL:            "https://api.openai.com/v1",
			Model:              "codex-mini-latest",
			PreviousResponseID: "resp_next",
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

	agent := NewAgentWithHistory(completer, catalog, preloaded, 3)
	if _, err := agent.Run(context.Background(), "new user message"); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if len(completer.requests) != 1 {
		t.Fatalf("unexpected complete call count: got %d want 1", len(completer.requests))
	}
	if completer.requests[0].ConversationState.PreviousResponseID != "resp_prev" {
		t.Fatalf("unexpected previous_response_id: got %q want %q", completer.requests[0].ConversationState.PreviousResponseID, "resp_prev")
	}
	if agent.GetConversationState().PreviousResponseID != "resp_next" {
		t.Fatalf("unexpected updated conversation state: %+v", agent.GetConversationState())
	}
}

func TestRunInvalidToolCallClearsConversationStateForNextTurnReplay(t *testing.T) {
	completer := newFakeCompleter(
		&llm.CompletionResponse{
			Message: llm.Message{
				Role: llm.RoleAssistant,
				ToolCalls: []llm.ToolCall{
					{
						ID:        "",
						Name:      "echo",
						Arguments: []byte(`{}`),
					},
				},
			},
			FinishReason: llm.FinishToolCalls,
			ConversationState: llm.ConversationState{
				Provider:           llm.ProviderCodex,
				BaseURL:            "https://api.openai.com/v1",
				Model:              "codex-mini-latest",
				PreviousResponseID: "resp_invalid",
			},
		},
		&llm.CompletionResponse{
			Message:      llm.Message{Role: llm.RoleAssistant, Text: "done"},
			FinishReason: llm.FinishStop,
		},
	)
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

	agent := NewAgentWithHistory(completer, newFakeToolCatalog(), preloaded, 3)
	if _, err := agent.Run(context.Background(), "new user message"); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if len(completer.requests) != 2 {
		t.Fatalf("unexpected complete call count: got %d want %d", len(completer.requests), 2)
	}
	if got := completer.requests[0].ConversationState.PreviousResponseID; got != "resp_prev" {
		t.Fatalf("unexpected first previous_response_id: got %q want %q", got, "resp_prev")
	}
	if got := completer.requests[1].ConversationState.PreviousResponseID; got != "" {
		t.Fatalf("invalid tool-call rewrite should clear conversation state before replay, got %q", got)
	}
}

func TestRunFailureDoesNotCommitUserMessageOrNewMessages(t *testing.T) {
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
		t.Fatalf("failed turn should not produce committed new messages: %+v", got)
	}

	historyMessages := agent.history.Messages()
	if len(historyMessages) != 3 {
		t.Fatalf("unexpected history length after failed turn: got %d want %d", len(historyMessages), 3)
	}
	if historyMessages[len(historyMessages)-1].Text != "old assistant message" {
		t.Fatalf("failed turn should keep original history intact: %+v", historyMessages)
	}
}

func TestRunFailureRollsBackAssistantMessageAndConversationState(t *testing.T) {
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
		t.Fatalf("failed turn should not update conversation state: got %q want %q", got, "resp_prev")
	}
	if got := agent.GetNewMessages(); len(got) != 0 {
		t.Fatalf("failed turn should not expose assistant partials as new messages: %+v", got)
	}
	if len(agent.history.Messages()) != 3 {
		t.Fatalf("unexpected history length after failed assistant turn: got %d want %d", len(agent.history.Messages()), 3)
	}
}

// 场景：模型直接 stop，Agent 返回 assistant 文本。
func TestRunStop(t *testing.T) {
	completer := newFakeCompleter(newStopResponse("done"))
	catalog := newFakeToolCatalog()

	agent := newTestAgent(completer, catalog, 3)
	got, err := agent.Run(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if got != "done" {
		t.Fatalf("unexpected output: got %q want %q", got, "done")
	}
	if len(completer.requests) != 1 {
		t.Fatalf("unexpected complete call count: got %d want %d", len(completer.requests), 1)
	}
}

func TestRunLengthReturnsMessage(t *testing.T) {
	completer := newFakeCompleter(newLengthResponse("truncated-final"))
	catalog := newFakeToolCatalog()

	agent := newTestAgent(completer, catalog, 2)
	got, err := agent.Run(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if got != "truncated-final" {
		t.Fatalf("unexpected output: got %q want %q", got, "truncated-final")
	}
}

// 场景：未知 finish_reason 返回显式错误，并带 trace_id/turn。
func TestRunUnknownFinishReasonReturnsError(t *testing.T) {
	completer := newFakeCompleter(newAssistantResponse(llm.FinishReason("unknown_reason"), "unknown"))
	catalog := newFakeToolCatalog()

	agent := newTestAgent(completer, catalog, 1)
	_, err := agent.Run(context.Background(), "hello")
	if err == nil {
		t.Fatal("expected error but got nil")
	}
	if !strings.Contains(err.Error(), "unsupported finish_reason") {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(err.Error(), "trace_id=") {
		t.Fatalf("error should include trace_id: %v", err)
	}
	if !strings.Contains(err.Error(), "turn=0") {
		t.Fatalf("error should include turn: %v", err)
	}
}

// 场景：连续 tool_calls 超过 max turns 时退出。
func TestRunMaxTurnsExceeded(t *testing.T) {
	tool := newStaticTool("echo", "ok")
	completer := newFakeCompleter(
		newToolCallsResponse(newToolCall("call-1", "echo", `{}`)),
		newToolCallsResponse(newToolCall("call-2", "echo", `{}`)),
	)
	catalog := newFakeToolCatalog(tool)

	agent := newTestAgent(completer, catalog, 2)
	_, err := agent.Run(context.Background(), "hello")
	if err == nil {
		t.Fatal("expected error but got nil")
	}
	if !strings.Contains(err.Error(), "max turns exceeded: 2") {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(err.Error(), "trace_id=") {
		t.Fatalf("error should include trace_id: %v", err)
	}
}
