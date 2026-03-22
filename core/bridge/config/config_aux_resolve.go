package config

// auxConfig bundles the legacy helper projections that still expose
// individual runtime fields outside the main Config load path.
type auxConfig struct {
	SessionsPath          string
	RSS                   RSSConfig
	WebSearchTavilyAPIKey string
	Execution             ExecutionConfig
}

func loadAuxConfigFromEnv() (auxConfig, error) {
	env := currentEnv()
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return resolveAuxConfig(bridgeFileConfig{}, env)
	}
	return resolveAuxConfig(fileCfg, env)
}

func resolveAuxConfig(fileCfg bridgeFileConfig, env envSnapshot) (auxConfig, error) {
	rss, err := buildRSSConfig(fileCfg, env)
	if err != nil {
		return auxConfig{}, err
	}
	execution, err := resolveExecutionConfig(fileCfg, env)
	if err != nil {
		return auxConfig{}, err
	}
	return auxConfig{
		SessionsPath:          resolveSessionsPath(fileCfg, env),
		RSS:                   rss,
		WebSearchTavilyAPIKey: resolveWebSearchTavilyAPIKey(fileCfg, env),
		Execution:             execution,
	}, nil
}

func resolveExecutionConfig(fileCfg bridgeFileConfig, env envSnapshot) (ExecutionConfig, error) {
	persistent, err := resolveNativePersistent(fileCfg.NativePersistent, env)
	if err != nil {
		return ExecutionConfig{}, err
	}
	return ExecutionConfig{
		Persistent:             persistent,
		NativeBinaryPath:       resolveNativeBinaryPath(fileCfg, env),
		NativeBinaryRoots:      resolveNativeBinaryRoots(fileCfg, env),
		NativeBinaryCandidates: resolveNativeBinaryCandidates(fileCfg, env),
		AllowedReadPaths:       resolveNativeAllowedReadPaths(fileCfg, env),
		AllowedWritePaths:      resolveNativeAllowedWritePaths(fileCfg, env),
		ProjectRoot:            resolveProjectRoot(fileCfg, env),
	}, nil
}
