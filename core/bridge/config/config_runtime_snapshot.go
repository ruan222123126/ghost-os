package config

import "ghost-os/bridge/llm"

// runtimeConfig is the resolved runtime layer: persisted file DTO merged with
// env fallback, normalized once for execution and public exposure.
type runtimeConfig struct {
	ProviderName               string
	Provider                   llm.Provider
	APIKey                     string
	BaseURL                    string
	Model                      string
	ChatPath                   string
	ResponseOptions            llm.ResponseOptions
	CodexStatelessRetryEnabled bool
	NativePersistent           bool
	ProjectRoot                string
	ModelSelectionEnabled      bool
	ContextWindowTokens        int
	ResponseReserveTokens      int
	ModelContextWindowTokens   map[string]int
	ModelResponseReserveTokens map[string]int
	WebSearchTavilyURL         string
	WebSearchExaURL            string
	WebSearchTavilyAPIKey      string
	WebSearchExaAPIKey         string
	SessionHumanLogFullEnabled bool
	WebRooterEnabled           bool
	WebRooterBaseURL           string
	WebRooterAPIToken          string
	WebRooterTimeoutMS         int
	GraphQL                    GraphQLConfig
}
