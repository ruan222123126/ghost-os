package tools

import (
	"encoding/json"
	"fmt"
	"strings"

	"ghost-os/bridge/tools/internal/payloadutil"
)

func summarizeWriteChange(toolName string, args map[string]any) *scriptExecWriteChange {
	switch toolName {
	case "apply_diff":
		return summarizeApplyDiffWriteChange(args)
	case "write_file":
		return summarizeWriteFileChange(args)
	default:
		return nil
	}
}

func summarizeApplyDiffWriteChange(args map[string]any) *scriptExecWriteChange {
	if len(args) == 0 {
		return nil
	}
	change := &scriptExecWriteChange{
		Operation: "apply_diff",
		Path:      strings.TrimSpace(anyString(args["path"])),
	}
	diffSummary, _ := args["diff_summary"].(map[string]any)
	if len(diffSummary) > 0 {
		change.AddedLines = scriptExecInt(diffSummary["added_lines"])
		change.RemovedLines = scriptExecInt(diffSummary["removed_lines"])
		change.HunkRanges = summarizeHunkRanges(diffSummary["hunk_ranges"])
	}
	if change.Path == "" && len(change.HunkRanges) == 0 && change.AddedLines == 0 && change.RemovedLines == 0 {
		return nil
	}
	return change
}

func summarizeWriteFileChange(args map[string]any) *scriptExecWriteChange {
	if len(args) == 0 {
		return nil
	}
	change := &scriptExecWriteChange{
		Operation: "write_file",
		Path:      strings.TrimSpace(anyString(args["path"])),
		Mode:      strings.TrimSpace(anyString(args["mode"])),
	}
	contentSummary, _ := args["content_summary"].(map[string]any)
	if len(contentSummary) > 0 {
		change.ContentBytes = scriptExecInt(contentSummary["bytes"])
		change.ContentLines = scriptExecInt(contentSummary["lines"])
	}
	if change.ContentBytes == 0 {
		content := anyString(args["content"])
		change.ContentBytes = len(content)
		change.ContentLines = countLines(content)
		change.ContentPreview = truncateText(strings.TrimSpace(content), scriptExecMaxContentPreview)
	}
	if change.Path == "" && change.ContentBytes == 0 {
		return nil
	}
	return change
}

func summarizeResult(toolName string, raw string) string {
	result := strings.TrimSpace(raw)
	if result == "" {
		return ""
	}
	switch toolName {
	case "list_files":
		return summarizeJSONList(result, "entries")
	case "search_files":
		return summarizeJSONList(result, "matches")
	case "read_file":
		return fmt.Sprintf("returned %d lines", countLines(result))
	default:
		return truncateText(result, scriptExecMaxResultChars)
	}
}

func summarizeJSONList(raw string, noun string) string {
	var items []string
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return truncateText(raw, scriptExecMaxResultChars)
	}
	return fmt.Sprintf("%d %s", len(items), noun)
}

func truncateText(value string, maxChars int) string {
	if maxChars <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= maxChars {
		return value
	}
	return string(runes[:maxChars]) + "...(truncated)"
}

func countLines(content string) int {
	if strings.TrimSpace(content) == "" {
		return 0
	}
	return strings.Count(content, "\n") + 1
}

func anyString(value any) string {
	if text, ok := value.(string); ok {
		return text
	}
	if value == nil {
		return ""
	}
	return fmt.Sprint(value)
}

func normalizedToolName(raw string) string {
	name := strings.TrimSpace(raw)
	if name == "" {
		return "unknown"
	}
	return name
}

func scriptExecInt(raw any) int {
	if value, ok := payloadutil.NumericToInt(raw); ok {
		return value
	}
	return 0
}
