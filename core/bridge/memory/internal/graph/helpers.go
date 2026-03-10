package graph

import (
	"fmt"
	"math"
	"strings"

	"ghost-os/bridge/memory/internal/pathutil"
)

const (
	MemoryAnchorPreference = "preference"
	MemoryAnchorAvoidance  = "avoidance"
	MemoryAnchorEmotion    = "emotion"
	MemoryAnchorConstraint = "constraint"
	MemoryAnchorIdentity   = "identity"
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

func normalizeAnchor(anchor MemoryAnchor) MemoryAnchor {
	out := anchor
	out.Type = strings.ToLower(strings.TrimSpace(out.Type))
	out.Key = strings.TrimSpace(out.Key)
	out.Value = strings.TrimSpace(out.Value)
	out.Reason = strings.TrimSpace(out.Reason)
	out.SessionID = strings.TrimSpace(out.SessionID)
	out.Weight = clamp01(out.Weight)
	if !out.DetectedAt.IsZero() {
		out.DetectedAt = out.DetectedAt.UTC()
	}
	if !out.ExpiresAt.IsZero() {
		out.ExpiresAt = out.ExpiresAt.UTC()
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

func ensurePath(value string) (string, error) {
	resolved := pathutil.Resolve(value)
	if resolved == "" {
		return "", fmt.Errorf("empty path")
	}
	return resolved, nil
}
