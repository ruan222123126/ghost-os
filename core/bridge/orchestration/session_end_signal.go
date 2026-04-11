package orchestration

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
	probe, matched := probeSessionEndSignal(trimmed)
	if !matched {
		return trimmed, nil, nil
	}
	if len(probe) != 2 {
		return "", nil, errors.New("session end signal only allows signal and message fields")
	}
	parsed, err := decodeSessionEndSignalPayload(trimmed)
	if err != nil {
		return "", nil, err
	}
	return parsed.Message, &parsed, nil
}

func probeSessionEndSignal(raw string) (map[string]json.RawMessage, bool) {
	var probe map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &probe); err != nil {
		return nil, false
	}
	signalValueRaw, hasSignal := probe["signal"]
	if !hasSignal {
		return nil, false
	}
	var signalText string
	if err := json.Unmarshal(signalValueRaw, &signalText); err != nil {
		return nil, false
	}
	if strings.TrimSpace(signalText) != busAssistantSessionEndSignal {
		return nil, false
	}
	return probe, true
}

func decodeSessionEndSignalPayload(raw string) (assistantSessionEndSignalPayload, error) {
	var parsed assistantSessionEndSignalPayload
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return assistantSessionEndSignalPayload{}, fmt.Errorf("decode session end signal: %w", err)
	}
	parsed.Signal = strings.TrimSpace(parsed.Signal)
	parsed.Message = strings.TrimSpace(parsed.Message)
	if parsed.Signal != busAssistantSessionEndSignal {
		return assistantSessionEndSignalPayload{}, fmt.Errorf("session end signal must set signal=%q", busAssistantSessionEndSignal)
	}
	if parsed.Message == "" {
		return assistantSessionEndSignalPayload{}, errors.New("session end signal message is empty")
	}
	return parsed, nil
}
