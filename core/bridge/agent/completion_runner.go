package agent

import (
	"context"

	"ghost-os/bridge/llm"
	"ghost-os/bridge/streaming"
)

// completionRunner 封装一次模型调用，不负责 history 提交。
type completionRunner struct {
	completer Completer
	tools     ToolCatalog
	history   *History
}

func newCompletionRunner(completer Completer, toolCatalog ToolCatalog, history *History) completionRunner {
	return completionRunner{
		completer: completer,
		tools:     toolCatalog,
		history:   history,
	}
}

func (r completionRunner) complete(ctx context.Context, streamSink streaming.Sink, traceID string, sessionID string, turn int) (*llm.CompletionResponse, error) {
	req := llm.CompletionRequest{
		Messages:          r.history.Messages(),
		Tools:             r.tools.ToolDefs(),
		ConversationState: r.history.ConversationState(),
	}

	var (
		resp *llm.CompletionResponse
		err  error
	)
	if streamingCompleter, ok := r.completer.(llm.StreamingCompleter); ok && streamSink != nil {
		deltaBridge, bridgeErr := newLLMDeltaBridge(streamSink, traceID, sessionID, turn)
		if bridgeErr != nil {
			return nil, bridgeErr
		}
		resp, err = streamingCompleter.CompleteStream(ctx, req, deltaBridge)
	} else {
		resp, err = r.completer.Complete(ctx, req)
	}
	if err != nil {
		return nil, err
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
