package config

import (
	"fmt"
	"strings"
)

const (
	SessionTitleModeSessionID    = "session_id"
	SessionTitleModeFirstMessage = "first_message"
	SessionTitleModeAIGenerated  = "ai_generated"
)

func normalizeSessionTitleMode(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return defaultSessionTitleMode, nil
	}
	switch value {
	case SessionTitleModeSessionID,
		SessionTitleModeFirstMessage,
		SessionTitleModeAIGenerated:
		return value, nil
	default:
		return "", fmt.Errorf("invalid session_title_mode: %q", value)
	}
}

func resolveRuntimeSessionTitleMode(fileCfg bridgeFileConfig, fallback runtimeConfig) (string, error) {
	if fileCfg.SessionTitleMode == nil {
		return normalizeSessionTitleMode(fallback.SessionTitleMode)
	}
	return normalizeSessionTitleMode(*fileCfg.SessionTitleMode)
}
