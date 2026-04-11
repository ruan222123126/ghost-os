package orchestration

import (
	"strings"

	"ghost-os/bridge/agent"
	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/llm"
	bridgeruntime "ghost-os/bridge/runtime"
)

func applyTaskRuntimeOverridesToDependencies(
	deps agentRuntimeDependencies,
	runtimeOverrides *TaskRuntimeOverrides,
) (agentRuntimeDependencies, error) {
	normalizedOverrides, err := normalizeTaskRuntimeOverrides(runtimeOverrides)
	if err != nil {
		return agentRuntimeDependencies{}, err
	}
	if normalizedOverrides == nil {
		return deps, nil
	}
	cfg := deps.cfg
	overrideModel := strings.TrimSpace(normalizedOverrides.Model)
	cfg.Provider.Model = resolveTaskOverrideModel(cfg.Provider.Model, overrideModel)
	if len(normalizedOverrides.ToolAllowlist) > 0 {
		cfg.ToolSelector.Allowlist = append([]string(nil), normalizedOverrides.ToolAllowlist...)
		cfg.ToolSelector.AllowlistOnly = true
	}
	systemPrompt, err := bridgeruntime.BuildSystemPromptForCatalog(cfg, bridgeruntime.NewToolSelectionPolicy(cfg).ResidentCatalog(deps.registry))
	if err != nil {
		return agentRuntimeDependencies{}, err
	}
	deps.cfg = cfg
	if overrideModel != "" {
		deps.client = newAgentCompleterForConfig(cfg)
	}
	deps.systemPrompt = systemPrompt
	return deps, nil
}

func resolveTaskOverrideModel(defaultModel string, overrideModel string) string {
	if model := strings.TrimSpace(overrideModel); model != "" {
		return model
	}
	return strings.TrimSpace(defaultModel)
}

func newAgentCompleterForConfig(cfg bridgeconfig.Config) agent.Completer {
	return llm.NewClientWithOptions(llm.ClientOptions{
		Provider:                   cfg.Provider.Type,
		BaseURL:                    cfg.Provider.BaseURL,
		APIKey:                     cfg.Provider.APIKey,
		Model:                      strings.TrimSpace(cfg.Provider.Model),
		ChatPath:                   cfg.ChatPath,
		Headers:                    cfg.Provider.Headers,
		AnthropicVersion:           cfg.Provider.AnthropicVersion,
		AnthropicMaxTokens:         cfg.Provider.AnthropicMaxTokens,
		CodexStatelessRetryEnabled: cfg.CodexStatelessRetryEnabled,
		ResponseOptions:            llm.CloneResponseOptions(cfg.ResponseOptions),
	})
}
