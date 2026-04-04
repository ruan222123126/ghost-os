package tools

import "strings"

const toolTagResultMarker = "[TOOL_TAG_RESULT]"

type graphQLTextNormalizationResult struct {
	Document      string
	Recognized    bool
	LegacyGraphQL bool
	SanitizeKinds []string
}

func normalizeGraphQLTextDocument(
	text string,
	sanitizeKnownArtifacts bool,
) graphQLTextNormalizationResult {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return graphQLTextNormalizationResult{}
	}

	kinds := make([]string, 0, 2)
	if trimmed != text {
		kinds = append(kinds, "trim_whitespace")
	}
	if sanitizeKnownArtifacts {
		if stripped, ok := stripToolTagResultSuffix(trimmed); ok {
			trimmed = stripped
			kinds = append(kinds, "strip_internal_feedback_suffix")
		}
	}
	if looksLikeTaggedToolCall(trimmed) {
		return graphQLTextNormalizationResult{
			Document:      trimmed,
			Recognized:    true,
			SanitizeKinds: kinds,
		}
	}
	if looksLikeLegacyGraphQLToolCall(trimmed) {
		return graphQLTextNormalizationResult{
			Document:      trimmed,
			Recognized:    true,
			LegacyGraphQL: true,
			SanitizeKinds: kinds,
		}
	}
	return graphQLTextNormalizationResult{}
}

func stripToolTagResultSuffix(text string) (string, bool) {
	index := strings.Index(text, toolTagResultMarker)
	if index < 0 {
		return text, false
	}
	prefix := strings.TrimSpace(text[:index])
	if prefix == "" {
		return text, false
	}
	return prefix, true
}

func looksLikeTaggedToolCall(text string) bool {
	return strings.Contains(strings.TrimSpace(text), "<t:")
}

func looksLikeLegacyGraphQLToolCall(text string) bool {
	trimmed := strings.ToLower(strings.TrimSpace(text))
	return hasGraphQLKeywordPrefix(trimmed, "mutation") || hasGraphQLKeywordPrefix(trimmed, "query")
}

func hasGraphQLKeywordPrefix(text string, keyword string) bool {
	if !strings.HasPrefix(text, keyword) {
		return false
	}
	if len(text) == len(keyword) {
		return true
	}
	next := text[len(keyword)]
	return next == '{' || next == ' ' || next == '\n' || next == '\t' || next == '\r'
}
