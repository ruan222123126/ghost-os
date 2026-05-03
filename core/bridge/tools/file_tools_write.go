package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"ghost-os/bridge/tools/internal/payloadutil"
)

type WriteFileTool struct {
	execution ExecutionClient
}

type writeFileArgs struct {
	Path    string  `json:"path"`
	Content *string `json:"content"`
	Mode    *string `json:"mode,omitempty"`
}

func NewWriteFileTool(client ExecutionClient) Tool {
	return WriteFileTool{execution: client}
}

func (WriteFileTool) Name() string {
	return "write_file"
}

func (WriteFileTool) Description() string {
	return "Create, overwrite, or append UTF-8 text files inside allowed paths. Missing parent directories are created automatically."
}

func (WriteFileTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type":"object",
		"properties":{
			"path":{"type":"string","description":"File path to write."},
			"content":{"type":"string","description":"UTF-8 text content to write. Can be empty."},
			"mode":{"type":"string","enum":["write","append"],"description":"Write mode. Defaults to 'write'."}
		},
		"required":["path","content"],
		"additionalProperties":false
	}`)
}

func (t WriteFileTool) Execute(ctx context.Context, argsJSON json.RawMessage, traceID string) (string, error) {
	if t.execution == nil {
		return "", fmt.Errorf("execution client is not configured")
	}

	params, err := decodeWriteFileParams(argsJSON)
	if err != nil {
		return "", err
	}
	payload, err := t.execution.Call(ctx, "WRITE_FILE", params, traceID)
	if err != nil {
		return "", fmt.Errorf("execution WRITE_FILE failed: %w", err)
	}
	message, err := payloadutil.String(payload, "message")
	if err != nil {
		return "", fmt.Errorf("invalid WRITE_FILE payload: %w", err)
	}
	return message, nil
}

func decodeWriteFileParams(argsJSON json.RawMessage) (map[string]any, error) {
	var args writeFileArgs
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return nil, fmt.Errorf("decode args: %w", err)
	}
	path := strings.TrimSpace(args.Path)
	if path == "" {
		return nil, fmt.Errorf("path is required")
	}
	if args.Content == nil {
		return nil, fmt.Errorf("content is required")
	}

	params := map[string]any{
		"path":    path,
		"content": *args.Content,
	}
	if args.Mode != nil {
		params["mode"] = *args.Mode
	}
	return params, nil
}
