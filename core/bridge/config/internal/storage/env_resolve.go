package storage

import "strings"

func ResolveSessionsPath(fileCfg FileConfig, env EnvSnapshot, fallback string) string {
	return ValueOrEnvWithEnv(fileCfg.SessionsPath, env, "GHOST_SESSIONS_PATH", fallback)
}

func ResolveWebSearchTavilyAPIKey(fileCfg FileConfig, env EnvSnapshot) string {
	if fileCfg.WebSearchTavilyAPIKey != nil {
		return strings.TrimSpace(*fileCfg.WebSearchTavilyAPIKey)
	}
	return env.FirstNonEmpty("GHOST_WEB_SEARCH_TAVILY_API_KEY", "TAVILY_API_KEY")
}

func ResolveTasksPath(fileCfg FileConfig, env EnvSnapshot, fallback string) string {
	return ValueOrEnvWithEnv(fileCfg.TasksPath, env, "GHOST_TASKS_PATH", fallback)
}

func ResolveScriptExecSandboxMemoryMB(fileCfg FileConfig, env EnvSnapshot, fallback int, max int) (int, error) {
	return BoundedPositiveIntOrEnvWithEnv(
		fileCfg.ScriptExecSandboxMemoryMB,
		"script_exec_sandbox_memory_mb",
		env,
		"GHOST_SCRIPT_EXEC_SANDBOX_MEMORY_MB",
		fallback,
		max,
	)
}

func ResolveNativeBinaryPath(fileCfg FileConfig, env EnvSnapshot) string {
	if override := env.FirstNonEmpty("GHOST_NATIVE_BINARY_PATH_OVERRIDE", "GHOST_NATIVE_BIN_OVERRIDE"); override != "" {
		return override
	}
	if fileCfg.NativeBinaryPath != nil {
		return strings.TrimSpace(*fileCfg.NativeBinaryPath)
	}
	return env.FirstNonEmpty("GHOST_NATIVE_BINARY_PATH", "GHOST_NATIVE_BIN")
}

func ResolveNativeBinaryRoots(fileCfg FileConfig, env EnvSnapshot) []string {
	if fileCfg.NativeBinaryRoots != nil {
		return NormalizeConfiguredPathList(fileCfg.NativeBinaryRoots)
	}
	return NormalizeConfiguredPathList(ParseStringCSV(env.DefaultValue("GHOST_NATIVE_BINARY_ROOTS", "")))
}

func ResolveNativeBinaryCandidates(fileCfg FileConfig, env EnvSnapshot) []string {
	if fileCfg.NativeBinaryCandidates != nil {
		return NormalizeConfiguredPathList(fileCfg.NativeBinaryCandidates)
	}
	return NormalizeConfiguredPathList(ParseStringCSV(env.DefaultValue("GHOST_NATIVE_BINARY_CANDIDATES", "")))
}

func ResolveCodexCLIPath(fileCfg FileConfig, env EnvSnapshot) string {
	if override := env.Value("GHOST_CODEX_CLI_PATH"); override != "" {
		return strings.TrimSpace(override)
	}
	if fileCfg.CodexCLIPath != nil {
		return strings.TrimSpace(*fileCfg.CodexCLIPath)
	}
	return ""
}

func ResolveNodeBinPath(fileCfg FileConfig, env EnvSnapshot) string {
	if override := env.Value("GHOST_NODE_BIN_PATH"); override != "" {
		return strings.TrimSpace(override)
	}
	if fileCfg.NodeBinPath != nil {
		return strings.TrimSpace(*fileCfg.NodeBinPath)
	}
	return ""
}

func ResolveNativeAllowedReadPaths(fileCfg FileConfig, env EnvSnapshot) []string {
	if fileCfg.NativeAllowedReadPaths != nil {
		return NormalizeConfiguredPathList(fileCfg.NativeAllowedReadPaths)
	}
	return NormalizeConfiguredPathList(ParseStringCSV(env.DefaultValue("GHOST_NATIVE_ALLOWED_READ_PATHS", "")))
}

func ResolveNativeAllowedWritePaths(fileCfg FileConfig, env EnvSnapshot) []string {
	if fileCfg.NativeAllowedWritePaths != nil {
		return NormalizeConfiguredPathList(fileCfg.NativeAllowedWritePaths)
	}
	return NormalizeConfiguredPathList(ParseStringCSV(env.DefaultValue("GHOST_NATIVE_ALLOWED_WRITE_PATHS", "")))
}

func ResolveProjectRoot(fileCfg FileConfig, env EnvSnapshot) string {
	if fileCfg.ProjectRoot != nil {
		return strings.TrimSpace(*fileCfg.ProjectRoot)
	}
	return env.Value("GHOST_PROJECT_ROOT")
}
