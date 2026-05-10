package orchestration

import (
	"ghost-os/bridge/llm"
	"ghost-os/bridge/orchestration/internal/domain/sessionturn"
)

func sanitizeToolProtocolMessages(messages []llm.Message) []llm.Message {
	return sessionturn.SanitizeToolProtocolMessages(messages)
}
