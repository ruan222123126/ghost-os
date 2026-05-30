package config

import (
	"errors"
	"sync"
)

var (
	errProviderNameRequired    = errors.New("provider name is required")
	errProviderTypeInvalid     = errors.New("provider type must be one of: openai|anthropic|custom|codex")
	errProviderBaseURLRequired = errors.New("provider base_url is required")
	errProviderNotFound        = errors.New("provider not found")
	errProviderExists          = errors.New("provider already exists")
	errModelSelectionDisabled  = errors.New("model selection is disabled")
	errToolNameRequired        = errors.New("tool name is required")
	errToolNotFound            = errors.New("tool not found")
	errToolUpdateEmpty         = errors.New("at least one of enabled, prompt_override, or sandbox_memory_mb is required")
	errToolConfigInvalid       = errors.New("tool config is invalid")
	errSkillIDRequired         = errors.New("skill id is required")
)

// store 管理 bridge 运行态可变配置，避免直接写入进程环境变量。
type store struct {
	mu      sync.RWMutex
	runtime runtimeConfig
}

// newStoreFromEnv 用配置文件 + 环境变量回退初始化可热更新配置存储。
func newStoreFromEnv() (*store, error) {
	env := currentEnv()
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return nil, err
	}
	runtime, err := resolveRuntimeConfig(fileCfg, env)
	if err != nil {
		return nil, err
	}
	return &store{runtime: runtime}, nil
}

func (s *store) Config() (Config, error) {
	return loadConfigWithRuntime(s.runtimeSnapshot())
}

func (s *store) runtimeSnapshot() runtimeConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneRuntimeConfig(s.runtime)
}

func (s *store) RuntimeConfig() runtimeConfig {
	return s.runtimeSnapshot()
}

// Snapshot 返回可直接暴露给前端的安全配置视图。
func (s *store) Snapshot() Snapshot {
	return snapshotFromRuntimeConfig(s.runtimeSnapshot())
}

func (s *store) ListProviders() ([]ProviderRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	fileCfg, _, err := s.loadStoredFileConfigLocked()
	if err != nil {
		return nil, err
	}
	return providerRecordsFromConfigs(normalizeProviderConfigs(fileCfg.Providers, stringValue(fileCfg.Model))), nil
}

func (s *store) SystemPrompts() (SystemPromptFiles, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	fileCfg, _, err := s.loadStoredFileConfigLocked()
	if err != nil {
		return SystemPromptFiles{}, err
	}
	promptsDir, err := resolvePromptsDir(fileCfg, currentEnv())
	if err != nil {
		return SystemPromptFiles{}, err
	}
	return LoadSystemPromptFiles(promptsDir)
}
