package agent

import "ghost-os/bridge/llm"

func projectMessagesForProvider(messages []llm.Message, _ ToolCatalog) []llm.Message {
	return llm.ProjectMessagesForCompletion(messages)
}
