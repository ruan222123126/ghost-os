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
	errModelSelectionDisabled  = errors.New("model selection is disabled by tool_allowlist_only")
)

// ConfigStore 管理 bridge 运行态可变配置，避免直接写入进程环境变量。
type ConfigStore struct {
	mu      sync.RWMutex
	runtime runtimeConfig
}

// NewConfigStoreFromEnv 用配置文件 + 环境变量回退初始化可热更新配置存储。
func NewConfigStoreFromEnv() (*ConfigStore, error) {
	runtime, err := runtimeConfigFromEnv()
	if err != nil {
		return nil, err
	}
	return &ConfigStore{runtime: runtime}, nil
}

// RuntimeConfig 返回当前运行态配置快照（值语义）。
func (s *ConfigStore) RuntimeConfig() runtimeConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.runtime
}

// Snapshot 返回可直接暴露给前端的安全配置视图。
func (s *ConfigStore) Snapshot() configResponse {
	runtime := s.RuntimeConfig()
	return configResponse{
		Provider:                 activeProviderLabel(runtime),
		ProviderType:             string(runtime.Provider),
		BaseURL:                  runtime.BaseURL,
		Model:                    runtime.Model,
		ChatPath:                 runtime.ChatPath,
		APIKeySet:                runtime.APIKey != "",
		ModelSelectionEnabled:    runtime.ModelSelectionEnabled,
		GraphQLDefaultSource:     runtime.GraphQL.DefaultSource,
		GraphQLSources:           graphQLSourceSnapshots(runtime.GraphQL.Sources),
		WebSearchTavilyAPIKeySet: runtime.WebSearchTavilyAPIKey != "",
		WebSearchExaAPIKeySet:    runtime.WebSearchExaAPIKey != "",
	}
}

func (s *ConfigStore) ListProviders() []providerConfig {
	s.mu.Lock()
	defer s.mu.Unlock()

	fileCfg, _, _, err := s.loadMutationStateLocked()
	if err != nil {
		return nil
	}
	return cloneProviderConfigs(normalizeProviderConfigs(fileCfg.Providers, stringValue(fileCfg.Model)))
}

func graphQLSourceSnapshots(raw []GraphQLSourceConfig) []GraphQLSourceSnapshot {
	if len(raw) == 0 {
		return nil
	}

	out := make([]GraphQLSourceSnapshot, 0, len(raw))
	for _, source := range raw {
		out = append(out, GraphQLSourceSnapshot{
			Name:             source.Name,
			Description:      source.Description,
			Endpoint:         source.Endpoint,
			SchemaPath:       source.SchemaPath,
			TimeoutMS:        source.TimeoutMS,
			MaxResponseBytes: source.MaxResponseBytes,
			MaxDepth:         source.MaxDepth,
			MaxFields:        source.MaxFields,
			MaxRootFields:    source.MaxRootFields,
			MaxFragments:     source.MaxFragments,
			Headers:          cloneStringMap(source.Headers),
			APIKeySet:        source.APIKey != "",
			Domains:          graphQLDomainSnapshots(source.Domains),
		})
	}
	return out
}

func graphQLDomainSnapshots(raw []GraphQLDomainConfig) []GraphQLDomainSnapshot {
	if len(raw) == 0 {
		return nil
	}

	out := make([]GraphQLDomainSnapshot, 0, len(raw))
	for _, domain := range raw {
		out = append(out, GraphQLDomainSnapshot{
			Name:          domain.Name,
			Description:   domain.Description,
			RootQueries:   append([]string(nil), domain.RootQueries...),
			Types:         append([]string(nil), domain.Types...),
			MaxDepth:      domain.MaxDepth,
			MaxFields:     domain.MaxFields,
			MaxRootFields: domain.MaxRootFields,
		})
	}
	return out
}
