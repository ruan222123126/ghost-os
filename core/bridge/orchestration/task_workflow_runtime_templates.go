package orchestration

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var workflowTemplatePattern = regexp.MustCompile(`\$\{([^{}]+)\}`)

func resolveWorkflowTemplateValue(value any, vars map[string]any) (any, error) {
	switch typed := value.(type) {
	case string:
		return resolveWorkflowTemplateString(typed, vars)
	case map[string]any:
		return resolveWorkflowTemplateMap(typed, vars)
	case []any:
		return resolveWorkflowTemplateArray(typed, vars)
	default:
		return value, nil
	}
}

func resolveWorkflowTemplateMap(input map[string]any, vars map[string]any) (map[string]any, error) {
	output := make(map[string]any, len(input))
	for key, value := range input {
		resolvedValue, err := resolveWorkflowTemplateValue(value, vars)
		if err != nil {
			return nil, err
		}
		output[key] = resolvedValue
	}
	return output, nil
}

func resolveWorkflowTemplateArray(input []any, vars map[string]any) ([]any, error) {
	output := make([]any, len(input))
	for index, value := range input {
		resolvedValue, err := resolveWorkflowTemplateValue(value, vars)
		if err != nil {
			return nil, err
		}
		output[index] = resolvedValue
	}
	return output, nil
}

func resolveWorkflowTemplateString(input string, vars map[string]any) (any, error) {
	matches := workflowTemplatePattern.FindAllStringSubmatchIndex(input, -1)
	if len(matches) == 0 {
		return input, nil
	}
	if isWorkflowTemplateWholeString(input, matches[0]) && len(matches) == 1 {
		path := input[matches[0][2]:matches[0][3]]
		return resolveWorkflowTemplatePath(path, vars)
	}
	return resolveWorkflowTemplateCompositeString(input, matches, vars)
}

func isWorkflowTemplateWholeString(input string, match []int) bool {
	return len(match) >= 2 && match[0] == 0 && match[1] == len(input)
}

func resolveWorkflowTemplateCompositeString(
	input string,
	matches [][]int,
	vars map[string]any,
) (string, error) {
	var builder strings.Builder
	lastIndex := 0
	for _, match := range matches {
		builder.WriteString(input[lastIndex:match[0]])
		path := input[match[2]:match[3]]
		value, err := resolveWorkflowTemplatePath(path, vars)
		if err != nil {
			return "", err
		}
		builder.WriteString(stringifyWorkflowTemplateValue(value))
		lastIndex = match[1]
	}
	builder.WriteString(input[lastIndex:])
	return builder.String(), nil
}

func resolveWorkflowTemplatePath(path string, vars map[string]any) (any, error) {
	normalizedPath := strings.TrimSpace(path)
	if normalizedPath == "" {
		return nil, fmt.Errorf("workflow template variable path is empty")
	}
	segments := strings.Split(normalizedPath, ".")
	current := any(vars)
	for _, segment := range segments {
		key := strings.TrimSpace(segment)
		if key == "" {
			return nil, fmt.Errorf("workflow template variable path %q is invalid", normalizedPath)
		}
		next, ok := readWorkflowTemplateSegment(current, key)
		if !ok {
			return nil, fmt.Errorf("workflow template variable %q is not defined", normalizedPath)
		}
		current = next
	}
	return current, nil
}

func readWorkflowTemplateSegment(current any, segment string) (any, bool) {
	if values, ok := current.(map[string]any); ok {
		value, exists := values[segment]
		return value, exists
	}
	if values, ok := current.([]any); ok {
		index, err := strconv.Atoi(segment)
		if err != nil || index < 0 || index >= len(values) {
			return nil, false
		}
		return values[index], true
	}
	return nil, false
}

func stringifyWorkflowTemplateValue(value any) string {
	if value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return typed
	case bool:
		if typed {
			return "true"
		}
		return "false"
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return fmt.Sprintf("%v", value)
	}
	return string(encoded)
}
