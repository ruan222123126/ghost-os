package app

import (
	"context"
	"errors"
	"runtime"
	"strconv"
	"strings"

	"ghost-os/bridge/agent"
	ctxmgr "ghost-os/bridge/context"
	"ghost-os/bridge/execution"
	"ghost-os/bridge/llm"
	"ghost-os/bridge/session"
	"ghost-os/bridge/tools"
)

type agentRuntimeDependencies struct {
	cfg          Config
	client       *llm.Client
	registry     *tools.Registry
	systemPrompt string
}

// runAgent 组装最小可运行链路：配置 -> LLM 客户端 -> 工具目录 -> Agent。
func runAgent(ctx context.Context, userMessage string) (string, error) {
	return runAgentWithConfigStore(ctx, userMessage, nil, "")
}

func runAgentWithConfigStore(ctx context.Context, userMessage string, store *ConfigStore, traceID string) (string, error) {
	deps, err := buildAgentRuntimeDependencies(store)
	if err != nil {
		return "", err
	}

	a := agent.NewAgent(deps.client, deps.registry, deps.systemPrompt, deps.cfg.MaxTurns)
	return a.RunWithTraceID(ctx, userMessage, traceID)
}

func runAgentWithSession(
	ctx context.Context,
	userMessage string,
	sessionID string,
	traceID string,
	store *ConfigStore,
	sessionStore *session.Store,
) (string, string, error) {
	deps, err := buildAgentRuntimeDependencies(store)
	if err != nil {
		return "", "", err
	}

	trimmedSessionID := strings.TrimSpace(sessionID)
	var sess *session.Session
	if sessionStore != nil && trimmedSessionID != "" {
		sess, err = sessionStore.Load(trimmedSessionID)
		if err != nil && !errors.Is(err, session.ErrSessionNotFound) {
			return "", "", err
		}
	}

	if sess == nil {
		sess = session.NewSession(deps.systemPrompt)
	}

	contextLimit := session.GetContextLimit(deps.cfg.Provider, deps.cfg.Model)
	history := agent.NewHistoryFromMessages(sess.GetMessages(contextLimit))
	a := agent.NewAgentWithHistory(deps.client, deps.registry, history, deps.cfg.MaxTurns)

	response, err := a.RunWithTraceID(ctx, userMessage, traceID)
	if err != nil {
		return "", "", err
	}

	if sessionStore != nil {
		for _, msg := range a.GetNewMessages() {
			sess.AddMessage(msg)
		}
		if err := sessionStore.Save(sess); err != nil {
			return "", "", err
		}
		return response, sess.ID, nil
	}

	return response, "", nil
}

func buildAgentRuntimeDependencies(store *ConfigStore) (agentRuntimeDependencies, error) {
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

	client := llm.NewClientWithOptions(llm.ClientOptions{
		Provider:           cfg.Provider,
		BaseURL:            cfg.BaseURL,
		APIKey:             cfg.APIKey,
		Model:              cfg.Model,
		ChatPath:           cfg.ChatPath,
		Headers:            cfg.ProviderHeaders,
		AnthropicVersion:   cfg.AnthropicVersion,
		AnthropicMaxTokens: cfg.AnthropicMaxTokens,
	})

	executionClient := execution.NewNativeClient()
	registry := tools.NewRegistry()
	registry.Register(tools.NewScriptExecTool(executionClient))

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
	}, nil
}
