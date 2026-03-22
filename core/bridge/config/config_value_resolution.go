package config

import (
	"fmt"
	"strings"
	"time"
)

func valueOrEnv(raw *string, envName, fallback string) string {
	return valueOrEnvWithEnv(raw, currentEnv(), envName, fallback)
}

func valueOrEnvWithEnv(raw *string, env envSnapshot, envName, fallback string) string {
	if raw != nil {
		return strings.TrimSpace(*raw)
	}
	return env.defaultValue(envName, fallback)
}

func boolOrEnv(raw *bool, envName string, fallback bool) (bool, error) {
	return boolOrEnvWithEnv(raw, currentEnv(), envName, fallback)
}

func boolOrEnvWithEnv(raw *bool, env envSnapshot, envName string, fallback bool) (bool, error) {
	if raw != nil {
		return *raw, nil
	}
	return parseBoolValue(env.value(envName), envName, fallback)
}

func intOrEnv(raw *int, fieldName, envName string, fallback int) (int, error) {
	return intOrEnvWithEnv(raw, fieldName, currentEnv(), envName, fallback)
}

func intOrEnvWithEnv(raw *int, fieldName string, env envSnapshot, envName string, fallback int) (int, error) {
	if raw != nil {
		if *raw <= 0 {
			return 0, fmt.Errorf("invalid %s: must be > 0, got %d", resolutionFieldName(fieldName, envName), *raw)
		}
		return *raw, nil
	}
	return parsePositiveIntValue(env.value(envName), envName, fallback)
}

func floatOrEnv(raw *float64, fieldName, envName string, fallback float64) (float64, error) {
	return floatOrEnvWithEnv(raw, fieldName, currentEnv(), envName, fallback)
}

func floatOrEnvWithEnv(raw *float64, fieldName string, env envSnapshot, envName string, fallback float64) (float64, error) {
	if raw != nil {
		if *raw < 0 || *raw > 1 {
			return 0, fmt.Errorf(
				"invalid %s: must be between 0 and 1, got %v",
				resolutionFieldName(fieldName, envName),
				*raw,
			)
		}
		return *raw, nil
	}
	return parseFloatValue(env.value(envName), envName, fallback)
}

func durationOrEnv(raw *string, fieldName, envName string, fallback time.Duration) (time.Duration, error) {
	return durationOrEnvWithEnv(raw, fieldName, currentEnv(), envName, fallback)
}

func durationOrEnvWithEnv(raw *string, fieldName string, env envSnapshot, envName string, fallback time.Duration) (time.Duration, error) {
	if raw != nil {
		trimmed := strings.TrimSpace(*raw)
		if trimmed == "" {
			return 0, fmt.Errorf("invalid %s: expected duration, got empty string", resolutionFieldName(fieldName, envName))
		}
		value, err := time.ParseDuration(trimmed)
		if err != nil {
			return 0, fmt.Errorf(
				"invalid %s: expected duration, got %q: %w",
				resolutionFieldName(fieldName, envName),
				trimmed,
				err,
			)
		}
		if value <= 0 {
			return 0, fmt.Errorf("invalid %s: must be > 0, got %q", resolutionFieldName(fieldName, envName), trimmed)
		}
		return value, nil
	}
	return parseDurationValue(env.value(envName), envName, fallback)
}

func headersOrEnv(raw map[string]string) (map[string]string, error) {
	return headersOrEnvWithEnv(raw, currentEnv())
}

func headersOrEnvWithEnv(raw map[string]string, env envSnapshot) (map[string]string, error) {
	if len(raw) > 0 {
		return normalizeProviderHeaders(raw)
	}
	return parseProviderHeaders(env.value("GHOST_PROVIDER_HEADERS"))
}

func corsOriginsOrEnv(raw []string) []string {
	return corsOriginsOrEnvWithEnv(raw, currentEnv())
}

func corsOriginsOrEnvWithEnv(raw []string, env envSnapshot) []string {
	if raw != nil {
		return normalizeOrigins(raw)
	}
	return parseOriginsCSV(env.defaultValue("GHOST_CORS_ORIGINS", ""))
}

func resolutionFieldName(fieldName, envName string) string {
	if trimmed := strings.TrimSpace(fieldName); trimmed != "" {
		return trimmed
	}
	return strings.TrimSpace(envName)
}
