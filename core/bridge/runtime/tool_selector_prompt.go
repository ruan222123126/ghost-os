package runtime

import (
	"fmt"
	"strings"

	"ghost-os/bridge/llm"
)

func (ts *ToolSelector) selectorSystemPrompt() string {
	return `You are a tool selector for Ghost-OS.

Choose the MINIMAL sufficient tool subset for the current turn.

Rules:
- ALWAYS include ask_human.
- Decision hints are advisory, not mandatory.
- Prefer the current request and available tools over historical hints when they conflict.
- Do not infer unavailable tools from hints.
- Return mode="subset" only when the current request is focused and a small tool set is clearly sufficient.
- Return mode="all" otherwise, especially when uncertain or the hints are weak, stale, or not clearly applicable.
- Confidence must be 0.0-1.0.
- Output JSON only.

JSON schema:
{"mode":"subset|all","tools":["tool_name"],"confidence":0.85,"reason":"brief explanation"}`
}

func (ts *ToolSelector) buildSelectorPrompt(userMessage string, recentHistory []llm.Message, decisionHint string) string {
	var sb strings.Builder
	sb.WriteString("Available tools:\n")
	sb.WriteString(ts.metadata)
	sb.WriteString("\n\nRecent context:\n")
	writeSelectorHistory(&sb, recentHistory)
	writeSelectorHint(&sb, decisionHint)
	sb.WriteString("\nCurrent request:\n")
	sb.WriteString(truncateSelectorText(userMessage, 500))
	return sb.String()
}

func writeSelectorHistory(sb *strings.Builder, history []llm.Message) {
	if len(history) == 0 {
		sb.WriteString("(none)\n")
		return
	}

	for _, msg := range history {
		switch msg.Role {
		case llm.RoleUser:
			sb.WriteString(fmt.Sprintf("User: %s\n", truncateSelectorText(msg.Text, 200)))
		case llm.RoleAssistant:
			sb.WriteString(fmt.Sprintf("Assistant: %s\n", truncateSelectorText(msg.Text, 200)))
		}
	}
}

func writeSelectorHint(sb *strings.Builder, decisionHint string) {
	trimmedHint := truncateSelectorText(decisionHint, 500)
	if trimmedHint == "" {
		return
	}

	sb.WriteString("\nPrior similar experience:\n")
	sb.WriteString(trimmedHint)
	sb.WriteString("\n")
}

func truncateSelectorText(text string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}

	runes := []rune(strings.TrimSpace(text))
	if len(runes) <= maxLen {
		return string(runes)
	}
	return string(runes[:maxLen]) + "..."
}
