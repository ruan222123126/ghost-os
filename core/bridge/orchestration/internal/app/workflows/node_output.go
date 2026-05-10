package workflows

import (
	"encoding/json"
	"fmt"
	"strings"
)

func EncodeNodeOutputText(value any) string {
	encoded, err := json.Marshal(value)
	if err != nil {
		return strings.TrimSpace(fmt.Sprintf("%v", value))
	}
	return strings.TrimSpace(string(encoded))
}

func DecodeNodeOutput(output string) any {
	trimmed := strings.TrimSpace(output)
	if trimmed == "" {
		return ""
	}
	var decoded any
	if err := json.Unmarshal([]byte(trimmed), &decoded); err != nil {
		return trimmed
	}
	return decoded
}

func nodeToolCallID(nodeID string) string {
	return "workflow-" + strings.TrimSpace(nodeID)
}

func mapString(record map[string]any, key string) string {
	raw, ok := record[key]
	if !ok || raw == nil {
		return ""
	}
	value, ok := raw.(string)
	if !ok {
		return ""
	}
	return value
}
