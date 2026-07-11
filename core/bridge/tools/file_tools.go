package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"ghost-os/bridge/tools/internal/payloadutil"
)

type ApplyDiffTool struct {
	execution ExecutionClient
}

type applyDiffArgs struct {
	Path     string `json:"path"`
	DiffText string `json:"diff_text"`
}

type ListFilesTool struct {
	execution ExecutionClient
}

type listFilesArgs struct {
	Path string `json:"path,omitempty"`
}

func NewApplyDiffTool(client ExecutionClient) Tool {
	return ApplyDiffTool{execution: client}
}

func (ApplyDiffTool) Name() string {
	return "apply_diff"
}

func (ApplyDiffTool) Description() string {
	return "Apply a unified diff to one file."
}

func (ApplyDiffTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type":"object",
		"properties":{
			"path":{"type":"string","description":"File path to patch."},
			"diff_text":{"type":"string","description":"Unified diff text to apply to the target file."}
		},
		"required":["path","diff_text"],
		"additionalProperties":false
	}`)
}

func (t ApplyDiffTool) Execute(ctx context.Context, argsJSON json.RawMessage, traceID string) (string, error) {
	if t.execution == nil {
		return "", fmt.Errorf("execution client is not configured")
	}

	var args applyDiffArgs
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return "", fmt.Errorf("decode args: %w", err)
	}

	path := strings.TrimSpace(args.Path)
	if path == "" {
		return "", fmt.Errorf("path is required")
	}
	diffText := args.DiffText
	if strings.TrimSpace(diffText) == "" {
		return "", fmt.Errorf("diff_text is required")
	}

	payload, err := t.execution.Call(ctx, "APPLY_DIFF", map[string]any{
		"path":      path,
		"diff_text": diffText,
	}, traceID)
	if err != nil {
		return "", fmt.Errorf("execution APPLY_DIFF failed: %w", err)
	}

	message, err := payloadutil.String(payload, "message")
	if err != nil {
		return "", fmt.Errorf("invalid APPLY_DIFF payload: %w", err)
	}
	return message, nil
}

func NewListFilesTool(client ExecutionClient) Tool {
	return ListFilesTool{execution: client}
}

func (ListFilesTool) Name() string {
	return "list_files"
}

func (ListFilesTool) Description() string {
	return "List the direct children of a path. Directory entries end with '/'."
}

func (ListFilesTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type":"object",
		"properties":{
			"path":{"type":"string","description":"Directory path. Defaults to the current directory."}
		},
		"additionalProperties":false
	}`)
}

func (t ListFilesTool) Execute(ctx context.Context, argsJSON json.RawMessage, traceID string) (string, error) {
	if t.execution == nil {
		return "", fmt.Errorf("execution client is not configured")
	}

	var args listFilesArgs
	if len(argsJSON) > 0 {
		if err := json.Unmarshal(argsJSON, &args); err != nil {
			return "", fmt.Errorf("decode args: %w", err)
		}
	}

	params := map[string]any{}
	if path := strings.TrimSpace(args.Path); path != "" {
		params["path"] = path
	}

	payload, err := t.execution.Call(ctx, "LIST_FILES", params, traceID)
	if err != nil {
		return "", fmt.Errorf("execution LIST_FILES failed: %w", err)
	}

	path, err := payloadutil.String(payload, "path")
	if err != nil {
		return "", fmt.Errorf("invalid LIST_FILES payload: %w", err)
	}
	entries, err := payloadutil.StringSlice(payload, "entries")
	if err != nil {
		return "", fmt.Errorf("invalid LIST_FILES payload: %w", err)
	}

	if len(entries) == 0 {
		return fmt.Sprintf("Directory: %s\nEntries (0)\n(no entries)", path), nil
	}

	return fmt.Sprintf("Directory: %s\nEntries (%d)\n- %s", path, len(entries), strings.Join(entries, "\n- ")), nil
}
