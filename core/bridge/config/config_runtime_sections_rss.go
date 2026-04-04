package config

import "time"

func buildRSSConfig(fileCfg bridgeFileConfig, env envSnapshot) (RSSConfig, error) {
	settings, err := resolveRSSSettings(fileCfg, env)
	if err != nil {
		return RSSConfig{}, err
	}
	return RSSConfig{
		FeedsPath:           resolveRSSFeedsPath(fileCfg, env),
		InboxPath:           resolveRSSInboxPath(fileCfg, env),
		BriefingsPath:       resolveRSSBriefingsPath(fileCfg, env),
		ReportsPath:         resolveRSSReportsPath(fileCfg, env),
		PollEnabled:         settings.PollEnabled,
		PollInterval:        settings.PollInterval,
		PollMaxItemsPerFeed: settings.PollMaxItemsPerFeed,
		AIBatchSize:         settings.AIBatchSize,
		BriefingEnabled:     settings.BriefingEnabled,
		BriefingInterval:    settings.BriefingInterval,
	}, nil
}

type rssSettings struct {
	PollEnabled         bool
	PollInterval        time.Duration
	PollMaxItemsPerFeed int
	AIBatchSize         int
	BriefingEnabled     bool
	BriefingInterval    time.Duration
}

func resolveRSSSettings(fileCfg bridgeFileConfig, env envSnapshot) (rssSettings, error) {
	pollEnabled, briefingEnabled, err := resolveRSSFeatureFlags(fileCfg, env)
	if err != nil {
		return rssSettings{}, err
	}
	pollInterval, briefingInterval, err := resolveRSSIntervals(fileCfg, env)
	if err != nil {
		return rssSettings{}, err
	}
	pollMaxItemsPerFeed, aiBatchSize, err := resolveRSSItemLimits(fileCfg, env)
	if err != nil {
		return rssSettings{}, err
	}
	return rssSettings{
		PollEnabled:         pollEnabled,
		PollInterval:        pollInterval,
		PollMaxItemsPerFeed: pollMaxItemsPerFeed,
		AIBatchSize:         aiBatchSize,
		BriefingEnabled:     briefingEnabled,
		BriefingInterval:    briefingInterval,
	}, nil
}

func resolveRSSFeatureFlags(fileCfg bridgeFileConfig, env envSnapshot) (bool, bool, error) {
	pollEnabled, err := boolOrEnvWithEnv(fileCfg.RSSPollEnabled, env, "GHOST_RSS_POLL_ENABLED", true)
	if err != nil {
		return false, false, err
	}
	briefingEnabled, err := boolOrEnvWithEnv(fileCfg.RSSBriefingEnabled, env, "GHOST_RSS_BRIEFING_ENABLED", true)
	if err != nil {
		return false, false, err
	}
	return pollEnabled, briefingEnabled, nil
}

func resolveRSSIntervals(fileCfg bridgeFileConfig, env envSnapshot) (time.Duration, time.Duration, error) {
	pollInterval, err := durationOrEnvWithEnv(
		fileCfg.RSSPollInterval,
		"rss_poll_interval",
		env,
		"GHOST_RSS_POLL_INTERVAL",
		defaultRSSPollInterval,
	)
	if err != nil {
		return 0, 0, err
	}
	briefingInterval, err := durationOrEnvWithEnv(
		fileCfg.RSSBriefingInterval,
		"rss_briefing_interval",
		env,
		"GHOST_RSS_BRIEFING_INTERVAL",
		defaultRSSBriefingInterval,
	)
	if err != nil {
		return 0, 0, err
	}
	return pollInterval, briefingInterval, nil
}

func resolveRSSItemLimits(fileCfg bridgeFileConfig, env envSnapshot) (int, int, error) {
	pollMaxItemsPerFeed, err := intOrEnvWithEnv(
		fileCfg.RSSPollMaxItemsPerFeed,
		"rss_poll_max_items_per_feed",
		env,
		"GHOST_RSS_POLL_MAX_ITEMS_PER_FEED",
		defaultRSSPollMaxItemsPerFeed,
	)
	if err != nil {
		return 0, 0, err
	}
	aiBatchSize, err := intOrEnvWithEnv(
		fileCfg.RSSAIBatchSize,
		"rss_ai_batch_size",
		env,
		"GHOST_RSS_AI_BATCH_SIZE",
		defaultRSSAIBatchSize,
	)
	if err != nil {
		return 0, 0, err
	}
	return pollMaxItemsPerFeed, aiBatchSize, nil
}
