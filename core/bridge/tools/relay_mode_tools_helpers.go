package tools

import (
	"encoding/json"
	"fmt"
	"strings"
)

func relayRecordSchema(includeFinal bool) string {
	finalFields := ""
	finalRequired := ""
	if includeFinal {
		finalFields = `,"final_message":{"type":"string","description":"Final assistant reply to show the user."},"final_change_log":{"type":"string","description":"Concise final modification record."}`
		finalRequired = `,"final_message","final_change_log"`
	}
	return `{"type":"object","properties":{"did":{"type":"string"},"remaining":{"type":"string"},"failed_attempts":{"type":"array","items":{"type":"string"}},"next_step":{"type":"string"}` +
		`}` + finalFields + `},"required":["did","remaining","failed_attempts","next_step"` + finalRequired +
		`],"additionalProperties":false}`
}

func decodeRelayRecordArgs(raw json.RawMessage) (relayRecordArgs, error) {
	var args relayRecordArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return relayRecordArgs{}, fmt.Errorf("decode args: %w", err)
	}
	args.Did = strings.TrimSpace(args.Did)
	args.Remaining = strings.TrimSpace(args.Remaining)
	args.FailedAttempts = normalizeRelayFailedAttempts(args.FailedAttempts)
	args.NextStep = strings.TrimSpace(args.NextStep)
	if args.Did == "" || args.Remaining == "" || args.NextStep == "" {
		return relayRecordArgs{}, fmt.Errorf("did, remaining, and next_step are required")
	}
	return args, nil
}

func decodeRelayCompletePayload(raw json.RawMessage) (relayModeToolPayload, error) {
	var args relayCompleteArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return relayModeToolPayload{}, fmt.Errorf("decode args: %w", err)
	}
	base, err := relayCompleteBase(args)
	if err != nil {
		return relayModeToolPayload{}, err
	}
	base.FinalMessage = strings.TrimSpace(args.FinalMessage)
	base.FinalChangeLog = strings.TrimSpace(args.FinalChangeLog)
	if base.FinalMessage == "" || base.FinalChangeLog == "" {
		return relayModeToolPayload{}, fmt.Errorf("final_message and final_change_log are required")
	}
	return base, nil
}

func relayCompleteBase(args relayCompleteArgs) (relayModeToolPayload, error) {
	record := relayRecordArgs{
		Did:            strings.TrimSpace(args.Did),
		Remaining:      strings.TrimSpace(args.Remaining),
		FailedAttempts: normalizeRelayFailedAttempts(args.FailedAttempts),
		NextStep:       strings.TrimSpace(args.NextStep),
	}
	if record.Did == "" || record.Remaining == "" || record.NextStep == "" {
		return relayModeToolPayload{}, fmt.Errorf("did, remaining, and next_step are required")
	}
	return relayModeToolPayload{
		Status:         relayCompleteStatus,
		Did:            record.Did,
		Remaining:      record.Remaining,
		FailedAttempts: record.FailedAttempts,
		NextStep:       record.NextStep,
	}, nil
}

func encodeRelayPayload(payload relayModeToolPayload, toolName string) (string, error) {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("encode %s payload: %w", toolName, err)
	}
	return string(encoded), nil
}

func decodeRelayModeToolPayload(output string) (relayModeToolPayload, bool) {
	var payload relayModeToolPayload
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &payload); err != nil {
		return relayModeToolPayload{}, false
	}
	payload = normalizeRelayPayload(payload)
	if payload.Status == "" || payload.Did == "" || payload.Remaining == "" || payload.NextStep == "" {
		return relayModeToolPayload{}, false
	}
	if payload.Status == relayCompleteStatus && (payload.FinalMessage == "" || payload.FinalChangeLog == "") {
		return relayModeToolPayload{}, false
	}
	return payload, true
}

func normalizeRelayPayload(payload relayModeToolPayload) relayModeToolPayload {
	payload.Status = strings.TrimSpace(payload.Status)
	payload.Did = strings.TrimSpace(payload.Did)
	payload.Remaining = strings.TrimSpace(payload.Remaining)
	payload.FailedAttempts = normalizeRelayFailedAttempts(payload.FailedAttempts)
	payload.NextStep = strings.TrimSpace(payload.NextStep)
	payload.FinalMessage = strings.TrimSpace(payload.FinalMessage)
	payload.FinalChangeLog = strings.TrimSpace(payload.FinalChangeLog)
	return payload
}

func normalizeRelayFailedAttempts(input []string) []string {
	out := make([]string, 0, len(input))
	for _, raw := range input {
		item := strings.TrimSpace(raw)
		if item != "" {
			out = append(out, item)
		}
	}
	return out
}

func relayPayloadSignal(payload relayModeToolPayload) *IterationHandoffSignal {
	return &IterationHandoffSignal{
		Did:            payload.Did,
		Remaining:      payload.Remaining,
		FailedAttempts: append([]string(nil), payload.FailedAttempts...),
		NextStep:       payload.NextStep,
	}
}
