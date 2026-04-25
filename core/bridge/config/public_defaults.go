package config

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
	DefaultToolSelectorTimeoutMS   = defaultToolSelectorTimeoutMS
	DefaultToolSelectorConfidence  = defaultToolSelectorConfidence
	DefaultToolSelectorRecentMsgs  = defaultToolSelectorRecentMsgs
	DefaultToolSearchIdleTurns     = defaultToolSearchIdleTurns
)

var (
	ErrProviderNameRequired    = errProviderNameRequired
	ErrProviderTypeInvalid     = errProviderTypeInvalid
	ErrProviderBaseURLRequired = errProviderBaseURLRequired
	ErrProviderNotFound        = errProviderNotFound
	ErrProviderExists          = errProviderExists
	ErrModelSelectionDisabled  = errModelSelectionDisabled
	ErrToolNameRequired        = errToolNameRequired
	ErrToolNotFound            = errToolNotFound
	ErrToolUpdateEmpty         = errToolUpdateEmpty
	ErrSystemPromptUpdateEmpty = errSystemPromptUpdateEmpty
)
