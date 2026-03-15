package memoryaug

import (
	"fmt"
	"strings"
	"unicode"

	"ghost-os/bridge/memorystore"
)

var lowSignalTexts = map[string]bool{
	"hi": true, "hello": true, "thanks": true, "thank you": true, "ok": true,
	"okay": true, "yes": true, "no": true, "好的": true, "谢谢": true, "行": true,
}

func filterTurnMessages(messages []TurnMessage) []TurnMessage {
	filtered := make([]TurnMessage, 0, len(messages))
	for _, message := range messages {
		if !isSupportedLearningRole(message.Role) {
			continue
		}
		text := strings.TrimSpace(message.Text)
		if text == "" || isLowSignalMessage(text) || looksLikeEphemeralAssistantOutput(message.Role, text) {
			continue
		}
		filtered = append(filtered, TurnMessage{
			Role: normalizeRole(message.Role),
			Text: text,
		})
	}
	return filtered
}

func buildTranscriptText(messages []TurnMessage) string {
	lines := make([]string, 0, len(messages))
	for _, message := range messages {
		lines = append(lines, fmt.Sprintf("%s: %s", normalizeRole(message.Role), strings.TrimSpace(message.Text)))
	}
	return strings.Join(lines, "\n")
}

func isLowSignalMessage(text string) bool {
	normalized := normalizeText(text)
	if normalized == "" {
		return true
	}
	if lowSignalTexts[normalized] {
		return true
	}
	return len(strings.Fields(normalized)) <= 2 && len([]rune(normalized)) <= 8
}

func looksLikeEphemeralAssistantOutput(role string, text string) bool {
	if normalizeRole(role) != "assistant" {
		return false
	}
	lower := strings.ToLower(strings.TrimSpace(text))
	return strings.Contains(lower, "```") ||
		strings.Contains(lower, "trace_id=") ||
		strings.Contains(lower, "exit code") ||
		strings.Contains(lower, "applied patch") ||
		strings.Contains(lower, "$ ")
}

func computeTextScore(query string, entry memorystore.MemoryEntry) float64 {
	terms := textTerms(query)
	if len(terms) == 0 {
		return 0
	}
	text := normalizeText(entry.Summary + " " + entry.Content + " " + entry.ID)
	if text == "" {
		return 0
	}
	score := 0.0
	for _, term := range terms {
		if strings.Contains(text, term) {
			score += 1
		}
	}
	return score / float64(len(terms))
}

func memorySimilarity(left memorystore.MemoryEntry, right memorystore.MemoryEntry) float64 {
	leftText := normalizeText(left.Summary + " " + left.Content)
	rightText := normalizeText(right.Summary + " " + right.Content)
	if leftText == "" || rightText == "" {
		return 0
	}
	if leftText == rightText {
		return 1
	}
	leftTerms := textTerms(leftText)
	rightTerms := textTerms(rightText)
	if len(leftTerms) == 0 || len(rightTerms) == 0 {
		return 0
	}
	overlap := 0
	set := make(map[string]bool, len(leftTerms))
	for _, term := range leftTerms {
		set[term] = true
	}
	for _, term := range rightTerms {
		if set[term] {
			overlap++
		}
	}
	denominator := len(leftTerms) + len(rightTerms) - overlap
	if denominator <= 0 {
		return 0
	}
	return float64(overlap) / float64(denominator)
}

func normalizeText(raw string) string {
	var builder strings.Builder
	builder.Grow(len(raw))
	for _, r := range strings.ToLower(raw) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			builder.WriteRune(r)
			continue
		}
		builder.WriteByte(' ')
	}
	return strings.Join(strings.Fields(builder.String()), " ")
}

func textTerms(raw string) []string {
	normalized := normalizeText(raw)
	if normalized == "" {
		return nil
	}
	return strings.Fields(normalized)
}

func normalizeIDs(values []string) []string {
	out := make([]string, 0, len(values))
	seen := make(map[string]bool, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" || seen[trimmed] {
			continue
		}
		seen[trimmed] = true
		out = append(out, trimmed)
	}
	return out
}

func normalizeRole(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "assistant":
		return "assistant"
	default:
		return "user"
	}
}

func isSupportedLearningRole(raw string) bool {
	role := normalizeRole(raw)
	return role == "user" || role == "assistant"
}
