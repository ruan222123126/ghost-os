package agent

import (
	"context"
	"testing"

	"ghost-os/bridge/llm"
)

type stubAssistantTextHandler struct {
	calls []AssistantTextRequest
}

func (h *stubAssistantTextHandler) HandleAssistantText(
	_ context.Context,
	req AssistantTextRequest,
) (AssistantTextResult, error) {
	h.calls = append(h.calls, req)
	if req.Text != "invoke handler" {
		return AssistantTextResult{}, nil
	}
	return AssistantTextResult{
		Recognized: true,
		Tool: AssistantTextToolRef{
			Name:   "assistant_text_protocol",
			CallID: "assistant-text-call",
		},
		Feedback: []llm.Message{
			{Role: llm.RoleInternal, Text: "[ASSISTANT_TEXT_RESULT]\n{\"status\":\"ok\"}"},
		},
	}, nil
}

func TestRunLengthTrimsTerminalOutputButKeepsAssistantHistory(t *testing.T) {
	completer := newFakeCompleter(newLengthResponse("  truncated-final  \n"))
	agent := newTestAgent(completer, newFakeToolCatalog(), 1)

	got, err := agent.Run(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if got != "truncated-final" {
		t.Fatalf("unexpected output: got %q want %q", got, "truncated-final")
	}

	newMessages := agent.GetNewMessages()
	if len(newMessages) != 2 {
		t.Fatalf("unexpected new message count: got %d want %d", len(newMessages), 2)
	}
	if newMessages[1].Role != llm.RoleAssistant {
		t.Fatalf("unexpected assistant message: %+v", newMessages[1])
	}
	if newMessages[1].Text != "  truncated-final  \n" {
		t.Fatalf("assistant history should keep raw completion text, got %q", newMessages[1].Text)
	}
}

func TestRunLengthStillDispatchesAssistantTextHandler(t *testing.T) {
	completer := newFakeCompleter(
		newLengthResponse("invoke handler"),
		newStopResponse("done"),
	)
	handler := &stubAssistantTextHandler{}
	agent := newTestAgent(completer, newFakeToolCatalog(), 2)
	agent.AddAssistantTextHandler(handler)

	got, err := agent.Run(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if got != "done" {
		t.Fatalf("unexpected output: got %q want %q", got, "done")
	}
	if len(handler.calls) != 1 {
		t.Fatalf("unexpected handler call count: got %d want %d", len(handler.calls), 1)
	}

	secondTurnMessages := completer.requests[1].Messages
	if len(secondTurnMessages) < 4 {
		t.Fatalf("expected committed assistant-text turn before second completion, got %+v", secondTurnMessages)
	}
	last := secondTurnMessages[len(secondTurnMessages)-1]
	if last.Role != llm.RoleInternal {
		t.Fatalf("unexpected feedback message: %+v", last)
	}
	if last.Text != "[ASSISTANT_TEXT_RESULT]\n{\"status\":\"ok\"}" {
		t.Fatalf("unexpected feedback text: %q", last.Text)
	}
}
