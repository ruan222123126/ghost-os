package agent

import (
	"context"
	"errors"
	"strings"
	"testing"

	"ghost-os/bridge/llm"
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
