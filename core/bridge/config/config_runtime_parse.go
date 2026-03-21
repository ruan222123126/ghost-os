package config

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

func parseProviderHeaders(raw string) (map[string]string, error) {
	return parseNamedHeaders(raw, "GHOST_PROVIDER_HEADERS")
}

func parseNamedHeaders(raw string, envName string) (map[string]string, error) {
	text := strings.TrimSpace(raw)
	if text == "" {
		return nil, nil
	}

	var parsed map[string]string
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		return nil, fmt.Errorf("invalid %s: expected JSON object of string values: %w", strings.TrimSpace(envName), err)
	}

	out := make(map[string]string, len(parsed))
	for key, value := range parsed {
		k := strings.TrimSpace(key)
		if k == "" {
			return nil, fmt.Errorf("invalid %s: header key cannot be empty", strings.TrimSpace(envName))
		}
		out[k] = strings.TrimSpace(value)
	}

	return out, nil
}

func nativePersistentEnabledFromEnv() bool {
	env := CurrentEnv()
	fileCfg, _, err := loadBridgeFileConfig()
	if err == nil {
		return resolveNativePersistent(fileCfg.NativePersistent, env)
	}
	return resolveNativePersistent(nil, env)
}

func resolveNativePersistent(raw *bool, env Env) bool {
	if raw != nil {
		return *raw
	}
	for _, name := range []string{"GHOST_NATIVE_PERSISTENT", "GHOST_NATIVE_PERSISTENT_ENABLED"} {
		if rawValue := env.value(name); rawValue == "" {
			continue
		} else {
			return parseBoolValue(rawValue, false)
		}
	}
	return false
}

func parseBoolEnv(name string, fallback bool) bool {
	return parseBoolValue(CurrentEnv().value(name), fallback)
}

func parsePositiveIntEnv(name string, fallback int) int {
	return parsePositiveIntValue(CurrentEnv().value(name), fallback)
}

func parseFloatEnv(name string, fallback float64) float64 {
	return parseFloatValue(CurrentEnv().value(name), fallback)
}

func parseDurationEnv(name string, fallback time.Duration) time.Duration {
	return parseDurationValue(CurrentEnv().value(name), fallback)
}

func parseBoolValue(raw string, fallback bool) bool {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return fallback
	}
	value, err := strconv.ParseBool(trimmed)
	if err != nil {
		return fallback
	}
	return value
}

func parsePositiveIntValue(raw string, fallback int) int {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return fallback
	}
	value, err := strconv.Atoi(trimmed)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func parseFloatValue(raw string, fallback float64) float64 {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return fallback
	}
	value, err := strconv.ParseFloat(trimmed, 64)
	if err != nil || value < 0 || value > 1 {
		return fallback
	}
	return value
}

func parseDurationValue(raw string, fallback time.Duration) time.Duration {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return fallback
	}
	value, err := time.ParseDuration(trimmed)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}
