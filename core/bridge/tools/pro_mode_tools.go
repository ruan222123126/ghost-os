package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

const (
	proUpdateRecordStatus = "iteration_recorded"
	proCompleteStatus     = "iteration_completed"
)

type ProUpdateRecordTool struct{}

type proRecordArgs struct {
	Did       string `json:"did"`
	Remaining string `json:"remaining"`
}

type ProCompleteTool struct{}

type proCompleteArgs struct {
	Did            string `json:"did"`
	Remaining      string `json:"remaining"`
	FinalMessage   string `json:"final_message"`
	FinalChangeLog string `json:"final_change_log"`
}

type proModeToolPayload struct {
	Status         string `json:"status"`
	Did            string `json:"did"`
	Remaining      string `json:"remaining"`
	FinalMessage   string `json:"final_message,omitempty"`
	FinalChangeLog string `json:"final_change_log,omitempty"`
}

func NewProUpdateRecordTool() Tool {
	return ProUpdateRecordTool{}
}

func (ProUpdateRecordTool) Name() string {
	return "pro_update_record"
}

func (ProUpdateRecordTool) Description() string {
	return "End the current pro/prox iteration and hand off to a fresh-memory agent. Use it only after recording what you completed and what remains."
}

func (ProUpdateRecordTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type":"object",
		"properties":{
			"did":{"type":"string","description":"Short summary of what this iteration completed."},
			"remaining":{"type":"string","description":"Short summary of what still remains for the next fresh-memory agent."}
		},
		"required":["did","remaining"],
		"additionalProperties":false
	}`)
}

func (ProUpdateRecordTool) Execute(_ context.Context, argsJSON json.RawMessage, _ string) (string, error) {
	args, err := decodeProRecordArgs(argsJSON)
	if err != nil {
		return "", err
	}
	encoded, err := json.Marshal(proModeToolPayload{
		Status:    proUpdateRecordStatus,
		Did:       args.Did,
		Remaining: args.Remaining,
	})
	if err != nil {
		return "", fmt.Errorf("encode pro_update_record payload: %w", err)
	}
	return string(encoded), nil
}

func (ProUpdateRecordTool) InterpretResult(output string) ExecuteMeta {
	payload, ok := decodeProModeToolPayload(output)
	if !ok || payload.Status != proUpdateRecordStatus {
		return ExecuteMeta{}
	}
	return ExecuteMeta{
		Iteration: &IterationHandoffSignal{
			Did:       payload.Did,
			Remaining: payload.Remaining,
		},
	}
}

func NewProCompleteTool() Tool {
	return ProCompleteTool{}
}

func (ProCompleteTool) Name() string {
	return "pro_complete"
}

func (ProCompleteTool) Description() string {
	return "Finish a finite pro run only when the task is truly complete. Include the final user-facing message and the final change log."
}

func (ProCompleteTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type":"object",
		"properties":{
			"did":{"type":"string","description":"Short summary of what this final iteration completed."},
			"remaining":{"type":"string","description":"What remains. Use an explicit empty-state summary such as 'none' when complete."},
			"final_message":{"type":"string","description":"Final assistant reply to show the user."},
			"final_change_log":{"type":"string","description":"Concise final modification record summarizing the finished changes."}
		},
		"required":["did","remaining","final_message","final_change_log"],
		"additionalProperties":false
	}`)
}

func (ProCompleteTool) Execute(_ context.Context, argsJSON json.RawMessage, _ string) (string, error) {
	var args proCompleteArgs
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return "", fmt.Errorf("decode args: %w", err)
	}
	payload := proModeToolPayload{
		Status:         proCompleteStatus,
		Did:            strings.TrimSpace(args.Did),
		Remaining:      strings.TrimSpace(args.Remaining),
		FinalMessage:   strings.TrimSpace(args.FinalMessage),
		FinalChangeLog: strings.TrimSpace(args.FinalChangeLog),
	}
	if payload.Did == "" {
		return "", fmt.Errorf("did is required")
	}
	if payload.Remaining == "" {
		return "", fmt.Errorf("remaining is required")
	}
	if payload.FinalMessage == "" {
		return "", fmt.Errorf("final_message is required")
	}
	if payload.FinalChangeLog == "" {
		return "", fmt.Errorf("final_change_log is required")
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("encode pro_complete payload: %w", err)
	}
	return string(encoded), nil
}

func (ProCompleteTool) InterpretResult(output string) ExecuteMeta {
	payload, ok := decodeProModeToolPayload(output)
	if !ok || payload.Status != proCompleteStatus {
		return ExecuteMeta{}
	}
	return ExecuteMeta{
		Iteration: &IterationHandoffSignal{
			Did:            payload.Did,
			Remaining:      payload.Remaining,
			Completed:      true,
			FinalMessage:   payload.FinalMessage,
			FinalChangeLog: payload.FinalChangeLog,
		},
	}
}

func decodeProRecordArgs(raw json.RawMessage) (proRecordArgs, error) {
	var args proRecordArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return proRecordArgs{}, fmt.Errorf("decode args: %w", err)
	}
	args.Did = strings.TrimSpace(args.Did)
	args.Remaining = strings.TrimSpace(args.Remaining)
	if args.Did == "" {
		return proRecordArgs{}, fmt.Errorf("did is required")
	}
	if args.Remaining == "" {
		return proRecordArgs{}, fmt.Errorf("remaining is required")
	}
	return args, nil
}

func decodeProModeToolPayload(output string) (proModeToolPayload, bool) {
	var payload proModeToolPayload
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &payload); err != nil {
		return proModeToolPayload{}, false
	}
	payload.Status = strings.TrimSpace(payload.Status)
	payload.Did = strings.TrimSpace(payload.Did)
	payload.Remaining = strings.TrimSpace(payload.Remaining)
	payload.FinalMessage = strings.TrimSpace(payload.FinalMessage)
	payload.FinalChangeLog = strings.TrimSpace(payload.FinalChangeLog)
	if payload.Status == "" || payload.Did == "" || payload.Remaining == "" {
		return proModeToolPayload{}, false
	}
	if payload.Status == proCompleteStatus && (payload.FinalMessage == "" || payload.FinalChangeLog == "") {
		return proModeToolPayload{}, false
	}
	return payload, true
}
