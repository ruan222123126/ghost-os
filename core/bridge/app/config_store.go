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

func NewConfigStoreFromEnv() (*ConfigStore, error) {
	runtime, err := runtimeConfigFromEnv()
	if err != nil {
		return nil, err
	}
	return &ConfigStore{runtime: runtime}, nil
}

func (s *ConfigStore) RuntimeConfig() runtimeConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.runtime
}

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
