package storage

import (
	"fmt"
	"sort"
	"strings"
)

func NormalizeProviderHeaders(raw map[string]string) (map[string]string, error) {
	return NormalizeNamedHeaders(raw, "provider_headers")
}

func NormalizeResponseMetadata(raw map[string]string) (map[string]string, error) {
	if len(raw) == 0 {
		return nil, nil
	}

	out := make(map[string]string, len(raw))
	for key, value := range raw {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" {
			return nil, fmt.Errorf("invalid response_metadata: metadata key cannot be empty")
		}
		out[trimmedKey] = strings.TrimSpace(value)
	}
	return out, nil
}

func NormalizeNamedHeaders(raw map[string]string, fieldName string) (map[string]string, error) {
	if len(raw) == 0 {
		return nil, nil
	}

	out := make(map[string]string, len(raw))
	for key, value := range raw {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" {
			return nil, fmt.Errorf("invalid %s: header key cannot be empty", strings.TrimSpace(fieldName))
		}
		out[trimmedKey] = strings.TrimSpace(value)
	}
	return out, nil
}

func NormalizeOrigins(origins []string) []string {
	trimmed := make([]string, 0, len(origins))
	for _, origin := range origins {
		value := strings.TrimSpace(origin)
		if value == "" {
			continue
		}
		trimmed = append(trimmed, value)
	}
	return trimmed
}

func ParseOriginsCSV(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	return NormalizeOrigins(strings.Split(raw, ","))
}

func ParseStringCSV(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}

	values := make([]string, 0)
	for _, item := range strings.Split(raw, ",") {
		if value := strings.TrimSpace(item); value != "" {
			values = append(values, value)
		}
	}
	if len(values) == 0 {
		return nil
	}
	return values
}

func NormalizeStringList(raw []string) []string {
	seen := make(map[string]bool, len(raw))
	items := make([]string, 0, len(raw))
	for _, item := range raw {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" || seen[trimmed] {
			continue
		}
		seen[trimmed] = true
		items = append(items, trimmed)
	}
	return items
}

func NormalizeConfiguredNames(raw []string) []string {
	normalized := NormalizeStringList(raw)
	if len(normalized) == 0 {
		return nil
	}
	sort.Strings(normalized)
	return normalized
}
