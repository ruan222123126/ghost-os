package config

import (
	"os"
	"strings"
)

// Env 是配置解析阶段使用的环境快照，避免 resolve 过程中直接读取进程环境。
type Env map[string]string

func CurrentEnv() Env {
	return envFromEntries(os.Environ())
}

func envFromEntries(entries []string) Env {
	values := make(Env, len(entries))
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

func (env Env) value(name string) string {
	if env == nil {
		return ""
	}
	return strings.TrimSpace(env[name])
}

func (env Env) defaultValue(name string, fallback string) string {
	if value := env.value(name); value != "" {
		return value
	}
	return fallback
}

func (env Env) firstNonEmpty(names ...string) string {
	for _, name := range names {
		if value := env.value(name); value != "" {
			return value
		}
	}
	return ""
}
