package agent

import (
	"context"

	"ghost-os/bridge/llm"
)

// completionRunner 封装一次模型调用与 assistant 消息落历史。
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

func (r completionRunner) complete(ctx context.Context, streamSink EventSink, traceID string, turn int) (llm.Message, llm.FinishReason, error) {
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
		resp, err = streamingCompleter.CompleteStream(ctx, req, newLLMDeltaBridge(streamSink, traceID, turn))
	} else {
		resp, err = r.completer.Complete(ctx, req)
	}
	if err != nil {
		return llm.Message{}, "", err
	}

	msg := resp.Message
	if msg.Role == "" {
		msg.Role = llm.RoleAssistant
	}
	r.history.Append(msg)
	r.history.SetConversationState(resp.ConversationState)

	return msg, resp.FinishReason, nil
}
