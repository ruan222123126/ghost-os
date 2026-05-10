package config

import (
	"fmt"
	"strings"
)

const (
	RelayStopPolicyAIDecides = "ai_decides"
	RelayStopPolicyMaxRounds = "max_rounds"
)

type relayDefaultSettings struct {
	stopPolicy         string
	maxRounds          int
	executionTimeoutMS int
}

func defaultRelaySettings() (relayDefaultSettings, error) {
	return normalizeRelayDefaultSettings(relayDefaultSettings{
		stopPolicy:         defaultRelayStopPolicy,
		maxRounds:          defaultRelayMaxRounds,
		executionTimeoutMS: defaultRelayExecutionTimeoutMS,
	})
}

func resolveRuntimeRelayDefaults(fileCfg bridgeFileConfig, fallback runtimeConfig) (relayDefaultSettings, error) {
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
		settings.stopPolicy = defaultRelayStopPolicy
	}
	if !validRelayStopPolicy(settings.stopPolicy) {
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

func validRelayStopPolicy(policy string) bool {
	switch strings.TrimSpace(policy) {
	case RelayStopPolicyAIDecides, RelayStopPolicyMaxRounds:
		return true
	default:
		return false
	}
}
