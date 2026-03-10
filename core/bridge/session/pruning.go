package session

import (
	"strings"
	"unicode/utf8"

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
	stats := tokenEstimateStats{}
	stats.addText(strings.TrimSpace(msg.Text))
	for _, part := range msg.Content {
		switch strings.ToLower(strings.TrimSpace(part.Type)) {
		case llm.ContentTypeText:
			stats.addText(part.Text)
		case llm.ContentTypeImage:
			if part.Image != nil {
				stats.addText(part.Image.Path)
				stats.addText(part.Image.URL)
				stats.addText(part.Image.MimeType)
				stats.addText(part.Image.SHA256)
			}
		}
	}
	if msg.ToolCallID != "" {
		stats.addText(msg.ToolCallID)
	}
	for _, call := range msg.ToolCalls {
		stats.addText(call.ID)
		stats.addText(call.Name)
		stats.addText(string(call.Arguments))
	}

	if stats.isEmpty() {
		return 4
	}

	charCount := stats.charCount()
	var estimated int
	if stats.structured {
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
	if strings.Contains(modelName, "gpt") || strings.Contains(modelName, "o1") || strings.Contains(modelName, "o3") || strings.Contains(modelName, "codex") {
		return openAIContextTokens - openAIResponseReserve
	}

	switch normalizedProvider {
	case llm.ProviderAnthropic:
		return anthropicContextTokens - anthropicResponseReserve
	case llm.ProviderOpenAI, llm.ProviderCodex:
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

type tokenEstimateStats struct {
	bytes       int
	runes       int
	hasNonASCII bool
	structured  bool
	tail        string
}

func (s *tokenEstimateStats) addText(text string) {
	if text == "" {
		return
	}

	s.bytes += len(text)
	if !s.hasNonASCII {
		for i := 0; i < len(text); i++ {
			if text[i] >= utf8.RuneSelf {
				s.hasNonASCII = true
				break
			}
		}
	}
	if s.hasNonASCII {
		s.runes += utf8.RuneCountInString(text)
	} else {
		s.runes += len(text)
	}

	if !s.structured {
		candidate := text
		if s.tail != "" {
			candidate = s.tail + text
		}
		if containsStructuredContent(candidate) {
			s.structured = true
		}
	}

	s.tail = updateStructuredTail(s.tail, text)
}

func (s *tokenEstimateStats) isEmpty() bool {
	return s.bytes == 0 && s.runes == 0
}

func (s *tokenEstimateStats) charCount() int {
	if s.hasNonASCII {
		return s.runes
	}
	return s.bytes
}

func updateStructuredTail(prev, text string) string {
	if text == "" {
		return prev
	}

	const maxMarkerLen = 8
	combined := prev + text
	if len(combined) <= maxMarkerLen-1 {
		return combined
	}
	return combined[len(combined)-(maxMarkerLen-1):]
}
