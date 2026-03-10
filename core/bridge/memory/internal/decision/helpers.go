package decision

import (
	"encoding/json"
	"math"
	"strings"

	"ghost-os/bridge/llm"
)

func clamp01(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func max(a int, b int) int {
	if a >= b {
		return a
	}
	return b
}

func maxFloat(a, b float64) float64 {
	if a >= b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a <= b {
		return a
	}
	return b
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func uniqueStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}

func summarizeLine(content string, maxLen int) string {
	trimmed := strings.TrimSpace(content)
	if len(trimmed) <= maxLen {
		return trimmed
	}
	if maxLen <= 3 {
		return trimmed[:maxLen]
	}
	return trimmed[:maxLen-3] + "..."
}

func extractKeywords(input string) []string {
	words := strings.Fields(strings.ToLower(strings.TrimSpace(input)))
	if len(words) == 0 {
		return nil
	}
	stopWords := map[string]struct{}{"the": {}, "and": {}, "for": {}, "with": {}, "that": {}, "this": {}, "from": {}, "into": {}, "what": {}, "how": {}, "why": {}, "are": {}, "you": {}, "is": {}, "was": {}, "were": {}, "have": {}, "has": {}}
	seen := make(map[string]struct{}, len(words))
	out := make([]string, 0, 8)
	for _, word := range words {
		cleaned := strings.Trim(word, ",.!?:;()[]{}\"'")
		if len(cleaned) < 3 {
			continue
		}
		if _, isStop := stopWords[cleaned]; isStop {
			continue
		}
		if _, ok := seen[cleaned]; ok {
			continue
		}
		seen[cleaned] = struct{}{}
		out = append(out, cleaned)
		if len(out) >= 8 {
			break
		}
	}
	return out
}

func queryTerms(query MemoryQuery) []string {
	terms := make([]string, 0, len(query.Keywords)+4)
	seen := make(map[string]struct{}, len(query.Keywords)+4)
	appendTerm := func(value string) {
		trimmed := strings.ToLower(strings.TrimSpace(value))
		if trimmed == "" {
			return
		}
		if _, ok := seen[trimmed]; ok {
			return
		}
		seen[trimmed] = struct{}{}
		terms = append(terms, trimmed)
	}
	for _, keyword := range query.Keywords {
		appendTerm(keyword)
	}
	for _, keyword := range extractKeywords(query.SemanticQuery) {
		appendTerm(keyword)
	}
	return terms
}

func normalizeRecallText(text string) string {
	normalized := strings.ToLower(strings.TrimSpace(text))
	if normalized == "" {
		return ""
	}
	return strings.Join(strings.Fields(normalized), " ")
}

func messageToContent(msg llm.Message) string {
	if text := strings.TrimSpace(msg.Text); text != "" {
		return text
	}
	if len(msg.ToolCalls) == 0 {
		return ""
	}
	encoded, err := json.Marshal(msg.ToolCalls)
	if err != nil {
		return ""
	}
	return string(encoded)
}
