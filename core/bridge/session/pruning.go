package session

import "ghost-os/bridge/llm"

const defaultRecentMessagesToKeep = 10

type ContextLimitConfig struct {
	ContextWindowTokens        int
	ResponseReserveTokens      int
	ModelContextWindowTokens   map[string]int
	ModelResponseReserveTokens map[string]int
}

var (
	defaultTokenEstimator      = tokenEstimator{}
	defaultContextLimitPolicy  = newContextLimitResolver()
	defaultSessionMessagePrune = newMessagePruner(defaultRecentMessagesToKeep, defaultTokenEstimator.Estimate)
)

// EstimateTokens 基于文本长度做近似估算，避免引入 provider 专属依赖。
func EstimateTokens(msg llm.Message) int {
	return defaultTokenEstimator.Estimate(msg)
}

// PruneMessages 在超限时按“系统消息 + 最近消息优先”的策略裁剪。
func PruneMessages(messages []llm.Message, maxTokens int) []llm.Message {
	return defaultSessionMessagePrune.Prune(messages, maxTokens)
}

// GetContextLimit 返回 prompt token 预算（已预留输出空间）。
func GetContextLimit(provider llm.Provider, model string, cfg ContextLimitConfig) int {
	return defaultContextLimitPolicy.Resolve(provider, model, cfg)
}
