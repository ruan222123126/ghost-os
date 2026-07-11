package session

import (
	"encoding/json"
	"fmt"
	"strings"

	"ghost-os/bridge/llm"
)

const sessionHumanLogSummaryMaxRunes = 1200

type sessionHumanLogToolEnvelope struct {
	Status  string `json:"status"`
	Tool    string `json:"tool"`
	TraceID string `json:"trace_id"`
	Output  string `json:"output"`
	Error   string `json:"error"`
}

type sessionHumanLogCallView struct {
	Name      string
	Arguments string
}

type assistantTagCall struct {
	ID        string
	Arguments string
}

func renderSessionHumanLogDocument(meta sessionHumanLogMeta, body string) string {
	var builder strings.Builder
	builder.WriteString("# Session `")
	builder.WriteString(meta.SessionID)
	builder.WriteString("`\n\n")
	builder.WriteString(renderSessionHumanLogMetaMarker(meta))
	builder.WriteString("\n\n")
	builder.WriteString("- session_id: `")
	builder.WriteString(meta.SessionID)
	builder.WriteString("`\n")
	builder.WriteString("- created_at: `")
	builder.WriteString(meta.CreatedAt)
	builder.WriteString("`\n")
	builder.WriteString("- updated_at: `")
	builder.WriteString(meta.UpdatedAt)
	builder.WriteString("`\n")
	builder.WriteString("- export_mode: `")
	builder.WriteString(string(meta.ExportMode))
	builder.WriteString("`\n")
	builder.WriteString("- last_exported_index: `")
	fmt.Fprintf(&builder, "%d", meta.LastExportedIndex)
	builder.WriteString("`\n\n---\n\n")
	builder.WriteString(sessionHumanLogBodyMarker)
	builder.WriteString(body)
	return builder.String()
}

func renderSessionHumanLogMetaMarker(meta sessionHumanLogMeta) string {
	encoded, err := encodeSessionHumanLogMeta(meta)
	if err != nil {
		return sessionHumanLogMetaPrefix + `{"session_id":"","export_mode":"summary","last_exported_index":-1}` + sessionHumanLogMetaSuffix
	}
	return sessionHumanLogMetaPrefix + encoded + sessionHumanLogMetaSuffix
}

func renderSessionHumanLogBody(messages []IndexedMessage, mode sessionHumanLogMode) string {
	var builder strings.Builder
	for _, item := range messages {
		block := renderSessionHumanLogMessage(item, mode)
		if block == "" {
			continue
		}
		builder.WriteString(block)
	}
	return builder.String()
}

func renderSessionHumanLogMessage(item IndexedMessage, mode sessionHumanLogMode) string {
	switch item.Message.Role {
	case llm.RoleUser:
		return renderSessionHumanLogUserMessage(item)
	case llm.RoleAssistant:
		return renderSessionHumanLogAssistantMessage(item)
	case llm.RoleTool:
		return renderSessionHumanLogToolMessage(item, mode)
	default:
		return ""
	}
}

func renderSessionHumanLogUserMessage(item IndexedMessage) string {
	var builder strings.Builder
	writeSessionHumanLogBlockHeader(&builder, item.Index, llm.RoleUser)
	writeSessionHumanLogTextSection(&builder, "text", item.Message.Text)

	attachments := summarizeSessionHumanLogContent(item.Message.Content)
	if strings.TrimSpace(attachments) != "" {
		builder.WriteString("### attachments\n")
		builder.WriteString(attachments)
		builder.WriteString("\n")
	}
	builder.WriteString("\n")
	return builder.String()
}

func renderSessionHumanLogAssistantMessage(item IndexedMessage) string {
	visibleText, tagCalls := extractAssistantVisibleTextAndTagCalls(item.Message.Text)
	calls := buildSessionHumanLogAssistantCalls(item.Message.ToolCalls, tagCalls)

	var builder strings.Builder
	writeSessionHumanLogBlockHeader(&builder, item.Index, llm.RoleAssistant)
	writeSessionHumanLogTextSection(&builder, "text", visibleText)
	if len(calls) > 0 {
		builder.WriteString("### tool_calls\n")
		builder.WriteString(renderSessionHumanLogCalls(calls))
		builder.WriteString("\n")
	}
	builder.WriteString("\n")
	return builder.String()
}

func renderSessionHumanLogToolMessage(item IndexedMessage, mode sessionHumanLogMode) string {
	envelope, ok := parseSessionHumanLogToolEnvelope(item.Message.Text)
	meta := resolveSessionHumanLogToolMeta(item.Message.Text, envelope, ok)

	var builder strings.Builder
	writeSessionHumanLogBlockHeader(&builder, item.Index, llm.RoleTool)
	builder.WriteString("### tool\n`")
	builder.WriteString(meta.Tool)
	builder.WriteString("`\n\n")
	builder.WriteString("### status\n`")
	builder.WriteString(meta.Status)
	builder.WriteString("`\n\n")
	builder.WriteString("### trace_id\n`")
	builder.WriteString(meta.TraceID)
	builder.WriteString("`\n\n")

	if mode == sessionHumanLogModeFull {
		writeSessionHumanLogTextSection(&builder, "output", meta.Output)
		writeSessionHumanLogTextSection(&builder, "error", meta.Error)
	} else {
		writeSessionHumanLogTextSection(&builder, "result_summary", summarizeSessionHumanLogToolResult(meta))
	}
	builder.WriteString("\n")
	return builder.String()
}

func writeSessionHumanLogBlockHeader(builder *strings.Builder, index int, role llm.Role) {
	builder.WriteString("## [")
	fmt.Fprintf(builder, "%d", index)
	builder.WriteString("] `")
	builder.WriteString(string(role))
	builder.WriteString("`\n\n")
}

func writeSessionHumanLogTextSection(builder *strings.Builder, title string, text string) {
	builder.WriteString("### ")
	builder.WriteString(title)
	builder.WriteString("\n")
	builder.WriteString("```text\n")
	if strings.TrimSpace(text) == "" {
		builder.WriteString("(empty)\n")
	} else {
		builder.WriteString(text)
		if !strings.HasSuffix(text, "\n") {
			builder.WriteString("\n")
		}
	}
	builder.WriteString("```\n\n")
}

func renderSessionHumanLogCalls(calls []sessionHumanLogCallView) string {
	var builder strings.Builder
	for index, call := range calls {
		fmt.Fprintf(&builder, "%d. `%s`\n", index+1, call.Name)
		builder.WriteString("```json\n")
		builder.WriteString(normalizeSessionHumanLogJSON(call.Arguments))
		builder.WriteString("\n```\n")
	}
	return builder.String()
}

func buildSessionHumanLogAssistantCalls(calls []llm.ToolCall, tagCalls []assistantTagCall) []sessionHumanLogCallView {
	if len(calls) > 0 {
		out := make([]sessionHumanLogCallView, 0, len(calls))
		for _, call := range calls {
			out = append(out, sessionHumanLogCallView{
				Name:      strings.TrimSpace(call.Name),
				Arguments: string(call.Arguments),
			})
		}
		return out
	}
	if len(tagCalls) == 0 {
		return nil
	}
	out := make([]sessionHumanLogCallView, 0, len(tagCalls))
	for _, tagCall := range tagCalls {
		out = append(out, sessionHumanLogCallView{
			Name:      "tag:" + strings.TrimSpace(tagCall.ID),
			Arguments: tagCall.Arguments,
		})
	}
	return out
}

func parseSessionHumanLogToolEnvelope(raw string) (sessionHumanLogToolEnvelope, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return sessionHumanLogToolEnvelope{}, false
	}
	var envelope sessionHumanLogToolEnvelope
	if err := json.Unmarshal([]byte(trimmed), &envelope); err != nil {
		return sessionHumanLogToolEnvelope{}, false
	}
	if strings.TrimSpace(envelope.Status) == "" || strings.TrimSpace(envelope.Tool) == "" {
		return sessionHumanLogToolEnvelope{}, false
	}
	return envelope, true
}

func resolveSessionHumanLogToolMeta(raw string, envelope sessionHumanLogToolEnvelope, ok bool) sessionHumanLogToolEnvelope {
	if ok {
		return sessionHumanLogToolEnvelope{
			Status:  strings.TrimSpace(envelope.Status),
			Tool:    strings.TrimSpace(envelope.Tool),
			TraceID: strings.TrimSpace(envelope.TraceID),
			Output:  envelope.Output,
			Error:   envelope.Error,
		}
	}
	return sessionHumanLogToolEnvelope{
		Status: "unknown",
		Tool:   "unknown",
		Output: raw,
		Error:  "",
	}
}

func summarizeSessionHumanLogToolResult(meta sessionHumanLogToolEnvelope) string {
	source := strings.TrimSpace(meta.Output)
	if strings.TrimSpace(meta.Error) != "" {
		source = strings.TrimSpace(meta.Error)
	}
	if source == "" {
		source = "(empty)"
	}
	return truncateSessionHumanLogSummary(source, sessionHumanLogSummaryMaxRunes)
}

func truncateSessionHumanLogSummary(value string, maxRunes int) string {
	if maxRunes <= 0 {
		return value
	}
	runes := []rune(value)
	if len(runes) <= maxRunes {
		return value
	}
	return string(runes[:maxRunes]) + fmt.Sprintf(" ... (truncated, total_runes=%d)", len(runes))
}
