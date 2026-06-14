package tools

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

var (
	codexSessionIDPatternsOnce sync.Once
	codexSessionIDPatterns     []*regexp.Regexp
)

func parseCodexCLISessionID(line string) string {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return ""
	}
	if value := parseCodexCLIJSONSessionID(trimmed); value != "" {
		return value
	}
	return parseCodexCLITextSessionID(trimmed)
}

func parseCodexCLISessionIDFromOutput(output string) string {
	if strings.TrimSpace(output) == "" {
		return ""
	}
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if sessionID := parseCodexCLISessionID(line); sessionID != "" {
			return sessionID
		}
	}
	return ""
}

func parseCodexCLIJSONSessionID(trimmed string) string {
	if !strings.HasPrefix(trimmed, "{") || !strings.HasSuffix(trimmed, "}") {
		return ""
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(trimmed), &payload); err != nil {
		return ""
	}
	for _, field := range []string{"session_id", "sessionId", "thread_id", "threadId"} {
		raw, ok := payload[field]
		if !ok || raw == nil {
			continue
		}
		value, ok := raw.(string)
		if !ok {
			continue
		}
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func parseCodexCLITextSessionID(trimmed string) string {
	for _, pattern := range codexCLITextSessionPatterns() {
		matches := pattern.FindStringSubmatch(trimmed)
		if len(matches) < 2 {
			continue
		}
		value := strings.TrimSpace(matches[1])
		if value != "" {
			return value
		}
	}
	return ""
}

func codexCLITextSessionPatterns() []*regexp.Regexp {
	codexSessionIDPatternsOnce.Do(func() {
		codexSessionIDPatterns = []*regexp.Regexp{
			regexp.MustCompile(`(?i)session[_\s-]*id[:=]\s*([A-Za-z0-9_-]{6,})`),
			regexp.MustCompile(`(?i)session\s+id[:=]\s*([A-Za-z0-9_-]{6,})`),
			regexp.MustCompile(`(?i)session\s+([0-9a-fA-F-]{8,})`),
		}
	})
	return codexSessionIDPatterns
}

func formatUint(value uint64) string {
	return strconv.FormatUint(value, 10)
}
