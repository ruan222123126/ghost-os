package orchestration

import (
	"ghost-os/bridge/llm"
	"ghost-os/bridge/orchestration/internal/domain/sessionturn"
)

func estimateMessagesTokens(messages []llm.Message) int {
	return sessionturn.EstimateMessagesTokens(messages)
}
