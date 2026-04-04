package agent

import (
	"context"
	"encoding/json"
	"strings"
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
	if req.Text != "invoke handler" {
		return AssistantTextResult{}, nil
	}
	h.calls = append(h.calls, req)
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

type stubAssistantTextInvocationHandler struct{}

func (h stubAssistantTextInvocationHandler) HandleAssistantText(
	_ context.Context,
	req AssistantTextRequest,
) (AssistantTextResult, error) {
	if req.Text != "invoke tool handler" {
		return AssistantTextResult{}, nil
	}
	return AssistantTextResult{
		Recognized: true,
		Tool: AssistantTextToolRef{
			Name:   "web_search",
			CallID: "assistant-text-invoke-call",
		},
		Invocation: &AssistantTextToolInvocation{
			Arguments: json.RawMessage(`{"query":"OpenAI"}`),
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
	if last.Role != llm.RoleAssistant {
		t.Fatalf("unexpected feedback message: %+v", last)
	}
	if last.Text != "[ASSISTANT_TEXT_RESULT]\n{\"status\":\"ok\"}" {
		t.Fatalf("unexpected feedback text: %q", last.Text)
	}
}

func TestRunAssistantTextInvocationUsesHandlerFeedbackWithoutGraphQLSpecialCase(t *testing.T) {
	completer := newFakeCompleter(
		newLengthResponse("invoke tool handler"),
		newStopResponse("done"),
	)
	tool := newStaticTool("web_search", `{"items":[{"title":"OpenAI"}]}`)
	agent := newTestAgent(completer, newFakeToolCatalog(tool), 2)
	agent.AddAssistantTextHandler(stubAssistantTextInvocationHandler{})

	got, err := agent.Run(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if got != "done" {
		t.Fatalf("unexpected output: got %q want %q", got, "done")
	}
	if tool.callCount != 1 {
		t.Fatalf("expected web_search to execute once, got %d", tool.callCount)
	}

	secondTurnMessages := completer.requests[1].Messages
	if len(secondTurnMessages) < 4 {
		t.Fatalf("unexpected second turn messages: %+v", secondTurnMessages)
	}
	last := secondTurnMessages[len(secondTurnMessages)-1]
	if last.Role != llm.RoleAssistant {
		t.Fatalf("unexpected feedback message: %+v", last)
	}
	if last.Text != "[ASSISTANT_TEXT_RESULT]\n{\"status\":\"ok\"}" {
		t.Fatalf("unexpected feedback text: %q", last.Text)
	}
	if strings.Contains(last.Text, "[TOOL_TAG_RESULT]") {
		t.Fatalf("invocation handler feedback should not be rewritten as graphql feedback: %q", last.Text)
	}

	newMessages := agent.GetNewMessages()
	if len(newMessages) < 2 {
		t.Fatalf("unexpected committed messages: %+v", newMessages)
	}
	assistant := newMessages[1]
	if len(assistant.ToolCalls) != 1 {
		t.Fatalf("expected assistant invocation to commit tool_calls immediately, got %+v", assistant)
	}
	if assistant.ToolCalls[0].Name != "web_search" {
		t.Fatalf("unexpected committed tool call: %+v", assistant.ToolCalls[0])
	}
}
