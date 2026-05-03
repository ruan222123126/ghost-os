package orchestration

import (
	"bytes"
	"encoding/json"
	"log"
	"strings"
	"testing"

	"ghost-os/bridge/agent"
	"ghost-os/bridge/llm"
)

func TestProjectMessagesForModelKeepsRecentTwoCompressibleSpansRaw(t *testing.T) {
	messages := []llm.Message{{Role: llm.RoleSystem, Text: "system"}}
	messages = append(messages, toolSpan("c1", "web_search", `{"query":"first"}`, webSearchJSON("Alpha"))...)
	messages = append(messages, toolSpan("c2", "web_search", `{"query":"second"}`, webSearchJSON("Beta"))...)
	messages = append(messages, toolSpan("c3", "web_search", `{"query":"third"}`, webSearchJSON("Gamma"))...)

	projected := projectMessagesForModel(messages, messageProjectionOptions{MicrocompactEnabled: true, TraceID: "trace-recent"})

	assertNoOrphanToolResults(t, projected)
	if len(projected) != 6 {
		t.Fatalf("unexpected projected length: got %d want 6", len(projected))
	}
	if len(projected[1].ToolCalls) != 0 || !strings.Contains(projected[1].Text, `query="first"`) {
		t.Fatalf("expected first span to be summarized, got %+v", projected[1])
	}
	if len(projected[2].ToolCalls) != 1 || projected[2].ToolCalls[0].Name != "web_search" {
		t.Fatalf("expected second span to remain raw, got %+v", projected[2])
	}
	if len(projected[4].ToolCalls) != 1 || projected[4].ToolCalls[0].Name != "web_search" {
		t.Fatalf("expected third span to remain raw, got %+v", projected[4])
	}
}

func TestProjectMessagesForModelPreservesReadFileBody(t *testing.T) {
	messages := []llm.Message{{Role: llm.RoleSystem, Text: "system"}}
	messages = append(messages, toolSpan("read-old", "read_file", `{"path":"main.go"}`, readFileOutput())...)
	messages = append(messages, toolSpan("keep-1", "web_search", `{"query":"latest one"}`, webSearchJSON("One"))...)
	messages = append(messages, toolSpan("keep-2", "web_search", `{"query":"latest two"}`, webSearchJSON("Two"))...)

	projected := projectMessagesForModel(messages, messageProjectionOptions{MicrocompactEnabled: true, TraceID: "trace-read"})

	if !strings.Contains(projected[1].Text, "File: main.go") {
		t.Fatalf("expected compressed read_file to keep path, got %q", projected[1].Text)
	}
	if !strings.Contains(projected[1].Text, "Returned lines: 1-2") {
		t.Fatalf("expected compressed read_file to keep line range, got %q", projected[1].Text)
	}
	if !strings.Contains(projected[1].Text, "1 | package main") {
		t.Fatalf("expected compressed read_file to keep body, got %q", projected[1].Text)
	}
}

func TestProjectMessagesForModelRemovesScreenImageContentFromOldSpan(t *testing.T) {
	messages := []llm.Message{{Role: llm.RoleSystem, Text: "system"}}
	messages = append(messages, screenShotSpan("shot-1")...)
	messages = append(messages, toolSpan("keep-1", "read_file", `{"path":"one.txt"}`, readFileOutput())...)
	messages = append(messages, toolSpan("keep-2", "read_file", `{"path":"two.txt"}`, readFileOutput())...)

	projected := projectMessagesForModel(messages, messageProjectionOptions{MicrocompactEnabled: true, TraceID: "trace-screen"})

	if len(projected[1].Content) != 0 {
		t.Fatalf("expected compressed screen span to drop image content, got %+v", projected[1].Content)
	}
	if !strings.Contains(projected[1].Text, "screen_action screenshot") {
		t.Fatalf("expected screen summary, got %q", projected[1].Text)
	}
	if !strings.Contains(projected[1].Text, "artifact=image") {
		t.Fatalf("expected artifact hint in summary, got %q", projected[1].Text)
	}
}

func TestProjectMessagesForModelUsesStructuredScriptExecSummary(t *testing.T) {
	messages := []llm.Message{{Role: llm.RoleSystem, Text: "system"}}
	messages = append(messages, toolSpan("script-1", "script_exec", `{"script":"print(1)"}`, scriptExecJSON())...)
	messages = append(messages, toolSpan("keep-1", "read_file", `{"path":"one.txt"}`, readFileOutput())...)
	messages = append(messages, toolSpan("keep-2", "read_file", `{"path":"two.txt"}`, readFileOutput())...)

	projected := projectMessagesForModel(messages, messageProjectionOptions{MicrocompactEnabled: true, TraceID: "trace-script"})

	if !strings.Contains(projected[1].Text, "script_exec steps=2 failed=0 writes=1") {
		t.Fatalf("expected structured script_exec summary, got %q", projected[1].Text)
	}
	if strings.Contains(projected[1].Text, "very long raw script output") {
		t.Fatalf("expected raw script output to be compacted, got %q", projected[1].Text)
	}
}

func TestProjectMessagesForModelPreservesErrorDetails(t *testing.T) {
	messages := []llm.Message{{Role: llm.RoleSystem, Text: "system"}}
	messages = append(messages, toolSpanWithError("err-1", "web_search", `{"query":"boom"}`, "search backend unavailable")...)
	messages = append(messages, toolSpan("keep-1", "read_file", `{"path":"one.txt"}`, readFileOutput())...)
	messages = append(messages, toolSpan("keep-2", "read_file", `{"path":"two.txt"}`, readFileOutput())...)

	projected := projectMessagesForModel(messages, messageProjectionOptions{MicrocompactEnabled: true, TraceID: "trace-error"})

	if !strings.Contains(projected[1].Text, "web_search error: search backend unavailable") {
		t.Fatalf("expected compressed error text, got %q", projected[1].Text)
	}
}

func TestProjectMessagesForModelLeavesNonTargetToolSpanUntouched(t *testing.T) {
	messages := []llm.Message{{Role: llm.RoleSystem, Text: "system"}}
	messages = append(messages, toolSpan("ask-1", "ask_human", `{"prompt":"Ship?"}`, `{"status":"awaiting_human"}`)...)

	projected := projectMessagesForModel(messages, messageProjectionOptions{MicrocompactEnabled: true, TraceID: "trace-nontarget"})

	if len(projected) != len(messages) {
		t.Fatalf("expected non-target span to stay unchanged, got %d messages", len(projected))
	}
	if len(projected[1].ToolCalls) != 1 || projected[2].Role != llm.RoleTool {
		t.Fatalf("expected ask_human span to remain raw, got %+v %+v", projected[1], projected[2])
	}
}

func TestProjectMessagesForModelLogsSkipOnUnsupportedPayload(t *testing.T) {
	var buffer bytes.Buffer
	oldOutput := log.Writer()
	log.SetOutput(&buffer)
	t.Cleanup(func() { log.SetOutput(oldOutput) })

	messages := []llm.Message{{Role: llm.RoleSystem, Text: "system"}}
	messages = append(messages, toolSpan("bad-1", "web_search", `{"query":"broken"}`, "not-json")...)
	messages = append(messages, toolSpan("keep-1", "read_file", `{"path":"one.txt"}`, readFileOutput())...)
	messages = append(messages, toolSpan("keep-2", "read_file", `{"path":"two.txt"}`, readFileOutput())...)

	projected := projectMessagesForModel(messages, messageProjectionOptions{MicrocompactEnabled: true, TraceID: "trace-bad"})

	if len(projected) != len(messages) {
		t.Fatalf("expected unsupported span to stay raw, got %d messages", len(projected))
	}
	if !strings.Contains(buffer.String(), "trace_id=trace-bad microcompact skipped: unsupported payload") {
		t.Fatalf("expected skip log, got %q", buffer.String())
	}
}

func TestProjectMessagesForModelReducesTokenEstimate(t *testing.T) {
	messages := []llm.Message{{Role: llm.RoleSystem, Text: "system"}}
	messages = append(messages, toolSpan("c1", "web_search", `{"query":"one"}`, longWebSearchJSON())...)
	messages = append(messages, toolSpan("c2", "web_search", `{"query":"two"}`, longWebSearchJSON())...)
	messages = append(messages, toolSpan("c3", "web_search", `{"query":"three"}`, longWebSearchJSON())...)

	projected := projectMessagesForModel(messages, messageProjectionOptions{MicrocompactEnabled: true, TraceID: "trace-budget"})

	if estimateMessagesTokens(projected) >= estimateMessagesTokens(messages) {
		t.Fatalf("expected token estimate to drop: before=%d after=%d", estimateMessagesTokens(messages), estimateMessagesTokens(projected))
	}
}

func TestProjectMessagesForModelKeepsReasoningReplayTurnRaw(t *testing.T) {
	messages := []llm.Message{
		{Role: llm.RoleSystem, Text: "system"},
		{Role: llm.RoleUser, Text: "browser demo"},
		{
			Role:             llm.RoleAssistant,
			ReasoningContent: json.RawMessage(`"step 1"`),
			ToolCalls: []llm.ToolCall{{
				ID:        "call-1",
				Name:      "script_exec",
				Arguments: json.RawMessage(`{"script":"one"}`),
			}},
		},
		{
			Role:       llm.RoleTool,
			ToolCallID: "call-1",
			Text:       agent.FormatToolResult("script_exec", "trace-1", scriptExecJSON(), nil),
		},
		{
			Role:             llm.RoleAssistant,
			ReasoningContent: json.RawMessage(`"step 2"`),
			ToolCalls: []llm.ToolCall{{
				ID:        "call-2",
				Name:      "script_exec",
				Arguments: json.RawMessage(`{"script":"two"}`),
			}},
		},
		{
			Role:       llm.RoleTool,
			ToolCallID: "call-2",
			Text:       agent.FormatToolResult("script_exec", "trace-2", scriptExecJSON(), nil),
		},
		{Role: llm.RoleAssistant, Text: "done"},
	}

	projected := projectMessagesForModel(messages, messageProjectionOptions{MicrocompactEnabled: true, TraceID: "trace-replay"})

	if len(projected) != len(messages) {
		t.Fatalf("expected reasoning replay turn to remain raw, got %d want %d", len(projected), len(messages))
	}
	if len(projected[2].ToolCalls) != 1 || len(projected[4].ToolCalls) != 1 {
		t.Fatalf("expected tool call spans to remain raw, got %+v %+v", projected[2], projected[4])
	}
}

func toolSpan(callID string, toolName string, args string, output string) []llm.Message {
	return []llm.Message{
		{
			Role: llm.RoleAssistant,
			ToolCalls: []llm.ToolCall{{
				ID:        callID,
				Name:      toolName,
				Arguments: json.RawMessage(args),
			}},
		},
		{
			Role:       llm.RoleTool,
			ToolCallID: callID,
			Text:       agent.FormatToolResult(toolName, "trace-"+callID, output, nil),
		},
	}
}

func toolSpanWithError(callID string, toolName string, args string, errText string) []llm.Message {
	span := toolSpan(callID, toolName, args, "")
	span[1].Text = agent.FormatToolResult(toolName, "trace-"+callID, "", errString(errText))
	return span
}

func screenShotSpan(callID string) []llm.Message {
	span := toolSpan(callID, "screen_action", `{"action":"screenshot","params":{"display_id":2}}`, `{"action":"screenshot","display_id":2,"artifact":{"type":"image"}}`)
	span[1].Content = []llm.ContentPart{{Type: llm.ContentTypeImage, Image: &llm.ImageContent{Path: "/tmp/shot.png"}}}
	return span
}

func readFileOutput() string {
	return "File: main.go\nRequested lines: 1-2\nReturned lines: 1-2 of 2 total\n1 | package main\n2 | func main() {}"
}

func webSearchJSON(title string) string {
	return `[{"title":"` + title + `","url":"https://example.com/` + strings.ToLower(title) + `","snippet":"snippet"}]`
}

func longWebSearchJSON() string {
	return `[{"title":"Alpha","url":"https://example.com/a","snippet":"` + strings.Repeat("very long snippet ", 40) + `"}]`
}

func scriptExecJSON() string {
	return `{"script_output":"very long raw script output ` + strings.Repeat("tail ", 30) + `","steps":[{"tool":"read_file","status":"success","result_summary":"returned 20 lines"},{"tool":"apply_diff","status":"success","write_change":{"operation":"apply_diff","path":"main.go","added_lines":3,"removed_lines":1}}],"summary":{"step_count":2,"failed_steps":0,"write_steps":1}}`
}

func assertNoOrphanToolResults(t *testing.T, messages []llm.Message) {
	t.Helper()
	pending := map[string]bool{}
	for index, message := range messages {
		if message.Role == llm.RoleAssistant {
			for _, call := range message.ToolCalls {
				pending[call.ID] = true
			}
			continue
		}
		if message.Role != llm.RoleTool {
			continue
		}
		if !pending[message.ToolCallID] {
			t.Fatalf("orphan tool result at index %d: %+v", index, message)
		}
		delete(pending, message.ToolCallID)
	}
	if len(pending) != 0 {
		t.Fatalf("missing tool results for %v", pending)
	}
}

type errString string

func (e errString) Error() string {
	return string(e)
}
