package rss

import (
	"context"

	"ghost-os/bridge/llm"
)

type fakeSelectorCompleter struct {
	response       *llm.CompletionResponse
	err            error
	waitForContext bool
	requests       []llm.CompletionRequest
}

func (f *fakeSelectorCompleter) Complete(ctx context.Context, request llm.CompletionRequest) (*llm.CompletionResponse, error) {
	f.requests = append(f.requests, request)
	if f.waitForContext {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	return f.response, f.err
}
