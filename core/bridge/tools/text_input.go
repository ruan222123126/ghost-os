package tools

import (
	"context"
	"encoding/json"
	"fmt"
)

type TextInputTool struct {
	execution ExecutionClient
}

type textInputArgs struct {
	Text   string `json:"text"`
	Submit bool   `json:"submit,omitempty"`
}

func NewTextInputTool(client ExecutionClient) Tool {
	return TextInputTool{execution: client}
}

func (TextInputTool) Name() string {
	return "text_input"
}

func (TextInputTool) Description() string {
	return "Type text into the currently focused input field. Use only after the target field is focused; set submit=true to press Enter after typing."
}

func (TextInputTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type":"object",
		"properties":{
			"text":{"type":"string","minLength":1,"description":"Exact text to type into the focused field. Newlines are allowed."},
			"submit":{"type":"boolean","description":"Press Enter after typing the text."}
		},
		"required":["text"],
		"additionalProperties":false
	}`)
}

func (t TextInputTool) Execute(ctx context.Context, argsJSON json.RawMessage, traceID string) (string, error) {
	if t.execution == nil {
		return "", fmt.Errorf("execution client is not configured")
	}

	var args textInputArgs
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return "", fmt.Errorf("decode args: %w", err)
	}
	if args.Text == "" {
		return "", fmt.Errorf("text is required")
	}

	params := map[string]any{
		"text": args.Text,
	}
	if args.Submit {
		params["submit"] = true
	}

	payload, err := t.execution.Call(ctx, "TEXT_INPUT", params, traceID)
	if err != nil {
		return "", fmt.Errorf("execution TEXT_INPUT failed: %w", err)
	}

	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("encode payload: %w", err)
	}
	return string(encoded), nil
}
