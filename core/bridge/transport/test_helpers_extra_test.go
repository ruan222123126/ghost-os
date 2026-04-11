package transport

import (
	"context"
	"errors"
	"testing"

	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/llm"
	bridgeorchestration "ghost-os/bridge/orchestration"
	bridgerss "ghost-os/bridge/rss"
	rsssubscriptions "ghost-os/bridge/rss/subscriptions"
	"ghost-os/bridge/streaming"
	"ghost-os/bridge/tools"
)

const cancelledHumanDialogueMessage = "Conversation cancelled by user."

func mustAppEvent(
	t *testing.T,
	traceID string,
	sessionID string,
	turn int,
	stepID string,
	eventType streaming.EventType,
	payload any,
) streaming.Event {
	t.Helper()

	event, err := streaming.NewEvent(traceID, sessionID, turn, stepID, eventType, payload)
	if err != nil {
		t.Fatalf("NewEvent returned error: %v", err)
	}
	return event
}

func mustAppAssistantStepID(t *testing.T, turn int) string {
	t.Helper()

	stepID, err := streaming.AssistantStepID(turn)
	if err != nil {
		t.Fatalf("AssistantStepID returned error: %v", err)
	}
	return stepID
}

func mustAppToolStepID(t *testing.T, turn int, toolIndex int) string {
	t.Helper()

	stepID, err := streaming.ToolStepID(turn, toolIndex)
	if err != nil {
		t.Fatalf("ToolStepID returned error: %v", err)
	}
	return stepID
}

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

type testRSSInboxFetcher struct {
	byURL map[string]tools.RSSResult
	err   error
}

func (f testRSSInboxFetcher) Fetch(_ context.Context, feedURL string, _ int) (tools.RSSResult, error) {
	if f.err != nil {
		return tools.RSSResult{}, f.err
	}
	if result, ok := f.byURL[feedURL]; ok {
		return result, nil
	}
	return tools.RSSResult{}, nil
}

type testRSSInboxClassifier struct {
	decisions []bridgerss.RSSInboxClassification
	perFeed   map[string][]bridgerss.RSSInboxClassification
	err       error
}

func (c testRSSInboxClassifier) Classify(
	_ context.Context,
	feed rsssubscriptions.FeedSubscription,
	_ []bridgerss.RSSInboxCandidate,
	_ string,
) ([]bridgerss.RSSInboxClassification, error) {
	if c.err != nil {
		return nil, c.err
	}
	if len(c.perFeed) > 0 {
		if decisions, ok := c.perFeed[feed.URL]; ok {
			return decisions, nil
		}
	}
	return c.decisions, nil
}

type proTestRuntimeFactory struct {
	deps bridgeorchestration.RuntimeDependencies
	err  error
}

func (f proTestRuntimeFactory) Build(bridgeconfig.Store) (bridgeorchestration.RuntimeDependencies, error) {
	if f.err != nil {
		return bridgeorchestration.RuntimeDependencies{}, f.err
	}
	return f.deps, nil
}

type proTestCompleter struct {
	responses []*llm.CompletionResponse
	requests  []llm.CompletionRequest
}

func (f *proTestCompleter) Complete(_ context.Context, request llm.CompletionRequest) (*llm.CompletionResponse, error) {
	f.requests = append(f.requests, request)
	if len(f.responses) == 0 {
		return nil, errors.New("unexpected complete call")
	}
	response := f.responses[0]
	f.responses = f.responses[1:]
	return response, nil
}
