package config

import (
	"time"

	"ghost-os/bridge/llm"
)

type Store = ConfigStore
type RuntimeConfig = runtimeConfig
type ProviderRecord = providerConfig
type ProviderFileConfig = providerFileConfig
type FileConfig = bridgeFileConfig
type MemoryAugmentationSettings = MemoryAugmentationConfig

type Snapshot struct {
	Provider                 string                          `json:"provider"`
	ProviderType             string                          `json:"provider_type"`
	BaseURL                  string                          `json:"base_url"`
	Model                    string                          `json:"model"`
	ChatPath                 string                          `json:"chat_path"`
	APIKeySet                bool                            `json:"api_key_set"`
	ModelSelectionEnabled    bool                            `json:"model_selection_enabled"`
	GraphQLDefaultSource     string                          `json:"graphql_default_source"`
	GraphQLSources           []GraphQLSourceSnapshot         `json:"graphql_sources"`
	GraphQLMutationPolicies  []GraphQLMutationPolicySnapshot `json:"graphql_mutation_policies"`
	WebSearchTavilyAPIKeySet bool                            `json:"web_search_tavily_api_key_set"`
	WebSearchExaAPIKeySet    bool                            `json:"web_search_exa_api_key_set"`
}

type UpdateRequest struct {
	Provider                *string                      `json:"provider,omitempty"`
	APIKey                  *string                      `json:"api_key,omitempty"`
	BaseURL                 *string                      `json:"base_url,omitempty"`
	Model                   *string                      `json:"model,omitempty"`
	ChatPath                *string                      `json:"chat_path,omitempty"`
	GraphQLDefaultSource    *string                      `json:"graphql_default_source,omitempty"`
	GraphQLSources          []GraphQLSourceInput         `json:"graphql_sources,omitempty"`
	GraphQLSourceUpsert     *GraphQLSourceInput          `json:"graphql_source_upsert,omitempty"`
	GraphQLMutationPolicies []GraphQLMutationPolicyInput `json:"graphql_mutation_policies,omitempty"`
	WebSearchTavilyAPIKey   *string                      `json:"web_search_tavily_api_key,omitempty"`
	WebSearchExaAPIKey      *string                      `json:"web_search_exa_api_key,omitempty"`
	TraceID                 string                       `json:"trace_id,omitempty"`
}

type configResponse = Snapshot
type configUpdateRequest = UpdateRequest

const (
	DefaultProvider                = defaultProvider
	DefaultBaseURL                 = defaultBaseURL
	DefaultAnthropicBaseURL        = defaultAnthropicBaseURL
	DefaultModel                   = defaultModel
	DefaultPromptsPath             = defaultPromptsPath
	DefaultPromptsDir              = defaultPromptsDir
	DefaultSessionsPath            = defaultSessionsPath
	DefaultRSSFeedsPath            = defaultRSSFeedsPath
	DefaultRSSInboxPath            = defaultRSSInboxPath
	DefaultRSSBriefingsPath        = defaultRSSBriefingsPath
	DefaultRSSReportsPath          = defaultRSSReportsPath
	DefaultRSSPollInterval         = defaultRSSPollInterval
	DefaultRSSPollMaxItemsPerFeed  = defaultRSSPollMaxItemsPerFeed
	DefaultRSSAIBatchSize          = defaultRSSAIBatchSize
	DefaultRSSBriefingInterval     = defaultRSSBriefingInterval
	DefaultTasksPath               = defaultTasksPath
	DefaultAnthropicVersion        = defaultAnthropicVersion
	DefaultAnthropicMaxTokens      = defaultAnthropicMaxTokens
	DefaultProMaxIterations        = defaultProMaxIterations
	DefaultMaxTurns                = defaultMaxTurns
	DefaultWorkerMaxConcurrency    = defaultWorkerMaxConcurrency
	DefaultWorkerMaxFiles          = defaultWorkerMaxFiles
	DefaultWorkerMaxFileChunks     = defaultWorkerMaxFileChunks
	DefaultGraphQLTimeoutMS        = defaultGraphQLTimeoutMS
	DefaultGraphQLMaxResponseBytes = defaultGraphQLMaxResponseBytes
	DefaultGraphQLMaxDepth         = defaultGraphQLMaxDepth
	DefaultGraphQLMaxFields        = defaultGraphQLMaxFields
	DefaultGraphQLMaxRootFields    = defaultGraphQLMaxRootFields
	DefaultGraphQLMaxFragments     = defaultGraphQLMaxFragments
	DefaultGraphQLLegacySourceName = defaultGraphQLLegacySourceName
	DefaultToolSelectorTimeoutMS   = defaultToolSelectorTimeoutMS
	DefaultToolSelectorConfidence  = defaultToolSelectorConfidence
	DefaultToolSelectorRecentMsgs  = defaultToolSelectorRecentMsgs
	DefaultToolSearchIdleTurns     = defaultToolSearchIdleTurns
	DefaultMemoryRecallItems       = defaultMemoryRecallItems
	DefaultMemoryMinConfidence     = defaultMemoryMinConfidence
	DefaultMemoryUserScopeID       = defaultMemoryUserScopeID
)

var (
	ErrProviderNameRequired    = errProviderNameRequired
	ErrProviderTypeInvalid     = errProviderTypeInvalid
	ErrProviderBaseURLRequired = errProviderBaseURLRequired
	ErrProviderNotFound        = errProviderNotFound
	ErrProviderExists          = errProviderExists
	ErrModelSelectionDisabled  = errModelSelectionDisabled
)

func Load() (Config, error) {
	return LoadConfig()
}

func LoadWithRuntime(runtime RuntimeConfig) (Config, error) {
	return loadConfigWithRuntime(runtime)
}

func NewStoreFromEnv() (*ConfigStore, error) {
	return NewConfigStoreFromEnv()
}

func GetenvDefault(name, fallback string) string {
	return getenvDefault(name, fallback)
}

func ParseStringCSV(raw string) []string {
	return parseStringCSV(raw)
}

func RuntimeConfigFromEnv() (RuntimeConfig, error) {
	return runtimeConfigFromEnv()
}

func ProviderClientOptions(cfg Config, model string) llm.ClientOptions {
	return providerClientOptions(cfg, model)
}

func ValueOrEnv(raw *string, envName, fallback string) string {
	return valueOrEnv(raw, envName, fallback)
}

func EffectiveWorkerModel(cfg Config) string {
	if model := cfg.Worker.Model; model != "" {
		return model
	}
	return cfg.Provider.Model
}

func SessionsPathFromEnv() string {
	return sessionsPathFromEnv()
}

func RSSFeedsPathFromEnv() string {
	return rssFeedsPathFromEnv()
}

func RSSInboxPathFromEnv() string {
	return rssInboxPathFromEnv()
}

func RSSBriefingsPathFromEnv() string {
	return rssBriefingsPathFromEnv()
}

func RSSReportsPathFromEnv() string {
	return rssReportsPathFromEnv()
}

func RSSPollEnabledFromEnv() bool {
	return rssPollEnabledFromEnv()
}

func RSSPollIntervalFromEnv() time.Duration {
	return rssPollIntervalFromEnv()
}

func RSSPollMaxItemsPerFeedFromEnv() int {
	return rssPollMaxItemsPerFeedFromEnv()
}

func RSSAIBatchSizeFromEnv() int {
	return rssAIBatchSizeFromEnv()
}

func RSSBriefingEnabledFromEnv() bool {
	return rssBriefingEnabledFromEnv()
}

func RSSBriefingIntervalFromEnv() time.Duration {
	return rssBriefingIntervalFromEnv()
}

func CORSOriginsOrEnv(raw []string) []string {
	return corsOriginsOrEnv(raw)
}

func WebSearchTavilyAPIKeyFromEnv() string {
	return webSearchTavilyAPIKeyFromEnv()
}

func TasksPathFromEnv() string {
	return tasksPathFromEnv()
}

func NativeBinaryPathFromEnv() string {
	return nativeBinaryPathFromEnv()
}

func NativeBinaryRootsFromEnv() []string {
	return nativeBinaryRootsFromEnv()
}

func NativeBinaryCandidatesFromEnv() []string {
	return nativeBinaryCandidatesFromEnv()
}

func NativeAllowedReadPathsFromEnv() []string {
	return nativeAllowedReadPathsFromEnv()
}

func NativeAllowedWritePathsFromEnv() []string {
	return nativeAllowedWritePathsFromEnv()
}

func ProjectRootFromEnv() string {
	return projectRootFromEnv()
}

func NativePersistentEnabledFromEnv() bool {
	return nativePersistentEnabledFromEnv()
}

func ToolNameListOrEnv(raw []string, envName string) []string {
	return toolNameListOrEnv(raw, envName)
}

func NormalizeConfiguredToolLists(allowlist []string, blocklist []string) ([]string, []string, error) {
	return normalizeConfiguredToolLists(allowlist, blocklist)
}

func DefaultBaseURLForProvider(provider llm.Provider) string {
	return defaultBaseURLForProvider(provider)
}

func ResolveUserPath(pathValue string) (string, error) {
	return resolveUserPath(pathValue)
}

func ConfigPathFromEnv() string {
	return configPathFromEnv()
}

func ActiveProviderLabel(runtime RuntimeConfig) string {
	return activeProviderLabel(runtime)
}

func CloneOptionalStringPointer(raw *string) *string {
	return cloneOptionalStringPointer(raw)
}

func OptionalStringPointer(raw string) *string {
	return optionalStringPointer(raw)
}

func StringPointer(raw string) *string {
	return stringPointer(raw)
}

func StringValue(raw *string) string {
	return stringValue(raw)
}

func LoadBridgeFileConfig() (FileConfig, string, error) {
	return loadBridgeFileConfig()
}

func WriteBridgeFileConfig(path string, cfg FileConfig) error {
	return writeBridgeFileConfig(path, bridgeFileConfig(cfg))
}

func MigrateBridgeFileConfigIfLegacy() (string, bool, error) {
	return migrateBridgeFileConfigIfLegacy()
}

func NormalizeProviderConfigs(raw map[string]ProviderFileConfig, model string) []ProviderRecord {
	return normalizeProviderConfigs(raw, model)
}
