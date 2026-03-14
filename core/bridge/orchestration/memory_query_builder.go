package orchestration

import (
	"strings"

	"ghost-os/bridge/agent"
	"ghost-os/bridge/llm"
)

const (
	memoryQueryRecentMessages = 3
	memoryQueryMaxChars       = 420
	memoryQueryMaxLineChars   = 160
)

func buildMemoryRecallQuery(history *agent.History, userMessage string) string {
	currentLine := formatMemoryQueryLine(llm.Message{
		Role: llm.RoleUser,
		Text: userMessage,
	})
	if history == nil {
		return strings.TrimSpace(currentLine)
	}
	recent := collectRecentConversationMessages(history.Messages(), memoryQueryRecentMessages)
	lines := make([]string, 0, len(recent)+1)
	budget := memoryQueryMaxChars
	if currentLine != "" {
		lines = append(lines, currentLine)
		budget -= len(currentLine)
	}
	currentKey := dedupeMemoryQueryLine(currentLine)
	for i := len(recent) - 1; i >= 0; i-- {
		line := formatMemoryQueryLine(recent[i])
		if line == "" || dedupeMemoryQueryLine(line) == currentKey {
			continue
		}
		required := len(line)
		if len(lines) > 0 {
			required++
		}
		if budget-required < 0 {
			break
		}
		lines = append([]string{line}, lines...)
		budget -= required
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func collectRecentConversationMessages(messages []llm.Message, limit int) []llm.Message {
	if limit <= 0 || len(messages) == 0 {
		return nil
	}
	out := make([]llm.Message, 0, limit)
	for i := len(messages) - 1; i >= 0 && len(out) < limit; i-- {
		if !isMemoryQueryRole(messages[i].Role) {
			continue
		}
		if strings.TrimSpace(messages[i].Text) == "" {
			continue
		}
		out = append([]llm.Message{messages[i]}, out...)
	}
	return out
}

func isMemoryQueryRole(role llm.Role) bool {
	return role == llm.RoleUser || role == llm.RoleAssistant
}

func formatMemoryQueryLine(message llm.Message) string {
	text := compactMemoryQueryText(message.Text)
	if text == "" {
		return ""
	}
	return string(message.Role) + ": " + text
}

func compactMemoryQueryText(raw string) string {
	text := strings.Join(strings.Fields(strings.TrimSpace(raw)), " ")
	if len(text) <= memoryQueryMaxLineChars {
		return text
	}
	return strings.TrimSpace(text[:memoryQueryMaxLineChars-3]) + "..."
}

func dedupeMemoryQueryLine(line string) string {
	return strings.ToLower(strings.TrimSpace(line))
}
