package orchestration

import (
	"context"
	"errors"
	"os"
	"strings"

	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/llm"
)

type testRuntimeFactory struct {
	deps agentRuntimeDependencies
	err  error
}

type proTestRuntimeFactory = testRuntimeFactory

func (f testRuntimeFactory) Build(bridgeconfig.Store) (agentRuntimeDependencies, error) {
	if f.err != nil {
		return agentRuntimeDependencies{}, f.err
	}
	if strings.TrimSpace(f.deps.cfg.PromptsDir) == "" {
		f.deps.cfg.PromptsDir = os.Getenv("GHOST_PROMPTS_DIR")
	}
	return f.deps, nil
}

type testCompleter struct {
	responses []*llm.CompletionResponse
	requests  []llm.CompletionRequest
}

type proTestCompleter = testCompleter

func (f *testCompleter) Complete(_ context.Context, request llm.CompletionRequest) (*llm.CompletionResponse, error) {
	f.requests = append(f.requests, request)
	if len(f.responses) == 0 {
		return nil, errors.New("unexpected complete call")
	}
	response := f.responses[0]
	f.responses = f.responses[1:]
	return response, nil
}
