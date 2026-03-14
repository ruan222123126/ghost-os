package agent

import (
	"context"
	"encoding/json"
	"testing"

	"ghost-os/bridge/tools"
)

func TestRunReturnsIterationHandoff(t *testing.T) {
	tool := &fakeTool{
		name: "pro_update_record",
		execute: func(_ context.Context, _ json.RawMessage) (string, error) {
			return `{"status":"iteration_recorded","did":"inspected config","remaining":"apply patch"}`, nil
		},
		interpret: func(_ string) tools.ExecuteMeta {
			return tools.ExecuteMeta{
				Iteration: &tools.IterationHandoffSignal{
					Did:       "inspected config",
					Remaining: "apply patch",
				},
			}
		},
	}
	completer := newFakeCompleter(newToolCallsResponse(newToolCall("call-pro", "pro_update_record", `{"did":"inspected config","remaining":"apply patch"}`)))
	agent := newTestAgent(completer, newFakeToolCatalog(tool), 2)

	_, err := agent.Run(context.Background(), "pro fix config")
	if err == nil {
		t.Fatal("expected iteration handoff error")
	}

	handoffErr, ok := err.(*ErrIterationHandoff)
	if !ok {
		t.Fatalf("unexpected error type: %T", err)
	}
	if handoffErr.Did != "inspected config" || handoffErr.Remaining != "apply patch" {
		t.Fatalf("unexpected handoff payload: %+v", handoffErr)
	}
}
