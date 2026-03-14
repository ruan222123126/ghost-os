package orchestration

import (
	"time"

	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/llm"
)

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

func LoadConfig() (Config, error) {
	return bridgeconfig.Load()
}

func getenvDefault(name, fallback string) string {
	return bridgeconfig.GetenvDefault(name, fallback)
}

func parseStringCSV(raw string) []string {
	return bridgeconfig.ParseStringCSV(raw)
}

func runtimeConfigFromEnv() (runtimeConfig, error) {
	return bridgeconfig.RuntimeConfigFromEnv()
}

func loadConfigWithRuntime(runtime runtimeConfig) (Config, error) {
	return bridgeconfig.LoadWithRuntime(runtime)
}

func providerClientOptions(cfg Config, model string) llm.ClientOptions {
	return bridgeconfig.ProviderClientOptions(cfg, model)
}

func valueOrEnv(raw *string, envName, fallback string) string {
	return bridgeconfig.ValueOrEnv(raw, envName, fallback)
}

func sessionsPathFromEnv() string {
	return bridgeconfig.SessionsPathFromEnv()
}

func rssFeedsPathFromEnv() string {
	return bridgeconfig.RSSFeedsPathFromEnv()
}

func rssInboxPathFromEnv() string {
	return bridgeconfig.RSSInboxPathFromEnv()
}

func rssBriefingsPathFromEnv() string {
	return bridgeconfig.RSSBriefingsPathFromEnv()
}

func rssReportsPathFromEnv() string {
	return bridgeconfig.RSSReportsPathFromEnv()
}

func rssPollEnabledFromEnv() bool {
	return bridgeconfig.RSSPollEnabledFromEnv()
}

func rssPollIntervalFromEnv() time.Duration {
	return bridgeconfig.RSSPollIntervalFromEnv()
}

func rssPollMaxItemsPerFeedFromEnv() int {
	return bridgeconfig.RSSPollMaxItemsPerFeedFromEnv()
}

func rssAIBatchSizeFromEnv() int {
	return bridgeconfig.RSSAIBatchSizeFromEnv()
}

func rssBriefingEnabledFromEnv() bool {
	return bridgeconfig.RSSBriefingEnabledFromEnv()
}

func rssBriefingIntervalFromEnv() time.Duration {
	return bridgeconfig.RSSBriefingIntervalFromEnv()
}

func corsOriginsOrEnv(raw []string) []string {
	return bridgeconfig.CORSOriginsOrEnv(raw)
}

func webSearchTavilyAPIKeyFromEnv() string {
	return bridgeconfig.WebSearchTavilyAPIKeyFromEnv()
}

func tasksPathFromEnv() string {
	return bridgeconfig.TasksPathFromEnv()
}

func nativePersistentEnabledFromEnv() bool {
	return bridgeconfig.NativePersistentEnabledFromEnv()
}

func nativeBinaryPathFromEnv() string {
	return bridgeconfig.NativeBinaryPathFromEnv()
}

func nativeBinaryRootsFromEnv() []string {
	return bridgeconfig.NativeBinaryRootsFromEnv()
}

func nativeBinaryCandidatesFromEnv() []string {
	return bridgeconfig.NativeBinaryCandidatesFromEnv()
}

func nativeAllowedReadPathsFromEnv() []string {
	return bridgeconfig.NativeAllowedReadPathsFromEnv()
}

func nativeAllowedWritePathsFromEnv() []string {
	return bridgeconfig.NativeAllowedWritePathsFromEnv()
}

func projectRootFromEnv() string {
	return bridgeconfig.ProjectRootFromEnv()
}

func defaultBaseURLForProvider(provider llm.Provider) string {
	return bridgeconfig.DefaultBaseURLForProvider(provider)
}

func resolveUserPath(pathValue string) (string, error) {
	return bridgeconfig.ResolveUserPath(pathValue)
}

func configPathFromEnv() string {
	return bridgeconfig.ConfigPathFromEnv()
}

func activeProviderLabel(runtime runtimeConfig) string {
	return bridgeconfig.ActiveProviderLabel(runtime)
}

func cloneOptionalStringPointer(raw *string) *string {
	return bridgeconfig.CloneOptionalStringPointer(raw)
}

func optionalStringPointer(raw string) *string {
	return bridgeconfig.OptionalStringPointer(raw)
}

func stringPointer(raw string) *string {
	return bridgeconfig.StringPointer(raw)
}

func stringValue(raw *string) string {
	return bridgeconfig.StringValue(raw)
}

func loadBridgeFileConfig() (bridgeFileConfig, string, error) {
	return bridgeconfig.LoadBridgeFileConfig()
}

func writeBridgeFileConfig(path string, cfg bridgeFileConfig) error {
	return bridgeconfig.WriteBridgeFileConfig(path, bridgeconfig.FileConfig(cfg))
}

func migrateBridgeFileConfigIfLegacy() (string, bool, error) {
	return bridgeconfig.MigrateBridgeFileConfigIfLegacy()
}

func normalizeProviderConfigs(raw map[string]providerFileConfig, model string) []providerConfig {
	providers := bridgeconfig.NormalizeProviderConfigs(raw, model)
	out := make([]providerConfig, 0, len(providers))
	for _, provider := range providers {
		out = append(out, providerConfig(provider))
	}
	return out
}
