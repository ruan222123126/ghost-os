package config

import "strings"

func normalizeBridgeFileConfigForWrite(cfg bridgeFileConfig) (bridgeFileConfig, error) {
	out := cfg
	normalizeBridgeScalarFields(&out)
	if err := normalizeBridgeCollectionFields(&out); err != nil {
		return bridgeFileConfig{}, err
	}
	providers := normalizeProviderConfigs(out.Providers, stringValue(out.Model))
	out.Providers = providerConfigsToFileMap(providers)
	out.ActiveProvider = normalizedActiveProviderName(providers, out.ActiveProvider)
	return out, nil
}

func normalizedActiveProviderName(providers []providerConfig, preferred ...*string) *string {
	if len(providers) == 0 {
		return nil
	}
	for _, candidate := range preferred {
		name := strings.TrimSpace(stringValue(candidate))
		if providerIndexByName(providers, name) >= 0 {
			return stringPointer(name)
		}
	}
	return stringPointer(providers[0].Name)
}

func normalizeBridgeScalarFields(cfg *bridgeFileConfig) {
	cfg.ActiveProvider = cloneOptionalStringPointer(cfg.ActiveProvider)
	cfg.Model = cloneOptionalStringPointer(cfg.Model)
	cfg.ModelSelectionEnabled = cloneBoolPointer(cfg.ModelSelectionEnabled)
	cfg.ChatPath = cloneOptionalStringPointer(cfg.ChatPath)
	cfg.ResponsePromptCacheKey = cloneOptionalStringPointer(cfg.ResponsePromptCacheKey)
	cfg.ResponsePromptCacheRetention = cloneOptionalStringPointer(cfg.ResponsePromptCacheRetention)
	cfg.ResponseSafetyIdentifier = cloneOptionalStringPointer(cfg.ResponseSafetyIdentifier)
	cfg.ResponseStore = cloneBoolPointer(cfg.ResponseStore)
	cfg.CodexStatelessRetryEnabled = cloneBoolPointer(cfg.CodexStatelessRetryEnabled)
	cfg.ProjectRoot = cloneOptionalStringPointer(cfg.ProjectRoot)
	cfg.MaxTurns = cloneIntPointer(cfg.MaxTurns)
	cfg.ScriptExecSandboxMemoryMB = cloneIntPointer(cfg.ScriptExecSandboxMemoryMB)
	cfg.WorkerModel = cloneOptionalStringPointer(cfg.WorkerModel)
	cfg.PromptsPath = cloneOptionalStringPointer(cfg.PromptsPath)
	cfg.PromptsDir = cloneOptionalStringPointer(cfg.PromptsDir)
	cfg.TasksPath = cloneOptionalStringPointer(cfg.TasksPath)
	cfg.TaskExecutionTimeoutMS = cloneIntPointer(cfg.TaskExecutionTimeoutMS)
	cfg.SessionsPath = cloneOptionalStringPointer(cfg.SessionsPath)
	cfg.SessionHumanLogFullEnabled = cloneBoolPointer(cfg.SessionHumanLogFullEnabled)
	cfg.SessionSystemPromptVisible = cloneBoolPointer(cfg.SessionSystemPromptVisible)
	cfg.AssistantMarkdownEnabled = cloneBoolPointer(cfg.AssistantMarkdownEnabled)
	cfg.ToolCallCompactOutputEnabled = cloneBoolPointer(cfg.ToolCallCompactOutputEnabled)
	cfg.MemoryModeEnabled = cloneBoolPointer(cfg.MemoryModeEnabled)
	cfg.MicrocompactEnabled = cloneBoolPointer(cfg.MicrocompactEnabled)
	cfg.SessionTitleMode = cloneOptionalStringPointer(cfg.SessionTitleMode)
	cfg.RSSFeedsPath = cloneOptionalStringPointer(cfg.RSSFeedsPath)
	cfg.RSSInboxPath = cloneOptionalStringPointer(cfg.RSSInboxPath)
	cfg.RSSBriefingsPath = cloneOptionalStringPointer(cfg.RSSBriefingsPath)
	cfg.RSSReportsPath = cloneOptionalStringPointer(cfg.RSSReportsPath)
	cfg.RSSPollInterval = cloneOptionalStringPointer(cfg.RSSPollInterval)
	cfg.RSSBriefingInterval = cloneOptionalStringPointer(cfg.RSSBriefingInterval)
	cfg.WebSearchTavilyURL = cloneOptionalStringPointer(cfg.WebSearchTavilyURL)
	cfg.WebSearchExaURL = cloneOptionalStringPointer(cfg.WebSearchExaURL)
	cfg.WebSearchTavilyAPIKey = cloneOptionalStringPointer(cfg.WebSearchTavilyAPIKey)
	cfg.WebSearchExaAPIKey = cloneOptionalStringPointer(cfg.WebSearchExaAPIKey)
	cfg.LLMCompletionRetryCount = cloneIntPointer(cfg.LLMCompletionRetryCount)
	cfg.LLMCompletionRetryIntervalMS = cloneIntPointer(cfg.LLMCompletionRetryIntervalMS)
	cfg.AnthropicVersion = cloneOptionalStringPointer(cfg.AnthropicVersion)
	cfg.ToolSelectorMode = cloneOptionalStringPointer(cfg.ToolSelectorMode)
	cfg.ToolSelectorModel = cloneOptionalStringPointer(cfg.ToolSelectorModel)
	cfg.BindAddr = cloneOptionalStringPointer(cfg.BindAddr)
	cfg.APIToken = cloneOptionalStringPointer(cfg.APIToken)
	cfg.NativeBinaryPath = cloneOptionalStringPointer(cfg.NativeBinaryPath)
	cfg.CodexCLIPath = cloneOptionalStringPointer(cfg.CodexCLIPath)
	cfg.NodeBinPath = cloneOptionalStringPointer(cfg.NodeBinPath)
}

func normalizeBridgeCollectionFields(cfg *bridgeFileConfig) error {
	cfg.PromptsCoreFiles = normalizeConfiguredPathList(cfg.PromptsCoreFiles)
	cfg.PromptsRuntimeConstraintFiles = normalizeConfiguredPathList(cfg.PromptsRuntimeConstraintFiles)
	cfg.PromptsResponseRuleFiles = normalizeConfiguredPathList(cfg.PromptsResponseRuleFiles)
	cfg.NativeBinaryRoots = normalizeConfiguredPathList(cfg.NativeBinaryRoots)
	cfg.NativeBinaryCandidates = normalizeConfiguredPathList(cfg.NativeBinaryCandidates)
	cfg.NativeAllowedReadPaths = normalizeConfiguredPathList(cfg.NativeAllowedReadPaths)
	cfg.NativeAllowedWritePaths = normalizeConfiguredPathList(cfg.NativeAllowedWritePaths)
	providerHeaders, err := normalizeProviderHeaders(cfg.ProviderHeaders)
	if err != nil {
		return err
	}
	cfg.ProviderHeaders = providerHeaders
	responseMetadata, err := normalizeResponseMetadata(cfg.ResponseMetadata)
	if err != nil {
		return err
	}
	cfg.ResponseMetadata = responseMetadata
	cfg.CORSOrigins = normalizeOrigins(cfg.CORSOrigins)
	cfg.ToolAllowlist = normalizeConfiguredToolNames(cfg.ToolAllowlist)
	cfg.ToolBlocklist = normalizeConfiguredToolNames(cfg.ToolBlocklist)
	cfg.SkillBlocklist = normalizeStringList(cfg.SkillBlocklist)
	cfg.WorkflowToolAllowlist = normalizeConfiguredToolNames(cfg.WorkflowToolAllowlist)
	promptOverrides, err := normalizeToolPromptOverrides(cfg.ToolPromptOverrides)
	if err != nil {
		return err
	}
	cfg.ToolPromptOverrides = promptOverrides
	return nil
}

func normalizeStringList(raw []string) []string {
	seen := make(map[string]bool, len(raw))
	items := make([]string, 0, len(raw))
	for _, item := range raw {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" || seen[trimmed] {
			continue
		}
		seen[trimmed] = true
		items = append(items, trimmed)
	}
	return items
}
