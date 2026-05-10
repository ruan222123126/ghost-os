package orchestration

import (
	"context"
	"testing"

	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/llm"
	"ghost-os/bridge/tools"
)

func TestExecuteAgentActionTreatsProPrefixAsStandardMessage(t *testing.T) {
	_, service, sessionStore := newTestHandlerWithService(t, nil, nil)

	completer := &testCompleter{
		responses: []*llm.CompletionResponse{
			{
				Message:      llm.Message{Role: llm.RoleAssistant, Text: "standard response"},
				FinishReason: llm.FinishStop,
			},
		},
	}
	service.runtimeFactory = testRuntimeFactory{
		deps: agentRuntimeDependencies{
			cfg: bridgeconfig.Config{
				MaxTurns:         4,
				ProMaxIterations: 1,
				Provider:         bridgeconfig.ProviderConfig{Model: "gpt-4o"},
			},
			client:       completer,
			registry:     tools.NewRegistry(),
			systemPrompt: "system prompt",
		},
	}
	service.agentRunner = NewSessionAgentRunner(
		service.runtimeFactory,
		service.configStore,
		sessionStore,
		service.runRegistry,
	)

	payloadResult, err := service.executeAgentAction(context.Background(), agentParams{Message: "pro 1 fix config"}, "trace-pro-limit")
	if err != nil {
		t.Fatalf("executeAgentAction returned error: %v", err)
	}
	if payloadResult.Outcome != ServiceOutcomeSuccess {
		t.Fatalf("unexpected outcome: %s", payloadResult.Outcome)
	}
	payload := payloadResult.Payload.(agentResponse)
	if payload.Mode != "" || payload.StoppedBy != "" || payload.IterationCount != 0 {
		t.Fatalf("expected standard response without pro fields, got %+v", payload)
	}
	if payload.Message != "standard response" {
		t.Fatalf("unexpected message: %q", payload.Message)
	}
	if len(completer.requests) != 1 {
		t.Fatalf("unexpected complete request count: %d", len(completer.requests))
	}
	last := completer.requests[0].Messages[len(completer.requests[0].Messages)-1]
	if last.Text != "pro 1 fix config" {
		t.Fatalf("unexpected user message: %q", last.Text)
	}
}
