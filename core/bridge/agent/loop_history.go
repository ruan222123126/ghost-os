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
func appendUserMessage(history *History, userInput llm.Message) {
	if history == nil {
		return
	}
	if userMessage, ok := normalizeUserInputMessage(userInput); ok {
		history.Append(userMessage)
	}
}

func normalizeUserInputMessage(userInput llm.Message) (llm.Message, bool) {
	message := llm.CloneMessages([]llm.Message{userInput})
	if len(message) == 0 {
		return llm.Message{}, false
	}
	message[0].Role = llm.RoleUser
	message[0].Text = strings.TrimSpace(message[0].Text)
	if message[0].Text == "" && len(message[0].Content) == 0 {
		return llm.Message{}, false
	}
	return message[0], true
}

func acceptAssistantTurn(history *History, resp *llm.CompletionResponse) {
	if history == nil || resp == nil {
		return
	}

	history.Append(resp.Message)
	history.SetConversationState(resp.ConversationState)
}
