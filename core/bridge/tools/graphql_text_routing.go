package tools

import (
	"regexp"
	"strings"
)

var graphQLTextFencePattern = regexp.MustCompile("(?is)```(?:graphql|gql)?\\s*([\\s\\S]*?)```")

const graphQLToolResultMarker = "[GRAPHQL_TOOL_RESULT]"

type graphQLTextNormalizationResult struct {
	Document      string
	Recognized    bool
	SanitizeKinds []string
}

func extractGraphQLFenceContent(text string) string {
	match := graphQLTextFencePattern.FindStringSubmatch(text)
	if len(match) != 2 {
		return ""
	}
	return strings.TrimSpace(match[1])
}

func normalizeGraphQLTextDocument(
	text string,
	sanitizeKnownArtifacts bool,
) graphQLTextNormalizationResult {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return graphQLTextNormalizationResult{}
	}

	kinds := make([]string, 0, 3)
	if trimmed != text {
		kinds = append(kinds, "trim_whitespace")
	}
	if fenced := extractGraphQLFenceContent(trimmed); fenced != "" {
		if fenced != trimmed {
			kinds = append(kinds, "strip_code_fence")
		}
		trimmed = fenced
	}
	if !looksLikeGraphQLDocument(trimmed) {
		return graphQLTextNormalizationResult{}
	}
	if sanitizeKnownArtifacts {
		if stripped, ok := stripGraphQLToolResultSuffix(trimmed); ok {
			trimmed = stripped
			kinds = append(kinds, "strip_graphql_tool_result_suffix")
		}
	}
	return graphQLTextNormalizationResult{
		Document:      trimmed,
		Recognized:    true,
		SanitizeKinds: kinds,
	}
}

func stripGraphQLToolResultSuffix(text string) (string, bool) {
	index := strings.Index(text, graphQLToolResultMarker)
	if index < 0 {
		return text, false
	}
	prefix := strings.TrimSpace(text[:index])
	if prefix == "" || !looksLikeGraphQLDocument(prefix) || !strings.HasSuffix(prefix, "}") {
		return text, false
	}
	return prefix, true
}

func looksLikeGraphQLDocument(text string) bool {
	trimmed := strings.TrimSpace(strings.ToLower(text))
	return strings.HasPrefix(trimmed, "{") ||
		strings.HasPrefix(trimmed, "query") ||
		strings.HasPrefix(trimmed, "mutation")
}
