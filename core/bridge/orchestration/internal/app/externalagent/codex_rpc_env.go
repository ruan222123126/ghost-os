package externalagent

import (
	"os"
	"strings"
)

func codexEnv(pathOverride string) []string {
	env := os.Environ()
	hasRustLog := false
	hasPath := false
	for i, item := range env {
		if matchesEnvKey(item, "RUST_LOG") {
			hasRustLog = true
			if !strings.Contains(item, "codex_core::rollout::list=") {
				env[i] = envAssignment("RUST_LOG", envValue(item)+",codex_core::rollout::list=off")
			}
			continue
		}
		if pathOverride != "" && matchesEnvKey(item, "PATH") {
			hasPath = true
			env[i] = envAssignment(envKey(item), pathOverride)
		}
	}
	if !hasRustLog {
		env = append(env, "RUST_LOG=codex_core::rollout::list=off")
	}
	if pathOverride != "" && !hasPath {
		env = append(env, envAssignment("PATH", pathOverride))
	}
	return env
}

func matchesEnvKey(item string, key string) bool {
	return strings.EqualFold(envKey(item), key)
}

func envKey(item string) string {
	key, _, _ := strings.Cut(item, "=")
	return key
}

func envValue(item string) string {
	_, value, _ := strings.Cut(item, "=")
	return value
}

func envAssignment(key string, value string) string {
	return key + "=" + value
}
