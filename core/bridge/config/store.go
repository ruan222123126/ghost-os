package config

import (
	"errors"
	"strings"
	"sync"

	"ghost-os/bridge/config/internal/providers"
	configruntime "ghost-os/bridge/config/internal/runtime"
	"ghost-os/bridge/llm"
)

var (
	errProviderNameRequired    = providers.ErrNameRequired
	errProviderTypeInvalid     = providers.ErrTypeInvalid
	errProviderBaseURLRequired = providers.ErrBaseURLRequired
	errProviderNotFound        = providers.ErrNotFound
	errProviderExists          = providers.ErrExists
	errModelSelectionDisabled  = configruntime.ErrModelSelectionDisabled
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

// PublicSnapshot 返回给配置 UI/CLI 的可编辑配置视图。
func (s *store) PublicSnapshot() (Snapshot, error) {
	s.mu.RLock()
	runtime := cloneRuntimeConfig(s.runtime)
	fileCfg, _, err := s.loadStoredFileConfigLocked()
	s.mu.RUnlock()
	if err != nil {
		return Snapshot{}, err
	}

	snapshot := snapshotFromRuntimeConfig(runtime)
	snapshot.ChatPath = publicChatPathValue(runtime, fileCfg)
	snapshot.ProjectRoot = publicProjectRootValue(fileCfg)
	return snapshot, nil
}

func publicChatPathValue(runtime runtimeConfig, fileCfg bridgeFileConfig) string {
	if chatPath := stringValue(fileCfg.ChatPath); chatPath != "" {
		return chatPath
	}
	if runtime.ChatPath == defaultChatPathForProvider(runtime.Provider) {
		return ""
	}
	return runtime.ChatPath
}

func publicProjectRootValue(fileCfg bridgeFileConfig) string {
	return stringValue(fileCfg.ProjectRoot)
}

func defaultChatPathForProvider(provider llm.Provider) string {
	switch provider.Normalized() {
	case llm.ProviderAnthropic:
		return "/v1/messages"
	case llm.ProviderCodex:
		return "/responses"
	default:
		return "/chat/completions"
	}
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

type projectRootOverrideStore struct {
	Store
	projectRoot string
}

func WithProjectRootOverride(store Store, projectRoot string) Store {
	if store == nil {
		return nil
	}
	trimmed := strings.TrimSpace(projectRoot)
	if trimmed == "" {
		return store
	}
	return projectRootOverrideStore{
		Store:       store,
		projectRoot: trimmed,
	}
}

func (s projectRootOverrideStore) Config() (Config, error) {
	cfg, err := s.Store.Config()
	if err != nil {
		return Config{}, err
	}
	cfg.ProjectRoot = s.projectRoot
	return cfg, nil
}
