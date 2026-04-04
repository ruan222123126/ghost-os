package config

func buildProviderConfig(runtime runtimeConfig, fileCfg bridgeFileConfig, env envSnapshot, headers map[string]string) (ProviderConfig, error) {
	anthropicMaxTokens, err := intOrEnvWithEnv(
		fileCfg.AnthropicMaxTokens,
		"anthropic_max_tokens",
		env,
		"GHOST_ANTHROPIC_MAX_TOKENS",
		defaultAnthropicMaxTokens,
	)
	if err != nil {
		return ProviderConfig{}, err
	}
	return ProviderConfig{
		Type:                       runtime.Provider,
		APIKey:                     runtime.APIKey,
		BaseURL:                    runtime.BaseURL,
		Model:                      runtime.Model,
		Headers:                    headers,
		AnthropicVersion:           valueOrEnvWithEnv(fileCfg.AnthropicVersion, env, "GHOST_ANTHROPIC_VERSION", defaultAnthropicVersion),
		AnthropicMaxTokens:         anthropicMaxTokens,
		ContextWindowTokens:        runtime.ContextWindowTokens,
		ResponseReserveTokens:      runtime.ResponseReserveTokens,
		ModelContextWindowTokens:   cloneModelTokenOverrides(runtime.ModelContextWindowTokens),
		ModelResponseReserveTokens: cloneModelTokenOverrides(runtime.ModelResponseReserveTokens),
	}, nil
}

func buildWorkerConfig(fileCfg bridgeFileConfig, env envSnapshot) (WorkerConfig, error) {
	maxConcurrency, err := intOrEnvWithEnv(
		fileCfg.WorkerMaxConcurrency,
		"worker_max_concurrency",
		env,
		"GHOST_WORKER_MAX_CONCURRENCY",
		defaultWorkerMaxConcurrency,
	)
	if err != nil {
		return WorkerConfig{}, err
	}
	maxFiles, err := intOrEnvWithEnv(fileCfg.WorkerMaxFiles, "worker_max_files", env, "GHOST_WORKER_MAX_FILES", defaultWorkerMaxFiles)
	if err != nil {
		return WorkerConfig{}, err
	}
	maxFileChunks, err := intOrEnvWithEnv(
		fileCfg.WorkerMaxFileChunks,
		"worker_max_file_chunks",
		env,
		"GHOST_WORKER_MAX_FILE_CHUNKS",
		defaultWorkerMaxFileChunks,
	)
	if err != nil {
		return WorkerConfig{}, err
	}
	return WorkerConfig{
		Model:          valueOrEnvWithEnv(fileCfg.WorkerModel, env, "GHOST_WORKER_MODEL", ""),
		MaxConcurrency: maxConcurrency,
		MaxFiles:       maxFiles,
		MaxFileChunks:  maxFileChunks,
	}, nil
}
