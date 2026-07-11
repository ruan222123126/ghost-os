package agent

import (
	"context"
	"errors"
	"strings"
	"testing"

	"ghost-os/bridge/llm"
	"ghost-os/bridge/streaming"
)

func TestRunToolTurnCommitsMessagesBeforeLaterCompletionError(t *testing.T) {
	tool := newStaticTool("echo", "tool-ok")
	completer := newFakeCompleter(
		newToolCallsResponse(newToolCall("call-1", "echo", `{"input":"hi"}`)),
	)
	agent := newTestAgent(completer, newFakeToolCatalog(tool), 3)

	_, err := agent.Run(context.Background(), "hello")
	if err == nil {
		t.Fatal("expected completion error after tool turn")
	}

	newMessages := agent.GetNewMessages()
	if len(newMessages) != 3 {
		t.Fatalf("expected committed tool turn messages, got %+v", newMessages)
	}
	if newMessages[0].Role != llm.RoleUser || newMessages[0].Text != "hello" {
		t.Fatalf("unexpected committed user message: %+v", newMessages[0])
	}
	if newMessages[1].Role != llm.RoleAssistant || len(newMessages[1].ToolCalls) != 1 {
		t.Fatalf("unexpected committed assistant tool call: %+v", newMessages[1])
	}
	if newMessages[2].Role != llm.RoleTool || newMessages[2].ToolCallID != "call-1" {
		t.Fatalf("unexpected committed tool result: %+v", newMessages[2])
	}

	toolResult := decodeToolResult(t, newMessages[2].Text)
	if toolResult.Status != "success" || toolResult.Tool != "echo" || toolResult.Output != "tool-ok" {
		t.Fatalf("unexpected tool result payload: %+v", toolResult)
	}
}

func TestRunToolTurnCommitsMessagesBeforeMaxTurnsExceeded(t *testing.T) {
	tool := newStaticTool("echo", "tool-ok")
	completer := newFakeCompleter(
		newToolCallsResponse(newToolCall("call-1", "echo", `{"input":"hi"}`)),
	)
	agent := newTestAgent(completer, newFakeToolCatalog(tool), 1)

	_, err := agent.Run(context.Background(), "hello")
	if err == nil {
		t.Fatal("expected max turns error after tool turn")
	}
	if !strings.Contains(err.Error(), "max turns exceeded: 1") {
		t.Fatalf("unexpected error: %v", err)
	}

	newMessages := agent.GetNewMessages()
	if len(newMessages) != 3 {
		t.Fatalf("expected committed tool turn messages, got %+v", newMessages)
	}
	if newMessages[2].Role != llm.RoleTool || newMessages[2].ToolCallID != "call-1" {
		t.Fatalf("unexpected committed tool result: %+v", newMessages[2])
	}
}

func TestRunToolFailureCommitsToolResultWhenFinishEventEmitFails(t *testing.T) {
	toolErr := errors.New("native shutdown")
	sinkErr := errors.New("client disconnected")
	tool := newErrorTool("codex_cli", toolErr)
	completer := newFakeCompleter(
		newToolCallsResponse(newToolCall("call-1", "codex_cli", `{"op":"status"}`)),
	)
	agent := newTestAgent(completer, newFakeToolCatalog(tool), 2)
	sink := failOnEventTypeSink{eventType: streaming.EventToolCallFinished, err: sinkErr}

	_, err := agent.RunMessageStreamWithTraceID(
		context.Background(),
		llm.Message{Role: llm.RoleUser, Text: "check status"},
		"trace-tool-finish-emit",
		sink,
	)

	if !errors.Is(err, sinkErr) {
		t.Fatalf("expected sink error, got %v", err)
	}
	newMessages := agent.GetNewMessages()
	if len(newMessages) != 3 {
		t.Fatalf("expected committed user/assistant/tool messages, got %+v", newMessages)
	}
	if newMessages[2].Role != llm.RoleTool || newMessages[2].ToolCallID != "call-1" {
		t.Fatalf("unexpected committed tool result: %+v", newMessages[2])
	}
	result := decodeToolResult(t, newMessages[2].Text)
	if result.Status != "error" || !strings.Contains(result.Error, toolErr.Error()) {
		t.Fatalf("unexpected tool error result: %+v", result)
	}
}

type failOnEventTypeSink struct {
	eventType streaming.EventType
	err       error
}

func (f failOnEventTypeSink) Emit(_ context.Context, event streaming.Event) (streaming.Event, error) {
	if event.Type == f.eventType {
		return streaming.Event{}, f.err
	}
	return event, nil
}
