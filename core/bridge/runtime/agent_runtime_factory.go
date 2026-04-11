package runtime

import (
	"os"
	"strings"

	"ghost-os/bridge/agent"
	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/memoryaug"
	"ghost-os/bridge/tools"
)

type agentRuntimeDependencies struct {
	cfg          Config
	client       agent.Completer
	registry     *tools.Registry
	systemPrompt string
	memoryPlan   memoryaug.IntentPlanner
	memoryRecall memoryaug.RecallService
	memoryLearn  memoryaug.LearningService
	cleanup      func()
}

type AgentRuntimeFactory interface {
	Build(store *ConfigStore) (agentRuntimeDependencies, error)
}

type defaultAgentRuntimeFactory struct{}

type taskAwareAgentRuntimeFactory struct {
	taskManager tools.TaskManager
}

type runtimeBuildComponents struct {
	cfg       Config
	clients   runtimeClients
	resources runtimeToolResources
	registry  *tools.Registry
}

func newAgentRuntimeFactory() AgentRuntimeFactory {
	return defaultAgentRuntimeFactory{}
}

func newAgentRuntimeFactoryWithTaskManager(taskManager tools.TaskManager) AgentRuntimeFactory {
	if taskManager == nil {
		return newAgentRuntimeFactory()
	}
	return taskAwareAgentRuntimeFactory{
		taskManager: taskManager,
	}
}

// Build 组装运行 Agent 所需的配置、模型客户端与工具注册表。
func (f defaultAgentRuntimeFactory) Build(store *ConfigStore) (agentRuntimeDependencies, error) {
	return buildAgentRuntimeDependencies(store, nil)
}

func (f taskAwareAgentRuntimeFactory) Build(store *ConfigStore) (agentRuntimeDependencies, error) {
	return buildAgentRuntimeDependencies(store, f.taskManager)
}

func buildAgentRuntimeDependencies(
	store *ConfigStore,
	taskManager tools.TaskManager,
) (agentRuntimeDependencies, error) {
	cfg, err := loadAgentRuntimeBuildConfig(store)
	if err != nil {
		return agentRuntimeDependencies{}, err
	}
	components, err := newRuntimeBuildComponents(cfg, store, taskManager)
	if err != nil {
		return agentRuntimeDependencies{}, err
	}
	return finalizeAgentRuntimeDependencies(components)
}

func newRuntimeBuildComponents(
	cfg Config,
	store *ConfigStore,
	taskManager tools.TaskManager,
) (runtimeBuildComponents, error) {
	resources, err := newRuntimeToolResources(cfg)
	if err != nil {
		return runtimeBuildComponents{}, err
	}
	registry := tools.NewRegistry()
	clients := newRuntimeClients(cfg)
	registerCoreTools(coreToolOptions{
		store:       store,
		cfg:         cfg,
		registry:    registry,
		clients:     clients,
		resources:   resources,
		taskManager: taskManager,
	})
	return runtimeBuildComponents{
		cfg:       cfg,
		clients:   clients,
		resources: resources,
		registry:  registry,
	}, nil
}

func finalizeAgentRuntimeDependencies(
	components runtimeBuildComponents,
) (agentRuntimeDependencies, error) {
	memoryResources, err := setupMemoryAugmentation(components.cfg, components.registry)
	if err != nil {
		closeRuntimeToolResources(components.resources)
		return agentRuntimeDependencies{}, err
	}
	systemPrompt, err := buildRuntimeSystemPrompt(components.cfg, components.registry)
	if err != nil {
		memoryResources.Close()
		closeRuntimeToolResources(components.resources)
		return agentRuntimeDependencies{}, err
	}
	return agentRuntimeDependencies{
		cfg:          components.cfg,
		client:       components.clients.primary,
		registry:     components.registry,
		systemPrompt: systemPrompt,
		memoryPlan:   memoryResources.planner,
		memoryRecall: memoryResources.recall,
		memoryLearn:  memoryResources.learn,
		cleanup: func() {
			memoryResources.Close()
			closeRuntimeToolResources(components.resources)
		},
	}, nil
}

func (d agentRuntimeDependencies) Close() {
	if d.cleanup != nil {
		d.cleanup()
	}
}

func loadAgentRuntimeBuildConfig(store *ConfigStore) (Config, error) {
	cfg, err := loadAgentRuntimeConfig(store)
	if err != nil {
		return Config{}, err
	}
	return withFileToolPromptOverrides(cfg)
}

func loadAgentRuntimeConfig(store *ConfigStore) (Config, error) {
	if store == nil {
		return bridgeconfig.Load()
	}
	return store.Config()
}

func withFileToolPromptOverrides(cfg Config) (Config, error) {
	overrides, err := bridgeconfig.LoadToolPromptOverrides(cfg.PromptsDir)
	if err != nil {
		return Config{}, err
	}
	cfg.ToolSelector.PromptOverrides = overrides
	return cfg, nil
}

func buildRuntimeSystemPrompt(cfg Config, registry *tools.Registry) (string, error) {
	catalog := newToolSelectionPolicy(cfg).residentCatalog(registry)
	return buildSystemPrompt(cfg, catalog)
}

func resolvePromptProjectRoot(projectRoot string) string {
	trimmed := strings.TrimSpace(projectRoot)
	if trimmed != "" {
		return trimmed
	}
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	return cwd
}
