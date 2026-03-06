package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// parseSessionEndSignal 只识别完整会话结束信号；普通文本/普通 JSON 均按普通回复返回。
func parseSessionEndSignal(raw string) (message string, signal *assistantSessionEndSignalPayload, err error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", nil, nil
	}

	var probe map[string]json.RawMessage
	if err := json.Unmarshal([]byte(trimmed), &probe); err != nil {
		return trimmed, nil, nil
	}

	signalValueRaw, hasSignal := probe["signal"]
	if !hasSignal {
		return trimmed, nil, nil
	}

	var signalValue any
	if err := json.Unmarshal(signalValueRaw, &signalValue); err != nil {
		return trimmed, nil, nil
	}
	signalText, isString := signalValue.(string)
	if !isString || strings.TrimSpace(signalText) != busAssistantSessionEndSignal {
		return trimmed, nil, nil
	}

	if len(probe) != 2 {
		return "", nil, errors.New("session end signal only allows signal and message fields")
	}

	var parsed assistantSessionEndSignalPayload
	if err := json.Unmarshal([]byte(trimmed), &parsed); err != nil {
		return "", nil, fmt.Errorf("decode session end signal: %w", err)
	}
	parsed.Signal = strings.TrimSpace(parsed.Signal)
	parsed.Message = strings.TrimSpace(parsed.Message)
	if parsed.Signal != busAssistantSessionEndSignal {
		return "", nil, fmt.Errorf("session end signal must set signal=%q", busAssistantSessionEndSignal)
	}
	if parsed.Message == "" {
		return "", nil, errors.New("session end signal message is empty")
	}

	return parsed.Message, &parsed, nil
}
