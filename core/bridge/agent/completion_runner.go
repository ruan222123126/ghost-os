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

const (
	completionRetryMaxAttempts = 2
	completionRetryBackoff     = 200 * time.Millisecond
)

// completionRunner 封装一次模型调用，不负责 history 提交。
type completionRunner struct {
	completer       Completer
	tools           ToolCatalog
	history         *History
	responseOptions llm.ResponseOptions
}

func newCompletionRunner(
	completer Completer,
	toolCatalog ToolCatalog,
	history *History,
	responseOptions llm.ResponseOptions,
) completionRunner {
	return completionRunner{
		completer:       completer,
		tools:           toolCatalog,
		history:         history,
		responseOptions: llm.CloneResponseOptions(responseOptions),
	}
}

func (r completionRunner) complete(ctx context.Context, streamSink streaming.Sink, traceID string, sessionID string, turn int) (*llm.CompletionResponse, error) {
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
	var lastErr error
	for attempt := 0; attempt < completionRetryMaxAttempts; attempt++ {
		resp, err := r.completer.Complete(ctx, req)
		if err == nil {
			return normalizeCompletionResponse(resp)
		}
		lastErr = err
		if !shouldRetryCompletion(ctx, err, attempt, false) {
			return nil, err
		}
		if err := waitCompletionRetry(ctx); err != nil {
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
	var lastErr error
	for attempt := 0; attempt < completionRetryMaxAttempts; attempt++ {
		deltaBridge, err := newLLMDeltaBridge(streamSink, traceID, sessionID, turn)
		if err != nil {
			return nil, err
		}
		resp, err := completer.CompleteStream(ctx, req, deltaBridge)
		if err == nil {
			return normalizeCompletionResponse(resp)
		}
		lastErr = err
		if !shouldRetryCompletion(ctx, err, attempt, deltaBridge.hasEmitted()) {
			return nil, err
		}
		if err := waitCompletionRetry(ctx); err != nil {
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

func shouldRetryCompletion(ctx context.Context, err error, attempt int, streamEmitted bool) bool {
	if attempt+1 >= completionRetryMaxAttempts {
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

func waitCompletionRetry(ctx context.Context) error {
	timer := time.NewTimer(completionRetryBackoff)
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
