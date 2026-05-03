package orchestration

import (
	"context"
	"encoding/json"
	"os"
	"strings"

	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/llm"
	bridgeruntime "ghost-os/bridge/runtime"
	"ghost-os/bridge/tools"
)

type fakeSelectorEngine struct {
	result bridgeruntime.ToolSelectorResult
}

func (f *fakeSelectorEngine) SelectTools(_ context.Context, _ string, _ []llm.Message, _ string, _ string) bridgeruntime.ToolSelectorResult {
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
	for _, name := range []string{"ask_human", "script_exec", "codex_cli", "web_search", "sfind"} {
		registry.Register(&runnerMockTool{name: name})
	}
	return registry
}

func newRunnerTestDeps(cfg bridgeconfig.Config) agentRuntimeDependencies {
	if strings.TrimSpace(cfg.PromptsDir) == "" {
		cfg.PromptsDir = os.Getenv("GHOST_PROMPTS_DIR")
	}
	return agentRuntimeDependencies{
		cfg:      cfg,
		registry: newRunnerTestRegistry(),
	}
}
