package workflow

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

func ResolveFindIconPath(path string, findIcon any) (any, bool, error) {
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
	return readFindIconPath(segments[1:], normalizedPath, findIcon)
}

func readFindIconPath(segments []string, normalizedPath string, current any) (any, bool, error) {
	for _, segment := range segments {
		next, ok := readFindIconSegment(current, strings.TrimSpace(segment))
		if !ok {
			return nil, true, fmt.Errorf("workflow variable %q is not defined", normalizedPath)
		}
		current = next
	}
	return current, true, nil
}

func readFindIconSegment(current any, segment string) (any, bool) {
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

func StringifyFindIconTemplateValue(value any) string {
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
