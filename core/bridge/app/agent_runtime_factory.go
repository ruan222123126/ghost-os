package app

import (
	"runtime"
	"strconv"
	"strings"

	"ghost-os/bridge/agent"
	"ghost-os/bridge/artifacts"
	ctxmgr "ghost-os/bridge/context"
	"ghost-os/bridge/llm"
	"ghost-os/bridge/tools"
)

type agentRuntimeDependencies struct {
	cfg          Config
	client       agent.Completer
	registry     *tools.Registry
	systemPrompt string
	cleanup      func()
}

type AgentRuntimeFactory interface {
	Build(store *ConfigStore) (agentRuntimeDependencies, error)
}

type defaultAgentRuntimeFactory struct{}

type taskAwareAgentRuntimeFactory struct {
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
	var runtimeCfg runtimeConfig
	var err error
	if store == nil {
		runtimeCfg, err = runtimeConfigFromEnv()
		if err != nil {
			return agentRuntimeDependencies{}, err
		}
	} else {
		runtimeCfg = store.RuntimeConfig()
	}

	cfg, err := loadConfigWithRuntime(runtimeCfg)
	if err != nil {
		return agentRuntimeDependencies{}, err
	}

	client := llm.NewClientWithOptions(providerClientOptions(cfg, cfg.Provider.Model))
	workerModel := strings.TrimSpace(cfg.Worker.Model)
	workerClient := llm.NewClientWithOptions(providerClientOptions(cfg, workerModel))

	registry := tools.NewRegistry()
	artifactStore, err := artifacts.NewSessionArtifactStoreFromEnv()
	if err != nil {
		return agentRuntimeDependencies{}, err
	}
	feedStore, err := tools.NewFeedStore(cfg.RSS.FeedsPath)
	if err != nil {
		return agentRuntimeDependencies{}, err
	}
	memoryTool, err := tools.NewMemoryManageToolFromEnv()
	if err != nil {
		return agentRuntimeDependencies{}, err
	}
	executionClient := newExecutionClient(executionClientConfigFromConfig(cfg))
	registry.Register(tools.NewListFilesTool(executionClient))
	registry.Register(tools.NewReadFileTool(executionClient))
	registry.Register(tools.NewReadAndSummarizeTool(executionClient, workerClient, tools.ReadAndSummarizeConfig{
		WorkerModel:      workerModel,
		MaxFiles:         cfg.Worker.MaxFiles,
		MaxParallel:      cfg.Worker.MaxConcurrency,
		DefaultMaxChunks: cfg.Worker.MaxFileChunks,
	}))
	registry.Register(tools.NewSearchFilesTool(executionClient))
	registry.Register(tools.NewSendFileTool(executionClient, artifactStore))
	registry.Register(tools.NewApplyDiffTool(executionClient))
	registry.Register(tools.NewBashExecTool(executionClient))
	registry.Register(tools.NewScriptExecTool(executionClient))
	if containsToolName(cfg.ToolSelector.Allowlist, "codex_cli") {
		registry.Register(tools.NewCodexCLITool(executionClient, cfg.NativePersistent))
	}
	registry.Register(tools.NewWebSearchTool(tools.WebSearchConfig{
		TavilyAPIKey: cfg.WebSearchTavilyAPIKey,
	}))
	registry.Register(tools.NewFeedSubscribeTool(feedStore))
	registry.Register(tools.NewFeedListTool(feedStore))
	registry.Register(tools.NewFeedUpdateTool(feedStore))
	registry.Register(tools.NewFeedUnsubscribeTool(feedStore))
	registry.Register(tools.NewRSSFetchTool())
	registry.Register(memoryTool)
	registry.Register(tools.NewScreenActionTool(executionClient))
	registry.Register(tools.NewBrowserControlTool(executionClient))
	registry.Register(tools.NewTextInputTool(executionClient))
	if taskManager != nil {
		registry.Register(tools.NewTaskManageTool(taskManager))
	}
	registry.Register(tools.NewAskHumanTool())

	promptManager, err := ctxmgr.NewPromptManager(cfg.PromptsPath)
	if err != nil {
		promptManager = ctxmgr.NewPromptManagerWithDefault()
	}
	contextBuilder := ctxmgr.NewBuilder(promptManager, registry)
	systemPrompt := contextBuilder.BuildSystemPrompt(map[string]string{
		"os_type":     runtime.GOOS,
		"tools_count": strconv.Itoa(len(registry.ToolDefs())),
		"max_turns":   strconv.Itoa(cfg.MaxTurns),
	})

	return agentRuntimeDependencies{
		cfg:          cfg,
		client:       client,
		registry:     registry,
		systemPrompt: systemPrompt,
		cleanup: func() {
			if memoryTool != nil {
				_ = memoryTool.Close()
			}
			_ = closeExecutionClient(executionClient)
		},
	}, nil
}

func (d agentRuntimeDependencies) Close() {
	if d.cleanup != nil {
		d.cleanup()
	}
}
