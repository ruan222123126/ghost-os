package config

import (
	"fmt"
	"strings"
)

func normalizeProviderHeaders(raw map[string]string) (map[string]string, error) {
	return normalizeNamedHeaders(raw, "provider_headers")
}

func normalizeGraphQLHeaders(raw map[string]string) (map[string]string, error) {
	return normalizeNamedHeaders(raw, "graphql_headers")
}

func normalizeResponseMetadata(raw map[string]string) (map[string]string, error) {
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

func normalizeNamedHeaders(raw map[string]string, fieldName string) (map[string]string, error) {
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

func normalizeOrigins(origins []string) []string {
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

func parseOriginsCSV(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	return normalizeOrigins(strings.Split(raw, ","))
}

func parseStringCSV(raw string) []string {
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

func normalizeToolPromptOverrides(raw map[string]string) (map[string]string, error) {
	if len(raw) == 0 {
		return nil, nil
	}

	validNames := validConfiguredToolNames()
	out := make(map[string]string, len(raw))
	for key, value := range raw {
		name := strings.TrimSpace(key)
		if name == "" {
			return nil, fmt.Errorf("invalid tool_prompt_overrides: tool name cannot be empty")
		}
		if !validNames[name] {
			return nil, fmt.Errorf("unknown tool in tool_prompt_overrides: %s", name)
		}

		prompt := strings.TrimSpace(value)
		if prompt == "" {
			continue
		}
		out[name] = prompt
	}

	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}
