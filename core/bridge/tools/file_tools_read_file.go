package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"ghost-os/bridge/tools/internal/payloadutil"
)

const maxReadFileLines = 200

type ReadFileTool struct {
	execution ExecutionClient
}

type readFileArgs struct {
	Path      string `json:"path"`
	StartLine *int   `json:"start_line,omitempty"`
	EndLine   *int   `json:"end_line,omitempty"`
}

type readFileResult struct {
	Path               string
	RequestedStartLine int
	RequestedEndLine   int
	ReturnedStartLine  int
	ReturnedEndLine    int
	TotalLines         int
	Content            string
}

func NewReadFileTool(client ExecutionClient) Tool {
	return ReadFileTool{execution: client}
}

func (ReadFileTool) Name() string {
	return "read_file"
}

func (ReadFileTool) Description() string {
	return "Read file text, optionally by line range. Returns line-numbered text and reads at most 200 lines per call."
}

func (ReadFileTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type":"object",
		"properties":{
			"path":{"type":"string","description":"File path to read."},
			"start_line":{"type":"integer","minimum":1,"description":"Optional 1-based start line."},
			"end_line":{"type":"integer","minimum":1,"description":"Optional 1-based end line."}
		},
		"required":["path"],
		"additionalProperties":false
	}`)
}

func (t ReadFileTool) Execute(ctx context.Context, argsJSON json.RawMessage, traceID string) (string, error) {
	if t.execution == nil {
		return "", fmt.Errorf("execution client is not configured")
	}
	args, path, err := decodeReadFileArgs(argsJSON)
	if err != nil {
		return "", err
	}
	params, err := buildReadFileParams(path, args.StartLine, args.EndLine)
	if err != nil {
		return "", err
	}
	payload, err := t.execution.Call(ctx, "READ_FILE", params, traceID)
	if err != nil {
		return "", fmt.Errorf("execution READ_FILE failed: %w", err)
	}
	result, err := decodeReadFileResult(payload)
	if err != nil {
		return "", err
	}
	return formatReadFileResult(result), nil
}

func decodeReadFileArgs(argsJSON json.RawMessage) (readFileArgs, string, error) {
	var args readFileArgs
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return readFileArgs{}, "", fmt.Errorf("decode args: %w", err)
	}
	path := strings.TrimSpace(args.Path)
	if path == "" {
		return readFileArgs{}, "", fmt.Errorf("path is required")
	}
	return args, path, nil
}

func buildReadFileParams(path string, startLine *int, endLine *int) (map[string]any, error) {
	params := map[string]any{"path": path}
	start := 1
	if startLine != nil {
		if *startLine < 1 {
			return nil, fmt.Errorf("start_line must be >= 1")
		}
		start = *startLine
		params["start_line"] = *startLine
	}
	if endLine != nil {
		if *endLine < 1 {
			return nil, fmt.Errorf("end_line must be >= 1")
		}
		if *endLine < start {
			return nil, fmt.Errorf("end_line must be >= start_line")
		}
		if (*endLine-start)+1 > maxReadFileLines {
			return nil, fmt.Errorf("read range too large (max %d lines)", maxReadFileLines)
		}
		params["end_line"] = *endLine
	}
	return params, nil
}

func decodeReadFileResult(payload map[string]any) (readFileResult, error) {
	result := readFileResult{}
	var err error
	if result.Path, err = payloadutil.String(payload, "path"); err != nil {
		return readFileResult{}, wrapReadFilePayloadError(err)
	}
	if result.RequestedStartLine, err = payloadutil.Int(payload, "requested_start_line"); err != nil {
		return readFileResult{}, wrapReadFilePayloadError(err)
	}
	if result.RequestedEndLine, err = payloadutil.Int(payload, "requested_end_line"); err != nil {
		return readFileResult{}, wrapReadFilePayloadError(err)
	}
	if result.ReturnedStartLine, err = payloadutil.Int(payload, "returned_start_line"); err != nil {
		return readFileResult{}, wrapReadFilePayloadError(err)
	}
	if result.ReturnedEndLine, err = payloadutil.Int(payload, "returned_end_line"); err != nil {
		return readFileResult{}, wrapReadFilePayloadError(err)
	}
	if result.TotalLines, err = payloadutil.Int(payload, "total_lines"); err != nil {
		return readFileResult{}, wrapReadFilePayloadError(err)
	}
	if result.Content, err = payloadutil.String(payload, "content"); err != nil {
		return readFileResult{}, wrapReadFilePayloadError(err)
	}
	return result, nil
}

func wrapReadFilePayloadError(err error) error {
	return fmt.Errorf("invalid READ_FILE payload: %w", err)
}

func formatReadFileResult(result readFileResult) string {
	var builder strings.Builder
	fmt.Fprintf(&builder, "File: %s\n", result.Path)
	fmt.Fprintf(&builder, "Requested lines: %d-%d\n", result.RequestedStartLine, result.RequestedEndLine)
	if result.ReturnedStartLine == 0 || result.ReturnedEndLine == 0 {
		builder.WriteString(formatReadFileEmptyResult(result.TotalLines))
		return builder.String()
	}
	fmt.Fprintf(
		&builder,
		"Returned lines: %d-%d of %d total\n%s",
		result.ReturnedStartLine,
		result.ReturnedEndLine,
		result.TotalLines,
		result.Content,
	)
	return builder.String()
}

func formatReadFileEmptyResult(totalLines int) string {
	if totalLines == 0 {
		return "Returned lines: none (file is empty)\n(no content)"
	}
	return fmt.Sprintf("Returned lines: none (file has %d total lines)\n(no content)", totalLines)
}
