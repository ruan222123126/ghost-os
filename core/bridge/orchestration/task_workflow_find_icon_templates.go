package orchestration

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var workflowFindIconTemplatePattern = regexp.MustCompile(`\$\{([^{}]+)\}`)

func resolveWorkflowFindIconTemplateValue(value any, findIcon any) (any, error) {
	switch typed := value.(type) {
	case string:
		return resolveWorkflowFindIconTemplateString(typed, findIcon)
	case map[string]any:
		return resolveWorkflowFindIconTemplateMap(typed, findIcon)
	case []any:
		return resolveWorkflowFindIconTemplateArray(typed, findIcon)
	default:
		return value, nil
	}
}

func resolveWorkflowFindIconTemplateMap(input map[string]any, findIcon any) (map[string]any, error) {
	output := make(map[string]any, len(input))
	for key, value := range input {
		resolvedValue, err := resolveWorkflowFindIconTemplateValue(value, findIcon)
		if err != nil {
			return nil, err
		}
		output[key] = resolvedValue
	}
	return output, nil
}

func resolveWorkflowFindIconTemplateArray(input []any, findIcon any) ([]any, error) {
	output := make([]any, len(input))
	for index, value := range input {
		resolvedValue, err := resolveWorkflowFindIconTemplateValue(value, findIcon)
		if err != nil {
			return nil, err
		}
		output[index] = resolvedValue
	}
	return output, nil
}

func resolveWorkflowFindIconTemplateString(input string, findIcon any) (any, error) {
	matches := workflowFindIconTemplatePattern.FindAllStringSubmatchIndex(input, -1)
	if len(matches) == 0 {
		return input, nil
	}
	if matches[0][0] == 0 && matches[0][1] == len(input) && len(matches) == 1 {
		return resolveWorkflowFindIconTemplatePathOrLiteral(input[matches[0][2]:matches[0][3]], input, findIcon)
	}
	var builder strings.Builder
	lastIndex := 0
	for _, match := range matches {
		builder.WriteString(input[lastIndex:match[0]])
		resolved, changed, err := resolveWorkflowFindIconPath(input[match[2]:match[3]], findIcon)
		if err != nil {
			return nil, err
		}
		if changed {
			builder.WriteString(stringifyWorkflowFindIconTemplateValue(resolved))
		} else {
			builder.WriteString(input[match[0]:match[1]])
		}
		lastIndex = match[1]
	}
	builder.WriteString(input[lastIndex:])
	return builder.String(), nil
}

func resolveWorkflowFindIconTemplatePathOrLiteral(path string, literal string, findIcon any) (any, error) {
	resolved, changed, err := resolveWorkflowFindIconPath(path, findIcon)
	if err != nil {
		return nil, err
	}
	if !changed {
		return literal, nil
	}
	return resolved, nil
}

func resolveWorkflowFindIconPath(path string, findIcon any) (any, bool, error) {
	normalizedPath := strings.TrimSpace(path)
	if normalizedPath == "" {
		return nil, false, nil
	}
	segments := strings.Split(normalizedPath, ".")
	if strings.TrimSpace(segments[0]) != "find_icon" {
		return nil, false, nil
	}
	if findIcon == nil {
		return nil, true, fmt.Errorf("workflow variable %q is not defined", normalizedPath)
	}
	current := findIcon
	for _, segment := range segments[1:] {
		next, ok := readWorkflowFindIconSegment(current, strings.TrimSpace(segment))
		if !ok {
			return nil, true, fmt.Errorf("workflow variable %q is not defined", normalizedPath)
		}
		current = next
	}
	return current, true, nil
}

func readWorkflowFindIconSegment(current any, segment string) (any, bool) {
	if segment == "" {
		return nil, false
	}
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

func stringifyWorkflowFindIconTemplateValue(value any) string {
	if value == nil {
		return ""
	}
	if text, ok := value.(string); ok {
		return text
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return fmt.Sprintf("%v", value)
	}
	return string(encoded)
}
