package orchestration

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/llm"
	"ghost-os/bridge/tools"
)

type proTestRuntimeFactory struct {
	deps agentRuntimeDependencies
	err  error
}

func (f proTestRuntimeFactory) Build(bridgeconfig.Store) (agentRuntimeDependencies, error) {
	if f.err != nil {
		return agentRuntimeDependencies{}, f.err
	}
	if strings.TrimSpace(f.deps.cfg.PromptsDir) == "" {
		f.deps.cfg.PromptsDir = os.Getenv("GHOST_PROMPTS_DIR")
	}
	return f.deps, nil
}

type proTestCompleter struct {
	responses []*llm.CompletionResponse
	requests  []llm.CompletionRequest
}

func (f *proTestCompleter) Complete(_ context.Context, request llm.CompletionRequest) (*llm.CompletionResponse, error) {
	f.requests = append(f.requests, request)
	if len(f.responses) == 0 {
		return nil, errors.New("unexpected complete call")
	}
	response := f.responses[0]
	f.responses = f.responses[1:]
	return response, nil
}

func TestParseProModeRequest(t *testing.T) {
	request, matched, err := parseProModeRequest("pro fix config", 20)
	if err != nil {
		t.Fatalf("parse pro mode request: %v", err)
	}
	if !matched {
		t.Fatal("expected pro prefix to match")
	}
	if request.Mode != proModePro || request.MaxIterations != 20 || request.OriginalTask != "fix config" {
		t.Fatalf("unexpected request: %+v", request)
	}

	request, matched, err = parseProModeRequest("pro 7 finish task", 20)
	if err != nil {
		t.Fatalf("parse explicit pro iterations: %v", err)
	}
	if !matched {
		t.Fatal("expected explicit pro prefix to match")
	}
	if request.Mode != proModePro || request.MaxIterations != 7 || request.OriginalTask != "finish task" {
		t.Fatalf("unexpected explicit pro request: %+v", request)
	}

	_, matched, err = parseProModeRequest("prox finish task", 20)
	if err == nil || !strings.Contains(err.Error(), "prox mode has been removed") {
		t.Fatalf("expected removed prox error, got matched=%t err=%v", matched, err)
	}
	if !matched {
		t.Fatal("removed prox prefix should be handled explicitly")
	}
}

func TestExecuteAgentActionTreatsProPrefixAsStandardMessage(t *testing.T) {
	_, service, sessionStore := newTestHandlerWithService(t, nil, nil)

	completer := &proTestCompleter{
		responses: []*llm.CompletionResponse{
			{
				Message:      llm.Message{Role: llm.RoleAssistant, Text: "standard response"},
				FinishReason: llm.FinishStop,
			},
		},
	}
	service.runtimeFactory = proTestRuntimeFactory{
		deps: agentRuntimeDependencies{
			cfg: bridgeconfig.Config{
				MaxTurns:         4,
				ProMaxIterations: 1,
				PromptsPath:      "",
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
