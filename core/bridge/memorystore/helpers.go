package memorystore

import (
	"strings"
	"unicode"
)

const maxSummaryLength = 240

func normalizeScopeType(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case ScopeTypeSession:
		return ScopeTypeSession
	case ScopeTypeUser:
		return ScopeTypeUser
	default:
		return ""
	}
}

func normalizeMemoryType(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case MemoryTypeProfile:
		return MemoryTypeProfile
	case MemoryTypePreference:
		return MemoryTypePreference
	case MemoryTypeWorkflow:
		return MemoryTypeWorkflow
	case MemoryTypeFact:
		return MemoryTypeFact
	default:
		return ""
	}
}

func resolveMemoryType(raw string) string {
	memoryType := normalizeMemoryType(raw)
	if memoryType == "" {
		return MemoryTypeFact
	}
	return memoryType
}

func normalizeStatusList(values []string) []string {
	out := make([]string, 0, len(values))
	seen := make(map[string]bool, len(values))
	for _, value := range values {
		status := strings.ToLower(strings.TrimSpace(value))
		if status == "" || seen[status] {
			continue
		}
		if status != MemoryStatusActive && status != MemoryStatusSuperseded && status != MemoryStatusDeleted {
			continue
		}
		seen[status] = true
		out = append(out, status)
	}
	return out
}

func normalizeConfidence(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}

func buildSearchTerms(query string) []string {
	normalized := normalizeSearchText(query)
	if normalized == "" {
		return nil
	}
	parts := strings.Fields(normalized)
	if len(parts) == 0 {
		return nil
	}
	terms := make([]string, 0, minInt(len(parts)+1, 7))
	terms = append(terms, normalized)
	for _, part := range parts {
		if len(terms) >= 7 {
			break
		}
		terms = append(terms, part)
	}
	return normalizeIdentifierList(terms)
}

func normalizeSearchText(raw string) string {
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

func summarizeText(preferred string, fallback string) string {
	text := strings.TrimSpace(preferred)
	if text == "" {
		text = strings.TrimSpace(fallback)
	}
	if len(text) <= maxSummaryLength {
		return text
	}
	return strings.TrimSpace(text[:maxSummaryLength-3]) + "..."
}

func minInt(left int, right int) int {
	if left < right {
		return left
	}
	return right
}
