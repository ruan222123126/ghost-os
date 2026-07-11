package session

import (
	"strings"

	"ghost-os/bridge/llm"
)

const (
	openAIContextTokens          = 8000
	openAIResponseReserve        = 2000
	openAIModernContextTokens    = 128000
	openAI32KContextTokens       = 32768
	openAI16KContextTokens       = 16384
	anthropicContextTokens       = 100000
	anthropicModernContextTokens = 200000
	anthropicResponseReserve     = 4000
	customContextTokens          = 4000
)

type modelContextRule struct {
	pattern string
	tokens  int
}

type contextLimitResolver struct {
	modelRules []modelContextRule
}

func newContextLimitResolver() contextLimitResolver {
	return contextLimitResolver{
		modelRules: []modelContextRule{
			{pattern: "gpt-4o", tokens: openAIModernContextTokens},
			{pattern: "gpt-4.1", tokens: openAIModernContextTokens},
			{pattern: "gpt-4-turbo", tokens: openAIModernContextTokens},
			{pattern: "gpt-4-0125", tokens: openAIModernContextTokens},
			{pattern: "gpt-4-1106", tokens: openAIModernContextTokens},
			{pattern: "gpt-4-32k", tokens: openAI32KContextTokens},
			{pattern: "gpt-3.5-16k", tokens: openAI16KContextTokens},
			{pattern: "o1", tokens: openAIModernContextTokens},
			{pattern: "o3", tokens: openAIModernContextTokens},
			{pattern: "codex", tokens: openAIModernContextTokens},
			{pattern: "claude-3", tokens: anthropicModernContextTokens},
			{pattern: "claude-2", tokens: anthropicContextTokens},
		},
	}
}

func (r contextLimitResolver) Resolve(provider llm.Provider, model string, cfg ContextLimitConfig) int {
	normalizedProvider := provider.Normalized()
	modelName := strings.ToLower(strings.TrimSpace(model))
	contextWindow, responseReserve := r.resolveDefaults(normalizedProvider, modelName)
	contextWindow = applyPositiveOverride(contextWindow, cfg.ContextWindowTokens)
	responseReserve = applyPositiveOverride(responseReserve, cfg.ResponseReserveTokens)
	contextWindow = applyPositiveOverride(contextWindow, matchModelTokenOverride(cfg.ModelContextWindowTokens, modelName))
	responseReserve = applyPositiveOverride(responseReserve, matchModelTokenOverride(cfg.ModelResponseReserveTokens, modelName))
	return resolvePromptBudget(contextWindow, responseReserve)
}

func (r contextLimitResolver) resolveDefaults(provider llm.Provider, model string) (int, int) {
	contextWindow := r.contextWindowForProvider(provider, model)
	responseReserve := responseReserveForProvider(provider)
	return contextWindow, responseReserve
}

func (r contextLimitResolver) contextWindowForProvider(provider llm.Provider, model string) int {
	if ruleTokens := r.modelRuleTokens(model); ruleTokens > 0 {
		return ruleTokens
	}
	switch provider {
	case llm.ProviderAnthropic:
		return anthropicContextTokens
	case llm.ProviderOpenAI, llm.ProviderCodex:
		return openAIContextTokens
	default:
		return customContextTokens
	}
}

func (r contextLimitResolver) modelRuleTokens(model string) int {
	for _, rule := range r.modelRules {
		if strings.Contains(model, rule.pattern) {
			return rule.tokens
		}
	}
	return 0
}

func responseReserveForProvider(provider llm.Provider) int {
	switch provider {
	case llm.ProviderAnthropic:
		return anthropicResponseReserve
	case llm.ProviderOpenAI, llm.ProviderCodex:
		return openAIResponseReserve
	default:
		return 0
	}
}

func resolvePromptBudget(contextWindow int, responseReserve int) int {
	if contextWindow <= 0 {
		contextWindow = customContextTokens
	}
	if responseReserve <= 0 || responseReserve >= contextWindow {
		return contextWindow
	}
	return contextWindow - responseReserve
}

func applyPositiveOverride(current int, override int) int {
	if override > 0 {
		return override
	}
	return current
}

func matchModelTokenOverride(overrides map[string]int, model string) int {
	if len(overrides) == 0 || model == "" {
		return 0
	}
	if value, ok := overrides[model]; ok && value > 0 {
		return value
	}
	return matchModelPrefixOverride(overrides, model)
}

func matchModelPrefixOverride(overrides map[string]int, model string) int {
	for key, value := range overrides {
		if value <= 0 || !strings.HasSuffix(key, "*") {
			continue
		}
		prefix := strings.TrimSuffix(key, "*")
		if prefix != "" && strings.HasPrefix(model, prefix) {
			return value
		}
	}
	return 0
}
