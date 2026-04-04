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

type SearchFilesTool struct {
	execution ExecutionClient
}

type searchFilesArgs struct {
	Keyword       string `json:"keyword"`
	DirPath       string `json:"dir_path,omitempty"`
	CaseSensitive *bool  `json:"case_sensitive,omitempty"`
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

func NewSearchFilesTool(client ExecutionClient) Tool {
	return SearchFilesTool{execution: client}
}

func (SearchFilesTool) Name() string {
	return "search_files"
}

func (SearchFilesTool) Description() string {
	return "Search files for text or regex matches. Returns up to 100 matching lines with file paths and line numbers."
}

func (SearchFilesTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type":"object",
		"properties":{
			"keyword":{"type":"string","description":"Keyword or regex pattern to search for."},
			"dir_path":{"type":"string","description":"Directory root to search from. Defaults to the current directory."},
			"case_sensitive":{"type":"boolean","description":"Whether matching should be case-sensitive. Defaults to true."}
		},
		"required":["keyword"],
		"additionalProperties":false
	}`)
}

func (t SearchFilesTool) Execute(ctx context.Context, argsJSON json.RawMessage, traceID string) (string, error) {
	if t.execution == nil {
		return "", fmt.Errorf("execution client is not configured")
	}

	var args searchFilesArgs
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return "", fmt.Errorf("decode args: %w", err)
	}

	keyword := strings.TrimSpace(args.Keyword)
	if keyword == "" {
		return "", fmt.Errorf("keyword is required")
	}

	params := map[string]any{"keyword": keyword}
	if dirPath := strings.TrimSpace(args.DirPath); dirPath != "" {
		params["dir_path"] = dirPath
	}
	if args.CaseSensitive != nil {
		params["case_sensitive"] = *args.CaseSensitive
	}

	payload, err := t.execution.Call(ctx, "SEARCH_FILES", params, traceID)
	if err != nil {
		return "", fmt.Errorf("execution SEARCH_FILES failed: %w", err)
	}

	dirPath, err := payloadutil.String(payload, "dir_path")
	if err != nil {
		return "", fmt.Errorf("invalid SEARCH_FILES payload: %w", err)
	}
	matches, err := payloadutil.StringSlice(payload, "matches")
	if err != nil {
		return "", fmt.Errorf("invalid SEARCH_FILES payload: %w", err)
	}

	if len(matches) == 0 {
		return fmt.Sprintf("Search root: %s\nMatches (0)\n(no matches)", dirPath), nil
	}

	return fmt.Sprintf("Search root: %s\nMatches (%d)\n%s", dirPath, len(matches), strings.Join(matches, "\n")), nil
}
