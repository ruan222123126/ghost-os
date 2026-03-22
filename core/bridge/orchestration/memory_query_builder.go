package orchestration

import (
	"strings"

	"ghost-os/bridge/agent"
	"ghost-os/bridge/llm"
	"ghost-os/bridge/memoryaug"
)

const (
	memoryPlannerRecentMessages = 3
	memoryPlannerMaxLineChars   = 160
)

func buildPlannerRecentMessages(history *agent.History, userMessage string) []memoryaug.TurnMessage {
	lines := collectRecentConversationMessages(history, memoryPlannerRecentMessages)
	out := make([]memoryaug.TurnMessage, 0, len(lines)+1)
	for _, message := range lines {
		out = append(out, memoryaug.TurnMessage{
			Role: string(message.Role),
			Text: compactPlannerText(message.Text),
		})
	}
	trimmedUserMessage := compactPlannerText(userMessage)
	if trimmedUserMessage != "" {
		out = append(out, memoryaug.TurnMessage{
			Role: string(llm.RoleUser),
			Text: trimmedUserMessage,
		})
	}
	return out
}

func collectRecentConversationMessages(history *agent.History, limit int) []llm.Message {
	if history == nil || limit <= 0 {
		return nil
	}
	messages := history.Messages()
	out := make([]llm.Message, 0, limit)
	for i := len(messages) - 1; i >= 0 && len(out) < limit; i-- {
		if !isPlannerRole(messages[i].Role) {
			continue
		}
		text := compactPlannerText(messages[i].Text)
		if text == "" {
			continue
		}
		out = append([]llm.Message{{Role: messages[i].Role, Text: text}}, out...)
	}
	return out
}

func isPlannerRole(role llm.Role) bool {
	return role == llm.RoleUser || role == llm.RoleAssistant
}

func compactPlannerText(raw string) string {
	text := strings.Join(strings.Fields(strings.TrimSpace(raw)), " ")
	if len(text) <= memoryPlannerMaxLineChars {
		return text
	}
	return strings.TrimSpace(text[:memoryPlannerMaxLineChars-3]) + "..."
}
