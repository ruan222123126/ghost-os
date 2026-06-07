package storage

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"
)

func LoadToml(path string) (FileConfig, error) {
	resolvedPath, err := ResolveUserPath(path)
	if err != nil {
		return FileConfig{}, fmt.Errorf("resolve config path: %w", err)
	}

	raw, err := os.ReadFile(resolvedPath)
	if err != nil {
		return FileConfig{}, err
	}
	if strings.TrimSpace(string(raw)) == "" {
		return FileConfig{}, nil
	}

	var cfg FileConfig
	md, err := toml.Decode(string(raw), &cfg)
	if err != nil {
		return FileConfig{}, fmt.Errorf("parse config file %s: %w", resolvedPath, err)
	}
	if err := ValidateRemovedConfigKeys(md.Undecoded()); err != nil {
		return FileConfig{}, fmt.Errorf("parse config file %s: %w", resolvedPath, err)
	}
	return cfg, nil
}

func Load() (FileConfig, string, error) {
	configPath := ConfigPathFromEnv()
	rawCfg, err := LoadToml(configPath)
	switch {
	case err == nil:
		resolvedPath, resolveErr := ResolveUserPath(configPath)
		if resolveErr != nil {
			return FileConfig{}, "", fmt.Errorf("resolve config path: %w", resolveErr)
		}
		normalized, normalizeErr := NormalizeForWrite(rawCfg)
		if normalizeErr != nil {
			return FileConfig{}, "", fmt.Errorf("normalize config file %s: %w", resolvedPath, normalizeErr)
		}
		return normalized, resolvedPath, nil
	case errors.Is(err, os.ErrNotExist):
		resolvedPath, resolveErr := ResolveUserPath(configPath)
		if resolveErr != nil {
			return FileConfig{}, "", fmt.Errorf("resolve config path: %w", resolveErr)
		}
		return FileConfig{}, resolvedPath, nil
	default:
		return FileConfig{}, configPath, fmt.Errorf("read config file %s: %w", configPath, err)
	}
}

func Write(path string, cfg FileConfig) error {
	return WriteToml(path, cfg)
}

func WriteToml(path string, cfg FileConfig) error {
	resolvedPath, err := ResolveUserPath(path)
	if err != nil {
		return fmt.Errorf("resolve config path: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(resolvedPath), 0o755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	normalized, normalizeErr := NormalizeForWrite(cfg)
	if normalizeErr != nil {
		return fmt.Errorf("normalize config file: %w", normalizeErr)
	}
	var buffer bytes.Buffer
	if err := toml.NewEncoder(&buffer).Encode(normalized); err != nil {
		return fmt.Errorf("encode config file: %w", err)
	}
	if err := os.WriteFile(resolvedPath, buffer.Bytes(), 0o600); err != nil {
		return fmt.Errorf("write config file %s: %w", resolvedPath, err)
	}
	return nil
}

func ValidateRemovedConfigKeys(keys []toml.Key) error {
	if len(keys) == 0 {
		return nil
	}

	removed := CollectRemovedConfigKeys(keys)
	if len(removed) == 0 {
		return nil
	}
	return fmt.Errorf("unsupported config fields: %s", strings.Join(removed, ", "))
}

func CollectRemovedConfigKeys(keys []toml.Key) []string {
	seen := make(map[string]struct{}, len(keys))
	out := make([]string, 0, len(keys))
	for _, key := range keys {
		name := RemovedConfigKeyName(key)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

func RemovedConfigKeyName(key toml.Key) string {
	if len(key) == 0 {
		return ""
	}

	switch key[0] {
	case "model_providers",
		"graphql_headers",
		"graphql_sources",
		"graphql_mutation_policies":
		return key[0]
	}
	if len(key) != 1 {
		return ""
	}

	switch key[0] {
	case "model_provider",
		"provider",
		"api_key",
		"base_url",
		"graphql_default_source",
		"graphql_tool_runtime_enabled",
		"graphql_text_sanitize_enabled",
		"graphql_enabled",
		"graphql_endpoint",
		"graphql_api_key",
		"graphql_schema_path",
		"graphql_timeout_ms",
		"graphql_max_response_bytes",
		"tool_prompt_overrides":
		return key[0]
	default:
		return ""
	}
}
