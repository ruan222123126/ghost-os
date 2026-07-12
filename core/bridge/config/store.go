package config

import (
	"errors"
	"strings"
	"sync"

	"ghost-os/bridge/config/internal/providers"
	configruntime "ghost-os/bridge/config/internal/runtime"
	"ghost-os/bridge/llm"
	"ghost-os/bridge/taskdefs"
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

func (s *store) WithRuntimeOverrides(providerName string, model string) (Store, error) {
	req := taskdefs.TaskRuntimeOverrides{}
	if trimmed := strings.TrimSpace(providerName); trimmed != "" {
		req.ProviderName = trimmed
	}
	if trimmed := strings.TrimSpace(model); trimmed != "" {
		req.Model = trimmed
	}
	return withRuntimeOverridesStore{Store: s, overrides: &req}, nil
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
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.listProvidersLocked(false)
}

func (s *store) ListProviderSyncRecords() ([]ProviderRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.listProvidersLocked(true)
}

func (s *store) GetProvider(name string) (ProviderRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	providers, err := s.listProvidersLocked(false)
	if err != nil {
		return ProviderRecord{}, err
	}
	target := strings.TrimSpace(name)
	if target == "" {
		return ProviderRecord{}, ErrProviderNotFound
	}
	for _, provider := range providers {
		if strings.EqualFold(strings.TrimSpace(provider.Name), target) {
			return provider, nil
		}
	}
	return ProviderRecord{}, ErrProviderNotFound
}

func (s *store) GetProviderByID(providerID string) (ProviderRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	providers, err := s.listProvidersLocked(false)
	if err != nil {
		return ProviderRecord{}, err
	}
	target := strings.TrimSpace(providerID)
	if target == "" {
		return ProviderRecord{}, ErrProviderNotFound
	}
	for _, provider := range providers {
		if strings.TrimSpace(provider.ProviderID) == target {
			return provider, nil
		}
	}
	return ProviderRecord{}, ErrProviderNotFound
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

type withRuntimeOverridesStore struct {
	Store
	overrides *taskdefs.TaskRuntimeOverrides
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

func (s withRuntimeOverridesStore) Config() (Config, error) {
	if s.Store == nil {
		return Config{}, errors.New("config store is nil")
	}
	cfg, err := s.Store.Config()
	if err != nil {
		return Config{}, err
	}
	normalized, err := appRuntimeOverridesToConfig(cfg, s.overrides, s.Store)
	if err != nil {
		return Config{}, err
	}
	return normalized, nil
}

func appRuntimeOverridesToConfig(
	cfg Config,
	overrides *taskdefs.TaskRuntimeOverrides,
	store Store,
) (Config, error) {
	if overrides == nil {
		return cfg, nil
	}
	if providerName := strings.TrimSpace(overrides.ProviderName); providerName != "" {
		providers, err := store.ListProviders()
		if err != nil {
			return Config{}, err
		}
		index := -1
		for candidateIndex, provider := range providers {
			if strings.EqualFold(strings.TrimSpace(provider.Name), providerName) {
				index = candidateIndex
				break
			}
		}
		if index >= 0 {
			provider := providers[index]
			cfg.Provider.Type = provider.Type
			cfg.Provider.APIKey = stringValue(provider.APIKey)
			cfg.Provider.BaseURL = provider.BaseURL
			cfg.Provider.ContextWindowTokens = provider.ContextWindowTokens
			cfg.Provider.ResponseReserveTokens = provider.ResponseReserveTokens
			cfg.Provider.ModelContextWindowTokens = cloneModelTokenOverrides(provider.ModelContextWindowTokens)
			cfg.Provider.ModelResponseReserveTokens = cloneModelTokenOverrides(provider.ModelResponseReserveTokens)
		}
	}
	if model := strings.TrimSpace(overrides.Model); model != "" {
		cfg.Provider.Model = model
	}
	if overrides.MaxTurns != nil {
		cfg.MaxTurns = *overrides.MaxTurns
	}
	return cfg, nil
}
