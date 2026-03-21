package config

import "time"

func getenvDefault(name, fallback string) string {
	return CurrentEnv().defaultValue(name, fallback)
}

// sessionsPathFromEnv 返回会话持久化目录。
func sessionsPathFromEnv() string {
	env := CurrentEnv()
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return env.defaultValue("GHOST_SESSIONS_PATH", defaultSessionsPath)
	}
	return resolveSessionsPath(fileCfg, env)
}

func rssFeedsPathFromEnv() string {
	env := CurrentEnv()
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return env.defaultValue("GHOST_RSS_FEEDS_PATH", defaultRSSFeedsPath)
	}
	return resolveRSSFeedsPath(fileCfg, env)
}

func rssInboxPathFromEnv() string {
	env := CurrentEnv()
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return env.defaultValue("GHOST_RSS_INBOX_PATH", defaultRSSInboxPath)
	}
	return resolveRSSInboxPath(fileCfg, env)
}

func rssBriefingsPathFromEnv() string {
	env := CurrentEnv()
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return env.defaultValue("GHOST_RSS_BRIEFINGS_PATH", defaultRSSBriefingsPath)
	}
	return resolveRSSBriefingsPath(fileCfg, env)
}

func rssReportsPathFromEnv() string {
	env := CurrentEnv()
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return env.defaultValue("GHOST_RSS_REPORTS_PATH", defaultRSSReportsPath)
	}
	return resolveRSSReportsPath(fileCfg, env)
}

func rssPollEnabledFromEnv() bool {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return parseBoolEnv("GHOST_RSS_POLL_ENABLED", true)
	}
	return boolOrEnv(fileCfg.RSSPollEnabled, "GHOST_RSS_POLL_ENABLED", true)
}

func rssPollIntervalFromEnv() time.Duration {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return parseDurationEnv("GHOST_RSS_POLL_INTERVAL", defaultRSSPollInterval)
	}
	return durationOrEnv(fileCfg.RSSPollInterval, "GHOST_RSS_POLL_INTERVAL", defaultRSSPollInterval)
}

func rssPollMaxItemsPerFeedFromEnv() int {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return parsePositiveIntEnv("GHOST_RSS_POLL_MAX_ITEMS_PER_FEED", defaultRSSPollMaxItemsPerFeed)
	}
	return intOrEnv(fileCfg.RSSPollMaxItemsPerFeed, "GHOST_RSS_POLL_MAX_ITEMS_PER_FEED", defaultRSSPollMaxItemsPerFeed)
}

func rssAIBatchSizeFromEnv() int {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return parsePositiveIntEnv("GHOST_RSS_AI_BATCH_SIZE", defaultRSSAIBatchSize)
	}
	return intOrEnv(fileCfg.RSSAIBatchSize, "GHOST_RSS_AI_BATCH_SIZE", defaultRSSAIBatchSize)
}

func rssBriefingEnabledFromEnv() bool {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return parseBoolEnv("GHOST_RSS_BRIEFING_ENABLED", true)
	}
	return boolOrEnv(fileCfg.RSSBriefingEnabled, "GHOST_RSS_BRIEFING_ENABLED", true)
}

func rssBriefingIntervalFromEnv() time.Duration {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return parseDurationEnv("GHOST_RSS_BRIEFING_INTERVAL", defaultRSSBriefingInterval)
	}
	return durationOrEnv(fileCfg.RSSBriefingInterval, "GHOST_RSS_BRIEFING_INTERVAL", defaultRSSBriefingInterval)
}

func webSearchTavilyAPIKeyFromEnv() string {
	env := CurrentEnv()
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return env.firstNonEmpty("GHOST_WEB_SEARCH_TAVILY_API_KEY", "TAVILY_API_KEY")
	}
	return resolveWebSearchTavilyAPIKey(fileCfg, env)
}

func tasksPathFromEnv() string {
	return resolveTasksPath(CurrentEnv())
}

func nativeBinaryPathFromEnv() string {
	env := CurrentEnv()
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return env.firstNonEmpty("GHOST_NATIVE_BINARY_PATH", "GHOST_NATIVE_BIN")
	}
	return resolveNativeBinaryPath(fileCfg, env)
}

func nativeBinaryRootsFromEnv() []string {
	env := CurrentEnv()
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return normalizeConfiguredPathList(parseStringCSV(env.defaultValue("GHOST_NATIVE_BINARY_ROOTS", "")))
	}
	return resolveNativeBinaryRoots(fileCfg, env)
}

func nativeBinaryCandidatesFromEnv() []string {
	env := CurrentEnv()
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return normalizeConfiguredPathList(parseStringCSV(env.defaultValue("GHOST_NATIVE_BINARY_CANDIDATES", "")))
	}
	return resolveNativeBinaryCandidates(fileCfg, env)
}

func nativeAllowedReadPathsFromEnv() []string {
	env := CurrentEnv()
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return normalizeConfiguredPathList(parseStringCSV(env.defaultValue("GHOST_NATIVE_ALLOWED_READ_PATHS", "")))
	}
	return resolveNativeAllowedReadPaths(fileCfg, env)
}

func nativeAllowedWritePathsFromEnv() []string {
	env := CurrentEnv()
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return normalizeConfiguredPathList(parseStringCSV(env.defaultValue("GHOST_NATIVE_ALLOWED_WRITE_PATHS", "")))
	}
	return resolveNativeAllowedWritePaths(fileCfg, env)
}

func projectRootFromEnv() string {
	env := CurrentEnv()
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return env.value("GHOST_PROJECT_ROOT")
	}
	return resolveProjectRoot(fileCfg, env)
}

func firstNonEmptyEnv(names ...string) string {
	return CurrentEnv().firstNonEmpty(names...)
}
