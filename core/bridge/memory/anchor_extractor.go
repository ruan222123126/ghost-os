package memory

import (
	"regexp"
	"strings"
	"time"

	"ghost-os/bridge/llm"
)

type anchorPattern struct {
	anchorType string
	weight     float64
	pattern    *regexp.Regexp
}

var anchorPatterns = []anchorPattern{
	{anchorType: MemoryAnchorPreference, weight: 0.78, pattern: regexp.MustCompile(`(?i)(?:\bi\s+(?:like|love|prefer)\b|\bprefer\b|以后都用|以后用|都用|请用)\s*[:：-]?\s*([^\n.!?;，。]+)`)},
	{anchorType: MemoryAnchorAvoidance, weight: 0.88, pattern: regexp.MustCompile(`(?i)(?:\bdon't\b|\bdo not\b|\bnever\b|\bavoid\b|\bdislike\b|\bhate\b|不要再|不要|别再|最讨厌)\s*[:：-]?\s*([^\n.!?;，。]+)`)},
	{anchorType: MemoryAnchorConstraint, weight: 0.9, pattern: regexp.MustCompile(`(?i)(?:\bmust\b|\bneed to\b|\brequired\b|必须|务必|一定要|不能)\s*[:：-]?\s*([^\n.!?;，。]+)`)},
	{anchorType: MemoryAnchorIdentity, weight: 0.76, pattern: regexp.MustCompile(`(?i)(?:\bi am\b|\bi'm\b|\bmy role is\b|我是|我负责|我通常|我习惯)\s*[:：-]?\s*([^\n.!?;，。]+)`)},
	{anchorType: MemoryAnchorEmotion, weight: 0.72, pattern: regexp.MustCompile(`(?i)(?:\bi(?:'m| am)?\s+(?:frustrated|annoyed|upset|happy)\b|\bi\s+(?:hate|love)\b|痛点是|太棒了|很烦|很满意|很不满)\s*[:：-]?\s*([^\n.!?;，。]+)`)},
}

func extractAnchors(messages []llm.Message, sessionID string, detectedAt time.Time, minWeight float64, worker AnchorExtractor, useWorker bool) []MemoryAnchor {
	ruleAnchors := extractRuleAnchors(messages, sessionID, detectedAt)
	merged := cloneAnchors(ruleAnchors)
	if useWorker && worker != nil {
		if workerAnchors, err := worker.ExtractAnchors(messages); err == nil {
			for _, anchor := range workerAnchors {
				normalized := normalizeAnchor(anchor)
				if normalized.SessionID == "" {
					normalized.SessionID = strings.TrimSpace(sessionID)
				}
				if normalized.DetectedAt.IsZero() {
					normalized.DetectedAt = detectedAt.UTC()
				}
				if normalized.ExpiresAt.IsZero() {
					normalized.ExpiresAt = defaultAnchorExpiry(normalized.Type, normalized.DetectedAt)
				}
				merged = append(merged, normalized)
			}
		}
	}
	return filterAnchorsByWeight(merged, minWeight)
}

func extractRuleAnchors(messages []llm.Message, sessionID string, detectedAt time.Time) []MemoryAnchor {
	anchors := make([]MemoryAnchor, 0, len(messages))
	for _, msg := range messages {
		text := strings.TrimSpace(messageToContent(msg))
		if text == "" {
			continue
		}
		for _, pattern := range anchorPatterns {
			matches := pattern.pattern.FindAllStringSubmatch(text, -1)
			for _, match := range matches {
				if len(match) < 2 {
					continue
				}
				rawValue := cleanAnchorValue(match[1])
				if rawValue == "" {
					continue
				}
				key, value := inferAnchorKeyValue(pattern.anchorType, rawValue)
				if value == "" {
					continue
				}
				anchors = append(anchors, MemoryAnchor{
					Type:       pattern.anchorType,
					Key:        key,
					Value:      value,
					Weight:     pattern.weight,
					Reason:     summarizeLine(text, 180),
					DetectedAt: detectedAt.UTC(),
					ExpiresAt:  defaultAnchorExpiry(pattern.anchorType, detectedAt.UTC()),
					SessionID:  strings.TrimSpace(sessionID),
				})
			}
		}
	}
	return normalizeAnchors(anchors)
}

func defaultAnchorExpiry(anchorType string, detectedAt time.Time) time.Time {
	base := detectedAt.UTC()
	if base.IsZero() {
		base = time.Now().UTC()
	}
	switch strings.ToLower(strings.TrimSpace(anchorType)) {
	case MemoryAnchorEmotion:
		return base.Add(72 * time.Hour)
	case MemoryAnchorPreference:
		return base.Add(180 * 24 * time.Hour)
	case MemoryAnchorAvoidance, MemoryAnchorConstraint, MemoryAnchorIdentity:
		return base.Add(365 * 24 * time.Hour)
	default:
		return base.Add(30 * 24 * time.Hour)
	}
}

func cleanAnchorValue(raw string) string {
	trimmed := strings.TrimSpace(raw)
	trimmed = strings.Trim(trimmed, " \t\n\r,.;:!?，。！？；：\"'“”‘’()[]{}")
	return strings.Join(strings.Fields(trimmed), " ")
}

func inferAnchorKeyValue(anchorType string, raw string) (string, string) {
	lower := strings.ToLower(strings.TrimSpace(raw))
	switch {
	case strings.Contains(lower, "python"):
		return "language", "Python"
	case strings.Contains(lower, "golang") || lower == "go" || strings.HasPrefix(lower, "go ") || strings.Contains(lower, " go ") || strings.HasSuffix(lower, " go"):
		return "language", "Go"
	case strings.Contains(lower, "rust"):
		return "language", "Rust"
	case strings.Contains(lower, "java"):
		return "language", "Java"
	case strings.Contains(lower, "typescript") || lower == "ts" || strings.HasPrefix(lower, "ts ") || strings.Contains(lower, " ts ") || strings.HasSuffix(lower, " ts"):
		return "language", "TypeScript"
	case strings.Contains(lower, "javascript") || lower == "js" || strings.HasPrefix(lower, "js ") || strings.Contains(lower, " js ") || strings.HasSuffix(lower, " js"):
		return "language", "JavaScript"
	case strings.Contains(lower, "dark") || strings.Contains(lower, "深色"):
		return "ui_theme", "dark"
	case strings.Contains(lower, "light") || strings.Contains(lower, "浅色"):
		return "ui_theme", "light"
	case strings.Contains(lower, "concise") || strings.Contains(lower, "brief") || strings.Contains(lower, "简洁"):
		return "communication_style", "concise"
	case strings.Contains(lower, "detailed") || strings.Contains(lower, "详细"):
		return "communication_style", "detailed"
	case strings.Contains(lower, "production") || strings.Contains(lower, "prod") || strings.Contains(lower, "生产"):
		return "environment", cleanAnchorValue(raw)
	case strings.Contains(lower, "database") || strings.Contains(lower, "db") || strings.Contains(lower, "数据库"):
		return "database", cleanAnchorValue(raw)
	}
	defaultKey := anchorType
	if defaultKey == "" {
		defaultKey = "memory"
	}
	return defaultKey, cleanAnchorValue(raw)
}
