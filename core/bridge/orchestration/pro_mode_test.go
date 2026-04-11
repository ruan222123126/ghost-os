package orchestration

import (
	"context"
	"errors"
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

	request, matched, err = parseProModeRequest("prox 7 finish task", 20)
	if err != nil {
		t.Fatalf("parse prox mode request: %v", err)
	}
	if !matched {
		t.Fatal("expected prox prefix to match")
	}
	if request.Mode != proModeProx || request.MaxIterations != 7 || request.Unlimited {
		t.Fatalf("unexpected prox request: %+v", request)
	}
}

func TestExecuteAgentActionRunsFiniteProMode(t *testing.T) {
	handler, service, sessionStore := newTestHandlerWithService(t, nil, nil)
	_ = handler

	completer := &proTestCompleter{
		responses: []*llm.CompletionResponse{
			{
				Message: llm.Message{
					Role: llm.RoleAssistant,
					ToolCalls: []llm.ToolCall{{
						ID:        "call-1",
						Name:      "pro_update_record",
						Arguments: []byte(`{"did":"inspected config","remaining":"apply patch and run tests"}`),
					}},
				},
				FinishReason: llm.FinishToolCalls,
			},
			{
				Message: llm.Message{
					Role: llm.RoleAssistant,
					ToolCalls: []llm.ToolCall{{
						ID:        "call-2",
						Name:      "pro_complete",
						Arguments: []byte(`{"did":"patched config and ran tests","remaining":"none","final_message":"done","final_change_log":"updated config and verified tests"}`),
					}},
				},
				FinishReason: llm.FinishToolCalls,
			},
		},
	}
	service.runtimeFactory = proTestRuntimeFactory{
		deps: agentRuntimeDependencies{
			cfg: bridgeconfig.Config{
				MaxTurns:         4,
				ProMaxIterations: 2,
				PromptsPath:      "",
				Provider:         bridgeconfig.ProviderConfig{Model: "gpt-4o"},
			},
			client:       completer,
			registry:     tools.NewRegistry(),
			systemPrompt: "system prompt",
		},
	}

	payloadResult, err := service.executeAgentAction(context.Background(), agentParams{Message: "pro fix config"}, "trace-pro")
	if err != nil {
		t.Fatalf("executeAgentAction returned error: %v", err)
	}
	if code := legacyStatusFromServiceOutcome(payloadResult.Outcome); code != 200 {
		t.Fatalf("unexpected status code: %d", code)
	}

	payload, ok := payloadResult.Payload.(agentResponse)
	if !ok {
		t.Fatalf("unexpected payload type: %T", payloadResult.Payload)
	}
	if payload.Mode != proModePro {
		t.Fatalf("unexpected mode: %q", payload.Mode)
	}
	if payload.StoppedBy != proModeStopCompleted {
		t.Fatalf("unexpected stopped_by: %q", payload.StoppedBy)
	}
	if payload.IterationCount != 2 {
		t.Fatalf("unexpected iteration_count: %d", payload.IterationCount)
	}
	if payload.FinalChangeLog != "updated config and verified tests" {
		t.Fatalf("unexpected final change log: %q", payload.FinalChangeLog)
	}
	if len(payload.IterationSummary) != 2 {
		t.Fatalf("unexpected iteration summary length: %d", len(payload.IterationSummary))
	}

	if len(completer.requests) != 2 {
		t.Fatalf("unexpected complete request count: %d", len(completer.requests))
	}
	secondRequest := completer.requests[1]
	last := secondRequest.Messages[len(secondRequest.Messages)-1]
	if !strings.Contains(last.Text, "did: inspected config") {
		t.Fatalf("second iteration prompt should include prior record, got %q", last.Text)
	}

	if payload.SessionID == "" {
		t.Fatal("expected session_id")
	}
	loaded, err := sessionStore.Load(payload.SessionID)
	if err != nil {
		t.Fatalf("load saved session: %v", err)
	}
	if loaded.IterationRuntime == nil || loaded.IterationRuntime.Status != proModeStatusCompleted {
		t.Fatalf("unexpected iteration runtime: %+v", loaded.IterationRuntime)
	}
	if len(loaded.IterationRuntime.Records) != 2 {
		t.Fatalf("unexpected stored record count: %d", len(loaded.IterationRuntime.Records))
	}
	if loaded.Messages[len(loaded.Messages)-1].Role != llm.RoleAssistant {
		t.Fatalf("expected final assistant message, got %+v", loaded.Messages[len(loaded.Messages)-1])
	}
}

func TestExecuteAgentActionStopsAtProMaxIterations(t *testing.T) {
	_, service, _ := newTestHandlerWithService(t, nil, nil)

	completer := &proTestCompleter{
		responses: []*llm.CompletionResponse{
			{
				Message: llm.Message{
					Role: llm.RoleAssistant,
					ToolCalls: []llm.ToolCall{{
						ID:        "call-1",
						Name:      "pro_update_record",
						Arguments: []byte(`{"did":"inspected config","remaining":"apply patch"}`),
					}},
				},
				FinishReason: llm.FinishToolCalls,
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

	payloadResult, err := service.executeAgentAction(context.Background(), agentParams{Message: "pro 1 fix config"}, "trace-pro-limit")
	if err != nil {
		t.Fatalf("executeAgentAction returned error: %v", err)
	}
	if code := legacyStatusFromServiceOutcome(payloadResult.Outcome); code != 200 {
		t.Fatalf("unexpected status code: %d", code)
	}
	payload := payloadResult.Payload.(agentResponse)
	if payload.StoppedBy != proModeStopMaxLimit {
		t.Fatalf("unexpected stopped_by: %q", payload.StoppedBy)
	}
	if !strings.Contains(payload.Message, "max_iterations=1") {
		t.Fatalf("expected max limit message, got %q", payload.Message)
	}
}
