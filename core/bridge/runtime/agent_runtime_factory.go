package runtime

import (
	"os"
	goruntime "runtime"
	"strconv"
	"strings"

	"ghost-os/bridge/agent"
	"ghost-os/bridge/artifacts"
	ctxmgr "ghost-os/bridge/context"
	"ghost-os/bridge/execution"
	"ghost-os/bridge/llm"
	"ghost-os/bridge/memoryaug"
	"ghost-os/bridge/memorystore"
	"ghost-os/bridge/tools"
)

type agentRuntimeDependencies struct {
	cfg          Config
	client       agent.Completer
	registry     *tools.Registry
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
	feedStore       *tools.FeedStore
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
	feedStore, err := tools.NewFeedStore(cfg.RSS.FeedsPath)
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
	if containsToolName(opts.cfg.ToolSelector.Allowlist, "codex_cli") {
		opts.registry.Register(tools.NewCodexCLITool(opts.resources.executionClient, opts.cfg.NativePersistent))
	}
	opts.registry.Register(tools.NewWebSearchTool(tools.WebSearchConfig{TavilyAPIKey: opts.cfg.WebSearchTavilyAPIKey}))
	opts.registry.Register(tools.NewFeedSubscribeTool(opts.resources.feedStore))
	opts.registry.Register(tools.NewFeedListTool(opts.resources.feedStore))
	opts.registry.Register(tools.NewFeedUpdateTool(opts.resources.feedStore))
	opts.registry.Register(tools.NewFeedUnsubscribeTool(opts.resources.feedStore))
	opts.registry.Register(tools.NewRSSFetchTool())
	opts.registry.Register(tools.NewScreenActionTool(opts.resources.executionClient))
	opts.registry.Register(tools.NewBrowserControlTool(opts.resources.executionClient))
	opts.registry.Register(tools.NewTextInputTool(opts.resources.executionClient))
	if opts.taskManager != nil {
		opts.registry.Register(tools.NewTaskManageTool(opts.taskManager))
	}
	opts.registry.Register(tools.NewAskHumanTool())
}

func setupMemoryAugmentation(cfg Config, registry *tools.Registry) (memoryRuntimeResources, error) {
	settings := memorySettingsFromConfig(cfg.MemoryAugmentation)
	if !settings.Enabled {
		return memoryRuntimeResources{}, nil
	}
	store, err := memorystore.NewStoreWithOptions(memorystore.StoreOptions{
		Path:               cfg.MemoryPath,
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

func memorySettingsFromConfig(cfg MemoryAugmentationConfig) memoryaug.Settings {
	return memoryaug.Settings{
		Enabled:             cfg.Enabled,
		LearningEnabled:     cfg.LearningEnabled,
		RecallEnabled:       cfg.RecallEnabled,
		MaxRecallItems:      cfg.MaxRecallItems,
		MinConfidence:       cfg.MinConfidence,
		SessionScopeEnabled: cfg.SessionScopeEnabled,
		UserScopeEnabled:    cfg.UserScopeEnabled,
		LLMModel:            cfg.LLMModel,
		UserScopeID:         cfg.UserScopeID,
	}
}

func buildRuntimeSystemPrompt(cfg Config, registry *tools.Registry) (string, error) {
	promptManager, err := ctxmgr.NewPromptManagerWithOptions(ctxmgr.PromptLoadOptions{
		ConfigPath: cfg.PromptsPath,
		CoreDir:    cfg.PromptsDir,
		CoreFiles:  cfg.PromptsCoreFiles,
	})
	if err != nil {
		if len(cfg.PromptsCoreFiles) > 0 {
			return "", err
		}
		promptManager = ctxmgr.NewPromptManagerWithDefault()
	}
	contextBuilder := ctxmgr.NewBuilder(promptManager, registry)
	return contextBuilder.BuildSystemPrompt(map[string]string{
		"os_type":      goruntime.GOOS,
		"tools_count":  strconv.Itoa(len(registry.ToolDefs())),
		"max_turns":    strconv.Itoa(cfg.MaxTurns),
		"project_root": resolvePromptProjectRoot(cfg.ProjectRoot),
	}), nil
}

func closeRuntimeToolResources(resources runtimeToolResources) {
	_ = closeExecutionClient(resources.executionClient)
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
