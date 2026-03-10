package session

import (
	"strings"

	"ghost-os/bridge/llm"
)

const (
	defaultRecentMessagesToKeep = 10

	openAIContextTokens          = 8000
	openAIResponseReserve        = 2000
	openAIModernContextTokens    = 128000
	openAI32KContextTokens       = 32768
	openAI16KContextTokens       = 16384
	anthropicContextTokens       = 100000
	anthropicModernContextTokens = 200000
	anthropicResponseReserve     = 4000
	customContextTokens          = 4000
)

type messageSpan struct {
	start  int
	end    int
	tokens int
}

type ContextLimitConfig struct {
	ContextWindowTokens        int
	ResponseReserveTokens      int
	ModelContextWindowTokens   map[string]int
	ModelResponseReserveTokens map[string]int
}

type modelContextRule struct {
	pattern string
	tokens  int
}

var defaultModelContextRules = []modelContextRule{
	{pattern: "gpt-4o", tokens: openAIModernContextTokens},
	{pattern: "gpt-4.1", tokens: openAIModernContextTokens},
	{pattern: "gpt-4-turbo", tokens: openAIModernContextTokens},
	{pattern: "gpt-4-0125", tokens: openAIModernContextTokens},
	{pattern: "gpt-4-1106", tokens: openAIModernContextTokens},
	{pattern: "gpt-4-32k", tokens: openAI32KContextTokens},
	{pattern: "gpt-3.5-16k", tokens: openAI16KContextTokens},
	{pattern: "o1", tokens: openAIModernContextTokens},
	{pattern: "o3", tokens: openAIModernContextTokens},
	{pattern: "codex", tokens: openAIModernContextTokens},
	{pattern: "claude-3", tokens: anthropicModernContextTokens},
	{pattern: "claude-2", tokens: anthropicContextTokens},
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

	if total > maxTokens {
		kept := 0
		for _, keepSpan := range keep {
			if keepSpan {
				kept++
			}
		}
		for i, span := range spans {
			if total <= maxTokens || kept <= 1 {
				break
			}
			if !keep[i] {
				continue
			}
			keep[i] = false
			kept--
			total -= span.tokens
		}
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

	if total > maxTokens {
		pruned = clampMessagesToTokenLimit(pruned, maxTokens)
	}

	return llm.CloneMessages(pruned)
}

// GetContextLimit 返回 prompt token 预算（已预留输出空间）。
func GetContextLimit(provider llm.Provider, model string, cfg ContextLimitConfig) int {
	normalizedProvider := provider.Normalized()
	modelName := strings.ToLower(strings.TrimSpace(model))

	contextWindow := contextWindowForProvider(normalizedProvider, modelName)
	responseReserve := responseReserveForProvider(normalizedProvider)

	if cfg.ContextWindowTokens > 0 {
		contextWindow = cfg.ContextWindowTokens
	}
	if cfg.ResponseReserveTokens > 0 {
		responseReserve = cfg.ResponseReserveTokens
	}
	if override := matchModelTokenOverride(cfg.ModelContextWindowTokens, modelName); override > 0 {
		contextWindow = override
	}
	if override := matchModelTokenOverride(cfg.ModelResponseReserveTokens, modelName); override > 0 {
		responseReserve = override
	}

	if contextWindow == 0 {
		contextWindow = customContextTokens
	}

	limit := contextWindow
	if responseReserve > 0 && responseReserve < contextWindow {
		limit = contextWindow - responseReserve
	}
	if limit <= 0 {
		limit = contextWindow
	}
	return limit
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

func clampMessagesToTokenLimit(messages []llm.Message, maxTokens int) []llm.Message {
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

	out := llm.CloneMessages(messages)
	start := 0
	systemTokens := 0
	if out[0].Role == llm.RoleSystem {
		systemTokens = EstimateTokens(out[0])
		start = 1
	}
	if systemTokens > maxTokens {
		out[0] = truncateMessage(out[0], maxTokens)
		return out[:1]
	}

	if start >= len(out) {
		return out
	}

	spanBudget := maxTokens - systemTokens
	if spanBudget <= 0 {
		return out[:start]
	}

	last := out[start:]
	truncated := truncateMessagesToBudget(last, spanBudget)
	if start == 0 {
		return truncated
	}
	merged := make([]llm.Message, 0, 1+len(truncated))
	merged = append(merged, out[0])
	merged = append(merged, truncated...)
	return merged
}

func truncateMessagesToBudget(messages []llm.Message, budget int) []llm.Message {
	if len(messages) == 0 {
		return nil
	}
	if budget <= 0 {
		return nil
	}

	out := llm.CloneMessages(messages)
	total := 0
	weights := make([]int, len(out))
	for i, msg := range out {
		tokens := EstimateTokens(msg)
		if tokens <= 0 {
			tokens = 1
		}
		weights[i] = tokens
		total += tokens
	}
	if total <= budget {
		return out
	}

	shares := make([]int, len(out))
	remaining := budget
	for i, weight := range weights {
		share := int(float64(weight) * float64(budget) / float64(total))
		if share < 4 {
			share = 4
		}
		shares[i] = share
		remaining -= share
	}
	for remaining < 0 {
		adjusted := false
		for i := 0; i < len(shares) && remaining < 0; i++ {
			if shares[i] > 4 {
				shares[i]--
				remaining++
				adjusted = true
			}
		}
		if !adjusted {
			break
		}
	}
	if remaining > 0 && len(shares) > 0 {
		shares[len(shares)-1] += remaining
	}

	for i := range out {
		out[i] = truncateMessage(out[i], shares[i])
	}

	total = 0
	for _, msg := range out {
		total += EstimateTokens(msg)
	}
	if total <= budget {
		return out
	}

	for i := range out {
		if total <= budget {
			break
		}
		current := EstimateTokens(out[i])
		remainingBudget := budget - (total - current)
		if remainingBudget < 4 {
			remainingBudget = 4
		}
		out[i] = truncateMessage(out[i], remainingBudget)
		total = 0
		for _, msg := range out {
			total += EstimateTokens(msg)
		}
	}

	return out
}

func truncateMessage(msg llm.Message, maxTokens int) llm.Message {
	if maxTokens <= 0 {
		return msg
	}

	out := llm.CloneMessages([]llm.Message{msg})[0]
	if EstimateTokens(out) <= maxTokens {
		return out
	}

	out = truncateMessageText(out, maxTokens)
	if EstimateTokens(out) <= maxTokens {
		return out
	}

	out = dropImageContent(out)
	if EstimateTokens(out) <= maxTokens {
		return out
	}

	if len(out.ToolCalls) > 0 {
		for i := range out.ToolCalls {
			out.ToolCalls[i].Arguments = []byte(`{"_truncated":true}`)
		}
	}
	if EstimateTokens(out) <= maxTokens {
		return out
	}

	out.Text = "[truncated]"
	if len(out.Content) > 0 {
		for i := range out.Content {
			out.Content[i].Text = ""
			out.Content[i].Image = nil
		}
	}
	if EstimateTokens(out) > maxTokens {
		out.Content = nil
	}
	return out
}

func truncateMessageText(msg llm.Message, maxTokens int) llm.Message {
	estimated := EstimateTokens(msg)
	if estimated <= maxTokens {
		return msg
	}

	ratio := float64(maxTokens) / float64(estimated)
	if ratio <= 0 {
		ratio = 0.05
	}
	msg.Text = truncateByRatio(msg.Text, ratio)
	for i := range msg.Content {
		if strings.EqualFold(strings.TrimSpace(msg.Content[i].Type), llm.ContentTypeText) {
			msg.Content[i].Text = truncateByRatio(msg.Content[i].Text, ratio)
		}
	}
	return msg
}

func dropImageContent(msg llm.Message) llm.Message {
	if len(msg.Content) == 0 {
		return msg
	}
	filtered := make([]llm.ContentPart, 0, len(msg.Content))
	for _, part := range msg.Content {
		if strings.EqualFold(strings.TrimSpace(part.Type), llm.ContentTypeImage) {
			continue
		}
		filtered = append(filtered, part)
	}
	msg.Content = filtered
	return msg
}

func truncateByRatio(value string, ratio float64) string {
	if value == "" {
		return value
	}
	runes := []rune(value)
	limit := int(float64(len(runes)) * ratio)
	if limit <= 0 {
		limit = 1
	}
	if limit >= len(runes) {
		return value
	}
	return string(runes[:limit])
}

func contextWindowForProvider(provider llm.Provider, model string) int {
	if model != "" {
		for _, rule := range defaultModelContextRules {
			if strings.Contains(model, rule.pattern) {
				return rule.tokens
			}
		}
	}

	switch provider {
	case llm.ProviderAnthropic:
		return anthropicContextTokens
	case llm.ProviderOpenAI, llm.ProviderCodex:
		return openAIContextTokens
	default:
		return customContextTokens
	}
}

func responseReserveForProvider(provider llm.Provider) int {
	switch provider {
	case llm.ProviderAnthropic:
		return anthropicResponseReserve
	case llm.ProviderOpenAI, llm.ProviderCodex:
		return openAIResponseReserve
	default:
		return 0
	}
}

func matchModelTokenOverride(overrides map[string]int, model string) int {
	if len(overrides) == 0 || model == "" {
		return 0
	}
	if value, ok := overrides[model]; ok && value > 0 {
		return value
	}
	for key, value := range overrides {
		if value <= 0 || !strings.HasSuffix(key, "*") {
			continue
		}
		prefix := strings.TrimSuffix(key, "*")
		if prefix != "" && strings.HasPrefix(model, prefix) {
			return value
		}
	}
	return 0
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
