package runtime

import (
	"context"
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

func TestAgentRuntimeFactoryRegistersGraphQLToolsWhenConfigured(t *testing.T) {
	tempDir := setupRuntimeFactoryTestEnv(t)
	t.Setenv("GHOST_TOOL_SEARCH_ENABLED", "true")
	writeRuntimeGraphQLConfig(t, tempDir, bridgeconfig.FileConfig{
		GraphQLDefaultSource: bridgeconfig.OptionalStringPointer("crm"),
		GraphQLSources: []bridgeconfig.GraphQLSourceFileConfig{{
			Name:             "crm",
			Endpoint:         "https://graphql.test/query",
			SchemaPath:       writeRuntimeGraphQLSchema(t, tempDir),
			TimeoutMS:        3000,
			MaxResponseBytes: 4096,
			MaxDepth:         5,
			MaxFields:        24,
			MaxRootFields:    2,
			MaxFragments:     3,
			Domains: []bridgeconfig.GraphQLDomainFileConfig{{
				Name:        "viewer",
				RootQueries: []string{"viewer"},
				Types:       []string{"Viewer", "MutationPayload"},
			}},
		}},
		GraphQLMutationPolicies: []bridgeconfig.GraphQLMutationPolicyFileConfig{{
			Name:              "update_viewer",
			Source:            "crm",
			Domain:            "viewer",
			RootMutation:      "updateViewer",
			IdempotencyMode:   "header",
			IdempotencyHeader: "Idempotency-Key",
		}},
	})
	store := newRuntimeTestStore(t)

	deps, err := newAgentRuntimeFactory().Build(store)
	if err != nil {
		t.Fatalf("build runtime deps: %v", err)
	}
	t.Cleanup(deps.Close)

	for _, name := range []string{"graphql_query", "graphql_schema_lookup", "graphql_mutation"} {
		if deps.registry.Get(name) == nil {
			t.Fatalf("expected %s to be registered", name)
		}
	}

	visible := tools.StaticVisibleToolNames(tools.CatalogToolNames(deps.registry), toolVisibilityOptions(deps.cfg))
	if containsRuntimeTool(visible, "graphql_query") || containsRuntimeTool(visible, "graphql_schema_lookup") || containsRuntimeTool(visible, "graphql_mutation") {
		t.Fatalf("expected graphql tools to stay out of the static tool surface, got %v", visible)
	}
	candidates := tools.SearchCandidateToolNames(tools.CatalogToolNames(deps.registry), nil, toolVisibilityOptions(deps.cfg))
	if !containsRuntimeTool(candidates, "graphql_query") || !containsRuntimeTool(candidates, "graphql_schema_lookup") || !containsRuntimeTool(candidates, "graphql_mutation") {
		t.Fatalf("expected graphql tools to be discoverable via tfind, got %v", candidates)
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
	writeRuntimeGraphQLConfig(t, tempDir, bridgeconfig.FileConfig{
		GraphQLSources: []bridgeconfig.GraphQLSourceFileConfig{{
			Name:             "crm",
			Endpoint:         "https://graphql.test/query",
			SchemaPath:       "",
			TimeoutMS:        3000,
			MaxResponseBytes: 4096,
			MaxDepth:         5,
			MaxFields:        24,
			MaxRootFields:    2,
			MaxFragments:     3,
		}},
	})
	if _, err := bridgeconfig.NewStoreFromEnv(); err == nil {
		t.Fatal("expected missing graphql schema snapshot error")
	}
}

func TestAgentRuntimeFactoryFailsOnInvalidGraphQLSchemaSnapshot(t *testing.T) {
	tempDir := setupRuntimeFactoryTestEnv(t)
	badPath := filepath.Join(tempDir, "bad-graphql-schema.json")
	if err := os.WriteFile(badPath, []byte(`{"root_queries":[{"name":"","return_type":"Viewer"}],"types":[]}`), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	writeRuntimeGraphQLConfig(t, tempDir, bridgeconfig.FileConfig{
		GraphQLSources: []bridgeconfig.GraphQLSourceFileConfig{{
			Name:             "crm",
			Endpoint:         "https://graphql.test/query",
			SchemaPath:       badPath,
			TimeoutMS:        3000,
			MaxResponseBytes: 4096,
			MaxDepth:         5,
			MaxFields:        24,
			MaxRootFields:    2,
			MaxFragments:     3,
		}},
	})
	store := newRuntimeTestStore(t)

	_, err := newAgentRuntimeFactory().Build(store)
	if err == nil {
		t.Fatal("expected invalid graphql schema snapshot error")
	}
}

func TestAgentRuntimeFactoryFailsOnInvalidGraphQLMutationPolicy(t *testing.T) {
	tempDir := setupRuntimeFactoryTestEnv(t)
	writeRuntimeGraphQLConfig(t, tempDir, bridgeconfig.FileConfig{
		GraphQLSources: []bridgeconfig.GraphQLSourceFileConfig{{
			Name:             "crm",
			Endpoint:         "https://graphql.test/query",
			SchemaPath:       writeRuntimeGraphQLSchema(t, tempDir),
			TimeoutMS:        3000,
			MaxResponseBytes: 4096,
			MaxDepth:         5,
			MaxFields:        24,
			MaxRootFields:    2,
			MaxFragments:     3,
			Domains: []bridgeconfig.GraphQLDomainFileConfig{{
				Name:        "viewer",
				RootQueries: []string{"viewer"},
				Types:       []string{"Viewer", "MutationPayload"},
			}},
		}},
		GraphQLMutationPolicies: []bridgeconfig.GraphQLMutationPolicyFileConfig{{
			Name:              "bad_policy",
			Source:            "crm",
			Domain:            "viewer",
			RootMutation:      "archiveViewer",
			IdempotencyMode:   "header",
			IdempotencyHeader: "Idempotency-Key",
		}},
	})
	store := newRuntimeTestStore(t)

	if _, err := newAgentRuntimeFactory().Build(store); err == nil {
		t.Fatal("expected invalid graphql mutation policy error")
	}
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

func writeRuntimeGraphQLConfig(t *testing.T, tempDir string, cfg bridgeconfig.FileConfig) {
	t.Helper()
	configPath := filepath.Join(tempDir, "config.toml")
	if err := bridgeconfig.WriteBridgeFileConfig(configPath, cfg); err != nil {
		t.Fatalf("WriteBridgeFileConfig: %v", err)
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
