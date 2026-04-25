package config

import "strings"

func buildToolSelectorConfig(fileCfg bridgeFileConfig, env envSnapshot) (ToolSelectorConfig, error) {
	settings, err := resolveToolSelectorSettings(fileCfg, env)
	if err != nil {
		return ToolSelectorConfig{}, err
	}
	return ToolSelectorConfig{
		Enabled:         settings.Enabled,
		Mode:            strings.ToLower(valueOrEnvWithEnv(fileCfg.ToolSelectorMode, env, "GHOST_TOOL_SELECTOR_MODE", "llm")),
		Model:           valueOrEnvWithEnv(fileCfg.ToolSelectorModel, env, "GHOST_TOOL_SELECTOR_MODEL", ""),
		TimeoutMS:       settings.TimeoutMS,
		Confidence:      settings.Confidence,
		Shadow:          settings.Shadow,
		RecentMsgs:      settings.RecentMsgs,
		AllowlistOnly:   settings.AllowlistOnly,
		Allowlist:       toolNameListOrEnvWithEnv(fileCfg.ToolAllowlist, env, "GHOST_TOOL_ALLOWLIST"),
		Blocklist:       toolNameListOrEnvWithEnv(fileCfg.ToolBlocklist, env, "GHOST_TOOL_BLOCKLIST"),
		PromptOverrides: cloneStringMap(fileCfg.ToolPromptOverrides),
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
