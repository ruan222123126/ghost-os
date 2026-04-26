package session

import (
	"strings"

	"ghost-os/bridge/llm"
)

const (
	emptyMessageTokenCost  = 4
	messageTokenOverhead   = 4
	structuredTokenDivisor = 2
	plainTextTokenDivisor  = 4
	imageTokenOverhead     = 500
	minimumEstimatedTokens = 1
)

var structuredContentMarkers = []string{
	"{", "}", "[", "]", "=>", "::", "func ", "package ", "import ", "SELECT ", "INSERT ", "\"tool\"",
}

type tokenEstimator struct{}

func (tokenEstimator) Estimate(msg llm.Message) int {
	text := collectMessageText(msg)
	if text == "" {
		return emptyMessageTokenCost
	}
	estimated := estimateTextTokens(text)
	estimated += estimateImageOverhead(msg.Content)
	return estimated + messageTokenOverhead
}

func collectMessageText(msg llm.Message) string {
	var builder strings.Builder
	appendTrimmed(&builder, msg.Text)
	appendTrimmed(&builder, string(msg.ReasoningContent))
	appendContentText(&builder, msg.Content)
	appendTrimmed(&builder, msg.ToolCallID)
	appendToolCalls(&builder, msg.ToolCalls)
	return builder.String()
}

func appendTrimmed(builder *strings.Builder, value string) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return
	}
	builder.WriteString(trimmed)
}

func appendContentText(builder *strings.Builder, content []llm.ContentPart) {
	for _, part := range content {
		switch strings.ToLower(strings.TrimSpace(part.Type)) {
		case llm.ContentTypeText:
			builder.WriteString(part.Text)
		case llm.ContentTypeImage:
			builder.WriteString(imageDescriptorText(part.Image))
		}
	}
}

func imageDescriptorText(image *llm.ImageContent) string {
	if image == nil {
		return ""
	}
	return image.Path + image.URL + image.MimeType + image.SHA256
}

func appendToolCalls(builder *strings.Builder, calls []llm.ToolCall) {
	for _, call := range calls {
		builder.WriteString(call.ID)
		builder.WriteString(call.Name)
		builder.WriteString(string(call.Arguments))
	}
}

func estimateTextTokens(text string) int {
	charCount := len(text)
	divisor := plainTextTokenDivisor
	if containsStructuredContent(text) {
		divisor = structuredTokenDivisor
	}
	estimated := charCount / divisor
	if estimated < minimumEstimatedTokens {
		return minimumEstimatedTokens
	}
	return estimated
}

func estimateImageOverhead(content []llm.ContentPart) int {
	total := 0
	for _, part := range content {
		if strings.EqualFold(strings.TrimSpace(part.Type), llm.ContentTypeImage) {
			total += imageTokenOverhead
		}
	}
	return total
}

func containsStructuredContent(text string) bool {
	if strings.Contains(text, "```") {
		return true
	}
	for _, marker := range structuredContentMarkers {
		if strings.Contains(text, marker) {
			return true
		}
	}
	return false
}
