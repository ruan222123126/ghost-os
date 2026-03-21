package agent

import (
	"context"
	"errors"
	"strings"
	"testing"

	"ghost-os/bridge/llm"
	"ghost-os/bridge/streaming"
	"ghost-os/bridge/tools"
)

type capturingAwaitingGraphQLTextExecutor struct {
	toolCallIDs []string
}

func (e *capturingAwaitingGraphQLTextExecutor) Execute(
	ctx context.Context,
	_ string,
	_ string,
) (tools.GraphQLTextExecutionResult, error) {
	e.toolCallIDs = append(e.toolCallIDs, tools.ToolCallIDFromContext(ctx))
	return tools.GraphQLTextExecutionResult{
		Recognized: true,
		Meta: tools.ExecuteMeta{
			AwaitingHuman: &tools.AwaitingHumanSignal{
				QuestionID: "q-1",
				Prompt:     "Approve write?",
			},
		},
	}, nil
}

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
		newStopResponse("query { viewer { id } }"),
		newStopResponse("done"),
	)
	executor := &fakeGraphQLTextExecutor{
		results: []tools.GraphQLTextExecutionResult{{
			Recognized: true,
			Output:     `{"data":{"viewer":{"id":"1"}}}`,
		}},
	}
	agent := newTestAgent(completer, newFakeToolCatalog(), 3)
	agent.SetStrictToolCallProtocol(true)
	agent.AddAssistantTextHandler(NewGraphQLTextTurnHandler(executor))

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
	if !strings.Contains(last.Text, "[GRAPHQL_EXECUTION_RESULT]") {
		t.Fatalf("expected graphql execution feedback in second request, got %+v", last)
	}
}

func TestGraphQLTextTurnReturnsAwaitingHumanSignal(t *testing.T) {
	completer := newFakeCompleter(newStopResponse("mutation { updateViewer(input:{id:\"1\"}) { ok } }"))
	executor := &fakeGraphQLTextExecutor{
		results: []tools.GraphQLTextExecutionResult{{
			Recognized: true,
			Meta: tools.ExecuteMeta{
				AwaitingHuman: &tools.AwaitingHumanSignal{
					QuestionID: "q-1",
					Prompt:     "Approve write?",
				},
			},
		}},
	}
	agent := newTestAgent(completer, newFakeToolCatalog(), 2)
	agent.SetStrictToolCallProtocol(true)
	agent.AddAssistantTextHandler(NewGraphQLTextTurnHandler(executor))

	_, err := agent.Run(context.Background(), "hello")
	var awaitingErr *ErrAwaitingHuman
	if !errors.As(err, &awaitingErr) {
		t.Fatalf("expected ErrAwaitingHuman, got %v", err)
	}
	if awaitingErr.QuestionID != "q-1" {
		t.Fatalf("unexpected awaiting question id: %q", awaitingErr.QuestionID)
	}
}

func TestGraphQLTextTurnAwaitingHumanEventMatchesExecutorToolCallID(t *testing.T) {
	completer := newFakeCompleter(newStopResponse("mutation { updateViewer(input:{id:\"1\"}) { ok } }"))
	executor := &capturingAwaitingGraphQLTextExecutor{}
	agent := newTestAgent(completer, newFakeToolCatalog(), 2)
	sink := newRecordingEventSink()
	agent.SetStrictToolCallProtocol(true)
	agent.AddAssistantTextHandler(NewGraphQLTextTurnHandler(executor))

	_, err := agent.RunStreamWithTraceID(context.Background(), "hello", "trace-await", sink)
	var awaitingErr *ErrAwaitingHuman
	if !errors.As(err, &awaitingErr) {
		t.Fatalf("expected ErrAwaitingHuman, got %v", err)
	}
	if len(sink.events) != 4 {
		t.Fatalf("unexpected event count: got %d want 4", len(sink.events))
	}
	if len(executor.toolCallIDs) != 1 || executor.toolCallIDs[0] == "" {
		t.Fatalf("expected executor to observe tool call id, got %+v", executor.toolCallIDs)
	}
	want := executor.toolCallIDs[0]
	if got := eventToolCallID(t, sink.events[1]); got != want {
		t.Fatalf("unexpected tool_call_started id: got %q want %q", got, want)
	}
	if got := eventToolCallID(t, sink.events[2]); got != want {
		t.Fatalf("unexpected tool_call_finished id: got %q want %q", got, want)
	}
	if got := eventToolCallID(t, sink.events[3]); got != want {
		t.Fatalf("unexpected awaiting_human id: got %q want %q", got, want)
	}
	for index, event := range sink.events[1:4] {
		if got := eventToolName(t, event); got != tools.GraphQLTextMutationToolName {
			t.Fatalf(
				"unexpected event[%d] tool: got %q want %q",
				index+1,
				got,
				tools.GraphQLTextMutationToolName,
			)
		}
	}
}

func TestGraphQLTextTurnCommitsSuccessfulExecutionBeforeLaterCompletionError(t *testing.T) {
	completer := newFakeCompleter(
		newStopResponse(`mutation { updateViewer(input: {id: "1"}) { ok } }`),
	)
	executor := &fakeGraphQLTextExecutor{
		results: []tools.GraphQLTextExecutionResult{{
			Recognized: true,
			Output:     `{"status":"executed","intent_id":"intent-1"}`,
		}},
	}
	agent := newTestAgent(completer, newFakeToolCatalog(), 3)
	agent.SetStrictToolCallProtocol(true)
	agent.AddAssistantTextHandler(NewGraphQLTextTurnHandler(executor))

	_, err := agent.Run(context.Background(), "hello")
	if err == nil {
		t.Fatal("expected completion error after graphql text execution")
	}

	newMessages := agent.GetNewMessages()
	if len(newMessages) != 3 {
		t.Fatalf("expected committed graphql turn messages, got %+v", newMessages)
	}
	if newMessages[0].Role != llm.RoleUser || newMessages[0].Text != "hello" {
		t.Fatalf("unexpected committed user message: %+v", newMessages[0])
	}
	if newMessages[1].Role != llm.RoleAssistant || !strings.Contains(newMessages[1].Text, "updateViewer") {
		t.Fatalf("unexpected committed assistant graphql text: %+v", newMessages[1])
	}
	if newMessages[2].Role != llm.RoleInternal || !strings.Contains(newMessages[2].Text, "[GRAPHQL_EXECUTION_RESULT]") {
		t.Fatalf("unexpected committed graphql feedback: %+v", newMessages[2])
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
