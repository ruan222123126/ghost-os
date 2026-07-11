package runtime

import (
	"fmt"
	"strings"

	"ghost-os/bridge/config/internal/storage"
)

const (
	SessionTitleModeSessionID    = "session_id"
	SessionTitleModeFirstMessage = "first_message"
	SessionTitleModeAIGenerated  = "ai_generated"
)

const (
	RelayStopPolicyAIDecides = "ai_decides"
	RelayStopPolicyMaxRounds = "max_rounds"
)

const (
	ExternalCodexPermissionReadOnly = "read-only"
	ExternalCodexPermissionDefault  = "default"
	ExternalCodexPermissionSafeYolo = "safe-yolo"
	ExternalCodexPermissionYolo     = "yolo"
)

type relayDefaultSettings struct {
	stopPolicy         string
	maxRounds          int
	executionTimeoutMS int
}

func NormalizeSessionTitleMode(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return DefaultSessionTitleMode, nil
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

func resolveSessionTitleMode(fileCfg storage.FileConfig, fallback Snapshot) (string, error) {
	if fileCfg.SessionTitleMode == nil {
		return NormalizeSessionTitleMode(fallback.SessionTitleMode)
	}
	return NormalizeSessionTitleMode(*fileCfg.SessionTitleMode)
}

func NormalizeExternalCodexPermissionMode(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return DefaultExternalCodexPermissionMode, nil
	}
	switch value {
	case ExternalCodexPermissionReadOnly,
		ExternalCodexPermissionDefault,
		ExternalCodexPermissionSafeYolo,
		ExternalCodexPermissionYolo:
		return value, nil
	default:
		return "", fmt.Errorf("invalid external_codex_permission_mode: %q", value)
	}
}

func resolveExternalCodexPermissionMode(fileCfg storage.FileConfig, fallback Snapshot) (string, error) {
	if fileCfg.ExternalCodexPermissionMode == nil {
		return NormalizeExternalCodexPermissionMode(fallback.ExternalCodexPermissionMode)
	}
	return NormalizeExternalCodexPermissionMode(*fileCfg.ExternalCodexPermissionMode)
}

func defaultRelaySettings() (relayDefaultSettings, error) {
	return normalizeRelayDefaultSettings(relayDefaultSettings{
		stopPolicy:         DefaultRelayStopPolicy,
		maxRounds:          DefaultRelayMaxRounds,
		executionTimeoutMS: DefaultRelayExecutionTimeoutMS,
	})
}

func resolveRelayDefaults(fileCfg storage.FileConfig, fallback Snapshot) (relayDefaultSettings, error) {
	settings := relayDefaultSettings{
		stopPolicy:         fallback.RelayDefaultStopPolicy,
		maxRounds:          fallback.RelayDefaultMaxRounds,
		executionTimeoutMS: fallback.RelayDefaultExecutionTimeoutMS,
	}
	if fileCfg.RelayDefaultStopPolicy != nil {
		settings.stopPolicy = *fileCfg.RelayDefaultStopPolicy
	}
	if fileCfg.RelayDefaultMaxRounds != nil {
		settings.maxRounds = *fileCfg.RelayDefaultMaxRounds
	}
	if fileCfg.RelayDefaultExecutionTimeoutMS != nil {
		settings.executionTimeoutMS = *fileCfg.RelayDefaultExecutionTimeoutMS
	}
	return normalizeRelayDefaultSettings(settings)
}

func normalizeRelayDefaultSettings(settings relayDefaultSettings) (relayDefaultSettings, error) {
	settings.stopPolicy = strings.TrimSpace(settings.stopPolicy)
	if settings.stopPolicy == "" {
		settings.stopPolicy = DefaultRelayStopPolicy
	}
	if !ValidRelayStopPolicy(settings.stopPolicy) {
		return relayDefaultSettings{}, fmt.Errorf("invalid relay_default_stop_policy: %s", settings.stopPolicy)
	}
	if settings.maxRounds <= 0 {
		return relayDefaultSettings{}, fmt.Errorf("invalid relay_default_max_rounds: must be > 0")
	}
	if settings.executionTimeoutMS < 0 {
		return relayDefaultSettings{}, fmt.Errorf("invalid relay_default_execution_timeout_ms: must be >= 0")
	}
	return settings, nil
}

func ValidRelayStopPolicy(policy string) bool {
	switch strings.TrimSpace(policy) {
	case RelayStopPolicyAIDecides, RelayStopPolicyMaxRounds:
		return true
	default:
		return false
	}
}
