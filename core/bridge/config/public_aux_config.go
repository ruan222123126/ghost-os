package config

func LoadServerConfig() (ServerConfig, error) {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return ServerConfig{}, err
	}
	return ServerConfig{
		BindAddr:     valueOrEnv(fileCfg.BindAddr, "GHOST_BIND_ADDR", ""),
		APIToken:     valueOrEnv(fileCfg.APIToken, "GHOST_API_TOKEN", ""),
		CORSOrigins:  corsOriginsOrEnv(fileCfg.CORSOrigins),
		SessionsPath: resolveSessionsPath(fileCfg, CurrentEnv()),
	}, nil
}

func LoadExecutionConfig() ExecutionConfig {
	return ExecutionConfig{
		Persistent:             nativePersistentEnabledFromEnv(),
		NativeBinaryPath:       nativeBinaryPathFromEnv(),
		NativeBinaryRoots:      nativeBinaryRootsFromEnv(),
		NativeBinaryCandidates: nativeBinaryCandidatesFromEnv(),
		AllowedReadPaths:       nativeAllowedReadPathsFromEnv(),
		AllowedWritePaths:      nativeAllowedWritePathsFromEnv(),
		ProjectRoot:            projectRootFromEnv(),
	}
}

func LoadTaskConfig() TaskConfig {
	return TaskConfig{TasksPath: tasksPathFromEnv()}
}
