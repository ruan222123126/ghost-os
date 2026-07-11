package tools

import (
	"context"
	"encoding/json"
)

const (
	relayUpdateRecordStatus = "relay_recorded"
	relayCompleteStatus     = "relay_completed"
)

type RelayUpdateRecordTool struct{}
type RelayCompleteTool struct{}

type relayRecordArgs struct {
	Did            string   `json:"did"`
	Remaining      string   `json:"remaining"`
	FailedAttempts []string `json:"failed_attempts"`
	NextStep       string   `json:"next_step"`
}

type relayCompleteArgs struct {
	Did            string   `json:"did"`
	Remaining      string   `json:"remaining"`
	FailedAttempts []string `json:"failed_attempts"`
	NextStep       string   `json:"next_step"`
	FinalMessage   string   `json:"final_message"`
	FinalChangeLog string   `json:"final_change_log"`
}

type relayModeToolPayload struct {
	Status         string   `json:"status"`
	Did            string   `json:"did"`
	Remaining      string   `json:"remaining"`
	FailedAttempts []string `json:"failed_attempts,omitempty"`
	NextStep       string   `json:"next_step"`
	FinalMessage   string   `json:"final_message,omitempty"`
	FinalChangeLog string   `json:"final_change_log,omitempty"`
}

func NewRelayUpdateRecordTool() Tool {
	return RelayUpdateRecordTool{}
}

func (RelayUpdateRecordTool) Name() string {
	return "relay_update_record"
}

func (RelayUpdateRecordTool) Description() string {
	return "End the current relay worker iteration and hand off to a fresh-memory worker with a structured status record."
}

func (RelayUpdateRecordTool) Parameters() json.RawMessage {
	return json.RawMessage(relayRecordSchema(false))
}

func (RelayUpdateRecordTool) Execute(_ context.Context, argsJSON json.RawMessage, _ string) (string, error) {
	args, err := decodeRelayRecordArgs(argsJSON)
	if err != nil {
		return "", err
	}
	return encodeRelayPayload(relayModeToolPayload{
		Status:         relayUpdateRecordStatus,
		Did:            args.Did,
		Remaining:      args.Remaining,
		FailedAttempts: args.FailedAttempts,
		NextStep:       args.NextStep,
	}, "relay_update_record")
}

func (RelayUpdateRecordTool) InterpretResult(output string) ExecuteMeta {
	payload, ok := decodeRelayModeToolPayload(output)
	if !ok || payload.Status != relayUpdateRecordStatus {
		return ExecuteMeta{}
	}
	return ExecuteMeta{Iteration: relayPayloadSignal(payload)}
}

func NewRelayCompleteTool() Tool {
	return RelayCompleteTool{}
}

func (RelayCompleteTool) Name() string {
	return "relay_complete"
}

func (RelayCompleteTool) Description() string {
	return "Finish an AI-decides relay run only when the task is truly complete."
}

func (RelayCompleteTool) Parameters() json.RawMessage {
	return json.RawMessage(relayRecordSchema(true))
}

func (RelayCompleteTool) Execute(_ context.Context, argsJSON json.RawMessage, _ string) (string, error) {
	payload, err := decodeRelayCompletePayload(argsJSON)
	if err != nil {
		return "", err
	}
	return encodeRelayPayload(payload, "relay_complete")
}

func (RelayCompleteTool) InterpretResult(output string) ExecuteMeta {
	payload, ok := decodeRelayModeToolPayload(output)
	if !ok || payload.Status != relayCompleteStatus {
		return ExecuteMeta{}
	}
	signal := relayPayloadSignal(payload)
	signal.Completed = true
	signal.FinalMessage = payload.FinalMessage
	signal.FinalChangeLog = payload.FinalChangeLog
	return ExecuteMeta{Iteration: signal}
}
