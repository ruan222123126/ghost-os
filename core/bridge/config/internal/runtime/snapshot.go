package runtime

import "ghost-os/bridge/llm"

type Snapshot struct {
	ProviderName                   string
	Provider                       llm.Provider
	APIKey                         string
	BaseURL                        string
	Model                          string
	ChatPath                       string
	ResponseOptions                llm.ResponseOptions
	CodexStatelessRetryEnabled     bool
	NativePersistent               bool
	ProjectRoot                    string
	MaxTurns                       int
	TaskExecutionTimeoutMS         int
	RelayDefaultStopPolicy         string
	RelayDefaultMaxRounds          int
	RelayDefaultExecutionTimeoutMS int
	ModelSelectionEnabled          bool
	ContextWindowTokens            int
	ResponseReserveTokens          int
	ModelContextWindowTokens       map[string]int
	ModelResponseReserveTokens     map[string]int
	WebSearchTavilyURL             string
	WebSearchExaURL                string
	WebSearchTavilyAPIKey          string
	WebSearchExaAPIKey             string
	LLMCompletionRetryCount        int
	LLMCompletionRetryIntervalMS   int
	SessionHumanLogFullEnabled     bool
	SessionSystemPromptVisible     bool
	AssistantMarkdownEnabled       bool
	ToolCallCompactOutputEnabled   bool
	MemoryModeEnabled              bool
	MicrocompactEnabled            bool
	SessionTitleMode               string
}

func Clone(raw Snapshot) Snapshot {
	return Snapshot{
		ProviderName:                   raw.ProviderName,
		Provider:                       raw.Provider,
		APIKey:                         raw.APIKey,
		BaseURL:                        raw.BaseURL,
		Model:                          raw.Model,
		ChatPath:                       raw.ChatPath,
		ResponseOptions:                llm.CloneResponseOptions(raw.ResponseOptions),
		CodexStatelessRetryEnabled:     raw.CodexStatelessRetryEnabled,
		NativePersistent:               raw.NativePersistent,
		ProjectRoot:                    raw.ProjectRoot,
		MaxTurns:                       raw.MaxTurns,
		TaskExecutionTimeoutMS:         raw.TaskExecutionTimeoutMS,
		RelayDefaultStopPolicy:         raw.RelayDefaultStopPolicy,
		RelayDefaultMaxRounds:          raw.RelayDefaultMaxRounds,
		RelayDefaultExecutionTimeoutMS: raw.RelayDefaultExecutionTimeoutMS,
		ModelSelectionEnabled:          raw.ModelSelectionEnabled,
		ContextWindowTokens:            raw.ContextWindowTokens,
		ResponseReserveTokens:          raw.ResponseReserveTokens,
		ModelContextWindowTokens:       cloneModelTokenOverrides(raw.ModelContextWindowTokens),
		ModelResponseReserveTokens:     cloneModelTokenOverrides(raw.ModelResponseReserveTokens),
		WebSearchTavilyURL:             raw.WebSearchTavilyURL,
		WebSearchExaURL:                raw.WebSearchExaURL,
		WebSearchTavilyAPIKey:          raw.WebSearchTavilyAPIKey,
		WebSearchExaAPIKey:             raw.WebSearchExaAPIKey,
		LLMCompletionRetryCount:        raw.LLMCompletionRetryCount,
		LLMCompletionRetryIntervalMS:   raw.LLMCompletionRetryIntervalMS,
		SessionHumanLogFullEnabled:     raw.SessionHumanLogFullEnabled,
		SessionSystemPromptVisible:     raw.SessionSystemPromptVisible,
		AssistantMarkdownEnabled:       raw.AssistantMarkdownEnabled,
		ToolCallCompactOutputEnabled:   raw.ToolCallCompactOutputEnabled,
		MemoryModeEnabled:              raw.MemoryModeEnabled,
		MicrocompactEnabled:            raw.MicrocompactEnabled,
		SessionTitleMode:               raw.SessionTitleMode,
	}
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
