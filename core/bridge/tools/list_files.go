package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// ListFilesTool 用于列出目录项，作为最小文件系统感知能力。
type ListFilesTool struct{}

type listFilesArgs struct {
	Path string `json:"path"`
}

func NewListFilesTool() Tool {
	return ListFilesTool{}
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

// Execute 读取目录并返回换行分隔列表，目录项以 "/" 标识。
func (ListFilesTool) Execute(_ context.Context, argsJSON json.RawMessage) (string, error) {
	var args listFilesArgs
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return "", fmt.Errorf("decode args: %w", err)
	}

	path := strings.TrimSpace(args.Path)
	if path == "" {
		path = "."
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return "", fmt.Errorf("read dir %q: %w", path, err)
	}

	if len(entries) == 0 {
		return "(empty)", nil
	}

	out := make([]string, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() {
			name += "/"
		}
		out = append(out, name)
	}

	return strings.Join(out, "\n"), nil
}
