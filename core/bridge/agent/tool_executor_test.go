package agent

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"ghost-os/bridge/llm"
	"ghost-os/bridge/streaming"
	"ghost-os/bridge/tools"
)

func TestToolCallExecutorExecuteReturnsAwaitingHumanAfterSuccessEvent(t *testing.T) {
	sink := newRecordingEventSink()
	history := NewHistory("")
	executor := newToolCallExecutor(newFakeToolCatalog(newAwaitingHumanTool("approval_gate", "q-123", "Which database?")), history, nil, newAgentEventEmitter(sink, nil))

	stats, err := executor.execute(context.Background(), "trace-await", 2, []indexedToolCall{
		{index: 1, call: newToolCall("call-ask-1", "approval_gate", `{"prompt":"Which database?"}`)},
	})
	if stats.totalCalls != 1 || stats.executed != 1 {
		t.Fatalf("unexpected stats: %+v", stats)
	}

	var awaitingErr *ErrAwaitingHuman
	if !errors.As(err, &awaitingErr) {
		t.Fatalf("expected ErrAwaitingHuman, got %v", err)
	}
	if awaitingErr.QuestionID != "q-123" || awaitingErr.Prompt != "Which database?" {
		t.Fatalf("unexpected awaiting payload: %+v", awaitingErr)
	}
	if len(history.Messages()) != 0 {
		t.Fatalf("awaiting-human path should not append tool result: %+v", history.Messages())
	}
	if len(sink.events) != 3 {
		t.Fatalf("unexpected event count: got %d want %d", len(sink.events), 3)
	}

	stepID, err := streaming.ToolStepID(2, 1)
	if err != nil {
		t.Fatalf("ToolStepID returned error: %v", err)
	}
	for index, event := range sink.events {
		if event.StepID != stepID {
			t.Fatalf("unexpected step id for event[%d]: got %q want %q", index, event.StepID, stepID)
		}
	}
	if sink.events[0].Type != streaming.EventToolCallStarted {
		t.Fatalf("unexpected first event: %q", sink.events[0].Type)
	}
	if sink.events[1].Type != streaming.EventToolCallFinished {
		t.Fatalf("unexpected second event: %q", sink.events[1].Type)
	}
	if sink.events[2].Type != streaming.EventAwaitingHuman {
		t.Fatalf("unexpected third event: %q", sink.events[2].Type)
	}
}

func TestToolCallExecutorExecuteReturnsIterationHandoffWithoutToolResult(t *testing.T) {
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
	sink := newRecordingEventSink()
	history := NewHistory("")
	executor := newToolCallExecutor(newFakeToolCatalog(tool), history, nil, newAgentEventEmitter(sink, nil))

	stats, err := executor.execute(context.Background(), "trace-pro", 3, []indexedToolCall{
		{index: 0, call: newToolCall("call-pro-1", "pro_update_record", `{"did":"inspected config","remaining":"apply patch"}`)},
	})
	if stats.totalCalls != 1 || stats.executed != 1 {
		t.Fatalf("unexpected stats: %+v", stats)
	}

	var handoffErr *ErrIterationHandoff
	if !errors.As(err, &handoffErr) {
		t.Fatalf("expected ErrIterationHandoff, got %v", err)
	}
	if handoffErr.Did != "inspected config" || handoffErr.Remaining != "apply patch" {
		t.Fatalf("unexpected handoff payload: %+v", handoffErr)
	}
	if len(history.Messages()) != 0 {
		t.Fatalf("iteration handoff should not append tool result: %+v", history.Messages())
	}
	if len(sink.events) != 2 {
		t.Fatalf("unexpected event count: got %d want %d", len(sink.events), 2)
	}
	if sink.events[1].Type != streaming.EventToolCallFinished {
		t.Fatalf("unexpected final event: %q", sink.events[1].Type)
	}
	payload, ok := sink.events[1].Payload.(map[string]any)
	if !ok {
		t.Fatalf("unexpected payload type: %T", sink.events[1].Payload)
	}
	if payload["status"] != "success" {
		t.Fatalf("unexpected finish status: got %v want %q", payload["status"], "success")
	}
}

func TestToolCallExecutorExecuteAppendsMissingToolErrorEnvelope(t *testing.T) {
	sink := newRecordingEventSink()
	history := NewHistory("")
	executor := newToolCallExecutor(newFakeToolCatalog(), history, nil, newAgentEventEmitter(sink, nil))

	stats, err := executor.execute(context.Background(), "trace-missing", 1, []indexedToolCall{
		{index: 0, call: llm.ToolCall{
			ID:        "call-missing-1",
			Name:      "missing_tool",
			Arguments: json.RawMessage(`{"input":"hi"}`),
		}},
	})
	if err != nil {
		t.Fatalf("execute returned error: %v", err)
	}
	if stats.totalCalls != 1 || stats.executed != 0 {
		t.Fatalf("unexpected stats: %+v", stats)
	}

	messages := history.Messages()
	if len(messages) != 1 {
		t.Fatalf("unexpected history messages: %+v", messages)
	}
	if messages[0].Role != llm.RoleTool || messages[0].ToolCallID != "call-missing-1" {
		t.Fatalf("unexpected tool result message: %+v", messages[0])
	}
	result, ok := ParseToolResultEnvelope(messages[0].Text)
	if !ok {
		t.Fatalf("expected tool result envelope, got %q", messages[0].Text)
	}
	if result.Status != "error" || result.Tool != "missing_tool" {
		t.Fatalf("unexpected tool result: %+v", result)
	}
	if len(sink.events) != 2 {
		t.Fatalf("unexpected event count: got %d want %d", len(sink.events), 2)
	}
	payload, ok := sink.events[1].Payload.(map[string]any)
	if !ok {
		t.Fatalf("unexpected payload type: %T", sink.events[1].Payload)
	}
	if payload["tool"] != "missing_tool" {
		t.Fatalf("unexpected tool name in finish event: %+v", sink.events[1].Payload)
	}
}

func TestToolCallExecutorExecuteSingleClosesExplicitInvalidInvocation(t *testing.T) {
	sink := newRecordingEventSink()
	history := NewHistory("")
	executor := newToolCallExecutor(newFakeToolCatalog(newStaticTool("web_search", `{"items":[]}`)), history, nil, newAgentEventEmitter(sink, nil))

	outcome, err := executor.executeSingle(
		context.Background(),
		"trace-explicit-invalid",
		4,
		"web_search",
		"call-explicit-1",
		json.RawMessage(`[]`),
	)
	if err == nil {
		t.Fatal("expected executeSingle to fail for non-object arguments")
	}
	if outcome.executed {
		t.Fatalf("invalid explicit invocation should not execute: %+v", outcome)
	}
	if len(history.Messages()) != 0 {
		t.Fatalf("invalid explicit invocation should not append history: %+v", history.Messages())
	}
	if len(sink.events) != 2 {
		t.Fatalf("unexpected event count: got %d want %d", len(sink.events), 2)
	}
	if sink.events[0].Type != streaming.EventToolCallStarted {
		t.Fatalf("unexpected first event: %q", sink.events[0].Type)
	}
	if sink.events[1].Type != streaming.EventToolCallFinished {
		t.Fatalf("unexpected second event: %q", sink.events[1].Type)
	}
	payload, ok := sink.events[1].Payload.(map[string]any)
	if !ok {
		t.Fatalf("unexpected payload type: %T", sink.events[1].Payload)
	}
	if payload["status"] != "error" {
		t.Fatalf("unexpected finish status: got %v want %q", payload["status"], "error")
	}
	if payload["tool"] != "web_search" {
		t.Fatalf("unexpected tool in finish event: %+v", payload)
	}
	if payload["tool_call_id"] != "call-explicit-1" {
		t.Fatalf("unexpected tool_call_id in finish event: %+v", payload)
	}
}
