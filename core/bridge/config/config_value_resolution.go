package config

import (
	"os"
	"strings"
	"time"
)

func valueOrEnv(raw *string, envName, fallback string) string {
	if raw != nil {
		return strings.TrimSpace(*raw)
	}
	return getenvDefault(envName, fallback)
}

func boolOrEnv(raw *bool, envName string, fallback bool) bool {
	if raw != nil {
		return *raw
	}
	return parseBoolEnv(envName, fallback)
}

func intOrEnv(raw *int, envName string, fallback int) int {
	if raw != nil {
		if *raw > 0 {
			return *raw
		}
		return fallback
	}
	return parsePositiveIntEnv(envName, fallback)
}

func floatOrEnv(raw *float64, envName string, fallback float64) float64 {
	if raw != nil {
		if *raw >= 0 && *raw <= 1 {
			return *raw
		}
		return fallback
	}
	return parseFloatEnv(envName, fallback)
}

func durationOrEnv(raw *string, envName string, fallback time.Duration) time.Duration {
	if raw != nil {
		value, err := time.ParseDuration(strings.TrimSpace(*raw))
		if err != nil || value <= 0 {
			return fallback
		}
		return value
	}
	return parseDurationEnv(envName, fallback)
}

func headersOrEnv(raw map[string]string) (map[string]string, error) {
	if len(raw) > 0 {
		return normalizeProviderHeaders(raw)
	}
	return parseProviderHeaders(os.Getenv("GHOST_PROVIDER_HEADERS"))
}

func corsOriginsOrEnv(raw []string) []string {
	if raw != nil {
		return normalizeOrigins(raw)
	}
	return parseOriginsCSV(getenvDefault("GHOST_CORS_ORIGINS", ""))
}
