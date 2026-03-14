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
