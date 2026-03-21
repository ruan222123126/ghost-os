package config

import "strings"

func resolveSessionsPath(fileCfg bridgeFileConfig, env Env) string {
	return valueOrEnvWithEnv(fileCfg.SessionsPath, env, "GHOST_SESSIONS_PATH", defaultSessionsPath)
}

func resolveRSSFeedsPath(fileCfg bridgeFileConfig, env Env) string {
	return valueOrEnvWithEnv(fileCfg.RSSFeedsPath, env, "GHOST_RSS_FEEDS_PATH", defaultRSSFeedsPath)
}

func resolveRSSInboxPath(fileCfg bridgeFileConfig, env Env) string {
	return valueOrEnvWithEnv(fileCfg.RSSInboxPath, env, "GHOST_RSS_INBOX_PATH", defaultRSSInboxPath)
}

func resolveRSSBriefingsPath(fileCfg bridgeFileConfig, env Env) string {
	return valueOrEnvWithEnv(fileCfg.RSSBriefingsPath, env, "GHOST_RSS_BRIEFINGS_PATH", defaultRSSBriefingsPath)
}

func resolveRSSReportsPath(fileCfg bridgeFileConfig, env Env) string {
	return valueOrEnvWithEnv(fileCfg.RSSReportsPath, env, "GHOST_RSS_REPORTS_PATH", defaultRSSReportsPath)
}

func resolveWebSearchTavilyAPIKey(fileCfg bridgeFileConfig, env Env) string {
	if fileCfg.WebSearchTavilyAPIKey != nil {
		return strings.TrimSpace(*fileCfg.WebSearchTavilyAPIKey)
	}
	return env.firstNonEmpty("GHOST_WEB_SEARCH_TAVILY_API_KEY", "TAVILY_API_KEY")
}

func resolveTasksPath(env Env) string {
	return env.defaultValue("GHOST_TASKS_PATH", defaultTasksPath)
}

func resolveNativeBinaryPath(fileCfg bridgeFileConfig, env Env) string {
	if fileCfg.NativeBinaryPath != nil {
		return strings.TrimSpace(*fileCfg.NativeBinaryPath)
	}
	return env.firstNonEmpty("GHOST_NATIVE_BINARY_PATH", "GHOST_NATIVE_BIN")
}

func resolveNativeBinaryRoots(fileCfg bridgeFileConfig, env Env) []string {
	if fileCfg.NativeBinaryRoots != nil {
		return normalizeConfiguredPathList(fileCfg.NativeBinaryRoots)
	}
	return normalizeConfiguredPathList(parseStringCSV(env.defaultValue("GHOST_NATIVE_BINARY_ROOTS", "")))
}

func resolveNativeBinaryCandidates(fileCfg bridgeFileConfig, env Env) []string {
	if fileCfg.NativeBinaryCandidates != nil {
		return normalizeConfiguredPathList(fileCfg.NativeBinaryCandidates)
	}
	return normalizeConfiguredPathList(parseStringCSV(env.defaultValue("GHOST_NATIVE_BINARY_CANDIDATES", "")))
}

func resolveNativeAllowedReadPaths(fileCfg bridgeFileConfig, env Env) []string {
	if fileCfg.NativeAllowedReadPaths != nil {
		return normalizeConfiguredPathList(fileCfg.NativeAllowedReadPaths)
	}
	return normalizeConfiguredPathList(parseStringCSV(env.defaultValue("GHOST_NATIVE_ALLOWED_READ_PATHS", "")))
}

func resolveNativeAllowedWritePaths(fileCfg bridgeFileConfig, env Env) []string {
	if fileCfg.NativeAllowedWritePaths != nil {
		return normalizeConfiguredPathList(fileCfg.NativeAllowedWritePaths)
	}
	return normalizeConfiguredPathList(parseStringCSV(env.defaultValue("GHOST_NATIVE_ALLOWED_WRITE_PATHS", "")))
}

func resolveProjectRoot(fileCfg bridgeFileConfig, env Env) string {
	if fileCfg.ProjectRoot != nil {
		return strings.TrimSpace(*fileCfg.ProjectRoot)
	}
	return env.value("GHOST_PROJECT_ROOT")
}
