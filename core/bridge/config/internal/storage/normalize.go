package storage

func NormalizeForWrite(cfg FileConfig) (FileConfig, error) {
	out := cfg
	normalizeScalarFields(&out)
	if err := normalizeCollectionFields(&out); err != nil {
		return FileConfig{}, err
	}
	providers := NormalizeProviderConfigs(out.Providers, StringValue(out.Model))
	out.Providers = ProviderConfigsToFileMap(providers)
	out.ActiveProvider = NormalizedActiveProviderName(providers, out.ActiveProvider)
	return out, nil
}

func normalizeScalarFields(cfg *FileConfig) {
	cfg.ActiveProvider = CloneOptionalStringPointer(cfg.ActiveProvider)
	cfg.Model = CloneOptionalStringPointer(cfg.Model)
	cfg.ModelSelectionEnabled = CloneBoolPointer(cfg.ModelSelectionEnabled)
	cfg.ChatPath = CloneOptionalStringPointer(cfg.ChatPath)
	cfg.ResponsePromptCacheKey = CloneOptionalStringPointer(cfg.ResponsePromptCacheKey)
	cfg.ResponsePromptCacheRetention = CloneOptionalStringPointer(cfg.ResponsePromptCacheRetention)
	cfg.ResponseSafetyIdentifier = CloneOptionalStringPointer(cfg.ResponseSafetyIdentifier)
	cfg.ResponseStore = CloneBoolPointer(cfg.ResponseStore)
	cfg.CodexStatelessRetryEnabled = CloneBoolPointer(cfg.CodexStatelessRetryEnabled)
	cfg.ProjectRoot = CloneOptionalStringPointer(cfg.ProjectRoot)
	cfg.MaxTurns = CloneIntPointer(cfg.MaxTurns)
	cfg.ScriptExecSandboxMemoryMB = CloneIntPointer(cfg.ScriptExecSandboxMemoryMB)
	cfg.WorkerModel = CloneOptionalStringPointer(cfg.WorkerModel)
	cfg.PromptsPath = CloneOptionalStringPointer(cfg.PromptsPath)
	cfg.PromptsDir = CloneOptionalStringPointer(cfg.PromptsDir)
	cfg.TasksPath = CloneOptionalStringPointer(cfg.TasksPath)
	cfg.TaskExecutionTimeoutMS = CloneIntPointer(cfg.TaskExecutionTimeoutMS)
	cfg.SessionsPath = CloneOptionalStringPointer(cfg.SessionsPath)
	cfg.SessionHumanLogFullEnabled = CloneBoolPointer(cfg.SessionHumanLogFullEnabled)
	cfg.SessionSystemPromptVisible = CloneBoolPointer(cfg.SessionSystemPromptVisible)
	cfg.AssistantMarkdownEnabled = CloneBoolPointer(cfg.AssistantMarkdownEnabled)
	cfg.ToolCallCompactOutputEnabled = CloneBoolPointer(cfg.ToolCallCompactOutputEnabled)
	cfg.MemoryModeEnabled = CloneBoolPointer(cfg.MemoryModeEnabled)
	cfg.MicrocompactEnabled = CloneBoolPointer(cfg.MicrocompactEnabled)
	cfg.SessionTitleMode = CloneOptionalStringPointer(cfg.SessionTitleMode)
	cfg.WebSearchTavilyURL = CloneOptionalStringPointer(cfg.WebSearchTavilyURL)
	cfg.WebSearchExaURL = CloneOptionalStringPointer(cfg.WebSearchExaURL)
	cfg.WebSearchTavilyAPIKey = CloneOptionalStringPointer(cfg.WebSearchTavilyAPIKey)
	cfg.WebSearchExaAPIKey = CloneOptionalStringPointer(cfg.WebSearchExaAPIKey)
	cfg.LLMCompletionRetryCount = CloneIntPointer(cfg.LLMCompletionRetryCount)
	cfg.LLMCompletionRetryIntervalMS = CloneIntPointer(cfg.LLMCompletionRetryIntervalMS)
	cfg.AnthropicVersion = CloneOptionalStringPointer(cfg.AnthropicVersion)
	cfg.ToolSelectorMode = CloneOptionalStringPointer(cfg.ToolSelectorMode)
	cfg.ToolSelectorModel = CloneOptionalStringPointer(cfg.ToolSelectorModel)
	cfg.BindAddr = CloneOptionalStringPointer(cfg.BindAddr)
	cfg.APIToken = CloneOptionalStringPointer(cfg.APIToken)
	cfg.NativeBinaryPath = CloneOptionalStringPointer(cfg.NativeBinaryPath)
	cfg.CodexCLIPath = CloneOptionalStringPointer(cfg.CodexCLIPath)
	cfg.NodeBinPath = CloneOptionalStringPointer(cfg.NodeBinPath)
	cfg.MobileWebRTC.Enabled = CloneBoolPointer(cfg.MobileWebRTC.Enabled)
	cfg.MobileWebRTC.SignalingURL = CloneOptionalStringPointer(cfg.MobileWebRTC.SignalingURL)
	cfg.MobileWebRTC.SignalingToken = CloneOptionalStringPointer(cfg.MobileWebRTC.SignalingToken)
	cfg.MobileWebRTC.PCID = CloneOptionalStringPointer(cfg.MobileWebRTC.PCID)
	cfg.MobileWebRTC.CredentialStorePath = CloneOptionalStringPointer(cfg.MobileWebRTC.CredentialStorePath)
}

func normalizeCollectionFields(cfg *FileConfig) error {
	cfg.PromptsCoreFiles = NormalizeConfiguredPathList(cfg.PromptsCoreFiles)
	cfg.PromptsRuntimeConstraintFiles = NormalizeConfiguredPathList(cfg.PromptsRuntimeConstraintFiles)
	cfg.PromptsResponseRuleFiles = NormalizeConfiguredPathList(cfg.PromptsResponseRuleFiles)
	cfg.NativeBinaryRoots = NormalizeConfiguredPathList(cfg.NativeBinaryRoots)
	cfg.NativeBinaryCandidates = NormalizeConfiguredPathList(cfg.NativeBinaryCandidates)
	cfg.NativeAllowedReadPaths = NormalizeConfiguredPathList(cfg.NativeAllowedReadPaths)
	cfg.NativeAllowedWritePaths = NormalizeConfiguredPathList(cfg.NativeAllowedWritePaths)
	providerHeaders, err := NormalizeProviderHeaders(cfg.ProviderHeaders)
	if err != nil {
		return err
	}
	cfg.ProviderHeaders = providerHeaders
	responseMetadata, err := NormalizeResponseMetadata(cfg.ResponseMetadata)
	if err != nil {
		return err
	}
	cfg.ResponseMetadata = responseMetadata
	cfg.CORSOrigins = NormalizeOrigins(cfg.CORSOrigins)
	cfg.ToolAllowlist = NormalizeConfiguredNames(cfg.ToolAllowlist)
	cfg.ToolBlocklist = NormalizeConfiguredNames(cfg.ToolBlocklist)
	cfg.SkillBlocklist = NormalizeStringList(cfg.SkillBlocklist)
	cfg.WorkflowToolAllowlist = NormalizeConfiguredNames(cfg.WorkflowToolAllowlist)
	cfg.MobileWebRTC.ICEServers = NormalizeMobileICEServers(cfg.MobileWebRTC.ICEServers)
	return nil
}

func NormalizeMobileICEServers(raw []MobileICEFileConfig) []MobileICEFileConfig {
	if len(raw) == 0 {
		return nil
	}
	out := make([]MobileICEFileConfig, 0, len(raw))
	for _, server := range raw {
		urls := NormalizeStringList(server.URLs)
		if len(urls) == 0 {
			continue
		}
		out = append(out, MobileICEFileConfig{
			URLs:       urls,
			Username:   CloneOptionalStringPointer(server.Username),
			Credential: CloneOptionalStringPointer(server.Credential),
		})
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
