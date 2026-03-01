package app

import (
	"fmt"
	"strings"
	"sync"
)

// ConfigStore 管理 bridge 运行态可变配置，避免直接写入进程环境变量。
type ConfigStore struct {
	mu      sync.RWMutex
	runtime runtimeConfig
}

// NewConfigStoreFromEnv 用环境变量初始化可热更新配置存储。
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
		Provider:  string(runtime.Provider),
		BaseURL:   runtime.BaseURL,
		Model:     runtime.Model,
		ChatPath:  runtime.ChatPath,
		APIKeySet: runtime.APIKey != "",
	}
}

// Update 仅覆盖请求中显式给出的字段，并在落库前执行归一化。
func (s *ConfigStore) Update(req configUpdateRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	next := s.runtime
	if req.Provider != nil {
		provider := normalizeProvider(*req.Provider)
		if !provider.Valid() {
			return fmt.Errorf("invalid provider %q, expected one of: openai|anthropic|custom", strings.TrimSpace(*req.Provider))
		}
		next.Provider = provider
	}
	if req.APIKey != nil {
		next.APIKey = strings.TrimSpace(*req.APIKey)
	}
	if req.BaseURL != nil {
		next.BaseURL = strings.TrimSpace(*req.BaseURL)
	}
	if req.Model != nil {
		next.Model = strings.TrimSpace(*req.Model)
	}
	if req.ChatPath != nil {
		next.ChatPath = strings.TrimSpace(*req.ChatPath)
	}

	s.runtime = normalizeRuntimeConfig(next)
	return nil
}
