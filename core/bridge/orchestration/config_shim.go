package orchestration

import bridgeconfig "ghost-os/bridge/config"

type Config = bridgeconfig.Config
type ProviderConfig = bridgeconfig.ProviderConfig
type RSSConfig = bridgeconfig.RSSConfig
type WorkerConfig = bridgeconfig.WorkerConfig
type ToolSelectorConfig = bridgeconfig.ToolSelectorConfig
type ToolSearchConfig = bridgeconfig.ToolSearchConfig
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
		GraphqlDefaultSource:     snapshot.GraphQLDefaultSource,
		GraphqlSources:           graphQLSourceResponses(snapshot.GraphQLSources),
		GraphqlMutationPolicies:  graphQLMutationPolicyResponses(snapshot.GraphQLMutationPolicies),
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
		Provider:                req.Provider,
		APIKey:                  req.APIKey,
		BaseURL:                 req.BaseURL,
		Model:                   req.Model,
		ChatPath:                req.ChatPath,
		GraphQLDefaultSource:    req.GraphqlDefaultSource,
		GraphQLSources:          graphQLSourceInputs(req.GraphqlSources),
		GraphQLSourceUpsert:     graphQLSourceInputPointer(req.GraphqlSourceUpsert),
		GraphQLMutationPolicies: graphQLMutationPolicyInputs(req.GraphqlMutationPolicies),
		WebSearchTavilyAPIKey:   req.WebSearchTavilyAPIKey,
		WebSearchExaAPIKey:      req.WebSearchExaAPIKey,
		TraceID:                 req.TraceID,
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

func graphQLSourceResponses(raw []bridgeconfig.GraphQLSourceSnapshot) []graphqlSourceResponse {
	if len(raw) == 0 {
		return []graphqlSourceResponse{}
	}

	out := make([]graphqlSourceResponse, 0, len(raw))
	for _, source := range raw {
		out = append(out, graphqlSourceResponse{
			Name:             source.Name,
			Description:      source.Description,
			Endpoint:         source.Endpoint,
			SchemaPath:       source.SchemaPath,
			TimeoutMs:        source.TimeoutMS,
			MaxResponseBytes: source.MaxResponseBytes,
			MaxDepth:         source.MaxDepth,
			MaxFields:        source.MaxFields,
			MaxRootFields:    source.MaxRootFields,
			MaxFragments:     source.MaxFragments,
			Headers:          cloneStringMap(source.Headers),
			APIKeySet:        source.APIKeySet,
			Domains:          graphQLDomainResponses(source.Domains),
		})
	}
	return out
}

func graphQLDomainResponses(raw []bridgeconfig.GraphQLDomainSnapshot) []graphqlDomainResponse {
	if len(raw) == 0 {
		return nil
	}

	out := make([]graphqlDomainResponse, 0, len(raw))
	for _, domain := range raw {
		out = append(out, graphqlDomainResponse{
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

func graphQLMutationPolicyResponses(
	raw []bridgeconfig.GraphQLMutationPolicySnapshot,
) []graphqlMutationPolicyResponse {
	if len(raw) == 0 {
		return []graphqlMutationPolicyResponse{}
	}

	out := make([]graphqlMutationPolicyResponse, 0, len(raw))
	for _, policy := range raw {
		out = append(out, graphqlMutationPolicyResponse{
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
