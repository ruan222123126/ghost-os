package runtime

import "ghost-os/bridge/llm"

const (
	DefaultProvider                     = llm.ProviderOpenAI
	DefaultBaseURL                      = "https://api.openai.com/v1"
	DefaultAnthropicBaseURL             = "https://api.anthropic.com"
	DefaultModel                        = "gpt-4o"
	DefaultTaskExecutionTimeoutMS       = 300000
	DefaultRelayStopPolicy              = RelayStopPolicyAIDecides
	DefaultRelayMaxRounds               = 20
	DefaultRelayExecutionTimeoutMS      = 0
	DefaultMaxTurns                     = 20
	DefaultLLMCompletionRetryCount      = 1
	DefaultLLMCompletionRetryIntervalMS = 200
	DefaultModelSelectionEnabled        = true
	DefaultSessionSystemPromptVisible   = true
	DefaultAssistantMarkdownEnabled     = true
	DefaultToolCallCompactOutputEnabled = false
	DefaultMemoryModeEnabled            = false
	DefaultMicrocompactEnabled          = false
	DefaultSessionTitleMode             = SessionTitleModeSessionID
)
