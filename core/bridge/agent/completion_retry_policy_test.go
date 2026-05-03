package agent

import (
	"context"
	"testing"
	"time"

	"ghost-os/bridge/llm"
)

func TestCompletionRunnerUsesConfiguredRetryCount(t *testing.T) {
	completer := &retrySequenceCompleter{
		results: []retryCompleteResult{
			{err: transientNetError{message: "temporary network error"}},
			{err: transientNetError{message: "temporary network error"}},
			{resp: newStopResponse("ok")},
		},
	}
	runner := newCompletionRunnerWithPolicy(
		completer,
		newFakeToolCatalog(),
		NewHistoryFromMessages([]llm.Message{{Role: llm.RoleUser, Text: "hello"}}),
		llm.ResponseOptions{},
		NewCompletionRetryPolicy(2, 0),
	)

	resp, err := runner.complete(context.Background(), nil, "trace-retry", "", 0)
	if err != nil {
		t.Fatalf("expected retry success, got error: %v", err)
	}
	if resp == nil || resp.Message.Text != "ok" {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if len(completer.requests) != 3 {
		t.Fatalf("expected 3 attempts, got %d", len(completer.requests))
	}
}

func TestCompletionRunnerDisablesRetryWhenRetryCountZero(t *testing.T) {
	completer := &retrySequenceCompleter{
		results: []retryCompleteResult{
			{err: transientNetError{message: "temporary network error"}},
			{resp: newStopResponse("should not reach")},
		},
	}
	runner := newCompletionRunnerWithPolicy(
		completer,
		newFakeToolCatalog(),
		NewHistoryFromMessages([]llm.Message{{Role: llm.RoleUser, Text: "hello"}}),
		llm.ResponseOptions{},
		NewCompletionRetryPolicy(0, 0),
	)

	if _, err := runner.complete(context.Background(), nil, "trace-retry", "", 0); err == nil {
		t.Fatal("expected retry to be disabled")
	}
	if len(completer.requests) != 1 {
		t.Fatalf("expected 1 attempt, got %d", len(completer.requests))
	}
}

func TestAgentSetCompletionRetryPolicyAppliesToRuns(t *testing.T) {
	completer := &retrySequenceCompleter{
		results: []retryCompleteResult{
			{err: transientNetError{message: "temporary network error"}},
			{resp: newStopResponse("should not reach")},
		},
	}
	runAgent := NewAgentWithHistory(
		completer,
		newFakeToolCatalog(),
		NewHistoryFromMessages([]llm.Message{{Role: llm.RoleUser, Text: "hello"}}),
		1,
	)
	runAgent.SetCompletionRetryPolicy(NewCompletionRetryPolicy(0, time.Millisecond))

	if _, err := runAgent.RunMessage(context.Background(), llm.Message{
		Role: llm.RoleUser,
		Text: "retry disabled",
	}); err == nil {
		t.Fatal("expected run to fail without retry")
	}
	if len(completer.requests) != 1 {
		t.Fatalf("expected 1 completion attempt, got %d", len(completer.requests))
	}
}
