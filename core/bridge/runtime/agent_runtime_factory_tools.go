package runtime

import (
	"ghost-os/bridge/agent"
	"ghost-os/bridge/artifacts"
	"ghost-os/bridge/llm"
	"ghost-os/bridge/skills"
	"ghost-os/bridge/tools"
)

type runtimeClients struct {
	primary agent.Completer
}

type runtimeToolResources struct {
	artifactStore              *artifacts.SessionArtifactStore
	workspaceExecutionClient   Client
	interactionExecutionClient Client
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
	workspaceCfg := executionClientConfigFromConfig(cfg)
	interactionCfg := interactionExecutionClientConfigFromConfig(cfg)
	return runtimeToolResources{
		artifactStore:              artifactStore,
		workspaceExecutionClient:   newExecutionClient(workspaceCfg),
		interactionExecutionClient: newExecutionClient(interactionCfg),
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
	registerRuntimeFileTools(opts)
	opts.registry.Register(
		tools.NewScriptExecTool(
			opts.resources.workspaceExecutionClient,
			opts.cfg.ScriptExecSandboxMemoryMB,
		),
	)
	opts.registry.Register(tools.NewBashExecTool(opts.resources.workspaceExecutionClient))
	opts.registry.Register(
		tools.NewCodexCLITool(
			opts.resources.workspaceExecutionClient,
			opts.cfg.CodexCLIPath,
			opts.cfg.NodeBinPath,
		),
	)
}

func registerRuntimeFileTools(opts coreToolOptions) {
	opts.registry.Register(tools.NewListFilesTool(opts.resources.workspaceExecutionClient))
	opts.registry.Register(tools.NewReadFileTool(opts.resources.workspaceExecutionClient))
	opts.registry.Register(tools.NewSearchFilesTool(opts.resources.workspaceExecutionClient))
	opts.registry.Register(tools.NewWriteFileTool(opts.resources.workspaceExecutionClient))
	opts.registry.Register(tools.NewApplyDiffTool(opts.resources.workspaceExecutionClient))
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
			opts.resources.interactionExecutionClient,
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
				tools.ToolSearchOptions{
					SkillConfig: skills.Config{
						ProjectRoot:    opts.cfg.ProjectRoot,
						SkillBlocklist: append([]string(nil), opts.cfg.SkillBlocklist...),
					},
				},
			),
		)
	}
}

func closeRuntimeToolResources(resources runtimeToolResources) {
	_ = closeExecutionClient(resources.workspaceExecutionClient)
	_ = closeExecutionClient(resources.interactionExecutionClient)
}
