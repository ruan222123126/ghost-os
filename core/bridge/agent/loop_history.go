package agent

import (
	"strings"

	"ghost-os/bridge/llm"
)

func (a *Agent) commitTurn(history *History) {
	if a == nil || history == nil {
		return
	}
	a.history = history
}

// appendUserMessage 只负责把用户输入追加到会话历史。
func appendUserMessage(history *History, userMessage string) {
	if history == nil {
		return
	}

	trimmed := strings.TrimSpace(userMessage)
	if trimmed == "" {
		return
	}

	history.Append(llm.Message{
		Role: llm.RoleUser,
		Text: trimmed,
	})
}

func acceptAssistantTurn(history *History, resp *llm.CompletionResponse) {
	if history == nil || resp == nil {
		return
	}

	history.Append(resp.Message)
	history.SetConversationState(resp.ConversationState)
}
