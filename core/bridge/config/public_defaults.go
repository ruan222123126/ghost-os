package config

const (
	DefaultProvider                     = defaultProvider
	DefaultBaseURL                      = defaultBaseURL
	DefaultAnthropicBaseURL             = defaultAnthropicBaseURL
	DefaultModel                        = defaultModel
	DefaultPromptsPath                  = defaultPromptsPath
	DefaultPromptsDir                   = defaultPromptsDir
	DefaultSessionsPath                 = defaultSessionsPath
	DefaultRSSFeedsPath                 = defaultRSSFeedsPath
	DefaultRSSInboxPath                 = defaultRSSInboxPath
	DefaultRSSBriefingsPath             = defaultRSSBriefingsPath
	DefaultRSSReportsPath               = defaultRSSReportsPath
	DefaultRSSPollInterval              = defaultRSSPollInterval
	DefaultRSSPollMaxItemsPerFeed       = defaultRSSPollMaxItemsPerFeed
	DefaultRSSAIBatchSize               = defaultRSSAIBatchSize
	DefaultRSSBriefingInterval          = defaultRSSBriefingInterval
	DefaultTasksPath                    = defaultTasksPath
	DefaultTaskExecutionTimeoutMS       = defaultTaskExecutionTimeoutMS
	DefaultAnthropicVersion             = defaultAnthropicVersion
	DefaultAnthropicMaxTokens           = defaultAnthropicMaxTokens
	DefaultProMaxIterations             = defaultProMaxIterations
	DefaultRelayStopPolicy              = defaultRelayStopPolicy
	DefaultRelayMaxRounds               = defaultRelayMaxRounds
	DefaultRelayExecutionTimeoutMS      = defaultRelayExecutionTimeoutMS
	DefaultMaxTurns                     = defaultMaxTurns
	DefaultLLMCompletionRetryCount      = defaultLLMCompletionRetryCount
	DefaultLLMCompletionRetryIntervalMS = defaultLLMCompletionRetryIntervalMS
	DefaultModelSelectionEnabled        = defaultModelSelectionEnabled
	DefaultWorkerMaxConcurrency         = defaultWorkerMaxConcurrency
	DefaultWorkerMaxFiles               = defaultWorkerMaxFiles
	DefaultWorkerMaxFileChunks          = defaultWorkerMaxFileChunks
	DefaultScriptExecSandboxMemoryMB    = defaultScriptExecSandboxMemoryMB
	DefaultSessionSystemPromptVisible   = defaultSessionSystemPromptVisible
	DefaultToolSelectorTimeoutMS        = defaultToolSelectorTimeoutMS
	DefaultToolSelectorConfidence       = defaultToolSelectorConfidence
	DefaultToolSelectorRecentMsgs       = defaultToolSelectorRecentMsgs
	DefaultToolSearchIdleTurns          = defaultToolSearchIdleTurns
	DefaultAssistantMarkdownEnabled     = defaultAssistantMarkdownEnabled
	DefaultToolCallCompactOutputEnabled = defaultToolCallCompactOutputEnabled
	DefaultMemoryModeEnabled            = defaultMemoryModeEnabled
	DefaultMicrocompactEnabled          = defaultMicrocompactEnabled
	DefaultSessionTitleMode             = defaultSessionTitleMode
)

var (
	ErrProviderNameRequired       = errProviderNameRequired
	ErrProviderTypeInvalid        = errProviderTypeInvalid
	ErrProviderBaseURLRequired    = errProviderBaseURLRequired
	ErrProviderNotFound           = errProviderNotFound
	ErrProviderExists             = errProviderExists
	ErrModelSelectionDisabled     = errModelSelectionDisabled
	ErrToolNameRequired           = errToolNameRequired
	ErrToolNotFound               = errToolNotFound
	ErrToolUpdateEmpty            = errToolUpdateEmpty
	ErrToolConfigInvalid          = errToolConfigInvalid
	ErrSystemPromptUpdateEmpty    = errSystemPromptUpdateEmpty
	ErrSystemPromptUpdateConflict = errSystemPromptUpdateConflict
	ErrSystemPromptLibraryInvalid = errSystemPromptLibraryInvalid
	ErrPresetInvalid              = errPresetInvalid
	ErrPresetIDRequired           = errPresetIDRequired
	ErrPresetNotFound             = errPresetNotFound
	ErrPresetUpdateEmpty          = errPresetUpdateEmpty
)
