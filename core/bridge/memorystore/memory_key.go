package memorystore

import (
	"strings"
	"unicode"
)

const maxMemoryKeyLength = 96

func normalizeMemoryKey(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	var builder strings.Builder
	builder.Grow(len(trimmed))
	lastUnderscore := false
	for _, r := range strings.ToLower(trimmed) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			builder.WriteRune(r)
			lastUnderscore = false
			continue
		}
		if lastUnderscore {
			continue
		}
		builder.WriteByte('_')
		lastUnderscore = true
	}
	value := strings.Trim(builder.String(), "_")
	if len(value) <= maxMemoryKeyLength {
		return value
	}
	return strings.Trim(value[:maxMemoryKeyLength], "_")
}

func defaultMemoryKey(memoryType string, summary string) string {
	normalizedType := resolveMemoryType(memoryType)
	normalizedSummary := normalizeMemoryKey(summary)
	if normalizedSummary == "" {
		return ""
	}
	return normalizedType + ":" + normalizedSummary
}

func NormalizeMemoryKey(raw string) string {
	return normalizeMemoryKey(raw)
}

func DefaultMemoryKey(memoryType string, summary string) string {
	return defaultMemoryKey(memoryType, summary)
}
