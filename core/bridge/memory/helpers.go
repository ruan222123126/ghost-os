package memory

import "strings"

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

	stopWords := map[string]struct{}{
		"the": {}, "and": {}, "for": {}, "with": {}, "that": {}, "this": {}, "from": {}, "into": {}, "what": {}, "how": {}, "why": {}, "are": {}, "you": {}, "is": {}, "was": {}, "were": {}, "have": {}, "has": {},
	}

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
