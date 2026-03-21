package agent

import (
	"context"
	"strings"
	"testing"

	"ghost-os/bridge/llm"
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
