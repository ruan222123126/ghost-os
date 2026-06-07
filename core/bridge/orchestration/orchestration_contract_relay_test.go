package orchestration

import (
	"context"
	"encoding/json"
	"errors"

	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/llm"
	apprelay "ghost-os/bridge/orchestration/internal/app/agentturn/relay"
	"ghost-os/bridge/session"
	"ghost-os/bridge/tools"
	"net/http"
	"strings"
	"testing"
)

func TestRelayTaskCreateAppliesConfiguredDefaults(t *testing.T) {
	_, service, _ := newTestHandlerWithService(t, nil, nil)
	_, err := service.executeConfigUpdateAction(configUpdateRequest{
		RelayDefaultStopPolicy:         stringPointer(taskRelayStopPolicyMaxRounds),
		RelayDefaultMaxRounds:          intPointer(7),
		RelayDefaultExecutionTimeoutMs: intPointer(0),
	}, "trace-relay-defaults-config")
	if err != nil {
		t.Fatalf("config update failed: %v", err)
	}

	createdRaw, code, err := service.executeTaskCreateAction(taskCreateParams{
		TaskKind:        taskKindAgentMessage,
		Message:         "run relay task",
		AgentMode:       taskAgentModeRelay,
		IntervalSeconds: 60,
	}, "trace-relay-defaults-task")
	if err != nil {
		t.Fatalf("task create failed: %v", err)
	}
	if code != http.StatusCreated {
		t.Fatalf("unexpected status code: got %d want %d", code, http.StatusCreated)
	}
	created := createdRaw.(taskPayload)
	if created.Relay == nil {
		t.Fatal("expected relay defaults")
	}
	if created.Relay.StopPolicy != taskRelayStopPolicyMaxRounds || created.Relay.MaxRounds != 7 {
		t.Fatalf("unexpected relay defaults: %+v", created.Relay)
	}
	if created.Relay.ExecutionTimeoutMS == nil || *created.Relay.ExecutionTimeoutMS != 0 {
		t.Fatalf("unexpected relay execution timeout: %+v", created.Relay.ExecutionTimeoutMS)
	}
}

func TestRelayTaskCreateAppliesAIDecidesMaxRoundDefault(t *testing.T) {
	_, service, _ := newTestHandlerWithService(t, nil, nil)
	_, err := service.executeConfigUpdateAction(configUpdateRequest{
		RelayDefaultStopPolicy:         stringPointer(taskRelayStopPolicyAIDecides),
		RelayDefaultMaxRounds:          intPointer(6),
		RelayDefaultExecutionTimeoutMs: intPointer(0),
	}, "trace-relay-ai-defaults-config")
	if err != nil {
		t.Fatalf("config update failed: %v", err)
	}

	createdRaw, code, err := service.executeTaskCreateAction(taskCreateParams{
		TaskKind:        taskKindAgentMessage,
		Message:         "run relay task",
		AgentMode:       taskAgentModeRelay,
		IntervalSeconds: 60,
	}, "trace-relay-ai-defaults-task")
	if err != nil {
		t.Fatalf("task create failed: %v", err)
	}
	if code != http.StatusCreated {
		t.Fatalf("unexpected status code: got %d want %d", code, http.StatusCreated)
	}
	created := createdRaw.(taskPayload)
	if created.Relay == nil {
		t.Fatal("expected relay defaults")
	}
	if created.Relay.StopPolicy != taskRelayStopPolicyAIDecides || created.Relay.MaxRounds != 6 {
		t.Fatalf("unexpected relay defaults: %+v", created.Relay)
	}
}

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
	if result.StoppedBy != apprelay.StopMaxRounds {
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
	if result.StoppedBy != apprelay.StopCompleted || result.Message != "done" {
		t.Fatalf("unexpected result: %+v", result)
	}
	if len(result.Records) != 1 || len(completer.requests) != 1 {
		t.Fatalf("unexpected rounds: records=%d requests=%d", len(result.Records), len(completer.requests))
	}
}

func TestRelayClearsTurnDraftOnCompletionHandoff(t *testing.T) {
	store := newTempSessionStore(t)
	sess := session.NewSession("system")
	completer := &proTestCompleter{
		responses: []*llm.CompletionResponse{
			relayCompleteResponse("call-relay-complete"),
		},
	}

	_, err := runRelayModeTestWithSession(t, store, sess, completer, TaskRelayConfig{
		StopPolicy: taskRelayStopPolicyAIDecides,
		MaxRounds:  5,
	})
	if err != nil {
		t.Fatalf("relay run failed: %v", err)
	}

	assertRelayTurnDraftCleared(t, store, sess.ID)
}

func TestRelayClearsTurnDraftOnCancellation(t *testing.T) {
	store := newTempSessionStore(t)
	sess := session.NewSession("system")
	completer := &draftStreamingCompleter{
		deltas: []llm.LLMDelta{
			{Kind: llm.DeltaKindThinking, Thinking: "still planning"},
		},
		runErr: context.Canceled,
	}

	_, err := runRelayModeTestWithSession(t, store, sess, completer, TaskRelayConfig{
		StopPolicy: taskRelayStopPolicyAIDecides,
		MaxRounds:  5,
	})
	if err == nil {
		t.Fatal("expected relay run to be cancelled")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("unexpected relay cancellation error: %v", err)
	}

	assertRelayTurnDraftCleared(t, store, sess.ID)
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
	if result.StoppedBy != apprelay.StopMaxRounds {
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

func TestRelayReturnsProtocolErrorToAIWhenRepairStillDoesNotHandoff(t *testing.T) {
	completer := &proTestCompleter{
		responses: []*llm.CompletionResponse{
			relayTextResponse("先给你一个普通答复。"),
			relayTextResponse("我还是直接回答。"),
			relayUpdateResponse("call-relay-1", "按错误提示补交接力记录", "继续下一轮"),
		},
	}
	result, err := runRelayModeTest(t, completer, TaskRelayConfig{
		StopPolicy: taskRelayStopPolicyMaxRounds,
		MaxRounds:  1,
	})
	if err != nil {
		t.Fatalf("relay run failed: %v", err)
	}
	if len(result.Records) != 1 || len(completer.requests) != 3 {
		t.Fatalf("unexpected rounds: records=%d requests=%d", len(result.Records), len(completer.requests))
	}
	correctionRequest := completer.requests[2]
	if correctionRequest.ToolChoice != "required" {
		t.Fatalf("unexpected correction tool_choice: %+v", correctionRequest)
	}
	lastMessage := correctionRequest.Messages[len(correctionRequest.Messages)-1]
	if !strings.Contains(lastMessage.Text, "relay round 1 ended without relay handoff tool") {
		t.Fatalf("unexpected correction prompt: %q", lastMessage.Text)
	}
}

func TestRelayReturnsErrorWhenProtocolCorrectionStillDoesNotHandoff(t *testing.T) {
	completer := &proTestCompleter{
		responses: []*llm.CompletionResponse{
			relayTextResponse("先给你一个普通答复。"),
			relayTextResponse("我还是直接回答。"),
			relayTextResponse("第三次仍然没有调用工具。"),
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
	if len(completer.requests) != 3 {
		t.Fatalf("unexpected request count: got %d want 3", len(completer.requests))
	}
}

func runRelayModeTest(
	t *testing.T,
	completer *proTestCompleter,
	relay TaskRelayConfig,
) (apprelay.Result, error) {
	t.Helper()

	store := newTempSessionStore(t)
	sess := session.NewSession("system")
	return runRelayModeTestWithSession(t, store, sess, completer, relay)
}

func runRelayModeTestWithSession(
	t *testing.T,
	store *session.Store,
	sess *session.Session,
	completer llm.Completer,
	relay TaskRelayConfig,
) (apprelay.Result, error) {
	t.Helper()

	runner := apprelay.Runner{SessionStore: store}
	deps := apprelay.RuntimeDependencies{
		Config: bridgeconfig.Config{
			MaxTurns: 4,
		},
		Client:               completer,
		Registry:             tools.NewRegistry(),
		SystemPrompt:         "system",
		SystemPromptOverride: true,
	}
	return runner.Run(context.Background(), apprelay.RunRequest{
		Deps:    deps,
		Relay:   relay,
		Task:    ScheduledTask{Message: "finish task"},
		Session: sess,
		TraceID: "trace-relay-test",
	})
}

func assertRelayTurnDraftCleared(t *testing.T, store *session.Store, sessionID string) {
	t.Helper()

	loaded, err := store.Load(sessionID)
	if err != nil {
		t.Fatalf("load relay session: %v", err)
	}
	if loaded.TurnDraft != nil {
		t.Fatalf("expected relay turn_draft to be cleared, got %+v", loaded.TurnDraft)
	}
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
