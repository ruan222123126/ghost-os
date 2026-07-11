package agent

import (
	"context"
	"errors"
	"strings"
	"testing"

	"ghost-os/bridge/llm"
	"ghost-os/bridge/streaming"
)

func TestRunRejectsInvalidAgentRuntime(t *testing.T) {
	cases := []struct {
		name  string
		agent *Agent
		want  string
	}{
		{name: "nil agent", agent: nil, want: errAgentRequired.Error()},
		{name: "nil completer", agent: &Agent{tools: newFakeToolCatalog(), history: NewHistory("system"), maxTurns: 1}, want: errAgentCompleterRequired.Error()},
		{name: "nil tool catalog", agent: &Agent{completer: newFakeCompleter(newStopResponse("done")), history: NewHistory("system"), maxTurns: 1}, want: errAgentToolCatalogMissing.Error()},
		{name: "nil history", agent: &Agent{completer: newFakeCompleter(newStopResponse("done")), tools: newFakeToolCatalog(), maxTurns: 1}, want: errAgentHistoryRequired.Error()},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := tc.agent.Run(context.Background(), "hello")
			if err == nil {
				t.Fatal("expected error but got nil")
			}
			if !strings.Contains(err.Error(), "initialize agent runtime") || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestRunStreamRejectsInvalidAgentRuntimeWithErrorEvent(t *testing.T) {
	sink := newRecordingEventSink()
	agent := &Agent{
		tools:    newFakeToolCatalog(),
		history:  NewHistory("system"),
		maxTurns: 1,
	}

	_, err := agent.RunMessageStreamWithTraceID(
		context.Background(),
		llm.Message{Role: llm.RoleUser, Text: "hello"},
		"trace-invalid",
		sink,
	)
	if err == nil {
		t.Fatal("expected error but got nil")
	}
	if !strings.Contains(err.Error(), errAgentCompleterRequired.Error()) {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sink.events) != 1 || sink.events[0].Type != streaming.EventError {
		t.Fatalf("unexpected events: %+v", sink.events)
	}

	stepID, stepErr := streaming.AssistantStepID(0)
	if stepErr != nil {
		t.Fatalf("AssistantStepID returned error: %v", stepErr)
	}
	if sink.events[0].StepID != stepID {
		t.Fatalf("unexpected step id: got %q want %q", sink.events[0].StepID, stepID)
	}
}

func TestRunStreamRejectsInvalidAgentRuntimePreservesRunErrorOnEmitFailure(t *testing.T) {
	sinkErr := errors.New("sink unavailable")
	sink := failingEventSink{err: sinkErr}
	agent := &Agent{
		tools:    newFakeToolCatalog(),
		history:  NewHistory("system"),
		maxTurns: 1,
	}

	_, err := agent.RunMessageStreamWithTraceID(
		context.Background(),
		llm.Message{Role: llm.RoleUser, Text: "hello"},
		"trace-invalid-emit",
		sink,
	)
	if err == nil {
		t.Fatal("expected error but got nil")
	}
	if !strings.Contains(err.Error(), "initialize agent runtime") {
		t.Fatalf("missing runtime init error context: %v", err)
	}
	if !strings.Contains(err.Error(), errAgentCompleterRequired.Error()) {
		t.Fatalf("missing root cause in error: %v", err)
	}
	if !errors.Is(err, sinkErr) {
		t.Fatalf("expected joined sink error, got: %v", err)
	}
}

type failingEventSink struct {
	err error
}

func (f failingEventSink) Emit(_ context.Context, _ streaming.Event) (streaming.Event, error) {
	return streaming.Event{}, f.err
}
