package orchestration

import (
	"context"
	"encoding/json"
	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/llm"
	bridgeruntime "ghost-os/bridge/runtime"
	"ghost-os/bridge/session"
	"ghost-os/bridge/tools"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func applyContractTestBaseEnv(t *testing.T, tempDir string) {
	t.Helper()

	t.Setenv("GHOST_CONFIG_PATH", filepath.Join(tempDir, "config.toml"))
	t.Setenv("GHOST_API_KEY", "test-key")
	t.Setenv("GHOST_TASKS_PATH", filepath.Join(tempDir, "tasks"))
	t.Setenv("GHOST_PROMPTS_DIR", filepath.Join(tempDir, "prompts"))
}

func applyContractTestRSSEnv(t *testing.T, tempDir string) {
	t.Helper()

	t.Setenv("GHOST_RSS_POLL_ENABLED", "false")
	t.Setenv("GHOST_RSS_BRIEFING_ENABLED", "false")
	t.Setenv("GHOST_RSS_FEEDS_PATH", filepath.Join(tempDir, "rss", "feeds.json"))
	t.Setenv("GHOST_RSS_INBOX_PATH", filepath.Join(tempDir, "rss", "inbox.json"))
}

func newRequestRuntimeTestService(t *testing.T, factory AgentRuntimeFactory) *bridgeService {
	t.Helper()

	tempDir := t.TempDir()
	applyContractTestBaseEnv(t, tempDir)
	applyContractTestRSSEnv(t, tempDir)
	t.Setenv("GHOST_PROVIDER", "custom")
	t.Setenv("GHOST_BASE_URL", "https://initial.example/v1")

	store, err := bridgeconfig.NewStoreFromEnv()
	if err != nil {
		t.Fatalf("NewStoreFromEnv: %v", err)
	}
	sessionStore, err := session.NewStore(filepath.Join(tempDir, "sessions"))
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	service := newBridgeServiceWithStreamExecutor(store, sessionStore, nil, nil)
	service.runtimeFactory = factory
	if runner, ok := service.agentRunner.(*SessionAgentRunner); ok {
		runner.runtimeFactory = factory
	}
	t.Cleanup(service.Close)
	return service
}

func newTestHandlerWithService(t *testing.T, executor agentExecutorFunc, streamExecutor agentStreamExecutorFunc) (http.Handler, *bridgeService, *session.Store) {
	t.Helper()

	tempDir := t.TempDir()
	applyContractTestBaseEnv(t, tempDir)
	applyContractTestRSSEnv(t, tempDir)

	if executor == nil {
		executor = func(_ context.Context, _ string, _ string, _ string, _ bridgeconfig.Store, _ *session.Store) (string, string, error) {
			return "ok", "session-test", nil
		}
	}

	store, err := bridgeconfig.NewStoreFromEnv()
	if err != nil {
		t.Fatalf("new config store: %v", err)
	}

	sessionStore, err := session.NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("new session store: %v", err)
	}

	service := newBridgeServiceWithStreamExecutor(store, sessionStore, executor, streamExecutor)
	if err := service.StartBackgroundRuntimes(); err != nil {
		t.Fatalf("start background runtimes: %v", err)
	}
	t.Cleanup(service.Close)
	return nil, service, sessionStore
}

func newTempSessionStore(t *testing.T) *session.Store {
	t.Helper()

	t.Setenv("GHOST_PROMPTS_DIR", filepath.Join(t.TempDir(), "prompts"))
	store, err := session.NewStore(filepath.Join(t.TempDir(), "sessions"))
	if err != nil {
		t.Fatalf("new session store: %v", err)
	}
	return store
}

func boolPointer(value bool) *bool {
	return &value
}

func intPointer(value int) *int {
	return &value
}

func stringPointer(value string) *string {
	return &value
}

func containsToolName(names []string, target string) bool {
	for _, name := range names {
		if name == target {
			return true
		}
	}
	return false
}

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
