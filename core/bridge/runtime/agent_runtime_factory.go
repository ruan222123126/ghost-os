package runtime

import (
	"os"
	"strings"

	"ghost-os/bridge/agent"
	"ghost-os/bridge/artifacts"
	"ghost-os/bridge/execution"
	"ghost-os/bridge/llm"
	"ghost-os/bridge/memoryaug"
	"ghost-os/bridge/memorystore"
	rsssubscriptions "ghost-os/bridge/rss/subscriptions"
	"ghost-os/bridge/tools"
)

type agentRuntimeDependencies struct {
	cfg          Config
	client       agent.Completer
	registry     *tools.Registry
	graphQL      *tools.GraphQLSourceRegistry
	systemPrompt string
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

type runtimeClients struct {
	primary     agent.Completer
	worker      agent.Completer
	workerModel string
}

type runtimeToolResources struct {
	artifactStore   *artifacts.SessionArtifactStore
	executionClient execution.Client
	feedStore       *rsssubscriptions.FeedStore
}

type memoryRuntimeResources struct {
	recall  memoryaug.RecallService
	learn   memoryaug.LearningService
	cleanup func()
}

type coreToolOptions struct {
	store       *ConfigStore
	cfg         Config
	registry    *tools.Registry
	clients     runtimeClients
	resources   runtimeToolResources
	taskManager tools.TaskManager
}

func newAgentRuntimeFactory() AgentRuntimeFactory {
	return defaultAgentRuntimeFactory{}
}

func newAgentRuntimeFactoryWithTaskManager(taskManager tools.TaskManager) AgentRuntimeFactory {
	if taskManager == nil {
		return newAgentRuntimeFactory()
	}
	return taskAwareAgentRuntimeFactory{taskManager: taskManager}
}

// Build 组装运行 Agent 所需的配置、模型客户端与工具注册表。
func (defaultAgentRuntimeFactory) Build(store *ConfigStore) (agentRuntimeDependencies, error) {
	return buildAgentRuntimeDependencies(store, nil)
}

func (f taskAwareAgentRuntimeFactory) Build(store *ConfigStore) (agentRuntimeDependencies, error) {
	return buildAgentRuntimeDependencies(store, f.taskManager)
}

func buildAgentRuntimeDependencies(store *ConfigStore, taskManager tools.TaskManager) (agentRuntimeDependencies, error) {
	cfg, err := loadAgentRuntimeConfig(store)
	if err != nil {
		return agentRuntimeDependencies{}, err
	}
	clients := newRuntimeClients(cfg)
	resources, err := newRuntimeToolResources(cfg)
	if err != nil {
		return agentRuntimeDependencies{}, err
	}
	registry := tools.NewRegistry()
	registerCoreTools(coreToolOptions{
		store:       store,
		cfg:         cfg,
		registry:    registry,
		clients:     clients,
		resources:   resources,
		taskManager: taskManager,
	})
	graphQLRegistry, err := buildGraphQLSourceRegistry(cfg)
	if err != nil {
		closeRuntimeToolResources(resources)
		return agentRuntimeDependencies{}, err
	}
	memoryResources, err := setupMemoryAugmentation(cfg, registry)
	if err != nil {
		closeRuntimeToolResources(resources)
		return agentRuntimeDependencies{}, err
	}
	systemPrompt, err := buildRuntimeSystemPrompt(cfg, registry)
	if err != nil {
		memoryResources.Close()
		closeRuntimeToolResources(resources)
		return agentRuntimeDependencies{}, err
	}
	return agentRuntimeDependencies{
		cfg:          cfg,
		client:       clients.primary,
		registry:     registry,
		graphQL:      graphQLRegistry,
		systemPrompt: systemPrompt,
		memoryRecall: memoryResources.recall,
		memoryLearn:  memoryResources.learn,
		cleanup: func() {
			memoryResources.Close()
			closeRuntimeToolResources(resources)
		},
	}, nil
}

func (d agentRuntimeDependencies) Close() {
	if d.cleanup != nil {
		d.cleanup()
	}
}

func (d memoryRuntimeResources) Close() {
	if d.cleanup != nil {
		d.cleanup()
	}
}

func loadAgentRuntimeConfig(store *ConfigStore) (Config, error) {
	if store == nil {
		runtimeCfg, err := runtimeConfigFromEnv()
		if err != nil {
			return Config{}, err
		}
		return loadConfigWithRuntime(runtimeCfg)
	}
	return loadConfigWithRuntime(store.RuntimeConfig())
}

func newRuntimeClients(cfg Config) runtimeClients {
	workerModel := strings.TrimSpace(cfg.Worker.Model)
	return runtimeClients{
		primary:     llm.NewClientWithOptions(providerClientOptions(cfg, cfg.Provider.Model)),
		worker:      llm.NewClientWithOptions(providerClientOptions(cfg, workerModel)),
		workerModel: workerModel,
	}
}

func newRuntimeToolResources(cfg Config) (runtimeToolResources, error) {
	artifactStore, err := artifacts.NewSessionArtifactStoreFromEnv()
	if err != nil {
		return runtimeToolResources{}, err
	}
	feedStore, err := rsssubscriptions.NewFeedStore(cfg.RSS.FeedsPath)
	if err != nil {
		return runtimeToolResources{}, err
	}
	return runtimeToolResources{
		artifactStore:   artifactStore,
		executionClient: newExecutionClient(executionClientConfigFromConfig(cfg)),
		feedStore:       feedStore,
	}, nil
}

func registerCoreTools(opts coreToolOptions) {
	opts.registry.Register(tools.NewReadAndSummarizeTool(opts.resources.executionClient, opts.clients.worker, tools.ReadAndSummarizeConfig{
		WorkerModel:      opts.clients.workerModel,
		MaxFiles:         opts.cfg.Worker.MaxFiles,
		MaxParallel:      opts.cfg.Worker.MaxConcurrency,
		DefaultMaxChunks: opts.cfg.Worker.MaxFileChunks,
	}))
	opts.registry.Register(tools.NewSendFileTool(opts.resources.executionClient, opts.resources.artifactStore))
	opts.registry.Register(tools.NewScriptExecTool(opts.resources.executionClient))
	opts.registry.Register(tools.NewSetProjectRootTool(opts.store, opts.resources.executionClient, opts.cfg.NativeAllowedReadPaths, opts.cfg.NativeAllowedWritePaths))
	opts.registry.Register(tools.NewCodexCLITool(opts.resources.executionClient, opts.cfg.NativePersistent))
	opts.registry.Register(tools.NewWebSearchTool(tools.WebSearchConfig{
		TavilyAPIKey: opts.cfg.WebSearchTavilyAPIKey,
		ExaAPIKey:    opts.cfg.WebSearchExaAPIKey,
	}))
	opts.registry.Register(tools.NewFeedManageTool(opts.resources.feedStore))
	opts.registry.Register(tools.NewRSSFetchTool())
	opts.registry.Register(tools.NewScreenActionTool(opts.resources.executionClient))
	opts.registry.Register(tools.NewBrowserControlTool(opts.resources.executionClient))
	opts.registry.Register(tools.NewTextInputTool(opts.resources.executionClient))
	if opts.taskManager != nil {
		opts.registry.Register(tools.NewTaskManageTool(opts.taskManager))
	}
	if opts.cfg.ToolSearch.Enabled {
		opts.registry.Register(tools.NewToolSearchTool(opts.registry, toolVisibilityOptions(opts.cfg), opts.cfg.ToolSearch.IdleTurns))
	}
	opts.registry.Register(tools.NewAskHumanTool())
}

func setupMemoryAugmentation(cfg Config, registry *tools.Registry) (memoryRuntimeResources, error) {
	settings := memorySettingsFromConfig(cfg)
	if !settings.Enabled {
		return memoryRuntimeResources{}, nil
	}
	store, err := memorystore.NewStoreWithOptions(memorystore.StoreOptions{
		Path:               strings.TrimSpace(os.Getenv("GHOST_MEMORY_PATH")),
		DefaultUserScopeID: settings.UserScopeID,
	})
	if err != nil {
		return memoryRuntimeResources{}, err
	}
	memoryClient := llm.NewClientWithOptions(providerClientOptions(cfg, strings.TrimSpace(settings.LLMModel)))
	recall := memoryaug.NewRecallService(settings, store)
	learn := memoryaug.NewLearningService(settings, store, memoryaug.NewLLMExtractor(memoryClient))
	registry.Register(tools.NewMemoryManageTool(store))
	registry.Register(tools.NewMemoryLearnedListTool(store))
	registry.Register(tools.NewMemoryRecallDebugTool(recall, settings.UserScopeID))
	return memoryRuntimeResources{
		recall: recall,
		learn:  learn,
		cleanup: func() {
			_ = store.Close()
		},
	}, nil
}

func memorySettingsFromConfig(cfg Config) memoryaug.Settings {
	return memoryaug.Settings{
		Enabled:             cfg.MemoryAugmentation.Enabled,
		LearningEnabled:     cfg.MemoryAugmentation.LearningEnabled,
		RecallEnabled:       cfg.MemoryAugmentation.RecallEnabled,
		MaxRecallItems:      cfg.MemoryAugmentation.MaxRecallItems,
		MinConfidence:       cfg.MemoryAugmentation.MinConfidence,
		SessionScopeEnabled: cfg.MemoryAugmentation.SessionScopeEnabled,
		UserScopeEnabled:    cfg.MemoryAugmentation.UserScopeEnabled,
		LLMModel:            cfg.MemoryAugmentation.LLMModel,
		UserScopeID:         cfg.MemoryAugmentation.UserScopeID,
	}
}

func buildRuntimeSystemPrompt(cfg Config, registry *tools.Registry) (string, error) {
	catalog := newToolSelectionPolicy(cfg).scopeCatalog(registry)
	return buildSystemPrompt(cfg, catalog)
}

func closeRuntimeToolResources(resources runtimeToolResources) {
	_ = closeExecutionClient(resources.executionClient)
}

func buildGraphQLSourceRegistry(cfg Config) (*tools.GraphQLSourceRegistry, error) {
	if len(cfg.GraphQL.Sources) == 0 {
		return nil, nil
	}
	return tools.NewGraphQLSourceRegistry(graphQLRegistryConfig(cfg))
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
