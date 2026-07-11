package sessionturn

import (
	"strings"
	"time"

	"ghost-os/bridge/llm"
)

type ProviderContext struct {
	Type                       llm.Provider
	BaseURL                    string
	Model                      string
	ContextWindowTokens        int
	ResponseReserveTokens      int
	ModelContextWindowTokens   map[string]int
	ModelResponseReserveTokens map[string]int
}

type ResolvedHumanQuestion struct {
	QuestionID string
	ToolName   string
	Prompt     string
	ToolCallID string
	TraceID    string
	Answer     string
	Summary    string
	AskedAt    time.Time
	AnsweredAt time.Time
}

func AgentMessageForResolvedHumanTool(
	toolCallID string,
	toolName string,
	traceID string,
	output string,
) llm.Message {
	return llm.Message{
		Role:       llm.RoleTool,
		ToolCallID: toolCallID,
		Text:       llm.FormatToolResult(toolName, traceID, output, nil),
	}
}

// MessagesWithSystemPrompt 在请求构建阶段覆盖 system prompt，避免修改会话持久化内容。
func MessagesWithSystemPrompt(messages []llm.Message, systemPrompt string) []llm.Message {
	out := llm.CloneMessages(messages)
	prompt := strings.TrimSpace(systemPrompt)
	if prompt == "" {
		return out
	}

	desired := llm.Message{
		Role: llm.RoleSystem,
		Text: prompt,
	}

	switch {
	case len(out) == 0:
		return []llm.Message{desired}
	case out[0].Role == llm.RoleSystem:
		out[0] = desired
		return out
	default:
		updated := make([]llm.Message, 0, len(out)+1)
		updated = append(updated, desired)
		updated = append(updated, out...)
		return updated
	}
}
