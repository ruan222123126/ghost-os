package config

import "time"

func getenvDefault(name, fallback string) string {
	return currentEnv().defaultValue(name, fallback)
}

// sessionsPathFromEnv 返回会话持久化目录。
func sessionsPathFromEnv() (string, error) {
	aux, err := loadAuxConfigFromEnv()
	if err != nil {
		return "", err
	}
	return aux.SessionsPath, nil
}

func rssFeedsPathFromEnv() (string, error) {
	aux, err := loadAuxConfigFromEnv()
	if err != nil {
		return "", err
	}
	return aux.RSS.FeedsPath, nil
}

func rssInboxPathFromEnv() (string, error) {
	aux, err := loadAuxConfigFromEnv()
	if err != nil {
		return "", err
	}
	return aux.RSS.InboxPath, nil
}

func rssBriefingsPathFromEnv() (string, error) {
	aux, err := loadAuxConfigFromEnv()
	if err != nil {
		return "", err
	}
	return aux.RSS.BriefingsPath, nil
}

func rssReportsPathFromEnv() (string, error) {
	aux, err := loadAuxConfigFromEnv()
	if err != nil {
		return "", err
	}
	return aux.RSS.ReportsPath, nil
}

func rssPollEnabledFromEnv() (bool, error) {
	aux, err := loadAuxConfigFromEnv()
	if err != nil {
		return false, err
	}
	return aux.RSS.PollEnabled, nil
}

func rssPollIntervalFromEnv() (time.Duration, error) {
	aux, err := loadAuxConfigFromEnv()
	if err != nil {
		return 0, err
	}
	return aux.RSS.PollInterval, nil
}

func rssPollMaxItemsPerFeedFromEnv() (int, error) {
	aux, err := loadAuxConfigFromEnv()
	if err != nil {
		return 0, err
	}
	return aux.RSS.PollMaxItemsPerFeed, nil
}

func rssAIBatchSizeFromEnv() (int, error) {
	aux, err := loadAuxConfigFromEnv()
	if err != nil {
		return 0, err
	}
	return aux.RSS.AIBatchSize, nil
}

func rssBriefingEnabledFromEnv() (bool, error) {
	aux, err := loadAuxConfigFromEnv()
	if err != nil {
		return false, err
	}
	return aux.RSS.BriefingEnabled, nil
}

func rssBriefingIntervalFromEnv() (time.Duration, error) {
	aux, err := loadAuxConfigFromEnv()
	if err != nil {
		return 0, err
	}
	return aux.RSS.BriefingInterval, nil
}

func webSearchTavilyAPIKeyFromEnv() (string, error) {
	aux, err := loadAuxConfigFromEnv()
	if err != nil {
		return "", err
	}
	return aux.WebSearchTavilyAPIKey, nil
}

func tasksPathFromEnv() string {
	return resolveTasksPath(currentEnv())
}

func nativeBinaryPathFromEnv() (string, error) {
	aux, err := loadAuxConfigFromEnv()
	if err != nil {
		return "", err
	}
	return aux.Execution.NativeBinaryPath, nil
}

func nativeBinaryRootsFromEnv() ([]string, error) {
	aux, err := loadAuxConfigFromEnv()
	if err != nil {
		return nil, err
	}
	return aux.Execution.NativeBinaryRoots, nil
}

func nativeBinaryCandidatesFromEnv() ([]string, error) {
	aux, err := loadAuxConfigFromEnv()
	if err != nil {
		return nil, err
	}
	return aux.Execution.NativeBinaryCandidates, nil
}

func nativeAllowedReadPathsFromEnv() ([]string, error) {
	aux, err := loadAuxConfigFromEnv()
	if err != nil {
		return nil, err
	}
	return aux.Execution.AllowedReadPaths, nil
}

func nativeAllowedWritePathsFromEnv() ([]string, error) {
	aux, err := loadAuxConfigFromEnv()
	if err != nil {
		return nil, err
	}
	return aux.Execution.AllowedWritePaths, nil
}

func projectRootFromEnv() (string, error) {
	aux, err := loadAuxConfigFromEnv()
	if err != nil {
		return "", err
	}
	return aux.Execution.ProjectRoot, nil
}

func firstNonEmptyEnv(names ...string) string {
	return currentEnv().firstNonEmpty(names...)
}
