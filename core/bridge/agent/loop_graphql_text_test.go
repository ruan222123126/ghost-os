package agent

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"ghost-os/bridge/llm"
	"ghost-os/bridge/streaming"
	"ghost-os/bridge/tools"
)

func TestStrictGraphQLTextModeRejectsToolCalls(t *testing.T) {
	completer := newFakeCompleter(newToolCallsResponse(newToolCall("call-1", "echo", `{}`)))
	agent := newTestAgent(completer, newFakeToolCatalog(), 1)
	agent.SetStrictToolCallProtocol(true)

	_, err := agent.Run(context.Background(), "hello")
	if err == nil {
		t.Fatal("expected strict graphql text mode to reject tool calls")
	}
	if !strings.Contains(err.Error(), "strict text protocol rejects tool_calls finish reason") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGraphQLTextTurnExecutesAndFeedsBackResult(t *testing.T) {
	completer := newFakeCompleter(
		newStopResponse(`query { web_search(query: "OpenAI") }`),
		newStopResponse("done"),
	)
	tool := newStaticTool("web_search", `{"items":[{"title":"OpenAI"}]}`)
	tool.semantics = llm.ToolSemantics{ReadOnly: true}
	catalog := newFakeToolCatalog(tool)
	agent := newTestAgent(completer, catalog, 3)
	agent.SetStrictToolCallProtocol(true)
	agent.AddAssistantTextHandler(NewGraphQLTextTurnHandler(tools.NewGraphQLTextExecutor(catalog)))

	output, err := agent.Run(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if output != "done" {
		t.Fatalf("unexpected output: got %q want %q", output, "done")
	}
	if len(completer.requests) != 2 {
		t.Fatalf("unexpected complete call count: got %d want %d", len(completer.requests), 2)
	}
	last := completer.requests[1].Messages[len(completer.requests[1].Messages)-1]
	if !strings.Contains(last.Text, "[GRAPHQL_TOOL_RESULT]") {
		t.Fatalf("expected graphql tool result feedback in second request, got %+v", last)
	}
	if tool.callCount != 1 {
		t.Fatalf("expected web_search to execute once, got %d", tool.callCount)
	}
}

func TestGraphQLTextTurnReturnsAwaitingHumanSignal(t *testing.T) {
	completer := newFakeCompleter(newStopResponse(`mutation { ask_human(prompt: "Approve write?") }`))
	catalog := newFakeToolCatalog(newAwaitingHumanTool("ask_human", "q-1", "Approve write?"))
	agent := newTestAgent(completer, catalog, 2)
	agent.SetStrictToolCallProtocol(true)
	agent.AddAssistantTextHandler(NewGraphQLTextTurnHandler(tools.NewGraphQLTextExecutor(catalog)))

	_, err := agent.Run(context.Background(), "hello")
	var awaitingErr *ErrAwaitingHuman
	if !errors.As(err, &awaitingErr) {
		t.Fatalf("expected ErrAwaitingHuman, got %v", err)
	}
	if awaitingErr.QuestionID != "q-1" {
		t.Fatalf("unexpected awaiting question id: %q", awaitingErr.QuestionID)
	}
}

func TestGraphQLTextTurnAwaitingHumanEventUsesRealToolName(t *testing.T) {
	completer := newFakeCompleter(newStopResponse(`mutation { ask_human(prompt: "Approve write?") }`))
	catalog := newFakeToolCatalog(newAwaitingHumanTool("ask_human", "q-1", "Approve write?"))
	agent := newTestAgent(completer, catalog, 2)
	sink := newRecordingEventSink()
	agent.SetStrictToolCallProtocol(true)
	agent.AddAssistantTextHandler(NewGraphQLTextTurnHandler(tools.NewGraphQLTextExecutor(catalog)))

	_, err := agent.RunStreamWithTraceID(context.Background(), "hello", "trace-await", sink)
	var awaitingErr *ErrAwaitingHuman
	if !errors.As(err, &awaitingErr) {
		t.Fatalf("expected ErrAwaitingHuman, got %v", err)
	}
	if len(sink.events) != 4 {
		t.Fatalf("unexpected event count: got %d want 4", len(sink.events))
	}
	startID := eventToolCallID(t, sink.events[1])
	if got := eventToolCallID(t, sink.events[2]); got != startID {
		t.Fatalf("unexpected tool_call_finished id: got %q want %q", got, startID)
	}
	if got := eventToolCallID(t, sink.events[3]); got != startID {
		t.Fatalf("unexpected awaiting_human id: got %q want %q", got, startID)
	}
	for index, event := range sink.events[1:4] {
		if got := eventToolName(t, event); got != "ask_human" {
			t.Fatalf("unexpected event[%d] tool: got %q want %q", index+1, got, "ask_human")
		}
	}
}

func TestGraphQLTextTurnCommitsSuccessfulExecutionBeforeLaterCompletionError(t *testing.T) {
	completer := newFakeCompleter(newStopResponse(`query { web_search(query: "OpenAI") }`))
	tool := newStaticTool("web_search", `{"items":[{"title":"OpenAI"}]}`)
	tool.semantics = llm.ToolSemantics{ReadOnly: true}
	catalog := newFakeToolCatalog(tool)
	agent := newTestAgent(completer, catalog, 3)
	agent.SetStrictToolCallProtocol(true)
	agent.AddAssistantTextHandler(NewGraphQLTextTurnHandler(tools.NewGraphQLTextExecutor(catalog)))

	_, err := agent.Run(context.Background(), "hello")
	if err == nil {
		t.Fatal("expected completion error after graphql tool execution")
	}

	newMessages := agent.GetNewMessages()
	if len(newMessages) != 4 {
		t.Fatalf("expected committed graphql turn messages, got %+v", newMessages)
	}
	if newMessages[0].Role != llm.RoleUser || newMessages[0].Text != "hello" {
		t.Fatalf("unexpected committed user message: %+v", newMessages[0])
	}
	if newMessages[1].Role != llm.RoleAssistant || !strings.Contains(newMessages[1].Text, "web_search") {
		t.Fatalf("unexpected committed assistant graphql text: %+v", newMessages[1])
	}
	if newMessages[2].Role != llm.RoleTool {
		t.Fatalf("expected committed tool envelope, got %+v", newMessages[2])
	}
	if newMessages[3].Role != llm.RoleInternal || !strings.Contains(newMessages[3].Text, "[GRAPHQL_TOOL_RESULT]") {
		t.Fatalf("unexpected committed graphql feedback: %+v", newMessages[3])
	}
}

func TestGraphQLTextTurnFeedsStructuredProtocolErrorBackIntoNextRound(t *testing.T) {
	completer := newFakeCompleter(
		newStopResponse(`mutation { web_search(query: "OpenAI") }`),
		newStopResponse(`query { web_search(query: "OpenAI") }`),
		newStopResponse("done"),
	)
	tool := newStaticTool("web_search", `{"items":[{"title":"OpenAI"}]}`)
	tool.semantics = llm.ToolSemantics{ReadOnly: true}
	catalog := newFakeToolCatalog(tool)
	agent := newTestAgent(completer, catalog, 4)
	agent.SetStrictToolCallProtocol(true)
	agent.AddAssistantTextHandler(NewGraphQLTextTurnHandler(tools.NewGraphQLTextExecutor(catalog)))

	output, err := agent.Run(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if output != "done" {
		t.Fatalf("unexpected output: got %q want %q", output, "done")
	}
	if tool.callCount != 1 {
		t.Fatalf("expected web_search to execute once after correction, got %d", tool.callCount)
	}
	if len(completer.requests) != 3 {
		t.Fatalf("unexpected complete call count: got %d want %d", len(completer.requests), 3)
	}

	last := completer.requests[1].Messages[len(completer.requests[1].Messages)-1]
	if !strings.Contains(last.Text, "[GRAPHQL_TOOL_RESULT]") {
		t.Fatalf("expected graphql protocol error feedback in second request, got %+v", last)
	}
	payload := decodeGraphQLToolFeedback(t, last.Text)
	if payload["status"] != "error" {
		t.Fatalf("unexpected feedback status: %+v", payload)
	}
	if payload["kind"] != "wrong_operation" {
		t.Fatalf("unexpected feedback kind: %+v", payload)
	}
	if payload["tool"] != "web_search" {
		t.Fatalf("unexpected feedback tool: %+v", payload)
	}
	if payload["expected"] != "query" || payload["received"] != "mutation" {
		t.Fatalf("unexpected feedback operation payload: %+v", payload)
	}
}

func eventToolCallID(t *testing.T, event streaming.Event) string {
	t.Helper()

	payload, ok := event.Payload.(map[string]any)
	if !ok {
		t.Fatalf("unexpected payload type: %T", event.Payload)
	}
	toolCallID, _ := payload["tool_call_id"].(string)
	if toolCallID == "" {
		t.Fatalf("expected tool_call_id in payload: %+v", payload)
	}
	return toolCallID
}

func eventToolName(t *testing.T, event streaming.Event) string {
	t.Helper()

	payload, ok := event.Payload.(map[string]any)
	if !ok {
		t.Fatalf("unexpected payload type: %T", event.Payload)
	}
	toolName, _ := payload["tool"].(string)
	if toolName == "" {
		t.Fatalf("expected tool in payload: %+v", payload)
	}
	return toolName
}

func decodeGraphQLToolFeedback(t *testing.T, raw string) map[string]any {
	t.Helper()

	payloadText := strings.TrimSpace(strings.TrimPrefix(raw, "[GRAPHQL_TOOL_RESULT]"))
	var payload map[string]any
	if err := json.Unmarshal([]byte(payloadText), &payload); err != nil {
		t.Fatalf("decode graphql tool feedback: %v, raw=%q", err, raw)
	}
	return payload
}
