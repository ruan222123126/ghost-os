package runtime

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/tools"
)

func TestAgentRuntimeFactoryRegistersTaskManageWhenTaskManagerPresent(t *testing.T) {
	setupRuntimeFactoryTestEnv(t)
	store := newRuntimeTestStore(t)

	deps, err := newAgentRuntimeFactoryWithTaskManager(fakeTaskManager{}).Build(store)
	if err != nil {
		t.Fatalf("build runtime deps: %v", err)
	}
	t.Cleanup(deps.Close)

	if deps.registry.Get("task_manage") == nil {
		t.Fatal("expected task_manage to be registered when task manager is present")
	}
}

func TestAgentRuntimeFactorySkipsTaskManageWithoutTaskManager(t *testing.T) {
	setupRuntimeFactoryTestEnv(t)
	store := newRuntimeTestStore(t)

	deps, err := newAgentRuntimeFactory().Build(store)
	if err != nil {
		t.Fatalf("build runtime deps: %v", err)
	}
	t.Cleanup(deps.Close)

	if deps.registry.Get("task_manage") != nil {
		t.Fatal("expected task_manage to stay hidden without a task manager")
	}
}

func TestAgentRuntimeFactoryRegistersToolSearchWhenEnabled(t *testing.T) {
	setupRuntimeFactoryTestEnv(t)
	t.Setenv("GHOST_TOOL_SEARCH_ENABLED", "true")
	store := newRuntimeTestStore(t)

	deps, err := newAgentRuntimeFactory().Build(store)
	if err != nil {
		t.Fatalf("build runtime deps: %v", err)
	}
	t.Cleanup(deps.Close)

	if deps.registry.Get("tfind") == nil {
		t.Fatal("expected tfind to be registered when tool search is enabled")
	}
}

func TestAgentRuntimeFactorySkipsGraphQLToolsEvenWhenConfigured(t *testing.T) {
	tempDir := setupRuntimeFactoryTestEnv(t)
	t.Setenv("GHOST_TOOL_SEARCH_ENABLED", "true")
	writeRuntimeGraphQLConfig(t, tempDir, fmt.Sprintf(`
graphql_default_source = "crm"

[[graphql_sources]]
name = "crm"
endpoint = "https://graphql.test/query"
schema_path = %q
timeout_ms = 3000
max_response_bytes = 4096
max_depth = 5
max_fields = 24
max_root_fields = 2
max_fragments = 3

[[graphql_sources.domains]]
name = "viewer"
root_queries = ["viewer"]
types = ["Viewer", "MutationPayload"]

[[graphql_mutation_policies]]
name = "update_viewer"
source = "crm"
domain = "viewer"
root_mutation = "updateViewer"
idempotency_mode = "header"
idempotency_header = "Idempotency-Key"
`, writeRuntimeGraphQLSchema(t, tempDir)))
	store := newRuntimeTestStore(t)

	deps, err := newAgentRuntimeFactory().Build(store)
	if err != nil {
		t.Fatalf("build runtime deps: %v", err)
	}
	t.Cleanup(deps.Close)

	for _, name := range []string{"graphql_query", "graphql_schema_lookup", "graphql_mutation"} {
		if deps.registry.Get(name) != nil {
			t.Fatalf("expected %s to stay disabled", name)
		}
	}

	visible := tools.StaticVisibleToolNames(tools.CatalogToolNames(deps.registry), toolVisibilityOptions(deps.cfg))
	if containsRuntimeTool(visible, "graphql_query") || containsRuntimeTool(visible, "graphql_schema_lookup") || containsRuntimeTool(visible, "graphql_mutation") {
		t.Fatalf("expected graphql tools to stay out of the static tool surface, got %v", visible)
	}
	candidates := tools.SearchCandidateToolNames(tools.CatalogToolNames(deps.registry), nil, toolVisibilityOptions(deps.cfg))
	if containsRuntimeTool(candidates, "graphql_query") || containsRuntimeTool(candidates, "graphql_schema_lookup") || containsRuntimeTool(candidates, "graphql_mutation") {
		t.Fatalf("expected graphql tools to stay hidden from tfind candidates, got %v", candidates)
	}
}

func TestAgentRuntimeFactorySkipsGraphQLRegistrationWithoutSources(t *testing.T) {
	setupRuntimeFactoryTestEnv(t)
	store := newRuntimeTestStore(t)

	deps, err := newAgentRuntimeFactory().Build(store)
	if err != nil {
		t.Fatalf("build runtime deps: %v", err)
	}
	t.Cleanup(deps.Close)
	if deps.registry.Get("graphql_query") != nil || deps.registry.Get("graphql_schema_lookup") != nil {
		t.Fatal("expected graphql tools to stay unregistered without graphql sources")
	}
}

func TestAgentRuntimeFactoryFailsOnMissingGraphQLSchemaSnapshot(t *testing.T) {
	tempDir := setupRuntimeFactoryTestEnv(t)
	writeRuntimeGraphQLConfig(t, tempDir, `
[[graphql_sources]]
name = "crm"
endpoint = "https://graphql.test/query"
schema_path = ""
timeout_ms = 3000
max_response_bytes = 4096
max_depth = 5
max_fields = 24
max_root_fields = 2
max_fragments = 3
`)
	if _, err := bridgeconfig.NewStoreFromEnv(); err == nil {
		t.Fatal("expected missing graphql schema snapshot error")
	}
}

func TestAgentRuntimeFactoryIgnoresInvalidGraphQLSchemaSnapshot(t *testing.T) {
	tempDir := setupRuntimeFactoryTestEnv(t)
	badPath := filepath.Join(tempDir, "bad-graphql-schema.json")
	if err := os.WriteFile(badPath, []byte(`{"root_queries":[{"name":"","return_type":"Viewer"}],"types":[]}`), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	writeRuntimeGraphQLConfig(t, tempDir, fmt.Sprintf(`
[[graphql_sources]]
name = "crm"
endpoint = "https://graphql.test/query"
schema_path = %q
timeout_ms = 3000
max_response_bytes = 4096
max_depth = 5
max_fields = 24
max_root_fields = 2
max_fragments = 3
	`, badPath))
	store := newRuntimeTestStore(t)

	deps, err := newAgentRuntimeFactory().Build(store)
	if err != nil {
		t.Fatalf("build runtime deps: %v", err)
	}
	t.Cleanup(deps.Close)
}

func TestAgentRuntimeFactoryIgnoresInvalidGraphQLMutationPolicy(t *testing.T) {
	tempDir := setupRuntimeFactoryTestEnv(t)
	writeRuntimeGraphQLConfig(t, tempDir, fmt.Sprintf(`
[[graphql_sources]]
name = "crm"
endpoint = "https://graphql.test/query"
schema_path = %q
timeout_ms = 3000
max_response_bytes = 4096
max_depth = 5
max_fields = 24
max_root_fields = 2
max_fragments = 3

[[graphql_sources.domains]]
name = "viewer"
root_queries = ["viewer"]
types = ["Viewer", "MutationPayload"]

[[graphql_mutation_policies]]
name = "bad_policy"
source = "crm"
domain = "viewer"
root_mutation = "archiveViewer"
idempotency_mode = "header"
idempotency_header = "Idempotency-Key"
	`, writeRuntimeGraphQLSchema(t, tempDir)))
	store := newRuntimeTestStore(t)

	deps, err := newAgentRuntimeFactory().Build(store)
	if err != nil {
		t.Fatalf("build runtime deps: %v", err)
	}
	t.Cleanup(deps.Close)
}

func TestAgentRuntimeFactorySkipsMemoryAugmentationWhenDisabled(t *testing.T) {
	tempDir := setupRuntimeFactoryTestEnv(t)
	memoryPath := filepath.Join(tempDir, "disabled-memory", "memory.db")

	t.Setenv("GHOST_MEMORY_PATH", memoryPath)
	t.Setenv("GHOST_MEMORY_AUGMENTATION_ENABLED", "false")

	store := newRuntimeTestStore(t)
	deps, err := newAgentRuntimeFactory().Build(store)
	if err != nil {
		t.Fatalf("build runtime deps: %v", err)
	}
	t.Cleanup(deps.Close)

	if deps.memoryRecall != nil {
		t.Fatal("expected memory recall service to stay disabled")
	}
	if deps.memoryLearn != nil {
		t.Fatal("expected memory learning service to stay disabled")
	}
	for _, name := range []string{"memory_manage", "memory_learned_list", "memory_recall_debug"} {
		if deps.registry.Get(name) != nil {
			t.Fatalf("expected %s to stay hidden when memory augmentation is disabled", name)
		}
	}
	if _, err := os.Stat(memoryPath); !os.IsNotExist(err) {
		t.Fatalf("expected disabled memory augmentation to avoid creating sqlite db, got err=%v", err)
	}
}

type fakeTaskManager struct{}

func (fakeTaskManager) CreateAgentTask(context.Context, tools.TaskCreateRequest, string) (tools.TaskPayload, error) {
	return tools.TaskPayload{}, nil
}

func (fakeTaskManager) UpdateAgentTask(context.Context, tools.TaskUpdateRequest, string) (tools.TaskPayload, error) {
	return tools.TaskPayload{}, nil
}

func (fakeTaskManager) GetTask(context.Context, string, string) (tools.TaskPayload, error) {
	return tools.TaskPayload{}, nil
}

func (fakeTaskManager) ListTasks(context.Context, string) ([]tools.TaskPayload, error) {
	return nil, nil
}

func (fakeTaskManager) DeleteTask(context.Context, string, string) (tools.TaskDeleteResult, error) {
	return tools.TaskDeleteResult{}, nil
}

func newRuntimeTestStore(t *testing.T) *ConfigStore {
	t.Helper()

	store, err := bridgeconfig.NewStoreFromEnv()
	if err != nil {
		t.Fatalf("new config store: %v", err)
	}
	return store
}

func setupRuntimeFactoryTestEnv(t *testing.T) string {
	t.Helper()

	tempDir := t.TempDir()
	t.Setenv("GHOST_CONFIG_PATH", filepath.Join(tempDir, "config.toml"))
	t.Setenv("GHOST_API_KEY", "test-key")
	t.Setenv("GHOST_ARTIFACTS_PATH", filepath.Join(tempDir, "artifacts"))
	t.Setenv("GHOST_MEMORY_PATH", filepath.Join(tempDir, "memory", "memory.db"))
	t.Setenv("GHOST_RSS_FEEDS_PATH", filepath.Join(tempDir, "rss", "feeds.json"))
	return tempDir
}

func writeRuntimeGraphQLSchema(t *testing.T, tempDir string) string {
	t.Helper()

	path := filepath.Join(tempDir, "graphql-schema.json")
	if err := os.WriteFile(path, []byte(`{
  "root_queries": [{"name":"viewer","return_type":"Viewer"}],
  "root_mutations": [{"name":"updateViewer","return_type":"MutationPayload"}],
  "types": [
    {"name":"Viewer","fields":[{"name":"id","return_type":"ID!"}]},
    {"name":"MutationPayload","fields":[{"name":"ok","return_type":"Boolean!"}]}
  ]
}`), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return path
}

func writeRuntimeGraphQLConfig(t *testing.T, tempDir string, body string) {
	t.Helper()
	configPath := filepath.Join(tempDir, "config.toml")
	if err := os.WriteFile(configPath, []byte(body), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}

func containsRuntimeTool(names []string, target string) bool {
	for _, name := range names {
		if name == target {
			return true
		}
	}
	return false
}
