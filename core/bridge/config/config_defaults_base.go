package config

import "ghost-os/bridge/llm"

const (
	defaultProvider                     = llm.ProviderOpenAI
	defaultBaseURL                      = "https://api.openai.com/v1"
	defaultAnthropicBaseURL             = "https://api.anthropic.com"
	defaultModel                        = "gpt-4o"
	defaultPromptsPath                  = "prompts.yaml"
	defaultPromptsDir                   = "~/.ghost-os/prompts"
	defaultSessionsPath                 = "~/.ghost-os/sessions"
	defaultTasksPath                    = "~/.ghost-os/tasks"
	defaultAnthropicVersion             = "2023-06-01"
	defaultAnthropicMaxTokens           = 1024
	defaultProMaxIterations             = 20
	defaultMaxTurns                     = 20
	defaultLLMCompletionRetryCount      = 1
	defaultLLMCompletionRetryIntervalMS = 200
	defaultWorkerMaxConcurrency         = 4
	defaultWorkerMaxFiles               = 20
	defaultWorkerMaxFileChunks          = 4
	defaultScriptExecSandboxMemoryMB    = 256
	maxScriptExecSandboxMemoryMB        = 512
	defaultSessionSystemPromptVisible   = true
	defaultAssistantMarkdownEnabled     = true
	defaultToolCallCompactOutputEnabled = false
	defaultMemoryModeEnabled            = false
	defaultMicrocompactEnabled          = false
)
