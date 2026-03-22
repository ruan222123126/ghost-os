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

func nativePersistentEnabledFromEnv() (bool, error) {
	aux, err := loadAuxConfigFromEnv()
	if err != nil {
		return false, err
	}
	return aux.Execution.Persistent, nil
}

func resolveNativePersistent(raw *bool, env envSnapshot) (bool, error) {
	if raw != nil {
		return *raw, nil
	}
	for _, name := range []string{"GHOST_NATIVE_PERSISTENT", "GHOST_NATIVE_PERSISTENT_ENABLED"} {
		rawValue := strings.TrimSpace(env.value(name))
		if rawValue == "" {
			continue
		}
		return parseBoolValue(rawValue, name, false)
	}
	return false, nil
}

func parseBoolEnv(name string, fallback bool) (bool, error) {
	return parseBoolValue(currentEnv().value(name), name, fallback)
}

func parsePositiveIntEnv(name string, fallback int) (int, error) {
	return parsePositiveIntValue(currentEnv().value(name), name, fallback)
}

func parseFloatEnv(name string, fallback float64) (float64, error) {
	return parseFloatValue(currentEnv().value(name), name, fallback)
}

func parseDurationEnv(name string, fallback time.Duration) (time.Duration, error) {
	return parseDurationValue(currentEnv().value(name), name, fallback)
}

func parseBoolValue(raw, fieldName string, fallback bool) (bool, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return fallback, nil
	}
	value, err := strconv.ParseBool(trimmed)
	if err != nil {
		return false, fmt.Errorf("invalid %s: expected boolean, got %q", strings.TrimSpace(fieldName), trimmed)
	}
	return value, nil
}

func parsePositiveIntValue(raw, fieldName string, fallback int) (int, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return fallback, nil
	}
	value, err := strconv.Atoi(trimmed)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: expected positive integer, got %q", strings.TrimSpace(fieldName), trimmed)
	}
	if value <= 0 {
		return 0, fmt.Errorf("invalid %s: must be > 0, got %q", strings.TrimSpace(fieldName), trimmed)
	}
	return value, nil
}

func parseFloatValue(raw, fieldName string, fallback float64) (float64, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return fallback, nil
	}
	value, err := strconv.ParseFloat(trimmed, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: expected number in [0, 1], got %q", strings.TrimSpace(fieldName), trimmed)
	}
	if value < 0 || value > 1 {
		return 0, fmt.Errorf("invalid %s: must be between 0 and 1, got %q", strings.TrimSpace(fieldName), trimmed)
	}
	return value, nil
}

func parseDurationValue(raw, fieldName string, fallback time.Duration) (time.Duration, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return fallback, nil
	}
	value, err := time.ParseDuration(trimmed)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: expected duration, got %q: %w", strings.TrimSpace(fieldName), trimmed, err)
	}
	if value <= 0 {
		return 0, fmt.Errorf("invalid %s: must be > 0, got %q", strings.TrimSpace(fieldName), trimmed)
	}
	return value, nil
}
