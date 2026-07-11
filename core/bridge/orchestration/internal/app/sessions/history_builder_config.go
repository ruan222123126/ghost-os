package sessions

import (
	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/orchestration/internal/domain/sessionturn"
	"ghost-os/bridge/session"
)

func NewHistoryBuilderFromConfig(
	cfg bridgeconfig.Config,
	systemPrompt string,
	sessionStore *session.Store,
	traceID string,
) *HistoryBuilder {
	return NewHistoryBuilder(
		ProviderContextFromConfig(cfg.Provider),
		systemPrompt,
		sessionStore,
		cfg.ToolSearch.IdleTurns,
		cfg.MicrocompactEnabled,
		traceID,
	)
}

func ProviderContextFromConfig(provider bridgeconfig.ProviderConfig) sessionturn.ProviderContext {
	return sessionturn.ProviderContext{
		Type:                       provider.Type,
		BaseURL:                    provider.BaseURL,
		Model:                      provider.Model,
		ContextWindowTokens:        provider.ContextWindowTokens,
		ResponseReserveTokens:      provider.ResponseReserveTokens,
		ModelContextWindowTokens:   provider.ModelContextWindowTokens,
		ModelResponseReserveTokens: provider.ModelResponseReserveTokens,
	}
}
