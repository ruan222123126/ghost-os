package skills

import "strings"

func WithUpdatedBlocklist(raw []string, skillID string, enabled bool) []string {
	items := normalizeStringList(raw)
	if !enabled {
		return normalizeStringList(append(items, skillID))
	}

	filtered := make([]string, 0, len(items))
	for _, item := range items {
		if item != skillID {
			filtered = append(filtered, item)
		}
	}
	return normalizeStringList(filtered)
}

func normalizeStringList(raw []string) []string {
	out := make([]string, 0, len(raw))
	seen := make(map[string]bool, len(raw))
	for _, item := range raw {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" || seen[trimmed] {
			continue
		}
		seen[trimmed] = true
		out = append(out, trimmed)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
