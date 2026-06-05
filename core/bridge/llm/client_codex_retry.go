package llm

import (
	"errors"
	"strings"
)

func (c *Client) shouldRetryCodexStateless(request CompletionRequest, statusCode int, raw []byte) bool {
	if c.opts.Provider != ProviderCodex {
		return false
	}
	if !c.opts.CodexStatelessRetryEnabled {
		return false
	}
	if statusCode < 400 || statusCode >= 500 {
		return false
	}
	if strings.TrimSpace(request.ConversationState.PreviousResponseID) == "" {
		return false
	}
	return shouldRetryCodexStatelessByBody(raw)
}

func (c *Client) shouldPreferCodexStateless(request CompletionRequest) bool {
	if c.opts.Provider != ProviderCodex {
		return false
	}
	if !c.opts.CodexStatelessRetryEnabled {
		return false
	}
	if strings.TrimSpace(request.ConversationState.PreviousResponseID) == "" {
		return false
	}
	return true
}

func shouldRetryCodexStatelessByBody(raw []byte) bool {
	body := strings.TrimSpace(strings.ToLower(string(raw)))
	if body == "" {
		return false
	}

	// 兼容历史上游错误与 previous_response_id 失效场景。
	if strings.Contains(body, `"upstream_error"`) || strings.Contains(body, "previous_response_id") {
		return true
	}

	// 继续响应时只收到 function_call_output，且上游上下文已丢失对应 function_call。
	return strings.Contains(body, "no tool call found for function call output with call_id") ||
		strings.Contains(body, "no tool call found for function_call_output with call_id")
}

func (c *Client) shouldRetryCodexStatelessForError(request CompletionRequest, err error) bool {
	var statusErr *completionStatusError
	if !errors.As(err, &statusErr) {
		return false
	}
	return c.shouldRetryCodexStateless(request, statusErr.statusCode, statusErr.raw)
}
