package agent

import (
	"context"
	"errors"
	"strings"
	"testing"

	"ghost-os/bridge/llm"
	"ghost-os/bridge/streaming"
)

func TestCompletionRunnerCompleteDoesNotMutateHistory(t *testing.T) {
	completer := newFakeCompleter(&llm.CompletionResponse{
		Message: llm.Message{
			Text: "follow-up",
		},
		FinishReason: llm.FinishStop,
		ConversationState: llm.ConversationState{
			Provider:           llm.ProviderCodex,
			BaseURL:            "https://api.openai.com/v1",
			Model:              "codex-mini-latest",
			PreviousResponseID: "resp_next",
		},
	})
	history := NewHistoryFromMessages([]llm.Message{
		{Role: llm.RoleSystem, Text: "system prompt"},
		{Role: llm.RoleUser, Text: "hello"},
	})
	history.SetConversationState(llm.ConversationState{
		Provider:           llm.ProviderCodex,
		BaseURL:            "https://api.openai.com/v1",
		Model:              "codex-mini-latest",
		PreviousResponseID: "resp_prev",
	})

	runner := newCompletionRunner(completer, newFakeToolCatalog(), history, llm.ResponseOptions{})
	resp, err := runner.complete(context.Background(), nil, "trace-runner", "", 0)
	if err != nil {
		t.Fatalf("complete returned error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected response but got nil")
	}
	if resp.Message.Role != llm.RoleAssistant {
		t.Fatalf("expected assistant role normalization, got %q", resp.Message.Role)
	}
	if len(history.Messages()) != 2 {
		t.Fatalf("complete should not append assistant message to history, got %d messages", len(history.Messages()))
	}
	if got := history.ConversationState().PreviousResponseID; got != "resp_prev" {
		t.Fatalf("complete should not update conversation state: got %q want %q", got, "resp_prev")
	}
}

func TestCompletionRunnerPassesResponseOptionsToRequest(t *testing.T) {
	store := true
	options := llm.ResponseOptions{
		PromptCacheKey:       " cache-key ",
		PromptCacheRetention: " sticky ",
		SafetyIdentifier:     " user-123 ",
		Metadata: map[string]string{
			"trace": " session-1 ",
		},
		Store: &store,
	}
	completer := newFakeCompleter(&llm.CompletionResponse{
		Message:      llm.Message{Role: llm.RoleAssistant, Text: "ok"},
		FinishReason: llm.FinishStop,
	})
	history := NewHistoryFromMessages([]llm.Message{
		{Role: llm.RoleSystem, Text: "system prompt"},
		{Role: llm.RoleUser, Text: "hello"},
	})

	runner := newCompletionRunner(completer, newFakeToolCatalog(), history, options)
	if _, err := runner.complete(context.Background(), nil, "trace-runner", "", 0); err != nil {
		t.Fatalf("complete returned error: %v", err)
	}
	if len(completer.requests) != 1 {
		t.Fatalf("expected one completion request, got %d", len(completer.requests))
	}
	got := completer.requests[0].ResponseOptions
	if got.PromptCacheKey != "cache-key" {
		t.Fatalf("unexpected prompt_cache_key: got %q want %q", got.PromptCacheKey, "cache-key")
	}
	if got.PromptCacheRetention != "sticky" {
		t.Fatalf("unexpected prompt_cache_retention: got %q want %q", got.PromptCacheRetention, "sticky")
	}
	if got.SafetyIdentifier != "user-123" {
		t.Fatalf("unexpected safety_identifier: got %q want %q", got.SafetyIdentifier, "user-123")
	}
	if got.Metadata["trace"] != "session-1" {
		t.Fatalf("unexpected metadata map: %+v", got.Metadata)
	}
	if got.Store == nil || !*got.Store {
		t.Fatalf("unexpected store option: %+v", got.Store)
	}
}

func TestCompletionRunnerProjectsInternalMessagesForProvider(t *testing.T) {
	completer := newFakeCompleter(&llm.CompletionResponse{
		Message:      llm.Message{Role: llm.RoleAssistant, Text: "done"},
		FinishReason: llm.FinishStop,
	})
	history := NewHistoryFromMessages([]llm.Message{
		{Role: llm.RoleSystem, Text: "system prompt"},
		{Role: llm.RoleUser, Text: "hello"},
		{Role: llm.RoleInternal, Text: "[TOOL_TAG_RESULT]\n{\"tool\":\"web_search\",\"output\":{\"items\":[{\"title\":\"OpenAI\"}]}}"},
	})

	runner := newCompletionRunner(completer, newFakeToolCatalog(), history, llm.ResponseOptions{})
	if _, err := runner.complete(context.Background(), nil, "trace-runner", "", 0); err != nil {
		t.Fatalf("complete returned error: %v", err)
	}

	if len(completer.requests) != 1 {
		t.Fatalf("expected one completion request, got %d", len(completer.requests))
	}
	request := completer.requests[0]
	last := request.Messages[len(request.Messages)-1]
	if last.Role != llm.RoleAssistant {
		t.Fatalf("expected projected assistant role for provider request, got %+v", last)
	}
	if history.Messages()[2].Role != llm.RoleInternal {
		t.Fatalf("expected persisted history role to stay internal, got %+v", history.Messages()[2])
	}
}

func TestCompletionRunnerCompleteReturnsErrorOnNilResponse(t *testing.T) {
	runner := newCompletionRunner(
		newFakeCompleter(nil),
		newFakeToolCatalog(),
		NewHistoryFromMessages([]llm.Message{{Role: llm.RoleUser, Text: "hello"}}),
		llm.ResponseOptions{},
	)

	_, err := runner.complete(context.Background(), nil, "trace-runner", "", 0)
	if err == nil {
		t.Fatal("expected error but got nil")
	}
	if !strings.Contains(err.Error(), errCompletionRunnerResponseRequired.Error()) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCompletionRunnerRetriesNonStreamingOnTransientError(t *testing.T) {
	completer := &retrySequenceCompleter{
		results: []retryCompleteResult{
			{err: transientNetError{message: "temporary network error"}},
			{resp: newStopResponse("ok")},
		},
	}
	runner := newCompletionRunner(
		completer,
		newFakeToolCatalog(),
		NewHistoryFromMessages([]llm.Message{{Role: llm.RoleUser, Text: "hello"}}),
		llm.ResponseOptions{},
	)

	resp, err := runner.complete(context.Background(), nil, "trace-retry", "", 0)
	if err != nil {
		t.Fatalf("expected retry success, got error: %v", err)
	}
	if resp == nil || resp.Message.Text != "ok" {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if len(completer.requests) != 2 {
		t.Fatalf("expected 2 attempts, got %d", len(completer.requests))
	}
}

func TestCompletionRunnerDoesNotRetryNonStreamingOnNonTransientError(t *testing.T) {
	completer := &retrySequenceCompleter{
		results: []retryCompleteResult{
			{err: errors.New("bad request")},
			{resp: newStopResponse("should not reach")},
		},
	}
	runner := newCompletionRunner(
		completer,
		newFakeToolCatalog(),
		NewHistoryFromMessages([]llm.Message{{Role: llm.RoleUser, Text: "hello"}}),
		llm.ResponseOptions{},
	)

	_, err := runner.complete(context.Background(), nil, "trace-retry", "", 0)
	if err == nil {
		t.Fatal("expected non-transient error")
	}
	if len(completer.requests) != 1 {
		t.Fatalf("expected 1 attempt, got %d", len(completer.requests))
	}
}

func TestCompletionRunnerRetriesStreamingWhenNoDeltaEmitted(t *testing.T) {
	completer := &retryStreamingCompleter{
		attempts: []retryStreamResult{
			{err: transientNetError{message: "upstream timeout"}},
			{
				resp: newStopResponse("done"),
				deltas: []llm.LLMDelta{
					{Kind: llm.DeltaKindText, Text: "done"},
				},
			},
		},
	}
	history := NewHistoryFromMessages([]llm.Message{{Role: llm.RoleUser, Text: "hello"}})
	runner := newCompletionRunner(completer, newFakeToolCatalog(), history, llm.ResponseOptions{})
	sink := newRecordingEventSink()

	resp, err := runner.complete(context.Background(), sink, "trace-retry-stream", "session-1", 0)
	if err != nil {
		t.Fatalf("expected retry success, got error: %v", err)
	}
	if resp == nil || resp.Message.Text != "done" {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if len(completer.requests) != 2 {
		t.Fatalf("expected 2 attempts, got %d", len(completer.requests))
	}
	if got := countCompletionDeltaEvents(sink.events); got != 1 {
		t.Fatalf("expected exactly one completion_delta event after retry, got %d", got)
	}
}

func TestCompletionRunnerDoesNotRetryStreamingAfterDeltaEmitted(t *testing.T) {
	completer := &retryStreamingCompleter{
		attempts: []retryStreamResult{
			{
				err: transientNetError{message: "connection reset"},
				deltas: []llm.LLMDelta{
					{Kind: llm.DeltaKindText, Text: "partial"},
				},
			},
			{resp: newStopResponse("should not reach")},
		},
	}
	history := NewHistoryFromMessages([]llm.Message{{Role: llm.RoleUser, Text: "hello"}})
	runner := newCompletionRunner(completer, newFakeToolCatalog(), history, llm.ResponseOptions{})
	sink := newRecordingEventSink()

	_, err := runner.complete(context.Background(), sink, "trace-retry-stream", "session-1", 0)
	if err == nil {
		t.Fatal("expected streaming failure without retry after delta emission")
	}
	if len(completer.requests) != 1 {
		t.Fatalf("expected 1 attempt, got %d", len(completer.requests))
	}
	if got := countCompletionDeltaEvents(sink.events); got != 1 {
		t.Fatalf("expected one completion_delta event from first attempt, got %d", got)
	}
}

func countCompletionDeltaEvents(events []streaming.Event) int {
	count := 0
	for _, event := range events {
		if event.Type == streaming.EventCompletionDelta {
			count++
		}
	}
	return count
}

type retryCompleteResult struct {
	resp *llm.CompletionResponse
	err  error
}

type retrySequenceCompleter struct {
	results  []retryCompleteResult
	requests []llm.CompletionRequest
}

func (f *retrySequenceCompleter) Complete(_ context.Context, request llm.CompletionRequest) (*llm.CompletionResponse, error) {
	f.requests = append(f.requests, cloneCompletionRequest(request))
	if len(f.results) == 0 {
		return nil, errors.New("unexpected complete call")
	}
	result := f.results[0]
	f.results = f.results[1:]
	return result.resp, result.err
}

type retryStreamResult struct {
	resp   *llm.CompletionResponse
	err    error
	deltas []llm.LLMDelta
}

type retryStreamingCompleter struct {
	attempts []retryStreamResult
	requests []llm.CompletionRequest
}

func (f *retryStreamingCompleter) Complete(_ context.Context, request llm.CompletionRequest) (*llm.CompletionResponse, error) {
	f.requests = append(f.requests, cloneCompletionRequest(request))
	return nil, errors.New("unexpected complete call")
}

func (f *retryStreamingCompleter) CompleteStream(ctx context.Context, request llm.CompletionRequest, sink llm.LLMStreamSink) (*llm.CompletionResponse, error) {
	f.requests = append(f.requests, cloneCompletionRequest(request))
	if len(f.attempts) == 0 {
		return nil, errors.New("unexpected complete stream call")
	}
	attempt := f.attempts[0]
	f.attempts = f.attempts[1:]
	for _, delta := range attempt.deltas {
		if err := sink.OnDelta(ctx, delta); err != nil {
			return nil, err
		}
	}
	return attempt.resp, attempt.err
}

type transientNetError struct {
	message string
}

func (e transientNetError) Error() string {
	return e.message
}

func (transientNetError) Timeout() bool {
	return true
}

func (transientNetError) Temporary() bool {
	return true
}
