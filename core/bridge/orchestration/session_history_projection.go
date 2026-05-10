package orchestration

import (
	"ghost-os/bridge/llm"
	"ghost-os/bridge/orchestration/internal/domain/sessionturn"
)

type messageProjectionOptions = sessionturn.ProjectionOptions

func projectMessagesForModel(messages []llm.Message, options messageProjectionOptions) []llm.Message {
	return sessionturn.ProjectMessagesForModel(messages, options)
}
