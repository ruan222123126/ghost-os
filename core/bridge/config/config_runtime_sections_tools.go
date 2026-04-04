package config

import "strings"

func buildToolSelectorConfig(fileCfg bridgeFileConfig, env envSnapshot) (ToolSelectorConfig, error) {
	settings, err := resolveToolSelectorSettings(fileCfg, env)
	if err != nil {
		return ToolSelectorConfig{}, err
	}
	return ToolSelectorConfig{
		Enabled:       settings.Enabled,
		Mode:          strings.ToLower(valueOrEnvWithEnv(fileCfg.ToolSelectorMode, env, "GHOST_TOOL_SELECTOR_MODE", "llm")),
		Model:         valueOrEnvWithEnv(fileCfg.ToolSelectorModel, env, "GHOST_TOOL_SELECTOR_MODEL", ""),
		TimeoutMS:     settings.TimeoutMS,
		Confidence:    settings.Confidence,
		Shadow:        settings.Shadow,
		RecentMsgs:    settings.RecentMsgs,
		AllowlistOnly: settings.AllowlistOnly,
		Allowlist:     toolNameListOrEnvWithEnv(fileCfg.ToolAllowlist, env, "GHOST_TOOL_ALLOWLIST"),
		Blocklist:     toolNameListOrEnvWithEnv(fileCfg.ToolBlocklist, env, "GHOST_TOOL_BLOCKLIST"),
	}, nil
}

type toolSelectorSettings struct {
	Enabled       bool
	TimeoutMS     int
	Confidence    float64
	Shadow        bool
	RecentMsgs    int
	AllowlistOnly bool
}

func resolveToolSelectorSettings(fileCfg bridgeFileConfig, env envSnapshot) (toolSelectorSettings, error) {
	enabled, shadow, allowlistOnly, err := resolveToolSelectorFlags(fileCfg, env)
	if err != nil {
		return toolSelectorSettings{}, err
	}
	timeoutMS, recentMsgs, err := resolveToolSelectorIntSettings(fileCfg, env)
	if err != nil {
		return toolSelectorSettings{}, err
	}
	confidence, err := floatOrEnvWithEnv(
		fileCfg.ToolSelectorConfidence,
		"tool_selector_confidence",
		env,
		"GHOST_TOOL_SELECTOR_CONFIDENCE",
		defaultToolSelectorConfidence,
	)
	if err != nil {
		return toolSelectorSettings{}, err
	}
	return toolSelectorSettings{
		Enabled:       enabled,
		TimeoutMS:     timeoutMS,
		Confidence:    confidence,
		Shadow:        shadow,
		RecentMsgs:    recentMsgs,
		AllowlistOnly: allowlistOnly,
	}, nil
}

func resolveToolSelectorFlags(fileCfg bridgeFileConfig, env envSnapshot) (bool, bool, bool, error) {
	enabled, err := boolOrEnvWithEnv(fileCfg.ToolSelectorEnabled, env, "GHOST_TOOL_SELECTOR_ENABLED", false)
	if err != nil {
		return false, false, false, err
	}
	shadow, err := boolOrEnvWithEnv(fileCfg.ToolSelectorShadow, env, "GHOST_TOOL_SELECTOR_SHADOW", false)
	if err != nil {
		return false, false, false, err
	}
	allowlistOnly, err := boolOrEnvWithEnv(fileCfg.ToolAllowlistOnly, env, "GHOST_TOOL_ALLOWLIST_ONLY", false)
	if err != nil {
		return false, false, false, err
	}
	return enabled, shadow, allowlistOnly, nil
}

func resolveToolSelectorIntSettings(fileCfg bridgeFileConfig, env envSnapshot) (int, int, error) {
	timeoutMS, err := intOrEnvWithEnv(
		fileCfg.ToolSelectorTimeoutMS,
		"tool_selector_timeout_ms",
		env,
		"GHOST_TOOL_SELECTOR_TIMEOUT_MS",
		defaultToolSelectorTimeoutMS,
	)
	if err != nil {
		return 0, 0, err
	}
	recentMsgs, err := intOrEnvWithEnv(
		fileCfg.ToolSelectorRecentMsgs,
		"tool_selector_recent_messages",
		env,
		"GHOST_TOOL_SELECTOR_RECENT_MESSAGES",
		defaultToolSelectorRecentMsgs,
	)
	if err != nil {
		return 0, 0, err
	}
	return timeoutMS, recentMsgs, nil
}

func buildToolSearchConfig(fileCfg bridgeFileConfig, env envSnapshot) (ToolSearchConfig, error) {
	enabled, err := boolOrEnvWithEnv(fileCfg.ToolSearchEnabled, env, "GHOST_TOOL_SEARCH_ENABLED", false)
	if err != nil {
		return ToolSearchConfig{}, err
	}
	idleTurns, err := intOrEnvWithEnv(
		fileCfg.ToolSearchIdleTurns,
		"tool_search_idle_turns",
		env,
		"GHOST_TOOL_SEARCH_IDLE_TURNS",
		defaultToolSearchIdleTurns,
	)
	if err != nil {
		return ToolSearchConfig{}, err
	}
	return ToolSearchConfig{Enabled: enabled, IdleTurns: idleTurns}, nil
}

func buildMemoryAugmentationConfig(fileCfg bridgeFileConfig, env envSnapshot) (MemoryAugmentationConfig, error) {
	settings, err := resolveMemoryAugmentationSettings(fileCfg, env)
	if err != nil {
		return MemoryAugmentationConfig{}, err
	}
	return MemoryAugmentationConfig{
		Enabled:             settings.Enabled,
		LearningEnabled:     settings.LearningEnabled,
		RecallEnabled:       settings.RecallEnabled,
		MaxRecallItems:      settings.MaxRecallItems,
		MinConfidence:       settings.MinConfidence,
		SessionScopeEnabled: settings.SessionScopeEnabled,
		UserScopeEnabled:    settings.UserScopeEnabled,
		LLMModel:            valueOrEnvWithEnv(fileCfg.MemoryAugmentationLLMModel, env, "GHOST_MEMORY_AUGMENTATION_LLM_MODEL", ""),
		UserScopeID:         valueOrEnvWithEnv(fileCfg.MemoryAugmentationUserScopeID, env, "GHOST_MEMORY_AUGMENTATION_USER_SCOPE_ID", defaultMemoryUserScopeID),
	}, nil
}

type memoryAugmentationSettings struct {
	Enabled             bool
	LearningEnabled     bool
	RecallEnabled       bool
	MaxRecallItems      int
	MinConfidence       float64
	SessionScopeEnabled bool
	UserScopeEnabled    bool
}

func resolveMemoryAugmentationSettings(fileCfg bridgeFileConfig, env envSnapshot) (memoryAugmentationSettings, error) {
	enabled, learningEnabled, recallEnabled, err := resolveMemoryAugmentationFeatureFlags(fileCfg, env)
	if err != nil {
		return memoryAugmentationSettings{}, err
	}
	maxRecallItems, minConfidence, err := resolveMemoryAugmentationThresholds(fileCfg, env)
	if err != nil {
		return memoryAugmentationSettings{}, err
	}
	sessionScopeEnabled, userScopeEnabled, err := resolveMemoryAugmentationScopes(fileCfg, env)
	if err != nil {
		return memoryAugmentationSettings{}, err
	}
	return memoryAugmentationSettings{
		Enabled:             enabled,
		LearningEnabled:     learningEnabled,
		RecallEnabled:       recallEnabled,
		MaxRecallItems:      maxRecallItems,
		MinConfidence:       minConfidence,
		SessionScopeEnabled: sessionScopeEnabled,
		UserScopeEnabled:    userScopeEnabled,
	}, nil
}

func resolveMemoryAugmentationFeatureFlags(fileCfg bridgeFileConfig, env envSnapshot) (bool, bool, bool, error) {
	enabled, err := boolOrEnvWithEnv(fileCfg.MemoryAugmentationEnabled, env, "GHOST_MEMORY_AUGMENTATION_ENABLED", true)
	if err != nil {
		return false, false, false, err
	}
	learningEnabled, err := boolOrEnvWithEnv(
		fileCfg.MemoryAugmentationLearningEnabled,
		env,
		"GHOST_MEMORY_AUGMENTATION_LEARNING_ENABLED",
		true,
	)
	if err != nil {
		return false, false, false, err
	}
	recallEnabled, err := boolOrEnvWithEnv(
		fileCfg.MemoryAugmentationRecallEnabled,
		env,
		"GHOST_MEMORY_AUGMENTATION_RECALL_ENABLED",
		true,
	)
	if err != nil {
		return false, false, false, err
	}
	return enabled, learningEnabled, recallEnabled, nil
}

func resolveMemoryAugmentationThresholds(fileCfg bridgeFileConfig, env envSnapshot) (int, float64, error) {
	maxRecallItems, err := intOrEnvWithEnv(
		fileCfg.MemoryAugmentationMaxRecallItems,
		"memory_augmentation_max_recall_items",
		env,
		"GHOST_MEMORY_AUGMENTATION_MAX_RECALL_ITEMS",
		defaultMemoryRecallItems,
	)
	if err != nil {
		return 0, 0, err
	}
	minConfidence, err := floatOrEnvWithEnv(
		fileCfg.MemoryAugmentationMinConfidence,
		"memory_augmentation_min_confidence",
		env,
		"GHOST_MEMORY_AUGMENTATION_MIN_CONFIDENCE",
		defaultMemoryMinConfidence,
	)
	if err != nil {
		return 0, 0, err
	}
	return maxRecallItems, minConfidence, nil
}

func resolveMemoryAugmentationScopes(fileCfg bridgeFileConfig, env envSnapshot) (bool, bool, error) {
	sessionScopeEnabled, err := boolOrEnvWithEnv(
		fileCfg.MemoryAugmentationSessionScopeEnabled,
		env,
		"GHOST_MEMORY_AUGMENTATION_SESSION_SCOPE_ENABLED",
		true,
	)
	if err != nil {
		return false, false, err
	}
	userScopeEnabled, err := boolOrEnvWithEnv(
		fileCfg.MemoryAugmentationUserScopeEnabled,
		env,
		"GHOST_MEMORY_AUGMENTATION_USER_SCOPE_ENABLED",
		true,
	)
	if err != nil {
		return false, false, err
	}
	return sessionScopeEnabled, userScopeEnabled, nil
}
