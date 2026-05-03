package orchestration

import (
	"encoding/json"
	"net/url"
	"strings"
)

const (
	microcompactTopSearchResults = 3
	microcompactMaxTextPreview   = 120
)

func isMicrocompactTargetTool(name string) bool {
	switch strings.TrimSpace(name) {
	case "read_file", "web_search", "screen_action", "screen_control", "script_exec", "sfind", "codex_cli":
		return true
	default:
		return false
	}
}

func summarizeMicrocompactPair(pair microcompactToolPair) (string, error) {
	if strings.EqualFold(strings.TrimSpace(pair.envelope.Status), "error") {
		return summarizeMicrocompactError(pair), nil
	}
	switch pair.toolName {
	case "read_file":
		return summarizeReadFileResult(pair), nil
	case "web_search":
		return summarizeWebSearchResult(pair)
	case "screen_action", "screen_control":
		return summarizeScreenToolResult(pair)
	case "script_exec":
		return summarizeScriptExecResult(pair)
	case "sfind":
		return summarizeToolSearchResult(pair)
	case "codex_cli":
		return summarizeCodexCLIResult(pair)
	default:
		return "", microcompactUnsupported(pair.toolName, pair.toolCallID, "tool is not supported")
	}
}

func summarizeMicrocompactError(pair microcompactToolPair) string {
	message := strings.TrimSpace(pair.envelope.Error)
	if message == "" {
		message = "unknown error"
	}
	return pair.toolName + " error: " + message
}

func decodeJSONText[T any](raw string) (T, error) {
	var value T
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &value); err != nil {
		return value, err
	}
	return value, nil
}

func truncateMicrocompactText(value string, limit int) string {
	if limit <= 0 {
		return ""
	}
	runes := []rune(strings.TrimSpace(value))
	if len(runes) <= limit {
		return string(runes)
	}
	return string(runes[:limit]) + "..."
}

func nonEmptyString(value any) string {
	text, ok := value.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(text)
}

func optionalBool(value any) (bool, bool) {
	typed, ok := value.(bool)
	return typed, ok
}

func optionalInt(value any) (int, bool) {
	switch typed := value.(type) {
	case int:
		return typed, true
	case int32:
		return int(typed), true
	case int64:
		return int(typed), true
	case float64:
		rounded := int(typed)
		if float64(rounded) == typed {
			return rounded, true
		}
	}
	return 0, false
}

func optionalMap(value any) map[string]any {
	typed, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	return typed
}

func resultDomain(rawURL string) string {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(parsed.Host)
}
