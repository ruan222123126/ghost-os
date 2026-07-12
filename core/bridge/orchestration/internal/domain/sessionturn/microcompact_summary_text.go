package sessionturn

import (
	"fmt"
	"strings"

	"ghost-os/bridge/internal/stringutil"
)

type microcompactWebSearchArgs struct {
	Query string `json:"query"`
}

type microcompactWebSearchResult struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}

type microcompactScriptExecReport struct {
	ScriptOutput string                        `json:"script_output,omitempty"`
	Steps        []microcompactScriptExecStep  `json:"steps,omitempty"`
	Summary      microcompactScriptExecSummary `json:"summary"`
}

type microcompactScriptExecSummary struct {
	StepCount   int `json:"step_count"`
	FailedSteps int `json:"failed_steps"`
	WriteSteps  int `json:"write_steps"`
}

type microcompactScriptExecStep struct {
	Tool          string                             `json:"tool"`
	Status        string                             `json:"status"`
	ResultSummary string                             `json:"result_summary,omitempty"`
	Error         string                             `json:"error,omitempty"`
	WriteChange   *microcompactScriptExecWriteChange `json:"write_change,omitempty"`
}

type microcompactScriptExecWriteChange struct {
	Operation    string `json:"operation"`
	Path         string `json:"path,omitempty"`
	AddedLines   int    `json:"added_lines,omitempty"`
	RemovedLines int    `json:"removed_lines,omitempty"`
	ContentLines int    `json:"content_lines,omitempty"`
}

type microcompactCodexCLIArgs struct {
	Op        string `json:"op"`
	SessionID string `json:"session_id,omitempty"`
}

type microcompactCodexCLIResult struct {
	Status       string `json:"status"`
	CommandID    string `json:"command_id,omitempty"`
	SessionID    string `json:"session_id,omitempty"`
	ExitCode     *int   `json:"exit_code,omitempty"`
	OutputTail   string `json:"output_tail,omitempty"`
	FinalMessage string `json:"final_message,omitempty"`
	Message      string `json:"message,omitempty"`
}

func summarizeReadFileResult(pair microcompactToolPair) string {
	output := strings.TrimSpace(pair.envelope.Output)
	if output == "" {
		return "read_file returned no content."
	}
	return output
}

func summarizeWebSearchResult(pair microcompactToolPair) (string, error) {
	args, err := decodeJSONText[microcompactWebSearchArgs](string(pair.call.Arguments))
	if err != nil {
		return "", microcompactParseFailed(pair.toolName, pair.toolCallID)
	}
	results, err := decodeJSONText[[]microcompactWebSearchResult](pair.envelope.Output)
	if err != nil {
		return "", microcompactUnsupported(pair.toolName, pair.toolCallID, "web_search results")
	}
	query := truncateMicrocompactText(args.Query, microcompactMaxTextPreview)
	items := summarizeWebSearchItems(results)
	return fmt.Sprintf("web_search query=%q results=%d top=%s", query, len(results), items), nil
}

func summarizeWebSearchItems(results []microcompactWebSearchResult) string {
	if len(results) == 0 {
		return "none"
	}
	limit := len(results)
	if limit > microcompactTopSearchResults {
		limit = microcompactTopSearchResults
	}
	parts := make([]string, 0, limit)
	for _, item := range results[:limit] {
		label := strings.TrimSpace(item.Title)
		if label == "" {
			label = resultDomain(item.URL)
		}
		if label != "" {
			parts = append(parts, truncateMicrocompactText(label, microcompactMaxTextPreview))
		}
	}
	if len(parts) == 0 {
		return "none"
	}
	return strings.Join(parts, ", ")
}

func summarizeScriptExecResult(pair microcompactToolPair) (string, error) {
	report, err := decodeJSONText[microcompactScriptExecReport](pair.envelope.Output)
	if err != nil {
		return "", microcompactUnsupported(pair.toolName, pair.toolCallID, "script_exec report")
	}
	summary := fmt.Sprintf(
		"script_exec steps=%d failed=%d writes=%d",
		report.Summary.StepCount,
		report.Summary.FailedSteps,
		report.Summary.WriteSteps,
	)
	details := summarizeScriptExecDetails(report)
	if details == "" {
		return summary, nil
	}
	return summary + " | " + details, nil
}

func summarizeScriptExecDetails(report microcompactScriptExecReport) string {
	parts := make([]string, 0, len(report.Steps)+1)
	for _, step := range report.Steps {
		if detail := scriptExecStepDetail(step); detail != "" {
			parts = append(parts, detail)
		}
		if len(parts) == microcompactTopSearchResults {
			break
		}
	}
	if len(parts) > 0 {
		return strings.Join(parts, "; ")
	}
	return truncateMicrocompactText(report.ScriptOutput, microcompactMaxTextPreview)
}

func scriptExecStepDetail(step microcompactScriptExecStep) string {
	if detail := scriptExecWriteDetail(step.WriteChange); detail != "" {
		return detail
	}
	if strings.TrimSpace(step.Error) != "" {
		return step.Tool + ": " + truncateMicrocompactText(step.Error, microcompactMaxTextPreview)
	}
	if strings.TrimSpace(step.ResultSummary) != "" {
		return step.Tool + ": " + truncateMicrocompactText(step.ResultSummary, microcompactMaxTextPreview)
	}
	return ""
}

func scriptExecWriteDetail(change *microcompactScriptExecWriteChange) string {
	if change == nil {
		return ""
	}
	detail := change.Operation
	if path := strings.TrimSpace(change.Path); path != "" {
		detail += " path=" + path
	}
	switch {
	case change.AddedLines != 0 || change.RemovedLines != 0:
		detail += fmt.Sprintf(" +%d/-%d", change.AddedLines, change.RemovedLines)
	case change.ContentLines != 0:
		detail += fmt.Sprintf(" lines=%d", change.ContentLines)
	}
	return detail
}

func summarizeCodexCLIResult(pair microcompactToolPair) (string, error) {
	args, err := decodeJSONText[microcompactCodexCLIArgs](string(pair.call.Arguments))
	if err != nil {
		return "", microcompactParseFailed(pair.toolName, pair.toolCallID)
	}
	result, err := decodeJSONText[microcompactCodexCLIResult](pair.envelope.Output)
	if err != nil {
		return "", microcompactUnsupported(pair.toolName, pair.toolCallID, "codex_cli result")
	}
	parts := []string{
		"codex_cli",
		"op=" + strings.TrimSpace(args.Op),
		"status=" + strings.TrimSpace(result.Status),
	}
	appendCodexCLIField(&parts, "command_id", result.CommandID)
	appendCodexCLIField(&parts, "session_id", stringutil.FirstNonEmpty(result.SessionID, args.SessionID))
	if result.ExitCode != nil {
		parts = append(parts, fmt.Sprintf("exit_code=%d", *result.ExitCode))
	}
	if detail := stringutil.FirstNonEmpty(result.FinalMessage, result.Message, result.OutputTail); detail != "" {
		parts = append(parts, fmt.Sprintf("result=%q", truncateMicrocompactText(detail, microcompactMaxTextPreview)))
	}
	return strings.Join(parts, " "), nil
}

func appendCodexCLIField(parts *[]string, key string, value string) {
	if trimmed := strings.TrimSpace(value); trimmed != "" {
		*parts = append(*parts, key+"="+trimmed)
	}
}
