package orchestration

import (
	"context"
	"strings"
	"testing"

	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/llm"
	"ghost-os/bridge/session"
	"ghost-os/bridge/tools"
)

func TestOrchestrationOwnerPrivateSendReachesOnlyRecipient(t *testing.T) {
	executor := privateSendRecipientExecutor()
	_, service, _ := newTestHandlerWithService(t, executor, nil)
	installPrivateSendOwnerRuntime(service)

	run := runOrchestrationTaskNow(t, service, buildOwnerDefinition("agent-1", 3))
	groupOutput := findNodeOutput(t, run.Run.NodeResults, "group-1")
	dispatches := groupOutput["dispatch_results"].([]any)
	first := dispatches[0].(map[string]any)
	if first["action"] != "private_send" {
		t.Fatalf("expected private_send first, got %#v", first)
	}
	deliveries := first["private_deliveries"].([]any)
	if len(deliveries) != 1 || deliveries[0].(map[string]any)["content"] != "secret-role" {
		t.Fatalf("expected visible private delivery content, got %#v", first)
	}
	member := dispatches[1].(map[string]any)["member_results"].([]any)[0].(map[string]any)
	if member["content"] != "saw-secret" {
		t.Fatalf("expected recipient to see private content, got %#v", member)
	}
	assertRunTranscriptSession(t, service, run.Run.SessionIDOutput, nil, []string{"私聊投递: agent-2 <- secret-role"})
	loaded, err := service.sessionStore.Load(run.Run.SessionIDOutput)
	if err != nil {
		t.Fatalf("load run transcript session %q: %v", run.Run.SessionIDOutput, err)
	}
	transcript := sessionMessagesText(loaded.Messages)
	required := []string{
		"tool_call: orchestration_dispatch {\"action\":\"private_send\",\"private_messages\":[{\"content\":\"secret-role\",\"participant_id\":\"agent-2\"}]}",
		"tool: {\"status\":\"success\",\"tool\":\"orchestration_dispatch\"",
		"secret-role",
	}
	for _, expected := range required {
		if !strings.Contains(transcript, expected) {
			t.Fatalf("run transcript missing %q in:\n%s", expected, transcript)
		}
	}
}

func privateSendRecipientExecutor() agentExecutorFunc {
	return func(_ context.Context, message string, _ string, _ string, _ bridgeconfig.Store, _ *session.Store) (string, string, error) {
		if strings.Contains(message, "secret-role") {
			return "saw-secret", "member-session", nil
		}
		return "missing-secret", "member-session", nil
	}
}

func installPrivateSendOwnerRuntime(service *bridgeService) {
	completer := &proTestCompleter{responses: []*llm.CompletionResponse{
		dispatchToolResponse("dispatch-1", `{"action":"private_send","private_messages":[{"participant_id":"agent-2","content":"secret-role"}]}`),
		dispatchToolResponse("dispatch-2", `{"action":"public_once","participant_ids":["agent-2"],"instruction":"speak now"}`),
		dispatchToolResponse("dispatch-3", `{"action":"end_group"}`),
	}}
	service.runtimeFactory = proTestRuntimeFactory{
		deps: agentRuntimeDependencies{
			cfg:          bridgeconfig.Config{MaxTurns: 4, Provider: bridgeconfig.ProviderConfig{Model: "gpt-5.4"}},
			client:       completer,
			registry:     tools.NewRegistry(),
			systemPrompt: "system prompt",
		},
	}
}
