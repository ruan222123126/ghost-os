package agent

import (
	"context"
	"errors"
	"time"

	"ghost-os/bridge/llm"
	"ghost-os/bridge/streaming"
)

var (
	errCompletionRunnerCompleterRequired = errors.New("completion runner completer is nil")
	errCompletionRunnerHistoryRequired   = errors.New("completion runner history is nil")
	errCompletionRunnerResponseRequired  = errors.New("completion runner got nil response")
	errCompletionRunnerToolsRequired     = errors.New("completion runner tool catalog is nil")
)

// completionRunner 封装一次模型调用，不负责 history 提交。
type completionRunner struct {
	completer       Completer
	tools           ToolCatalog
	history         *History
	responseOptions llm.ResponseOptions
	retryPolicy     CompletionRetryPolicy
	attemptState    *completionAttemptState
}

func newCompletionRunner(
	completer Completer,
	toolCatalog ToolCatalog,
	history *History,
	responseOptions llm.ResponseOptions,
) completionRunner {
	return newCompletionRunnerWithPolicy(
		completer,
		toolCatalog,
		history,
		responseOptions,
		DefaultCompletionRetryPolicy(),
	)
}

func newCompletionRunnerWithPolicy(
	completer Completer,
	toolCatalog ToolCatalog,
	history *History,
	responseOptions llm.ResponseOptions,
	retryPolicy CompletionRetryPolicy,
) completionRunner {
	return completionRunner{
		completer:       completer,
		tools:           toolCatalog,
		history:         history,
		responseOptions: llm.CloneResponseOptions(responseOptions),
		retryPolicy:     retryPolicy,
	}
}

func (r completionRunner) withAttemptState(state *completionAttemptState) completionRunner {
	r.attemptState = state
	return r
}

func (r completionRunner) complete(ctx context.Context, streamSink streaming.Sink, traceID string, sessionID string, turn int) (*llm.CompletionResponse, error) {
	if r.attemptState != nil {
		r.attemptState.reset()
	}
	req, err := r.request()
	if err != nil {
		return nil, err
	}

	if streamingCompleter, ok := r.completer.(llm.StreamingCompleter); ok && streamSink != nil {
		return r.completeStreaming(ctx, streamingCompleter, req, streamSink, traceID, sessionID, turn)
	}
	return r.completeNonStreaming(ctx, req)
}

func (r completionRunner) completeNonStreaming(ctx context.Context, req llm.CompletionRequest) (*llm.CompletionResponse, error) {
	maxAttempts := r.retryPolicy.maxAttempts()
	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		resp, err := r.completer.Complete(ctx, req)
		if err == nil {
			return normalizeCompletionResponse(resp)
		}
		lastErr = err
		if !shouldRetryCompletion(ctx, err, attempt, maxAttempts, false) {
			return nil, err
		}
		if err := waitCompletionRetry(ctx, r.retryPolicy.retryIntervalOrZero()); err != nil {
			return nil, err
		}
	}
	return nil, lastErr
}

func (r completionRunner) completeStreaming(
	ctx context.Context,
	completer llm.StreamingCompleter,
	req llm.CompletionRequest,
	streamSink streaming.Sink,
	traceID string,
	sessionID string,
	turn int,
) (*llm.CompletionResponse, error) {
	maxAttempts := r.retryPolicy.maxAttempts()
	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if r.attemptState != nil {
			r.attemptState.reset()
		}
		deltaBridge, err := newLLMDeltaBridge(streamSink, traceID, sessionID, turn, r.attemptState)
		if err != nil {
			return nil, err
		}
		resp, err := completer.CompleteStream(ctx, req, deltaBridge)
		if err == nil {
			return normalizeCompletionResponse(resp)
		}
		lastErr = err
		if !shouldRetryCompletion(ctx, err, attempt, maxAttempts, deltaBridge.hasEmitted()) {
			return nil, err
		}
		if err := waitCompletionRetry(ctx, r.retryPolicy.retryIntervalOrZero()); err != nil {
			return nil, err
		}
	}
	return nil, lastErr
}

func normalizeCompletionResponse(resp *llm.CompletionResponse) (*llm.CompletionResponse, error) {
	if resp == nil {
		return nil, errCompletionRunnerResponseRequired
	}
	normalized := *resp
	if cloned := llm.CloneMessages([]llm.Message{resp.Message}); len(cloned) == 1 {
		normalized.Message = cloned[0]
	}
	if normalized.Message.Role == "" {
		normalized.Message.Role = llm.RoleAssistant
	}

	return &normalized, nil
}

func shouldRetryCompletion(
	ctx context.Context,
	err error,
	attempt int,
	maxAttempts int,
	streamEmitted bool,
) bool {
	if attempt+1 >= maxAttempts {
		return false
	}
	if streamEmitted {
		return false
	}
	if ctx.Err() != nil {
		return false
	}
	return llm.IsTransientCompletionError(err)
}

func waitCompletionRetry(ctx context.Context, retryInterval time.Duration) error {
	if retryInterval == 0 {
		return ctx.Err()
	}
	timer := time.NewTimer(retryInterval)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (r completionRunner) request() (llm.CompletionRequest, error) {
	if r.completer == nil {
		return llm.CompletionRequest{}, errCompletionRunnerCompleterRequired
	}
	if r.tools == nil {
		return llm.CompletionRequest{}, errCompletionRunnerToolsRequired
	}
	if r.history == nil {
		return llm.CompletionRequest{}, errCompletionRunnerHistoryRequired
	}
	return llm.CompletionRequest{
		Messages:          projectMessagesForProvider(r.history.Messages(), r.tools),
		Tools:             r.tools.ToolDefs(),
		ConversationState: r.history.ConversationState(),
		ResponseOptions:   llm.CloneResponseOptions(r.responseOptions),
	}, nil
}

func (r completionRunner) completionDeltaEmitted() bool {
	return r.attemptState != nil && r.attemptState.hasCompletionDeltaEmitted()
}
