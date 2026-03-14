package orchestration

import bridgeconfig "ghost-os/bridge/config"

type Config = bridgeconfig.Config
type ProviderConfig = bridgeconfig.ProviderConfig
type RSSConfig = bridgeconfig.RSSConfig
type WorkerConfig = bridgeconfig.WorkerConfig
type ToolSelectorConfig = bridgeconfig.ToolSelectorConfig
type MemoryAugmentationConfig = bridgeconfig.MemoryAugmentationConfig
type runtimeConfig = bridgeconfig.RuntimeConfig
type providerConfig = bridgeconfig.ProviderRecord
type providerFileConfig = bridgeconfig.ProviderFileConfig
type bridgeFileConfig = bridgeconfig.FileConfig

const (
	defaultProvider               = bridgeconfig.DefaultProvider
	defaultBaseURL                = bridgeconfig.DefaultBaseURL
	defaultAnthropicBaseURL       = bridgeconfig.DefaultAnthropicBaseURL
	defaultModel                  = bridgeconfig.DefaultModel
	defaultPromptsPath            = bridgeconfig.DefaultPromptsPath
	defaultPromptsDir             = bridgeconfig.DefaultPromptsDir
	defaultSessionsPath           = bridgeconfig.DefaultSessionsPath
	defaultRSSFeedsPath           = bridgeconfig.DefaultRSSFeedsPath
	defaultRSSInboxPath           = bridgeconfig.DefaultRSSInboxPath
	defaultRSSBriefingsPath       = bridgeconfig.DefaultRSSBriefingsPath
	defaultRSSReportsPath         = bridgeconfig.DefaultRSSReportsPath
	defaultRSSPollInterval        = bridgeconfig.DefaultRSSPollInterval
	defaultRSSPollMaxItemsPerFeed = bridgeconfig.DefaultRSSPollMaxItemsPerFeed
	defaultRSSAIBatchSize         = bridgeconfig.DefaultRSSAIBatchSize
	defaultRSSBriefingInterval    = bridgeconfig.DefaultRSSBriefingInterval
	defaultTasksPath              = bridgeconfig.DefaultTasksPath
	defaultAnthropicVersion       = bridgeconfig.DefaultAnthropicVersion
	defaultAnthropicMaxTokens     = bridgeconfig.DefaultAnthropicMaxTokens
	defaultProMaxIterations       = bridgeconfig.DefaultProMaxIterations
	defaultMaxTurns               = bridgeconfig.DefaultMaxTurns
	defaultWorkerMaxConcurrency   = bridgeconfig.DefaultWorkerMaxConcurrency
	defaultWorkerMaxFiles         = bridgeconfig.DefaultWorkerMaxFiles
	defaultWorkerMaxFileChunks    = bridgeconfig.DefaultWorkerMaxFileChunks
	defaultToolSelectorTimeoutMS  = bridgeconfig.DefaultToolSelectorTimeoutMS
	defaultToolSelectorConfidence = bridgeconfig.DefaultToolSelectorConfidence
	defaultToolSelectorRecentMsgs = bridgeconfig.DefaultToolSelectorRecentMsgs
	defaultMemoryRecallItems      = bridgeconfig.DefaultMemoryRecallItems
	defaultMemoryMinConfidence    = bridgeconfig.DefaultMemoryMinConfidence
	defaultMemoryUserScopeID      = bridgeconfig.DefaultMemoryUserScopeID
)

var (
	errProviderNameRequired    = bridgeconfig.ErrProviderNameRequired
	errProviderTypeInvalid     = bridgeconfig.ErrProviderTypeInvalid
	errProviderBaseURLRequired = bridgeconfig.ErrProviderBaseURLRequired
	errProviderNotFound        = bridgeconfig.ErrProviderNotFound
	errProviderExists          = bridgeconfig.ErrProviderExists
	errModelSelectionDisabled  = bridgeconfig.ErrModelSelectionDisabled
)

type ConfigStore struct {
	inner *bridgeconfig.Store
}

func (s *ConfigStore) unwrap() *bridgeconfig.Store {
	if s == nil || s.inner == nil {
		panic("config store is nil")
	}
	return s.inner
}

func NewConfigStoreFromEnv() (*ConfigStore, error) {
	inner, err := bridgeconfig.NewStoreFromEnv()
	if err != nil {
		return nil, err
	}
	return &ConfigStore{inner: inner}, nil
}

func (s *ConfigStore) Inner() *bridgeconfig.Store {
	return s.unwrap()
}

func (s *ConfigStore) RuntimeConfig() runtimeConfig {
	return s.unwrap().RuntimeConfig()
}

func (s *ConfigStore) Snapshot() configResponse {
	snapshot := s.unwrap().Snapshot()
	return configResponse{
		Provider:                 snapshot.Provider,
		ProviderType:             snapshot.ProviderType,
		BaseURL:                  snapshot.BaseURL,
		Model:                    snapshot.Model,
		ChatPath:                 snapshot.ChatPath,
		APIKeySet:                snapshot.APIKeySet,
		ModelSelectionEnabled:    snapshot.ModelSelectionEnabled,
		WebSearchTavilyAPIKeySet: snapshot.WebSearchTavilyAPIKeySet,
		WebSearchExaAPIKeySet:    snapshot.WebSearchExaAPIKeySet,
	}
}

func (s *ConfigStore) ListProviders() []providerConfig {
	providers := s.unwrap().ListProviders()
	out := make([]providerConfig, 0, len(providers))
	for _, provider := range providers {
		out = append(out, providerConfig(provider))
	}
	return out
}

func (s *ConfigStore) AddProvider(cfg providerConfig) error {
	return s.unwrap().AddProvider(bridgeconfig.ProviderRecord(cfg))
}

func (s *ConfigStore) UpdateProvider(name string, cfg providerConfig) error {
	return s.unwrap().UpdateProvider(name, bridgeconfig.ProviderRecord(cfg))
}

func (s *ConfigStore) DeleteProvider(name string) error {
	return s.unwrap().DeleteProvider(name)
}

func (s *ConfigStore) SetActiveProvider(name string) error {
	return s.unwrap().SetActiveProvider(name)
}

func (s *ConfigStore) Update(req configUpdateRequest) error {
	return s.unwrap().Update(bridgeconfig.UpdateRequest{
		Provider:              req.Provider,
		APIKey:                req.APIKey,
		BaseURL:               req.BaseURL,
		Model:                 req.Model,
		ChatPath:              req.ChatPath,
		WebSearchTavilyAPIKey: req.WebSearchTavilyAPIKey,
		WebSearchExaAPIKey:    req.WebSearchExaAPIKey,
		TraceID:               req.TraceID,
	})
}

func (s *ConfigStore) SetProjectRoot(path string) error {
	return s.unwrap().SetProjectRoot(path)
}

func cloneOptionalStringPointer(raw *string) *string {
	return bridgeconfig.CloneOptionalStringPointer(raw)
}

func stringValue(raw *string) string {
	return bridgeconfig.StringValue(raw)
}
