package rss

import (
	"html"
	"regexp"
	"strings"
	"time"
)

var htmlTagPattern = regexp.MustCompile(`(?s)<[^>]+>`)

var supportedTimeLayouts = []string{
	time.RFC3339Nano,
	time.RFC3339,
	time.RFC1123Z,
	time.RFC1123,
	time.RFC822Z,
	time.RFC822,
	time.RFC850,
	"Mon, 02 Jan 2006 15:04:05 MST",
	"Mon, 2 Jan 2006 15:04:05 MST",
	"2006-01-02T15:04:05-07:00",
	"2006-01-02T15:04:05Z0700",
}

func cleanText(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	trimmed = html.UnescapeString(trimmed)
	trimmed = htmlTagPattern.ReplaceAllString(trimmed, " ")
	return strings.Join(strings.Fields(trimmed), " ")
}

func parsePublishedTime(values ...string) (time.Time, bool) {
	for _, raw := range values {
		text := strings.TrimSpace(raw)
		if text == "" {
			continue
		}
		for _, layout := range supportedTimeLayouts {
			parsed, err := time.Parse(layout, text)
			if err == nil {
				return parsed, true
			}
		}
	}
	return time.Time{}, false
}

func synthesizeID(values ...string) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			parts = append(parts, value)
		}
	}
	return strings.Join(parts, " | ")
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}
