package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// ListFilesTool 通过 execution layer 列出目录项。
type ListFilesTool struct {
	execution ExecutionClient
}

type listFilesArgs struct {
	Path string `json:"path"`
}

func NewListFilesTool(client ExecutionClient) Tool {
	return ListFilesTool{execution: client}
}

func (ListFilesTool) Name() string {
	return "list_files"
}

func (ListFilesTool) Description() string {
	return "List files and directories for a given path."
}

func (ListFilesTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"path": {
				"type": "string",
				"description": "Directory path to list. Defaults to current directory.",
				"default": "."
			}
		},
		"additionalProperties": false
	}`)
}

// Execute 仅负责参数校验与编排，实际 OS 调用下沉到 execution layer。
func (t ListFilesTool) Execute(ctx context.Context, argsJSON json.RawMessage, traceID string) (string, error) {
	if t.execution == nil {
		return "", fmt.Errorf("execution client is not configured")
	}

	var args listFilesArgs
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return "", fmt.Errorf("decode args: %w", err)
	}

	path := strings.TrimSpace(args.Path)
	if path == "" {
		path = "."
	}

	payload, err := t.execution.Call(ctx, "LIST_FILES", map[string]any{"path": path}, traceID)
	if err != nil {
		return "", fmt.Errorf("execution LIST_FILES failed: %w", err)
	}

	entries, ok := payload["entries"].([]any)
	if !ok {
		return "", fmt.Errorf("invalid LIST_FILES payload: entries must be an array")
	}
	if len(entries) == 0 {
		return "(empty)", nil
	}

	out := make([]string, 0, len(entries))
	for _, entry := range entries {
		name, ok := entry.(string)
		if !ok {
			return "", fmt.Errorf("invalid LIST_FILES payload: entry must be a string")
		}
		out = append(out, name)
	}

	return strings.Join(out, "\n"), nil
}
