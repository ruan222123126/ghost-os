package runtime

import (
	"strings"
	"time"

	"ghost-os/bridge/agent"
	"ghost-os/bridge/artifacts"
	"ghost-os/bridge/execution"
	"ghost-os/bridge/llm"
	rsssubscriptions "ghost-os/bridge/rss/subscriptions"
	"ghost-os/bridge/tools"
)

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

type coreToolOptions struct {
	store       *ConfigStore
	cfg         Config
	registry    *tools.Registry
	clients     runtimeClients
	resources   runtimeToolResources
	taskManager tools.TaskManager
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
	registerRuntimeWorkerTool(opts)
	registerRuntimeExecutionTools(opts)
	registerRuntimeWebTools(opts)
	registerRuntimeInteractionTools(opts)
	registerRuntimeOptionalTools(opts)
	opts.registry.Register(tools.NewAskHumanTool())
}

func registerRuntimeWorkerTool(opts coreToolOptions) {
	opts.registry.Register(tools.NewReadAndSummarizeTool(opts.resources.executionClient, opts.clients.worker, tools.ReadAndSummarizeConfig{
		WorkerModel:      opts.clients.workerModel,
		MaxFiles:         opts.cfg.Worker.MaxFiles,
		MaxParallel:      opts.cfg.Worker.MaxConcurrency,
		DefaultMaxChunks: opts.cfg.Worker.MaxFileChunks,
	}))
}

func registerRuntimeExecutionTools(opts coreToolOptions) {
	opts.registry.Register(tools.NewSendFileTool(opts.resources.executionClient, opts.resources.artifactStore))
	opts.registry.Register(tools.NewScriptExecTool(opts.resources.executionClient))
	opts.registry.Register(tools.NewSetProjectRootTool(opts.store, opts.resources.executionClient, opts.cfg.NativeAllowedReadPaths, opts.cfg.NativeAllowedWritePaths))
	opts.registry.Register(tools.NewCodexCLITool(opts.resources.executionClient, opts.cfg.NativePersistent))
}

func registerRuntimeWebTools(opts coreToolOptions) {
	opts.registry.Register(tools.NewWebSearchTool(tools.WebSearchConfig{
		TavilyURL:    opts.cfg.WebSearchTavilyURL,
		ExaURL:       opts.cfg.WebSearchExaURL,
		TavilyAPIKey: opts.cfg.WebSearchTavilyAPIKey,
		ExaAPIKey:    opts.cfg.WebSearchExaAPIKey,
	}))
	if opts.cfg.WebRooterEnabled {
		opts.registry.Register(tools.NewWebRooterTool(tools.WebRooterConfig{
			BaseURL:  opts.cfg.WebRooterBaseURL,
			APIToken: opts.cfg.WebRooterAPIToken,
			Timeout:  time.Duration(opts.cfg.WebRooterTimeoutMS) * time.Millisecond,
		}))
	}
	opts.registry.Register(tools.NewFeedManageTool(opts.resources.feedStore))
	opts.registry.Register(tools.NewRSSFetchTool())
}

func registerRuntimeInteractionTools(opts coreToolOptions) {
	opts.registry.Register(
		tools.NewScreenControlTool(
			opts.resources.executionClient,
			opts.clients.primary,
			opts.resources.artifactStore,
		),
	)
	opts.registry.Register(tools.NewTextInputTool(opts.resources.executionClient))
}

func registerRuntimeOptionalTools(opts coreToolOptions) {
	if opts.taskManager != nil {
		opts.registry.Register(tools.NewTaskManageTool(opts.taskManager))
	}
	if opts.cfg.ToolSearch.Enabled {
		opts.registry.Register(
			tools.NewToolSearchTool(
				opts.registry,
				toolVisibilityOptions(opts.cfg),
				opts.cfg.ToolSearch.IdleTurns,
				tools.ToolSearchOptions{ProjectRoot: opts.cfg.ProjectRoot},
			),
		)
	}
}

func closeRuntimeToolResources(resources runtimeToolResources) {
	_ = closeExecutionClient(resources.executionClient)
}
