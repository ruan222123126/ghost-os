package orchestration

import (
	"strings"

	bridgeconfig "ghost-os/bridge/config"
)

type Config = bridgeconfig.Config
type ProviderConfig = bridgeconfig.ProviderConfig
type RSSConfig = bridgeconfig.RSSConfig
type WorkerConfig = bridgeconfig.WorkerConfig
type GraphQLConfig = bridgeconfig.GraphQLConfig
type ToolSelectorConfig = bridgeconfig.ToolSelectorConfig
type ToolSearchConfig = bridgeconfig.ToolSearchConfig
type MemoryAugmentationConfig = bridgeconfig.MemoryAugmentationConfig
type TaskConfig = bridgeconfig.TaskConfig
type providerConfig = bridgeconfig.ProviderRecord

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
	defaultToolSearchIdleTurns    = bridgeconfig.DefaultToolSearchIdleTurns
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
	inner bridgeconfig.Store
}

func (s *ConfigStore) unwrap() bridgeconfig.Store {
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

func (s *ConfigStore) Inner() bridgeconfig.Store {
	return s.unwrap()
}

func (s *ConfigStore) Config() (Config, error) {
	return s.unwrap().Config()
}

func (s *ConfigStore) Snapshot() configResponse {
	return configResponseFromSnapshot(s.unwrap().Snapshot())
}

func (s *ConfigStore) ListProviders() ([]providerConfig, error) {
	return s.unwrap().ListProviders()
}

func (s *ConfigStore) AddProvider(cfg providerConfig) error {
	return s.unwrap().AddProvider(cfg)
}

func (s *ConfigStore) UpdateProvider(name string, cfg providerConfig) error {
	return s.unwrap().UpdateProvider(name, cfg)
}

func (s *ConfigStore) DeleteProvider(name string) error {
	return s.unwrap().DeleteProvider(name)
}

func (s *ConfigStore) SetActiveProvider(name string) error {
	return s.unwrap().SetActiveProvider(name)
}

func (s *ConfigStore) Update(req configUpdateRequest) error {
	return s.unwrap().Update(bridgeconfig.UpdateRequest{
		Provider:                   req.Provider,
		APIKey:                     req.APIKey,
		BaseURL:                    req.BaseURL,
		Model:                      req.Model,
		ChatPath:                   req.ChatPath,
		GraphQLDefaultSource:       req.GraphqlDefaultSource,
		GraphQLToolRuntimeEnabled:  req.GraphqlToolRuntimeEnabled,
		GraphQLTextSanitizeEnabled: req.GraphqlTextSanitizeEnabled,
		GraphQLSources:             graphQLSourceInputs(req.GraphqlSources),
		GraphQLSourceUpsert:        graphQLSourceInputPointer(req.GraphqlSourceUpsert),
		GraphQLMutationPolicies:    graphQLMutationPolicyInputs(req.GraphqlMutationPolicies),
		WebRooterEnabled:           req.WebRooterEnabled,
		WebRooterBaseURL:           req.WebRooterBaseURL,
		WebRooterAPIToken:          req.WebRooterAPIToken,
		WebRooterTimeoutMS:         req.WebRooterTimeoutMs,
		WebSearchTavilyURL:         req.WebSearchTavilyURL,
		WebSearchExaURL:            req.WebSearchExaURL,
		WebSearchTavilyAPIKey:      req.WebSearchTavilyAPIKey,
		WebSearchExaAPIKey:         req.WebSearchExaAPIKey,
		TraceID:                    req.TraceID,
	})
}

func (s *ConfigStore) SetProjectRoot(path string) error {
	return s.unwrap().SetProjectRoot(path)
}

func cloneOptionalStringPointer(raw *string) *string {
	if raw == nil {
		return nil
	}
	value := strings.TrimSpace(*raw)
	if value == "" {
		return nil
	}
	return &value
}

func stringValue(raw *string) string {
	if raw == nil {
		return ""
	}
	return strings.TrimSpace(*raw)
}

func cloneStringMap(raw map[string]string) map[string]string {
	if len(raw) == 0 {
		return nil
	}
	out := make(map[string]string, len(raw))
	for key, value := range raw {
		out[key] = value
	}
	return out
}

func cloneModelTokenOverrides(raw map[string]int) map[string]int {
	if len(raw) == 0 {
		return nil
	}
	out := make(map[string]int, len(raw))
	for key, value := range raw {
		out[key] = value
	}
	return out
}

func graphQLSourceInputs(raw []graphqlSourceInput) []bridgeconfig.GraphQLSourceInput {
	if raw == nil {
		return nil
	}

	out := make([]bridgeconfig.GraphQLSourceInput, 0, len(raw))
	for _, source := range raw {
		out = append(out, bridgeconfig.GraphQLSourceInput{
			Name:             source.Name,
			Description:      source.Description,
			Endpoint:         source.Endpoint,
			APIKey:           source.APIKey,
			SchemaPath:       source.SchemaPath,
			TimeoutMS:        source.TimeoutMs,
			MaxResponseBytes: source.MaxResponseBytes,
			Headers:          cloneStringMap(source.Headers),
			MaxDepth:         source.MaxDepth,
			MaxFields:        source.MaxFields,
			MaxRootFields:    source.MaxRootFields,
			MaxFragments:     source.MaxFragments,
			Domains:          graphQLDomainInputs(source.Domains),
		})
	}
	return out
}

func graphQLSourceInputPointer(raw graphqlSourceInput) *bridgeconfig.GraphQLSourceInput {
	if raw.Name == "" {
		return nil
	}
	value := bridgeconfig.GraphQLSourceInput{
		Name:             raw.Name,
		Description:      raw.Description,
		Endpoint:         raw.Endpoint,
		APIKey:           raw.APIKey,
		SchemaPath:       raw.SchemaPath,
		TimeoutMS:        raw.TimeoutMs,
		MaxResponseBytes: raw.MaxResponseBytes,
		Headers:          cloneStringMap(raw.Headers),
		MaxDepth:         raw.MaxDepth,
		MaxFields:        raw.MaxFields,
		MaxRootFields:    raw.MaxRootFields,
		MaxFragments:     raw.MaxFragments,
		Domains:          graphQLDomainInputs(raw.Domains),
	}
	return &value
}

func graphQLDomainInputs(raw []graphqlDomainInput) []bridgeconfig.GraphQLDomainInput {
	if len(raw) == 0 {
		return nil
	}

	out := make([]bridgeconfig.GraphQLDomainInput, 0, len(raw))
	for _, domain := range raw {
		out = append(out, bridgeconfig.GraphQLDomainInput{
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

func graphQLMutationPolicyInputs(
	raw []graphqlMutationPolicyInput,
) []bridgeconfig.GraphQLMutationPolicyInput {
	if len(raw) == 0 {
		return nil
	}

	out := make([]bridgeconfig.GraphQLMutationPolicyInput, 0, len(raw))
	for _, policy := range raw {
		out = append(out, bridgeconfig.GraphQLMutationPolicyInput{
			Name:                    policy.Name,
			Description:             policy.Description,
			Source:                  policy.Source,
			Domain:                  policy.Domain,
			RootMutation:            policy.RootMutation,
			ApprovalRequired:        policy.ApprovalRequired,
			IdempotencyMode:         policy.IdempotencyMode,
			IdempotencyHeader:       policy.IdempotencyHeader,
			IdempotencyVariablePath: policy.IdempotencyVariablePath,
			MaxDepth:                policy.MaxDepth,
			MaxFields:               policy.MaxFields,
			MaxRootFields:           policy.MaxRootFields,
			MaxFragments:            policy.MaxFragments,
		})
	}
	return out
}
