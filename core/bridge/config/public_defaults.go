package config

import configruntime "ghost-os/bridge/config/internal/runtime"

const (
	defaultProvider                     = configruntime.DefaultProvider
	defaultBaseURL                      = configruntime.DefaultBaseURL
	defaultAnthropicBaseURL             = configruntime.DefaultAnthropicBaseURL
	defaultModel                        = configruntime.DefaultModel
	defaultPromptsPath                  = "prompts.yaml"
	defaultPromptsDir                   = "~/.ghost-os/prompts"
	defaultSessionsPath                 = "~/.ghost-os/sessions"
	defaultTasksPath                    = "~/.ghost-os/tasks"
	defaultTaskExecutionTimeoutMS       = configruntime.DefaultTaskExecutionTimeoutMS
	defaultAnthropicVersion             = "2023-06-01"
	defaultAnthropicMaxTokens           = 1024
	defaultRelayStopPolicy              = configruntime.DefaultRelayStopPolicy
	defaultRelayMaxRounds               = configruntime.DefaultRelayMaxRounds
	defaultRelayExecutionTimeoutMS      = configruntime.DefaultRelayExecutionTimeoutMS
	defaultExternalCodexPermissionMode  = configruntime.DefaultExternalCodexPermissionMode
	defaultMaxTurns                     = configruntime.DefaultMaxTurns
	defaultLLMCompletionRetryCount      = configruntime.DefaultLLMCompletionRetryCount
	defaultLLMCompletionRetryIntervalMS = configruntime.DefaultLLMCompletionRetryIntervalMS
	defaultModelSelectionEnabled        = configruntime.DefaultModelSelectionEnabled
	defaultWorkerMaxConcurrency         = 4
	defaultWorkerMaxFiles               = 20
	defaultWorkerMaxFileChunks          = 4
	defaultScriptExecSandboxMemoryMB    = 256
	maxScriptExecSandboxMemoryMB        = 512
	defaultSessionSystemPromptVisible   = configruntime.DefaultSessionSystemPromptVisible
	defaultAssistantMarkdownEnabled     = configruntime.DefaultAssistantMarkdownEnabled
	defaultToolCallCompactOutputEnabled = configruntime.DefaultToolCallCompactOutputEnabled
	defaultMemoryModeEnabled            = configruntime.DefaultMemoryModeEnabled
	defaultMicrocompactEnabled          = configruntime.DefaultMicrocompactEnabled
	defaultSessionTitleMode             = configruntime.DefaultSessionTitleMode
	defaultToolSelectorTimeoutMS        = 1500
	defaultToolSelectorConfidence       = 0.75
	defaultToolSelectorRecentMsgs       = 6
	defaultToolSearchIdleTurns          = 3
)

const (
	DefaultProvider                     = defaultProvider
	DefaultBaseURL                      = defaultBaseURL
	DefaultAnthropicBaseURL             = defaultAnthropicBaseURL
	DefaultModel                        = defaultModel
	DefaultPromptsPath                  = defaultPromptsPath
	DefaultPromptsDir                   = defaultPromptsDir
	DefaultSessionsPath                 = defaultSessionsPath
	DefaultTasksPath                    = defaultTasksPath
	DefaultTaskExecutionTimeoutMS       = defaultTaskExecutionTimeoutMS
	DefaultAnthropicVersion             = defaultAnthropicVersion
	DefaultAnthropicMaxTokens           = defaultAnthropicMaxTokens
	DefaultRelayStopPolicy              = defaultRelayStopPolicy
	DefaultRelayMaxRounds               = defaultRelayMaxRounds
	DefaultRelayExecutionTimeoutMS      = defaultRelayExecutionTimeoutMS
	DefaultExternalCodexPermissionMode  = defaultExternalCodexPermissionMode
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

const (
	RelayStopPolicyAIDecides = configruntime.RelayStopPolicyAIDecides
	RelayStopPolicyMaxRounds = configruntime.RelayStopPolicyMaxRounds
)

const (
	ExternalCodexPermissionReadOnly = configruntime.ExternalCodexPermissionReadOnly
	ExternalCodexPermissionDefault  = configruntime.ExternalCodexPermissionDefault
	ExternalCodexPermissionSafeYolo = configruntime.ExternalCodexPermissionSafeYolo
	ExternalCodexPermissionYolo     = configruntime.ExternalCodexPermissionYolo
)

const (
	SessionTitleModeSessionID    = configruntime.SessionTitleModeSessionID
	SessionTitleModeFirstMessage = configruntime.SessionTitleModeFirstMessage
	SessionTitleModeAIGenerated  = configruntime.SessionTitleModeAIGenerated
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
