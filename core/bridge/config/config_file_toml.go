package config

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
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
	if _, err := toml.Decode(string(raw), &cfg); err != nil {
		return bridgeFileConfig{}, fmt.Errorf("parse config file %s: %w", resolvedPath, err)
	}
	return cfg, nil
}

func loadBridgeFileConfig() (bridgeFileConfig, string, error) {
	cfg, path, _, err := loadBridgeFileConfigWithMigrationInfo()
	return cfg, path, err
}

// loadBridgeFileConfigWithMigrationInfo behaves like loadBridgeFileConfig but also reports
// whether the config file uses a legacy provider layout that can be migrated.
//
// IMPORTANT: This function is side-effect free; callers must explicitly invoke migration/write.
func loadBridgeFileConfigWithMigrationInfo() (bridgeFileConfig, string, bool, error) {
	configPath := configPathFromEnv()
	rawCfg, err := loadBridgeTomlConfig(configPath)
	switch {
	case err == nil:
		resolvedPath, resolveErr := resolveUserPath(configPath)
		if resolveErr != nil {
			return bridgeFileConfig{}, "", false, fmt.Errorf("resolve config path: %w", resolveErr)
		}
		needsMigration := hasLegacyProviderLayout(rawCfg)
		normalized := normalizeBridgeFileConfigForWrite(rawCfg)
		return normalized, resolvedPath, needsMigration, nil
	case errors.Is(err, os.ErrNotExist):
		resolvedPath, resolveErr := resolveUserPath(configPath)
		if resolveErr != nil {
			return bridgeFileConfig{}, "", false, fmt.Errorf("resolve config path: %w", resolveErr)
		}
		return bridgeFileConfig{}, resolvedPath, false, nil
	default:
		return bridgeFileConfig{}, configPath, false, fmt.Errorf("read config file %s: %w", configPath, err)
	}
}

// migrateBridgeFileConfigIfLegacy rewrites the configured TOML file to the current provider layout,
// but only when legacy provider fields are present.
//
// This migration is intentionally opt-in to avoid "read == write" surprises across server options
// and middleware that read config frequently.
func migrateBridgeFileConfigIfLegacy() (string, bool, error) {
	cfg, resolvedPath, needsMigration, err := loadBridgeFileConfigWithMigrationInfo()
	if err != nil {
		return resolvedPath, false, err
	}
	if !needsMigration {
		return resolvedPath, false, nil
	}
	if err := writeBridgeTomlConfig(configPathFromEnv(), cfg); err != nil {
		return resolvedPath, false, err
	}
	return resolvedPath, true, nil
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

	normalized := normalizeBridgeFileConfigForWrite(cfg)
	var buffer bytes.Buffer
	if err := toml.NewEncoder(&buffer).Encode(normalized); err != nil {
		return fmt.Errorf("encode config file: %w", err)
	}
	if err := os.WriteFile(resolvedPath, buffer.Bytes(), 0o600); err != nil {
		return fmt.Errorf("write config file %s: %w", resolvedPath, err)
	}
	return nil
}
