package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"ghost-os/bridge/tools/internal/payloadutil"
)

type SearchFilesTool struct {
	execution ExecutionClient
}

type searchFilesArgs struct {
	Query      string `json:"query"`
	Path       string `json:"path,omitempty"`
	MaxResults *int   `json:"max_results,omitempty"`
}

type searchFilesMatch struct {
	Path string `json:"path"`
	Line int    `json:"line"`
	Text string `json:"text"`
}

func NewSearchFilesTool(client ExecutionClient) Tool {
	return SearchFilesTool{execution: client}
}

func (SearchFilesTool) Name() string {
	return "search_files"
}

func (SearchFilesTool) Description() string {
	return "Search exact text under a directory and return stable path:line:text matches."
}

func (SearchFilesTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type":"object",
		"properties":{
			"query":{"type":"string","description":"Exact text to search for."},
			"path":{"type":"string","description":"Directory path. Defaults to the current directory."},
			"max_results":{"type":"integer","minimum":1,"description":"Maximum number of matches to return. Defaults to 50."}
		},
		"required":["query"],
		"additionalProperties":false
	}`)
}

func (t SearchFilesTool) Execute(ctx context.Context, argsJSON json.RawMessage, traceID string) (string, error) {
	if t.execution == nil {
		return "", fmt.Errorf("execution client is not configured")
	}

	args, query, err := decodeSearchFilesArgs(argsJSON)
	if err != nil {
		return "", err
	}
	params := buildSearchFilesParams(args, query)
	payload, err := t.execution.Call(ctx, "SEARCH_FILES", params, traceID)
	if err != nil {
		return "", fmt.Errorf("execution SEARCH_FILES failed: %w", err)
	}
	matches, err := decodeSearchFilesMatches(payload)
	if err != nil {
		return "", err
	}
	return formatSearchFilesResult(query, resolveSearchFilesPath(args.Path), matches), nil
}

func decodeSearchFilesArgs(argsJSON json.RawMessage) (searchFilesArgs, string, error) {
	var args searchFilesArgs
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return searchFilesArgs{}, "", fmt.Errorf("decode args: %w", err)
	}
	query := strings.TrimSpace(args.Query)
	if query == "" {
		return searchFilesArgs{}, "", fmt.Errorf("query is required")
	}
	if args.MaxResults != nil && *args.MaxResults < 1 {
		return searchFilesArgs{}, "", fmt.Errorf("max_results must be >= 1")
	}
	return args, query, nil
}

func buildSearchFilesParams(args searchFilesArgs, query string) map[string]any {
	params := map[string]any{
		"query": query,
		"path":  resolveSearchFilesPath(args.Path),
	}
	if args.MaxResults != nil {
		params["max_results"] = *args.MaxResults
	}
	return params
}

func resolveSearchFilesPath(raw string) string {
	path := strings.TrimSpace(raw)
	if path == "" {
		return "."
	}
	return path
}

func decodeSearchFilesMatches(payload map[string]any) ([]searchFilesMatch, error) {
	items, err := payloadutil.AnySlice(payload, "matches")
	if err != nil {
		return nil, fmt.Errorf("invalid SEARCH_FILES payload: %w", err)
	}

	matches := make([]searchFilesMatch, 0, len(items))
	for _, item := range items {
		match, err := decodeSearchFilesMatch(item)
		if err != nil {
			return nil, err
		}
		matches = append(matches, match)
	}
	return matches, nil
}

func decodeSearchFilesMatch(item any) (searchFilesMatch, error) {
	raw, err := json.Marshal(item)
	if err != nil {
		return searchFilesMatch{}, fmt.Errorf("invalid SEARCH_FILES payload: encode match: %w", err)
	}
	var match searchFilesMatch
	if err := json.Unmarshal(raw, &match); err != nil {
		return searchFilesMatch{}, fmt.Errorf("invalid SEARCH_FILES payload: decode match: %w", err)
	}
	if strings.TrimSpace(match.Path) == "" || match.Line < 1 {
		return searchFilesMatch{}, fmt.Errorf("invalid SEARCH_FILES payload: malformed match")
	}
	return match, nil
}

func formatSearchFilesResult(query string, path string, matches []searchFilesMatch) string {
	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("Query: %s\n", query))
	builder.WriteString(fmt.Sprintf("Path: %s\n", path))
	builder.WriteString(fmt.Sprintf("Matches (%d)", len(matches)))
	if len(matches) == 0 {
		builder.WriteString("\n(no matches)")
		return builder.String()
	}
	for _, item := range matches {
		builder.WriteString(fmt.Sprintf("\n- %s:%d: %s", item.Path, item.Line, item.Text))
	}
	return builder.String()
}
