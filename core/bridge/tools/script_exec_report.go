package tools

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

const (
	scriptExecMaxArgFields       = 8
	scriptExecMaxArgPreviewChars = 160
	scriptExecMaxResultChars     = 240
	scriptExecMaxHunkRanges      = 8
	scriptExecMaxContentPreview  = 120
)

type scriptExecReport struct {
	ScriptOutput string            `json:"script_output,omitempty"`
	Steps        []scriptExecStep  `json:"steps,omitempty"`
	Summary      scriptExecSummary `json:"summary"`
}

type scriptExecSummary struct {
	StepCount   int `json:"step_count"`
	FailedSteps int `json:"failed_steps"`
	WriteSteps  int `json:"write_steps"`
}

type scriptExecStep struct {
	Step          int                    `json:"step"`
	Tool          string                 `json:"tool"`
	Status        string                 `json:"status"`
	Args          map[string]any         `json:"args,omitempty"`
	ResultSummary string                 `json:"result_summary,omitempty"`
	Error         string                 `json:"error,omitempty"`
	WriteChange   *scriptExecWriteChange `json:"write_change,omitempty"`
}

type scriptExecWriteChange struct {
	Operation      string                `json:"operation"`
	Path           string                `json:"path,omitempty"`
	Mode           string                `json:"mode,omitempty"`
	AddedLines     int                   `json:"added_lines,omitempty"`
	RemovedLines   int                   `json:"removed_lines,omitempty"`
	ContentBytes   int                   `json:"content_bytes,omitempty"`
	ContentLines   int                   `json:"content_lines,omitempty"`
	ContentPreview string                `json:"content_preview,omitempty"`
	HunkRanges     []scriptExecHunkRange `json:"hunk_ranges,omitempty"`
}

type scriptExecHunkRange struct {
	OldStart int `json:"old_start"`
	OldCount int `json:"old_count"`
	NewStart int `json:"new_start"`
	NewCount int `json:"new_count"`
}

func formatScriptExecReport(output string, toolCallsLog []any) (string, error) {
	steps, summary := buildScriptExecSteps(toolCallsLog)
	report := scriptExecReport{
		ScriptOutput: strings.TrimSpace(output),
		Steps:        steps,
		Summary:      summary,
	}
	if report.ScriptOutput == "" && len(report.Steps) == 0 {
		report.ScriptOutput = "(no output)"
	}
	encoded, err := json.Marshal(report)
	if err != nil {
		return "", fmt.Errorf("encode script_exec report: %w", err)
	}
	return string(encoded), nil
}

func buildScriptExecSteps(rawLogs []any) ([]scriptExecStep, scriptExecSummary) {
	steps := make([]scriptExecStep, 0, len(rawLogs))
	summary := scriptExecSummary{StepCount: len(rawLogs)}
	for idx, raw := range rawLogs {
		step := buildScriptExecStep(idx+1, raw)
		if step.Status == "error" {
			summary.FailedSteps++
		}
		if step.WriteChange != nil {
			summary.WriteSteps++
		}
		steps = append(steps, step)
	}
	return steps, summary
}

func buildScriptExecStep(stepNumber int, raw any) scriptExecStep {
	callMap, ok := raw.(map[string]any)
	if !ok {
		return scriptExecStep{
			Step:   stepNumber,
			Tool:   "unknown",
			Status: "error",
			Error:  "invalid tool call log entry",
		}
	}

	toolName := normalizedToolName(anyString(callMap["tool"]))
	errText := strings.TrimSpace(anyString(callMap["error"]))
	status := "success"
	if errText != "" {
		status = "error"
	}

	argsMap, _ := callMap["args"].(map[string]any)
	return scriptExecStep{
		Step:          stepNumber,
		Tool:          toolName,
		Status:        status,
		Args:          summarizeArgs(argsMap),
		ResultSummary: summarizeResult(toolName, anyString(callMap["result"])),
		Error:         errText,
		WriteChange:   summarizeWriteChange(toolName, argsMap),
	}
}

func summarizeArgs(args map[string]any) map[string]any {
	if len(args) == 0 {
		return nil
	}
	keys := make([]string, 0, len(args))
	for key := range args {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	limit := len(keys)
	if limit > scriptExecMaxArgFields {
		limit = scriptExecMaxArgFields
	}
	out := make(map[string]any, limit+1)
	for _, key := range keys[:limit] {
		out[key] = summarizeArgValue(key, args[key])
	}
	if len(keys) > limit {
		out["truncated_fields"] = len(keys) - limit
	}
	return out
}

func summarizeArgValue(key string, value any) any {
	switch typed := value.(type) {
	case string:
		if key == "path" {
			return strings.TrimSpace(typed)
		}
		return truncateText(strings.TrimSpace(typed), scriptExecMaxArgPreviewChars)
	case bool, int, int64, float64:
		return typed
	case []any:
		return map[string]any{"count": len(typed)}
	case map[string]any:
		return summarizeNestedMap(key, typed)
	default:
		if value == nil {
			return nil
		}
		return truncateText(strings.TrimSpace(fmt.Sprint(value)), scriptExecMaxArgPreviewChars)
	}
}

func summarizeNestedMap(key string, value map[string]any) any {
	if key == "diff_summary" {
		return summarizeDiffSummaryMap(value)
	}
	if key == "content_summary" {
		return map[string]any{
			"bytes": scriptExecInt(value["bytes"]),
			"lines": scriptExecInt(value["lines"]),
		}
	}
	out := make(map[string]any)
	keys := make([]string, 0, len(value))
	for item := range value {
		keys = append(keys, item)
	}
	sort.Strings(keys)
	for _, item := range keys {
		out[item] = summarizeArgValue(item, value[item])
	}
	return out
}

func summarizeDiffSummaryMap(value map[string]any) map[string]any {
	return map[string]any{
		"hunk_count":    scriptExecInt(value["hunk_count"]),
		"added_lines":   scriptExecInt(value["added_lines"]),
		"removed_lines": scriptExecInt(value["removed_lines"]),
		"hunk_ranges":   summarizeHunkRanges(value["hunk_ranges"]),
	}
}

func summarizeHunkRanges(raw any) []scriptExecHunkRange {
	hunkList, ok := raw.([]any)
	if !ok {
		return nil
	}
	limit := len(hunkList)
	if limit > scriptExecMaxHunkRanges {
		limit = scriptExecMaxHunkRanges
	}
	ranges := make([]scriptExecHunkRange, 0, limit)
	for _, item := range hunkList[:limit] {
		hunk, ok := item.(map[string]any)
		if !ok {
			continue
		}
		ranges = append(ranges, scriptExecHunkRange{
			OldStart: scriptExecInt(hunk["old_start"]),
			OldCount: scriptExecInt(hunk["old_count"]),
			NewStart: scriptExecInt(hunk["new_start"]),
			NewCount: scriptExecInt(hunk["new_count"]),
		})
	}
	return ranges
}
