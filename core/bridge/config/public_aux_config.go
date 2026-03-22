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
		SessionsPath: resolveSessionsPath(fileCfg, currentEnv()),
	}, nil
}

func LoadExecutionConfig() (ExecutionConfig, error) {
	aux, err := loadAuxConfigFromEnv()
	if err != nil {
		return ExecutionConfig{}, err
	}
	return aux.Execution, nil
}

func LoadTaskConfig() TaskConfig {
	return TaskConfig{TasksPath: tasksPathFromEnv()}
}
