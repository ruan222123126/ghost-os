package memory

import "strings"

func normalizeGraphNamespace(namespace string) string {
	trimmed := strings.TrimSpace(namespace)
	if trimmed == "" {
		return defaultGraphNamespace
	}
	return trimmed
}
