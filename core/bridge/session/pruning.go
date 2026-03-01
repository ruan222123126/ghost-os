package session

import (
	"strings"

	"ghost-os/bridge/llm"
)

const (
	defaultRecentMessagesToKeep = 10

	openAIContextTokens      = 8000
	openAIResponseReserve    = 2000
	anthropicContextTokens   = 100000
	anthropicResponseReserve = 4000
	customContextTokens      = 4000
)

type messageSpan struct {
	start  int
	end    int
	tokens int
}

// EstimateTokens 基于文本长度做近似估算，避免引入 provider 专属依赖。
func EstimateTokens(msg llm.Message) int {
	text := strings.TrimSpace(msg.Text)
	for _, part := range msg.Content {
		switch strings.ToLower(strings.TrimSpace(part.Type)) {
		case llm.ContentTypeText:
			text += part.Text
		case llm.ContentTypeImage:
			if part.Image != nil {
				text += part.Image.Path + part.Image.URL + part.Image.MimeType + part.Image.SHA256
			}
		}
	}
	if msg.ToolCallID != "" {
		text += msg.ToolCallID
	}
	for _, call := range msg.ToolCalls {
		text += call.ID
		text += call.Name
		text += string(call.Arguments)
	}

	if text == "" {
		return 4
	}

	charCount := len(text)
	var estimated int
	if containsStructuredContent(text) {
		estimated = charCount / 2
	} else {
		estimated = charCount / 4
	}
	if estimated <= 0 {
		estimated = 1
	}
	for _, part := range msg.Content {
		if strings.EqualFold(strings.TrimSpace(part.Type), llm.ContentTypeImage) {
			// 图片输入在 provider 侧开销高于纯文本，按固定开销上调估算，减少上下文低估。
			estimated += 500
		}
	}

	// 加入每条消息固定开销，防止系统性低估。
	return estimated + 4
}

// PruneMessages 在超限时按“系统消息 + 最近消息优先”的策略裁剪。
func PruneMessages(messages []llm.Message, maxTokens int) []llm.Message {
	if len(messages) == 0 {
		return nil
	}
	if maxTokens <= 0 {
		return llm.CloneMessages(messages)
	}

	total := 0
	for _, msg := range messages {
		total += EstimateTokens(msg)
	}
	if total <= maxTokens {
		return llm.CloneMessages(messages)
	}

	systemIndex := -1
	start := 0
	if messages[0].Role == llm.RoleSystem {
		systemIndex = 0
		start = 1
	}

	recentStart := len(messages) - defaultRecentMessagesToKeep
	if recentStart < start {
		recentStart = start
	}

	spans := buildMessageSpans(messages, start)
	keep := make([]bool, len(spans))
	for i := range keep {
		keep[i] = true
	}

	for i, span := range spans {
		if total <= maxTokens {
			break
		}
		// 最近 N 条消息保护，不做删除。
		if span.end > recentStart {
			continue
		}
		keep[i] = false
		total -= span.tokens
	}

	pruned := make([]llm.Message, 0, len(messages))
	if systemIndex >= 0 {
		pruned = append(pruned, messages[systemIndex])
	}
	for i, span := range spans {
		if !keep[i] {
			continue
		}
		pruned = append(pruned, messages[span.start:span.end]...)
	}

	return llm.CloneMessages(pruned)
}

// GetContextLimit 返回保守的 prompt token 预算（已预留输出空间）。
func GetContextLimit(provider llm.Provider, model string) int {
	normalizedProvider := provider.Normalized()
	modelName := strings.ToLower(strings.TrimSpace(model))

	if strings.Contains(modelName, "claude") {
		return anthropicContextTokens - anthropicResponseReserve
	}
	if strings.Contains(modelName, "gpt") || strings.Contains(modelName, "o1") || strings.Contains(modelName, "o3") {
		return openAIContextTokens - openAIResponseReserve
	}

	switch normalizedProvider {
	case llm.ProviderAnthropic:
		return anthropicContextTokens - anthropicResponseReserve
	case llm.ProviderOpenAI:
		return openAIContextTokens - openAIResponseReserve
	default:
		return customContextTokens
	}
}

func buildMessageSpans(messages []llm.Message, start int) []messageSpan {
	if start >= len(messages) {
		return nil
	}

	spans := make([]messageSpan, 0, len(messages)-start)
	for i := start; i < len(messages); {
		spanEnd := i + 1
		if messages[i].Role == llm.RoleAssistant && len(messages[i].ToolCalls) > 0 {
			for spanEnd < len(messages) && messages[spanEnd].Role == llm.RoleTool {
				spanEnd++
			}
		}

		tokens := 0
		for _, msg := range messages[i:spanEnd] {
			tokens += EstimateTokens(msg)
		}
		spans = append(spans, messageSpan{
			start:  i,
			end:    spanEnd,
			tokens: tokens,
		})

		i = spanEnd
	}

	return spans
}

func containsStructuredContent(text string) bool {
	if strings.Contains(text, "```") {
		return true
	}

	markers := []string{
		"{", "}", "[", "]", "=>", "::", "func ", "package ", "import ", "SELECT ", "INSERT ", "\"tool\"",
	}
	for _, marker := range markers {
		if strings.Contains(text, marker) {
			return true
		}
	}
	return false
}
