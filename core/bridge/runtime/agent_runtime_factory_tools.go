package runtime

import (
	"ghost-os/bridge/agent"
	"ghost-os/bridge/artifacts"
	"ghost-os/bridge/execution"
	"ghost-os/bridge/llm"
	"ghost-os/bridge/tools"
)

type runtimeClients struct {
	primary agent.Completer
}

type runtimeToolResources struct {
	artifactStore   *artifacts.SessionArtifactStore
	executionClient execution.Client
}

type coreToolOptions struct {
	cfg       Config
	registry  *tools.Registry
	clients   runtimeClients
	resources runtimeToolResources
}

func newRuntimeClients(cfg Config) runtimeClients {
	return runtimeClients{
		primary: llm.NewClientWithOptions(providerClientOptions(cfg, cfg.Provider.Model)),
	}
}

func newRuntimeToolResources(cfg Config) (runtimeToolResources, error) {
	artifactStore, err := artifacts.NewSessionArtifactStoreFromEnv()
	if err != nil {
		return runtimeToolResources{}, err
	}
	execCfg := executionClientConfigFromConfig(cfg)
	return runtimeToolResources{
		artifactStore:   artifactStore,
		executionClient: newExecutionClient(execCfg),
	}, nil
}

func registerCoreTools(opts coreToolOptions) {
	registerRuntimeExecutionTools(opts)
	registerRuntimeWebTools(opts)
	registerRuntimeInteractionTools(opts)
	registerRuntimeOptionalTools(opts)
	opts.registry.Register(tools.NewAskHumanTool())
}

func registerRuntimeExecutionTools(opts coreToolOptions) {
	opts.registry.Register(tools.NewScriptExecTool(opts.resources.executionClient))
	opts.registry.Register(tools.NewCodexCLITool(opts.resources.executionClient, opts.cfg.NativePersistent))
}

func registerRuntimeWebTools(opts coreToolOptions) {
	opts.registry.Register(tools.NewWebSearchTool(tools.WebSearchConfig{
		TavilyURL:    opts.cfg.WebSearchTavilyURL,
		ExaURL:       opts.cfg.WebSearchExaURL,
		TavilyAPIKey: opts.cfg.WebSearchTavilyAPIKey,
		ExaAPIKey:    opts.cfg.WebSearchExaAPIKey,
	}))
}

func registerRuntimeInteractionTools(opts coreToolOptions) {
	opts.registry.Register(
		tools.NewScreenControlTool(
			opts.resources.executionClient,
			opts.clients.primary,
			opts.resources.artifactStore,
		),
	)
}

func registerRuntimeOptionalTools(opts coreToolOptions) {
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
