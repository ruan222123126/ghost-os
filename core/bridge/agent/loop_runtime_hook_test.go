package agent

import (
	"context"
	"fmt"
	"testing"

	"ghost-os/bridge/llm"
)

func TestBeforeCompletionHookRefreshesSystemPromptPerCompletion(t *testing.T) {
	completer := newFakeCompleter(
		newToolCallsResponse(newToolCall("call-1", "echo", `{}`)),
		newStopResponse("done"),
	)
	tool := newStaticTool("echo", "ok")
	agent := newTestAgent(completer, newFakeToolCatalog(tool), 3)
	agent.SetBeforeCompletionHook(func(_ context.Context, turn int, history *History) error {
		if history == nil {
			t.Fatal("expected history")
		}
		history.UpdateSystemPrompt(fmt.Sprintf("system-%d", turn))
		return nil
	})

	output, err := agent.Run(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if output != "done" {
		t.Fatalf("unexpected output: %q", output)
	}
	if len(completer.requests) != 2 {
		t.Fatalf("expected two completion requests, got %d", len(completer.requests))
	}
	for index, request := range completer.requests {
		if len(request.Messages) == 0 {
			t.Fatalf("request[%d] has no messages", index)
		}
		if got := request.Messages[0]; got.Role != llm.RoleSystem || got.Text != fmt.Sprintf("system-%d", index) {
			t.Fatalf("unexpected system prompt in request[%d]: %+v", index, got)
		}
	}
}
