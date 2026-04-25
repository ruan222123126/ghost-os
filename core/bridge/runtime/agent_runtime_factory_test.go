package runtime

import (
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

func TestAgentRuntimeFactorySkipsWebRooterWhenDisabled(t *testing.T) {
	setupRuntimeFactoryTestEnv(t)
	store := newRuntimeTestStore(t)

	deps, err := newAgentRuntimeFactory().Build(store)
	if err != nil {
		t.Fatalf("build runtime deps: %v", err)
	}
	t.Cleanup(deps.Close)

	if deps.registry.Get("web_rooter") != nil {
		t.Fatal("expected web_rooter to stay hidden when disabled")
	}
}

func TestAgentRuntimeFactoryRegistersWebRooterWhenEnabled(t *testing.T) {
	setupRuntimeFactoryTestEnv(t)
	t.Setenv("GHOST_WEB_ROOTER_ENABLED", "true")
	store := newRuntimeTestStore(t)

	deps, err := newAgentRuntimeFactory().Build(store)
	if err != nil {
		t.Fatalf("build runtime deps: %v", err)
	}
	t.Cleanup(deps.Close)

	if deps.registry.Get("web_rooter") == nil {
		t.Fatal("expected web_rooter to be registered")
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

func TestAgentRuntimeFactoryRegistersImageGenerate(t *testing.T) {
	setupRuntimeFactoryTestEnv(t)
	store := newRuntimeTestStore(t)

	deps, err := newAgentRuntimeFactory().Build(store)
	if err != nil {
		t.Fatalf("build runtime deps: %v", err)
	}
	t.Cleanup(deps.Close)

	if deps.registry.Get("image_generate") == nil {
		t.Fatal("expected image_generate to be registered")
	}
}

func TestAgentRuntimeFactoryLoadsToolPromptOverridesFromFiles(t *testing.T) {
	const (
		testDirPerm  = 0o755
		testFilePerm = 0o600
	)

	tempDir := setupRuntimeFactoryTestEnv(t)
	promptsDir := filepath.Join(tempDir, "prompts")
	toolPromptsDir := filepath.Join(promptsDir, "tools")
	if err := os.MkdirAll(toolPromptsDir, testDirPerm); err != nil {
		t.Fatalf("MkdirAll(%s): %v", toolPromptsDir, err)
	}
	promptPath := filepath.Join(toolPromptsDir, "script_exec.md")
	if err := os.WriteFile(promptPath, []byte("from prompt file"), testFilePerm); err != nil {
		t.Fatalf("WriteFile(%s): %v", promptPath, err)
	}
	store := newRuntimeTestStore(t)

	deps, err := newAgentRuntimeFactory().Build(store)
	if err != nil {
		t.Fatalf("build runtime deps: %v", err)
	}
	t.Cleanup(deps.Close)

	if got := deps.cfg.ToolSelector.PromptOverrides["script_exec"]; got != "from prompt file" {
		t.Fatalf("unexpected script_exec override: got %q want %q", got, "from prompt file")
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
	if containsToolName(visible, "graphql_query") || containsToolName(visible, "graphql_schema_lookup") || containsToolName(visible, "graphql_mutation") {
		t.Fatalf("expected graphql tools to stay out of the static tool surface, got %v", visible)
	}
	candidates := tools.SearchCandidateToolNames(tools.CatalogToolNames(deps.registry), nil, toolVisibilityOptions(deps.cfg))
	if containsToolName(candidates, "graphql_query") || containsToolName(candidates, "graphql_schema_lookup") || containsToolName(candidates, "graphql_mutation") {
		t.Fatalf("expected graphql tools to stay hidden from tfind candidates, got %v", candidates)
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
