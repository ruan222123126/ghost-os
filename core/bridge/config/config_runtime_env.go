package config

import (
	"os"
	"strings"
	"time"
)

func getenvDefault(name, fallback string) string {
	v := strings.TrimSpace(os.Getenv(name))
	if v == "" {
		return fallback
	}
	return v
}

// sessionsPathFromEnv 返回会话持久化目录。
func sessionsPathFromEnv() string {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return getenvDefault("GHOST_SESSIONS_PATH", defaultSessionsPath)
	}
	return valueOrEnv(fileCfg.SessionsPath, "GHOST_SESSIONS_PATH", defaultSessionsPath)
}

func rssFeedsPathFromEnv() string {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return getenvDefault("GHOST_RSS_FEEDS_PATH", defaultRSSFeedsPath)
	}
	return valueOrEnv(fileCfg.RSSFeedsPath, "GHOST_RSS_FEEDS_PATH", defaultRSSFeedsPath)
}

func rssInboxPathFromEnv() string {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return getenvDefault("GHOST_RSS_INBOX_PATH", defaultRSSInboxPath)
	}
	return valueOrEnv(fileCfg.RSSInboxPath, "GHOST_RSS_INBOX_PATH", defaultRSSInboxPath)
}

func rssBriefingsPathFromEnv() string {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return getenvDefault("GHOST_RSS_BRIEFINGS_PATH", defaultRSSBriefingsPath)
	}
	return valueOrEnv(fileCfg.RSSBriefingsPath, "GHOST_RSS_BRIEFINGS_PATH", defaultRSSBriefingsPath)
}

func rssReportsPathFromEnv() string {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return getenvDefault("GHOST_RSS_REPORTS_PATH", defaultRSSReportsPath)
	}
	return valueOrEnv(fileCfg.RSSReportsPath, "GHOST_RSS_REPORTS_PATH", defaultRSSReportsPath)
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
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return firstNonEmptyEnv("GHOST_WEB_SEARCH_TAVILY_API_KEY", "TAVILY_API_KEY")
	}
	if fileCfg.WebSearchTavilyAPIKey != nil {
		return strings.TrimSpace(*fileCfg.WebSearchTavilyAPIKey)
	}
	return firstNonEmptyEnv("GHOST_WEB_SEARCH_TAVILY_API_KEY", "TAVILY_API_KEY")
}

func tasksPathFromEnv() string {
	return getenvDefault("GHOST_TASKS_PATH", defaultTasksPath)
}

func nativeBinaryPathFromEnv() string {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return firstNonEmptyEnv("GHOST_NATIVE_BINARY_PATH", "GHOST_NATIVE_BIN")
	}
	if fileCfg.NativeBinaryPath != nil {
		return strings.TrimSpace(*fileCfg.NativeBinaryPath)
	}
	return firstNonEmptyEnv("GHOST_NATIVE_BINARY_PATH", "GHOST_NATIVE_BIN")
}

func nativeBinaryRootsFromEnv() []string {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return normalizeConfiguredPathList(parseStringCSV(getenvDefault("GHOST_NATIVE_BINARY_ROOTS", "")))
	}
	if fileCfg.NativeBinaryRoots != nil {
		return normalizeConfiguredPathList(fileCfg.NativeBinaryRoots)
	}
	return normalizeConfiguredPathList(parseStringCSV(getenvDefault("GHOST_NATIVE_BINARY_ROOTS", "")))
}

func nativeBinaryCandidatesFromEnv() []string {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return normalizeConfiguredPathList(parseStringCSV(getenvDefault("GHOST_NATIVE_BINARY_CANDIDATES", "")))
	}
	if fileCfg.NativeBinaryCandidates != nil {
		return normalizeConfiguredPathList(fileCfg.NativeBinaryCandidates)
	}
	return normalizeConfiguredPathList(parseStringCSV(getenvDefault("GHOST_NATIVE_BINARY_CANDIDATES", "")))
}

func nativeAllowedReadPathsFromEnv() []string {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return normalizeConfiguredPathList(parseStringCSV(getenvDefault("GHOST_NATIVE_ALLOWED_READ_PATHS", "")))
	}
	if fileCfg.NativeAllowedReadPaths != nil {
		return normalizeConfiguredPathList(fileCfg.NativeAllowedReadPaths)
	}
	return normalizeConfiguredPathList(parseStringCSV(getenvDefault("GHOST_NATIVE_ALLOWED_READ_PATHS", "")))
}

func nativeAllowedWritePathsFromEnv() []string {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return normalizeConfiguredPathList(parseStringCSV(getenvDefault("GHOST_NATIVE_ALLOWED_WRITE_PATHS", "")))
	}
	if fileCfg.NativeAllowedWritePaths != nil {
		return normalizeConfiguredPathList(fileCfg.NativeAllowedWritePaths)
	}
	return normalizeConfiguredPathList(parseStringCSV(getenvDefault("GHOST_NATIVE_ALLOWED_WRITE_PATHS", "")))
}

func projectRootFromEnv() string {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return strings.TrimSpace(os.Getenv("GHOST_PROJECT_ROOT"))
	}
	if fileCfg.ProjectRoot != nil {
		return strings.TrimSpace(*fileCfg.ProjectRoot)
	}
	return strings.TrimSpace(os.Getenv("GHOST_PROJECT_ROOT"))
}

func firstNonEmptyEnv(names ...string) string {
	for _, name := range names {
		if value := strings.TrimSpace(os.Getenv(name)); value != "" {
			return value
		}
	}
	return ""
}
