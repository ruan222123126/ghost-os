package orchestration

import (
	"context"
	"encoding/json"

	"ghost-os/bridge/llm"
	"ghost-os/bridge/tools"
)

type fakeSelectorEngine struct {
	result ToolSelectorResult
}

func (f *fakeSelectorEngine) SelectTools(_ context.Context, _ string, _ []llm.Message, _ string, _ string) ToolSelectorResult {
	return f.result
}

type runnerMockTool struct {
	name string
}

func (m *runnerMockTool) Name() string { return m.name }

func (m *runnerMockTool) Description() string { return "runner mock" }

func (m *runnerMockTool) Parameters() json.RawMessage { return json.RawMessage(`{"type":"object"}`) }

func (m *runnerMockTool) Execute(context.Context, json.RawMessage, string) (string, error) {
	return "", nil
}

func newRunnerTestRegistry() *tools.Registry {
	registry := tools.NewRegistry()
	for _, name := range []string{"ask_human", "script_exec", "send_file", "web_search"} {
		registry.Register(&runnerMockTool{name: name})
	}
	return registry
}

func newRunnerTestDeps(cfg Config) agentRuntimeDependencies {
	return agentRuntimeDependencies{
		cfg:      cfg,
		registry: newRunnerTestRegistry(),
	}
}
