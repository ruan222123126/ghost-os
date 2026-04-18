package config

import "ghost-os/bridge/llm"

type ProviderConfig struct {
	Type                       llm.Provider
	APIKey                     string
	BaseURL                    string
	Model                      string
	Headers                    map[string]string
	AnthropicVersion           string
	AnthropicMaxTokens         int
	ContextWindowTokens        int
	ResponseReserveTokens      int
	ModelContextWindowTokens   map[string]int
	ModelResponseReserveTokens map[string]int
}

type ProviderRecord struct {
	Name                       string
	Type                       llm.Provider
	BaseURL                    string
	APIKey                     *string
	Models                     []string
	ContextWindowTokens        int
	ResponseReserveTokens      int
	ModelContextWindowTokens   map[string]int
	ModelResponseReserveTokens map[string]int
}
