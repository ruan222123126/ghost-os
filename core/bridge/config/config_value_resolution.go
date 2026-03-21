package config

import (
	"strings"
	"time"
)

func valueOrEnv(raw *string, envName, fallback string) string {
	return valueOrEnvWithEnv(raw, CurrentEnv(), envName, fallback)
}

func valueOrEnvWithEnv(raw *string, env Env, envName, fallback string) string {
	if raw != nil {
		return strings.TrimSpace(*raw)
	}
	return env.defaultValue(envName, fallback)
}

func boolOrEnv(raw *bool, envName string, fallback bool) bool {
	return boolOrEnvWithEnv(raw, CurrentEnv(), envName, fallback)
}

func boolOrEnvWithEnv(raw *bool, env Env, envName string, fallback bool) bool {
	if raw != nil {
		return *raw
	}
	return parseBoolValue(env.value(envName), fallback)
}

func intOrEnv(raw *int, envName string, fallback int) int {
	return intOrEnvWithEnv(raw, CurrentEnv(), envName, fallback)
}

func intOrEnvWithEnv(raw *int, env Env, envName string, fallback int) int {
	if raw != nil {
		if *raw > 0 {
			return *raw
		}
		return fallback
	}
	return parsePositiveIntValue(env.value(envName), fallback)
}

func floatOrEnv(raw *float64, envName string, fallback float64) float64 {
	return floatOrEnvWithEnv(raw, CurrentEnv(), envName, fallback)
}

func floatOrEnvWithEnv(raw *float64, env Env, envName string, fallback float64) float64 {
	if raw != nil {
		if *raw >= 0 && *raw <= 1 {
			return *raw
		}
		return fallback
	}
	return parseFloatValue(env.value(envName), fallback)
}

func durationOrEnv(raw *string, envName string, fallback time.Duration) time.Duration {
	return durationOrEnvWithEnv(raw, CurrentEnv(), envName, fallback)
}

func durationOrEnvWithEnv(raw *string, env Env, envName string, fallback time.Duration) time.Duration {
	if raw != nil {
		value, err := time.ParseDuration(strings.TrimSpace(*raw))
		if err != nil || value <= 0 {
			return fallback
		}
		return value
	}
	return parseDurationValue(env.value(envName), fallback)
}

func headersOrEnv(raw map[string]string) (map[string]string, error) {
	return headersOrEnvWithEnv(raw, CurrentEnv())
}

func headersOrEnvWithEnv(raw map[string]string, env Env) (map[string]string, error) {
	if len(raw) > 0 {
		return normalizeProviderHeaders(raw)
	}
	return parseProviderHeaders(env.value("GHOST_PROVIDER_HEADERS"))
}

func corsOriginsOrEnv(raw []string) []string {
	return corsOriginsOrEnvWithEnv(raw, CurrentEnv())
}

func corsOriginsOrEnvWithEnv(raw []string, env Env) []string {
	if raw != nil {
		return normalizeOrigins(raw)
	}
	return parseOriginsCSV(env.defaultValue("GHOST_CORS_ORIGINS", ""))
}
