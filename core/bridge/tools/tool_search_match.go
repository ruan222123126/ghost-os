package tools

import (
	"regexp"
	"strings"
)

const minToolSearchTermLen = 2

var toolSearchWordSplitPattern = regexp.MustCompile(`[^a-z0-9]+`)

var toolSearchStopWords = map[string]bool{
	"a":            true,
	"an":           true,
	"and":          true,
	"availability": true,
	"available":    true,
	"can":          true,
	"check":        true,
	"current":      true,
	"currently":    true,
	"find":         true,
	"for":          true,
	"in":           true,
	"is":           true,
	"look":         true,
	"of":           true,
	"only":         true,
	"or":           true,
	"the":          true,
	"tool":         true,
	"tools":        true,
	"use":          true,
	"using":        true,
	"visible":      true,
}

func toolMatchesQuery(metadata ToolMetadata, query string) bool {
	normalizedQuery := normalizeToolSearchText(query)
	if normalizedQuery == "" {
		return true
	}
	document := toolSearchDocument(metadata)
	if strings.Contains(document, normalizedQuery) {
		return true
	}
	terms := toolSearchQueryTerms(normalizedQuery)
	if len(terms) == 0 {
		return false
	}
	if toolSearchPrimaryFieldMatch(metadata, terms) {
		return true
	}
	return toolSearchDocumentMatchCount(document, terms) >= 2
}

func toolSearchDocument(metadata ToolMetadata) string {
	parts := make([]string, 0, len(metadata.Tags)+2)
	parts = append(parts, normalizeToolSearchText(metadata.Name))
	parts = append(parts, normalizeToolSearchText(metadata.ShortDesc))
	for _, tag := range metadata.Tags {
		parts = append(parts, normalizeToolSearchText(tag))
	}
	return strings.Join(parts, " ")
}

func toolSearchPrimaryFieldMatch(metadata ToolMetadata, terms []string) bool {
	name := normalizeToolSearchText(metadata.Name)
	tags := normalizeToolSearchText(strings.Join(metadata.Tags, " "))
	for _, term := range terms {
		if strings.Contains(name, term) || strings.Contains(tags, term) {
			return true
		}
	}
	return false
}

func toolSearchQueryTerms(query string) []string {
	words := strings.Fields(query)
	terms := make([]string, 0, len(words))
	seen := make(map[string]bool, len(words))
	for _, word := range words {
		if len(word) < minToolSearchTermLen || toolSearchStopWords[word] || seen[word] {
			continue
		}
		seen[word] = true
		terms = append(terms, word)
	}
	return terms
}

func toolSearchDocumentMatchCount(document string, terms []string) int {
	count := 0
	for _, term := range terms {
		if strings.Contains(document, term) {
			count++
		}
	}
	return count
}

func normalizeToolSearchText(raw string) string {
	trimmed := strings.ToLower(strings.TrimSpace(raw))
	if trimmed == "" {
		return ""
	}
	normalized := toolSearchWordSplitPattern.ReplaceAllString(trimmed, " ")
	return strings.Join(strings.Fields(normalized), " ")
}
