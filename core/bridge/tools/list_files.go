package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

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

func (ListFilesTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "Directory path to list. Defaults to current directory.",
				"default":     ".",
			},
		},
		"additionalProperties": false,
	}
}

func (ListFilesTool) Execute(_ context.Context, argsJSON string) (string, error) {
	var args listFilesArgs
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
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
