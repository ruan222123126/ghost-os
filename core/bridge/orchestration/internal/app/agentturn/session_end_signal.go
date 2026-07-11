package agentturn

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"ghost-os/bridge/orchestration/internal/contracts/api"
	"ghost-os/bridge/orchestration/internal/contracts/bus"
)

// ParseSessionEndSignal only accepts the exact assistant session-end payload.
// Other text or JSON stays a normal assistant message.
func ParseSessionEndSignal(raw string) (message string, signal *api.AssistantSessionEndSignalPayload, err error) {
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
	if strings.TrimSpace(signalText) != bus.AssistantSessionEndSignal {
		return nil, false
	}
	return probe, true
}

func decodeSessionEndSignalPayload(raw string) (api.AssistantSessionEndSignalPayload, error) {
	var parsed api.AssistantSessionEndSignalPayload
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return api.AssistantSessionEndSignalPayload{}, fmt.Errorf("decode session end signal: %w", err)
	}
	parsed.Signal = strings.TrimSpace(parsed.Signal)
	parsed.Message = strings.TrimSpace(parsed.Message)
	if parsed.Signal != bus.AssistantSessionEndSignal {
		return api.AssistantSessionEndSignalPayload{}, fmt.Errorf("session end signal must set signal=%q", bus.AssistantSessionEndSignal)
	}
	if parsed.Message == "" {
		return api.AssistantSessionEndSignalPayload{}, errors.New("session end signal message is empty")
	}
	return parsed, nil
}
