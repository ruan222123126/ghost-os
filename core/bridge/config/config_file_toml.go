package config

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

func loadBridgeTomlConfig(path string) (bridgeFileConfig, error) {
	resolvedPath, err := resolveUserPath(path)
	if err != nil {
		return bridgeFileConfig{}, fmt.Errorf("resolve config path: %w", err)
	}

	raw, err := os.ReadFile(resolvedPath)
	if err != nil {
		return bridgeFileConfig{}, err
	}
	if strings.TrimSpace(string(raw)) == "" {
		return bridgeFileConfig{}, nil
	}

	var cfg bridgeFileConfig
	md, err := toml.Decode(string(raw), &cfg)
	if err != nil {
		return bridgeFileConfig{}, fmt.Errorf("parse config file %s: %w", resolvedPath, err)
	}
	if err := validateRemovedConfigKeys(md.Undecoded()); err != nil {
		return bridgeFileConfig{}, fmt.Errorf("parse config file %s: %w", resolvedPath, err)
	}
	return cfg, nil
}

func loadBridgeFileConfig() (bridgeFileConfig, string, error) {
	configPath := configPathFromEnv()
	rawCfg, err := loadBridgeTomlConfig(configPath)
	switch {
	case err == nil:
		resolvedPath, resolveErr := resolveUserPath(configPath)
		if resolveErr != nil {
			return bridgeFileConfig{}, "", fmt.Errorf("resolve config path: %w", resolveErr)
		}
		normalized, normalizeErr := normalizeBridgeFileConfigForWrite(rawCfg)
		if normalizeErr != nil {
			return bridgeFileConfig{}, "", fmt.Errorf("normalize config file %s: %w", resolvedPath, normalizeErr)
		}
		return normalized, resolvedPath, nil
	case errors.Is(err, os.ErrNotExist):
		resolvedPath, resolveErr := resolveUserPath(configPath)
		if resolveErr != nil {
			return bridgeFileConfig{}, "", fmt.Errorf("resolve config path: %w", resolveErr)
		}
		return bridgeFileConfig{}, resolvedPath, nil
	default:
		return bridgeFileConfig{}, configPath, fmt.Errorf("read config file %s: %w", configPath, err)
	}
}

func writeBridgeFileConfig(path string, cfg bridgeFileConfig) error {
	return writeBridgeTomlConfig(path, cfg)
}

func writeBridgeTomlConfig(path string, cfg bridgeFileConfig) error {
	resolvedPath, err := resolveUserPath(path)
	if err != nil {
		return fmt.Errorf("resolve config path: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(resolvedPath), 0o755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	normalized, normalizeErr := normalizeBridgeFileConfigForWrite(cfg)
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

func validateRemovedConfigKeys(keys []toml.Key) error {
	if len(keys) == 0 {
		return nil
	}

	removed := collectRemovedConfigKeys(keys)
	if len(removed) == 0 {
		return nil
	}
	return fmt.Errorf(
		"unsupported legacy config fields: %s; migrate to [providers] and graphql_sources/graphql_mutation_policies",
		strings.Join(removed, ", "),
	)
}

func collectRemovedConfigKeys(keys []toml.Key) []string {
	seen := make(map[string]struct{}, len(keys))
	out := make([]string, 0, len(keys))
	for _, key := range keys {
		name := removedConfigKeyName(key)
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

func removedConfigKeyName(key toml.Key) string {
	if len(key) == 0 {
		return ""
	}

	switch key[0] {
	case "model_providers", "graphql_headers":
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
		"graphql_enabled",
		"graphql_endpoint",
		"graphql_api_key",
		"graphql_schema_path",
		"graphql_timeout_ms",
		"graphql_max_response_bytes":
		return key[0]
	default:
		return ""
	}
}
