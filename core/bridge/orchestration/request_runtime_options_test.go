package orchestration

import (
	"context"
	"path/filepath"
	"testing"

	"ghost-os/bridge/agent"
	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/llm"
	"ghost-os/bridge/session"
	"ghost-os/bridge/tools"
)

func TestRequestProjectRootOverrideAppliesToStandardTurn(t *testing.T) {
	projectRoot := t.TempDir()
	factory := &projectRootCaptureFactory{
		client: &proTestCompleter{
			responses: []*llm.CompletionResponse{{
				Message:      llm.Message{Role: llm.RoleAssistant, Text: "ok"},
				FinishReason: llm.FinishStop,
			}},
		},
	}
	service := newRequestRuntimeTestService(t, factory)

	result, err := service.executeAgentAction(context.Background(), agentParams{
		Message:     "hello",
		ProjectRoot: projectRoot,
	}, "trace-standard-project-root")
	if err != nil {
		t.Fatalf("executeAgentAction: %v", err)
	}
	if result.Outcome != ServiceOutcomeSuccess {
		t.Fatalf("unexpected outcome: %s", result.Outcome)
	}
	assertCapturedProjectRoot(t, factory, projectRoot)
}

func TestRequestProjectRootOverrideAppliesToPlanMode(t *testing.T) {
	projectRoot := t.TempDir()
	factory := &projectRootCaptureFactory{
		client: &proTestCompleter{
			responses: []*llm.CompletionResponse{{
				Message: llm.Message{
					Role: llm.RoleAssistant,
					Text: "【任务编排】\n1. task_id=T1; objective=分析请求; inputs=用户消息; depends_on=none; executor=main_ai",
				},
				FinishReason: llm.FinishStop,
			}},
		},
	}
	service := newRequestRuntimeTestService(t, factory)

	result, err := service.executeAgentAction(context.Background(), agentParams{
		Mode:        "plan",
		Message:     "plan this task",
		ProjectRoot: projectRoot,
	}, "trace-plan-project-root")
	if err != nil {
		t.Fatalf("executeAgentAction: %v", err)
	}
	if result.Outcome != ServiceOutcomeSuccess {
		t.Fatalf("unexpected outcome: %s", result.Outcome)
	}
	assertCapturedProjectRoot(t, factory, projectRoot)
}

func TestRequestProjectRootOverrideAppliesToProPrefixedStandardTurn(t *testing.T) {
	projectRoot := t.TempDir()
	factory := &projectRootCaptureFactory{
		client: &proTestCompleter{
			responses: []*llm.CompletionResponse{{
				Message:      llm.Message{Role: llm.RoleAssistant, Text: "ok"},
				FinishReason: llm.FinishStop,
			}},
		},
	}
	service := newRequestRuntimeTestService(t, factory)

	result, err := service.executeAgentAction(context.Background(), agentParams{
		Message:     "pro fix config",
		ProjectRoot: projectRoot,
	}, "trace-pro-project-root")
	if err != nil {
		t.Fatalf("executeAgentAction: %v", err)
	}
	if result.Outcome != ServiceOutcomeSuccess {
		t.Fatalf("unexpected outcome: %s", result.Outcome)
	}
	assertCapturedProjectRoot(t, factory, projectRoot)
}

type projectRootCaptureFactory struct {
	client       agent.Completer
	projectRoots []string
}

func (f *projectRootCaptureFactory) Build(store bridgeconfig.Store) (agentRuntimeDependencies, error) {
	cfg, err := store.Config()
	if err != nil {
		return agentRuntimeDependencies{}, err
	}
	cfg.MaxTurns = 4
	cfg.ProMaxIterations = 2
	f.projectRoots = append(f.projectRoots, cfg.ProjectRoot)
	return agentRuntimeDependencies{
		cfg:          cfg,
		client:       f.client,
		registry:     tools.NewRegistry(),
		systemPrompt: "system prompt",
	}, nil
}

func newRequestRuntimeTestService(t *testing.T, factory AgentRuntimeFactory) *bridgeService {
	t.Helper()

	tempDir := t.TempDir()
	t.Setenv("GHOST_CONFIG_PATH", filepath.Join(tempDir, "config.toml"))
	t.Setenv("GHOST_PROVIDER", "custom")
	t.Setenv("GHOST_BASE_URL", "https://initial.example/v1")
	t.Setenv("GHOST_API_KEY", "test-key")

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

func assertCapturedProjectRoot(t *testing.T, factory *projectRootCaptureFactory, want string) {
	t.Helper()

	if len(factory.projectRoots) != 1 {
		t.Fatalf("expected one captured project_root, got %#v", factory.projectRoots)
	}
	if factory.projectRoots[0] != want {
		t.Fatalf("unexpected captured project_root: got %q want %q", factory.projectRoots[0], want)
	}
}
