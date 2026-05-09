package agent

import (
	"context"
	"testing"

	"ghost-os/bridge/orchestration/internal/ports"
	bridgeTasks "ghost-os/bridge/tasks"
)

func TestMemberAgentRunnerMapsSuccessAndRuntimeSnapshot(t *testing.T) {
	allowlistOnly := true
	runner := MemberAgentRunner{
		Invoker: fakeActionInvoker{payload: ports.AgentActionPayload{
			Kind:      ports.AgentActionPayloadSuccess,
			SessionID: "session-1",
			Message:   "ok",
		}},
	}

	result, err := runner.RunMemberTurn(context.Background(), ports.MemberTurnRequest{
		Round:     1,
		AgentID:   "agent-1",
		Title:     "A",
		Message:   "speak",
		SessionID: "session-0",
		RuntimeOverrides: &bridgeTasks.TaskRuntimeOverrides{
			ProviderName:      "anthropic-main",
			Model:             "claude",
			PresetID:          "preset-1",
			ToolAllowlistOnly: &allowlistOnly,
			ToolAllowlist:     []string{"script_exec"},
		},
	})

	if err != nil {
		t.Fatalf("run member turn: %v", err)
	}
	if result.Status != bridgeTasks.RunStatusSuccess || result.Content != "ok" {
		t.Fatalf("unexpected member result: %#v", result)
	}
	if result.RuntimeOverrides["preset_id"] != "preset-1" {
		t.Fatalf("expected runtime snapshot, got %#v", result.RuntimeOverrides)
	}
}

func TestMemberAgentRunnerMapsUnsupportedPayload(t *testing.T) {
	runner := MemberAgentRunner{
		Invoker: fakeActionInvoker{payload: ports.AgentActionPayload{
			Kind:     ports.AgentActionPayloadUnsupported,
			TypeName: "customPayload",
		}},
	}

	result, err := runner.RunMemberTurn(context.Background(), ports.MemberTurnRequest{
		Round:   1,
		AgentID: "agent-1",
		Title:   "A",
		Message: "speak",
	})

	if err != nil {
		t.Fatalf("run member turn: %v", err)
	}
	if result.Status != bridgeTasks.RunStatusError ||
		result.Error != "unsupported orchestration member response customPayload" {
		t.Fatalf("unexpected unsupported result: %#v", result)
	}
}

type fakeActionInvoker struct {
	payload ports.AgentActionPayload
	err     error
}

func (f fakeActionInvoker) ExecuteAgentAction(
	_ context.Context,
	_ ports.AgentActionRequest,
) (ports.AgentActionPayload, error) {
	return f.payload, f.err
}
