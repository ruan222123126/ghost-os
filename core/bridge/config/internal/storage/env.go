package storage

import (
	"os"
	"strings"
)

// EnvSnapshot captures process env once so config resolution stays deterministic.
type EnvSnapshot map[string]string

func CurrentEnv() EnvSnapshot {
	return EnvFromEntries(os.Environ())
}

func EnvFromEntries(entries []string) EnvSnapshot {
	values := make(EnvSnapshot, len(entries))
	for _, entry := range entries {
		name, value, found := strings.Cut(entry, "=")
		if !found {
			values[strings.TrimSpace(entry)] = ""
			continue
		}
		values[strings.TrimSpace(name)] = strings.TrimSpace(value)
	}
	return values
}

func (env EnvSnapshot) Value(name string) string {
	if env == nil {
		return ""
	}
	return strings.TrimSpace(env[name])
}

func (env EnvSnapshot) DefaultValue(name string, fallback string) string {
	if value := env.Value(name); value != "" {
		return value
	}
	return fallback
}

func (env EnvSnapshot) FirstNonEmpty(names ...string) string {
	for _, name := range names {
		if value := env.Value(name); value != "" {
			return value
		}
	}
	return ""
}

func ConfigPathFromEnv() string {
	return CurrentEnv().DefaultValue("GHOST_CONFIG_PATH", DefaultConfigPath)
}
