package config

import (
	"encoding/json"
	"fmt"
	"os"
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
	fileCfg, _, err := loadBridgeFileConfig()
	if err == nil {
		return resolveNativePersistent(fileCfg.NativePersistent)
	}
	return resolveNativePersistent(nil)
}

func resolveNativePersistent(raw *bool) bool {
	if raw != nil {
		return *raw
	}
	for _, name := range []string{"GHOST_NATIVE_PERSISTENT", "GHOST_NATIVE_PERSISTENT_ENABLED"} {
		rawValue := strings.TrimSpace(os.Getenv(name))
		if rawValue == "" {
			continue
		}
		enabled, err := strconv.ParseBool(rawValue)
		if err != nil {
			return false
		}
		return enabled
	}
	return false
}

func parseBoolEnv(name string, fallback bool) bool {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}
	value, err := strconv.ParseBool(raw)
	if err != nil {
		return fallback
	}
	return value
}

func parsePositiveIntEnv(name string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func parseFloatEnv(name string, fallback float64) float64 {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil || value < 0 || value > 1 {
		return fallback
	}
	return value
}

func parseDurationEnv(name string, fallback time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return fallback
	}
	value, err := time.ParseDuration(raw)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}
