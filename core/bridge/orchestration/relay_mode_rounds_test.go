package orchestration

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/llm"
	"ghost-os/bridge/session"
	"ghost-os/bridge/tools"
)

func TestRelayAIDecidesStopsAtMaxRounds(t *testing.T) {
	completer := &proTestCompleter{
		responses: []*llm.CompletionResponse{
			relayUpdateResponse("call-relay-1", "round 1", "more"),
			relayUpdateResponse("call-relay-2", "round 2", "more"),
		},
	}
	result, err := runRelayModeTest(t, completer, TaskRelayConfig{
		StopPolicy: taskRelayStopPolicyAIDecides,
		MaxRounds:  2,
	})
	if err != nil {
		t.Fatalf("relay run failed: %v", err)
	}
	if result.StoppedBy != relayModeStopMaxRounds {
		t.Fatalf("unexpected stop reason: %s", result.StoppedBy)
	}
	if len(result.Records) != 2 || len(completer.requests) != 2 {
		t.Fatalf("unexpected rounds: records=%d requests=%d", len(result.Records), len(completer.requests))
	}
	if !strings.Contains(result.Message, "Reached relay max_rounds=2") {
		t.Fatalf("unexpected message: %q", result.Message)
	}
}

func TestRelayAIDecidesCanComplete(t *testing.T) {
	completer := &proTestCompleter{
		responses: []*llm.CompletionResponse{
			relayCompleteResponse("call-relay-complete"),
		},
	}
	result, err := runRelayModeTest(t, completer, TaskRelayConfig{
		StopPolicy: taskRelayStopPolicyAIDecides,
		MaxRounds:  5,
	})
	if err != nil {
		t.Fatalf("relay run failed: %v", err)
	}
	if result.StoppedBy != relayModeStopCompleted || result.Message != "done" {
		t.Fatalf("unexpected result: %+v", result)
	}
	if len(result.Records) != 1 || len(completer.requests) != 1 {
		t.Fatalf("unexpected rounds: records=%d requests=%d", len(result.Records), len(completer.requests))
	}
}

func TestRelayRepairsPlainTextRoundIntoHandoff(t *testing.T) {
	completer := &proTestCompleter{
		responses: []*llm.CompletionResponse{
			relayTextResponse("我已经查到了广州天气，接下来整理结果。"),
			relayUpdateResponse("call-relay-1", "查到了广州天气", "整理成最终答复"),
		},
	}
	result, err := runRelayModeTest(t, completer, TaskRelayConfig{
		StopPolicy: taskRelayStopPolicyMaxRounds,
		MaxRounds:  1,
	})
	if err != nil {
		t.Fatalf("relay run failed: %v", err)
	}
	if result.StoppedBy != relayModeStopMaxRounds {
		t.Fatalf("unexpected stop reason: %s", result.StoppedBy)
	}
	if len(result.Records) != 1 || len(completer.requests) != 2 {
		t.Fatalf("unexpected rounds: records=%d requests=%d", len(result.Records), len(completer.requests))
	}
	repairRequest := completer.requests[1]
	if repairRequest.ToolChoice != "required" {
		t.Fatalf("unexpected repair tool_choice: %+v", repairRequest)
	}
	lastMessage := repairRequest.Messages[len(repairRequest.Messages)-1]
	if !strings.Contains(lastMessage.Text, "invalid for relay mode") {
		t.Fatalf("unexpected repair prompt: %q", lastMessage.Text)
	}
}

func TestRelayReturnsErrorWhenRepairStillDoesNotHandoff(t *testing.T) {
	completer := &proTestCompleter{
		responses: []*llm.CompletionResponse{
			relayTextResponse("先给你一个普通答复。"),
			relayTextResponse("我还是直接回答。"),
		},
	}
	_, err := runRelayModeTest(t, completer, TaskRelayConfig{
		StopPolicy: taskRelayStopPolicyMaxRounds,
		MaxRounds:  1,
	})
	if err == nil {
		t.Fatal("expected relay run to fail")
	}
	if !strings.Contains(err.Error(), "ended without relay handoff tool") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func runRelayModeTest(
	t *testing.T,
	completer *proTestCompleter,
	relay TaskRelayConfig,
) (relayModeResult, error) {
	t.Helper()

	store := newTempSessionStore(t)
	sess := session.NewSession("system")
	runner := relayModeRunner{sessionStore: store}
	deps := agentRuntimeDependencies{
		cfg: bridgeconfig.Config{
			MaxTurns: 4,
		},
		client:               completer,
		registry:             tools.NewRegistry(),
		systemPrompt:         "system",
		systemPromptOverride: true,
	}
	return runner.run(context.Background(), relayRunDeps{
		deps:    deps,
		relay:   relay,
		task:    ScheduledTask{Message: "finish task"},
		session: sess,
		traceID: "trace-relay-test",
	})
}

func relayUpdateResponse(callID string, did string, remaining string) *llm.CompletionResponse {
	return relayToolResponse(callID, "relay_update_record", map[string]any{
		"did":             did,
		"remaining":       remaining,
		"failed_attempts": []string{},
		"next_step":       "continue",
	})
}

func relayCompleteResponse(callID string) *llm.CompletionResponse {
	return relayToolResponse(callID, "relay_complete", map[string]any{
		"did":              "finished",
		"remaining":        "none",
		"failed_attempts":  []string{},
		"next_step":        "none",
		"final_message":    "done",
		"final_change_log": "finished task",
	})
}

func relayToolResponse(callID string, name string, args map[string]any) *llm.CompletionResponse {
	encoded, err := json.Marshal(args)
	if err != nil {
		panic(err)
	}
	return &llm.CompletionResponse{
		Message: llm.Message{
			Role: llm.RoleAssistant,
			ToolCalls: []llm.ToolCall{{
				ID:        callID,
				Name:      name,
				Arguments: encoded,
			}},
		},
		FinishReason: llm.FinishToolCalls,
	}
}

func relayTextResponse(text string) *llm.CompletionResponse {
	return &llm.CompletionResponse{
		Message: llm.Message{
			Role: llm.RoleAssistant,
			Text: text,
		},
		FinishReason: llm.FinishStop,
	}
}
