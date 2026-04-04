package runtime

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/tools"
)

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
	return WrapConfigStore(store)
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
	data := []byte(`{
  "root_queries": [{"name":"viewer","return_type":"Viewer"}],
  "root_mutations": [{"name":"updateViewer","return_type":"MutationPayload"}],
  "types": [
    {"name":"Viewer","fields":[{"name":"id","return_type":"ID!"}]},
    {"name":"MutationPayload","fields":[{"name":"ok","return_type":"Boolean!"}]}
  ]
}`)
	if err := os.WriteFile(path, data, 0o600); err != nil {
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

func containsToolName(names []string, target string) bool {
	for _, name := range names {
		if name == target {
			return true
		}
	}
	return false
}
