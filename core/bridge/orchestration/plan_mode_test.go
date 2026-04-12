package orchestration

import (
	"context"
	"strings"
	"testing"

	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/llm"
	"ghost-os/bridge/tools"
)

func TestExecuteAgentActionRunsPlanModeWithExplicitModePriority(t *testing.T) {
	_, service, _ := newTestHandlerWithService(t, nil, nil)
	completer := &proTestCompleter{
		responses: []*llm.CompletionResponse{
			{
				Message: llm.Message{
					Role: llm.RoleAssistant,
					Text: "【用户意图】\n- 拆解任务\n【任务编排】\n1. task_id=T1; objective=分析需求; inputs=用户消息; depends_on=none; executor=main_ai\n【执行顺序】\n1. 先分析再执行\n【完成判定】\n1. 任务清单完整",
				},
				FinishReason: llm.FinishStop,
			},
		},
	}
	service.runtimeFactory = proTestRuntimeFactory{
		deps: agentRuntimeDependencies{
			cfg: bridgeconfig.Config{
				MaxTurns: 4,
				Provider: bridgeconfig.ProviderConfig{Model: "gpt-4o"},
			},
			client:       completer,
			registry:     tools.NewRegistry(),
			systemPrompt: "system prompt",
		},
	}

	payloadAny, err := service.executeAgentAction(context.Background(), agentParams{
		Mode:    agentModePlan,
		Message: "pro fix config",
	}, "trace-plan")
	if err != nil {
		t.Fatalf("executeAgentAction returned error: %v", err)
	}
	if payloadAny.Outcome != ServiceOutcomeSuccess {
		t.Fatalf("unexpected outcome: %s", payloadAny.Outcome)
	}

	payload, ok := payloadAny.Payload.(agentResponse)
	if !ok {
		t.Fatalf("unexpected payload type: %T", payloadAny)
	}
	if payload.Mode != agentModePlan {
		t.Fatalf("unexpected mode: got %q want %q", payload.Mode, agentModePlan)
	}
	if payload.IterationCount != 0 || payload.StoppedBy != "" {
		t.Fatalf("plan mode should not carry pro fields: %+v", payload)
	}
	if len(completer.requests) != 1 {
		t.Fatalf("unexpected complete request count: %d", len(completer.requests))
	}
	req := completer.requests[0]
	if len(req.Tools) != 0 {
		t.Fatalf("plan mode must not expose tools, got %+v", req.Tools)
	}
	if got := req.Messages[len(req.Messages)-1].Text; got != "pro fix config" {
		t.Fatalf("unexpected user message: got %q want %q", got, "pro fix config")
	}
	if !strings.Contains(req.Messages[0].Text, "planning orchestrator") {
		t.Fatalf("unexpected system prompt: %q", req.Messages[0].Text)
	}
}

func TestExecuteAgentActionRejectsPlanModeToolCalls(t *testing.T) {
	_, service, _ := newTestHandlerWithService(t, nil, nil)
	completer := &proTestCompleter{
		responses: []*llm.CompletionResponse{
			{
				Message: llm.Message{
					Role: llm.RoleAssistant,
					ToolCalls: []llm.ToolCall{{
						ID:        "call-1",
						Name:      "script_exec",
						Arguments: []byte(`{"script":"echo hi"}`),
					}},
				},
				FinishReason: llm.FinishToolCalls,
			},
		},
	}
	service.runtimeFactory = proTestRuntimeFactory{
		deps: agentRuntimeDependencies{
			cfg: bridgeconfig.Config{
				MaxTurns: 4,
				Provider: bridgeconfig.ProviderConfig{Model: "gpt-4o"},
			},
			client:       completer,
			registry:     tools.NewRegistry(),
			systemPrompt: "system prompt",
		},
	}

	_, err := service.executeAgentAction(context.Background(), agentParams{
		Mode:    agentModePlan,
		Message: "make a plan",
	}, "trace-plan-toolcall")
	if err == nil {
		t.Fatal("expected error but got nil")
	}
	if code := legacyStatusFromServiceError(err); code != 500 {
		t.Fatalf("unexpected status code: got %d want %d", code, 500)
	}
	if !strings.Contains(err.Error(), "cannot contain tool calls") {
		t.Fatalf("unexpected error: %v", err)
	}
}
