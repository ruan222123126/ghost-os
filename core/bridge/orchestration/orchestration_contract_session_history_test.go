package orchestration

import (
	"bytes"
	"encoding/json"
	"ghost-os/bridge/agent"
	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/llm"
	"ghost-os/bridge/session"
	"log"
	"strings"
	"testing"
	"time"
)

func TestSessionHistoryBuilderKeepsLongHistoryWhenProviderContextWindowIsConfigured(t *testing.T) {
	sess := session.NewSession("system")
	for i := 0; i < 12; i++ {
		sess.AddMessage(llm.Message{
			Role: llm.RoleUser,
			Text: strings.Repeat("browser tool replay payload ", 120),
		})
	}

	before := sess.Messages
	builder := newSessionHistoryBuilder(
		bridgeconfig.ProviderConfig{
			Type:                llm.ProviderCustom,
			Model:               "deepseek-v4-pro",
			ContextWindowTokens: 1000000,
		},
		"system",
		nil,
		3,
		false,
		"",
	)
	history := builder.BuildHistory(sess)
	after := history.Messages()

	if len(after) != len(before) {
		t.Fatalf("unexpected pruned message count: got %d want %d", len(after), len(before))
	}
	if after[len(after)-1].Text != before[len(before)-1].Text {
		t.Fatalf("expected last message to remain intact")
	}
}

func TestSessionHistoryBuilder_BuildHistoryWithResolvedQuestionsReturnsAnsweredQuestions(t *testing.T) {
	sess := session.NewSession("system")
	createdAt := time.Date(2026, 3, 7, 11, 0, 0, 0, time.UTC)
	sess.AddMessage(llm.Message{
		Role: llm.RoleAssistant,
		ToolCalls: []llm.ToolCall{{
			ID:        "call-ask",
			Name:      "ask_human",
			Arguments: json.RawMessage(`{"prompt":"Ship now?"}`),
		}},
	})
	sess.AddPendingQuestion("q-1", session.PendingHumanQuestion{
		Prompt:     "Ship now?",
		ToolCallID: "call-ask",
		TraceID:    "trace-q1",
		CreatedAt:  createdAt,
	})
	if !sess.SetHumanAnswer("q-1", "yes") {
		t.Fatalf("expected human answer to be accepted")
	}

	builder := newSessionHistoryBuilder(
		bridgeconfig.ProviderConfig{Type: llm.ProviderOpenAI, Model: "gpt-4o"},
		"system",
		nil,
		3,
		false,
		"",
	)
	history, resolved := builder.BuildHistoryWithResolvedQuestions(sess)
	if len(resolved) != 1 {
		t.Fatalf("expected 1 resolved question, got %d", len(resolved))
	}
	if resolved[0].QuestionID != "q-1" || resolved[0].Prompt != "Ship now?" || resolved[0].Answer != "yes" {
		t.Fatalf("unexpected resolved question: %+v", resolved[0])
	}
	if resolved[0].AskedAt != createdAt {
		t.Fatalf("unexpected asked_at: got %s want %s", resolved[0].AskedAt, createdAt)
	}
	if resolved[0].AnsweredAt.IsZero() {
		t.Fatalf("expected answered_at to be populated")
	}
	if len(sess.PendingQuestions) != 0 || len(sess.HumanAnswers) != 0 {
		t.Fatalf("expected resolved question to be consumed, pending=%v answers=%v", sess.PendingQuestions, sess.HumanAnswers)
	}

	messages := history.Messages()
	if len(messages) == 0 {
		t.Fatalf("expected history messages")
	}
	last := messages[len(messages)-1]
	if last.Role != llm.RoleTool || last.ToolCallID != "call-ask" {
		t.Fatalf("expected injected tool message, got %+v", last)
	}
	envelope, ok := agent.ParseToolResultEnvelope(last.Text)
	if !ok {
		t.Fatalf("expected tool result envelope, got %q", last.Text)
	}
	if envelope.Tool != "ask_human" {
		t.Fatalf("unexpected tool name: %q", envelope.Tool)
	}
	if envelope.TraceID != "trace-q1" {
		t.Fatalf("unexpected trace id: %q", envelope.TraceID)
	}
}

func TestSessionHistoryBuilder_ProjectsToolSearchLoadSpanForModel(t *testing.T) {
	sess := session.NewSession("system")
	sess.AddMessage(llm.Message{
		Role: llm.RoleAssistant,
		ToolCalls: []llm.ToolCall{{
			ID:        "call-sfind-load",
			Name:      "sfind",
			Arguments: []byte(`{"action":"load","skill_names":["release_flow"]}`),
		}},
	})
	sess.AddMessage(llm.Message{
		Role:       llm.RoleTool,
		ToolCallID: "call-sfind-load",
		Text: agent.FormatToolResult(
			"sfind",
			"trace-sfind-load",
			`{"action":"load","kind":"skill","items":[{"name":"release_flow","status":"loaded","available_now":true}]}`,
			nil,
		),
	})
	sess.AddMessage(llm.Message{
		Role: llm.RoleAssistant,
		ToolCalls: []llm.ToolCall{{
			ID:        "call-keep-1",
			Name:      "read_file",
			Arguments: []byte(`{"path":"one.txt"}`),
		}},
	})
	sess.AddMessage(llm.Message{
		Role:       llm.RoleTool,
		ToolCallID: "call-keep-1",
		Text:       agent.FormatToolResult("read_file", "trace-keep-1", "File: one.txt", nil),
	})
	sess.AddMessage(llm.Message{
		Role: llm.RoleAssistant,
		ToolCalls: []llm.ToolCall{{
			ID:        "call-keep-2",
			Name:      "read_file",
			Arguments: []byte(`{"path":"two.txt"}`),
		}},
	})
	sess.AddMessage(llm.Message{
		Role:       llm.RoleTool,
		ToolCallID: "call-keep-2",
		Text:       agent.FormatToolResult("read_file", "trace-keep-2", "File: two.txt", nil),
	})

	builder := newSessionHistoryBuilder(
		bridgeconfig.ProviderConfig{Type: llm.ProviderOpenAI, Model: "gpt-4o"},
		"system",
		nil,
		3,
		true,
		"",
	)
	history := builder.BuildHistory(sess)
	messages := history.Messages()
	if len(messages) != 6 {
		t.Fatalf("unexpected message count: got %d want 6", len(messages))
	}
	if messages[1].Role != llm.RoleAssistant {
		t.Fatalf("expected projected assistant summary, got %+v", messages[1])
	}
	if !strings.Contains(messages[1].Text, "Loaded dynamic session skills via sfind: `release_flow`.") {
		t.Fatalf("unexpected projected summary: %q", messages[1].Text)
	}
	if !strings.Contains(messages[1].Text, "available now in the current user turn") {
		t.Fatalf("expected immediate-availability hint in projected summary, got %q", messages[1].Text)
	}
}

func TestSessionHistoryBuilder_KeepsToolSearchSearchSpanUnchanged(t *testing.T) {
	sess := session.NewSession("system")
	sess.AddMessage(llm.Message{
		Role: llm.RoleAssistant,
		ToolCalls: []llm.ToolCall{{
			ID:        "call-sfind-search",
			Name:      "sfind",
			Arguments: []byte(`{"action":"search","query":"web"}`),
		}},
	})
	sess.AddMessage(llm.Message{
		Role:       llm.RoleTool,
		ToolCallID: "call-sfind-search",
		Text: agent.FormatToolResult(
			"sfind",
			"trace-sfind-search",
			`{"action":"search","kind":"skill","items":[{"name":"release_flow","summary":"Release workflow."}]}`,
			nil,
		),
	})

	builder := newSessionHistoryBuilder(
		bridgeconfig.ProviderConfig{Type: llm.ProviderOpenAI, Model: "gpt-4o"},
		"system",
		nil,
		3,
		true,
		"",
	)
	history := builder.BuildHistory(sess)
	messages := history.Messages()
	if len(messages) != 3 {
		t.Fatalf("unexpected message count: got %d want 3", len(messages))
	}
	if len(messages[1].ToolCalls) != 1 || messages[1].ToolCalls[0].Name != "sfind" {
		t.Fatalf("expected sfind tool call to remain in history, got %+v", messages[1])
	}
	if messages[2].Role != llm.RoleTool || messages[2].ToolCallID != "call-sfind-search" {
		t.Fatalf("expected tool result to remain in history, got %+v", messages[2])
	}
}

func TestSessionHistoryBuilder_DropsOrphanToolMessageAfterCompletedAnswer(t *testing.T) {
	sess := session.NewSession("system")
	sess.AddMessage(llm.Message{Role: llm.RoleUser, Text: "看看影视飓风的粉丝数"})
	sess.AddMessage(llm.Message{
		Role: llm.RoleAssistant,
		ToolCalls: []llm.ToolCall{{
			ID:        "call-search-1",
			Name:      "web_search",
			Arguments: json.RawMessage(`{"query":"影视飓风 B站 粉丝数"}`),
		}},
	})
	sess.AddMessage(llm.Message{
		Role:       llm.RoleTool,
		ToolCallID: "call-search-1",
		Text:       agent.FormatToolResult("web_search", "trace-search-1", "ok", nil),
	})
	sess.AddMessage(llm.Message{
		Role: llm.RoleAssistant,
		Text: "粉丝数是 1612.6 万。",
	})
	sess.AddMessage(llm.Message{
		Role:       llm.RoleTool,
		ToolCallID: "call-orphan-1",
		Text:       agent.FormatToolResult("bash_exec", "trace-orphan-1", "orphan", nil),
	})

	builder := newSessionHistoryBuilder(
		bridgeconfig.ProviderConfig{Type: llm.ProviderOpenAI, Model: "gpt-4o"},
		"system",
		nil,
		3,
		false,
		"",
	)
	history := builder.BuildHistory(sess)
	messages := history.Messages()

	if len(messages) != 5 {
		t.Fatalf("unexpected sanitized message count: got %d want %d", len(messages), 5)
	}
	last := messages[len(messages)-1]
	if last.Role != llm.RoleAssistant || last.Text != "粉丝数是 1612.6 万。" {
		t.Fatalf("expected orphan tool message to be removed, got %+v", last)
	}
	if messages[2].Role != llm.RoleAssistant || len(messages[2].ToolCalls) != 1 {
		t.Fatalf("expected valid tool-call assistant message to remain intact, got %+v", messages[2])
	}
	if messages[3].Role != llm.RoleTool || messages[3].ToolCallID != "call-search-1" {
		t.Fatalf("expected matching tool result to remain intact, got %+v", messages[3])
	}
}

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

func TestProjectMessagesForModelUsesCodexCLIFinalMessage(t *testing.T) {
	messages := []llm.Message{{Role: llm.RoleSystem, Text: "system"}}
	messages = append(messages, toolSpan("codex-1", "codex_cli", `{"op":"status"}`, codexCLIDoneJSON())...)
	messages = append(messages, toolSpan("keep-1", "read_file", `{"path":"one.txt"}`, readFileOutput())...)
	messages = append(messages, toolSpan("keep-2", "read_file", `{"path":"two.txt"}`, readFileOutput())...)

	projected := projectMessagesForModel(messages, messageProjectionOptions{MicrocompactEnabled: true, TraceID: "trace-codex"})

	if !strings.Contains(projected[1].Text, `result="child completed cleanly"`) {
		t.Fatalf("expected codex final message in summary, got %q", projected[1].Text)
	}
	if strings.Contains(projected[1].Text, "usage tail only") {
		t.Fatalf("expected final_message to win over output_tail, got %q", projected[1].Text)
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

func codexCLIDoneJSON() string {
	return `{"status":"done","command_id":"codex-cli-1","exit_code":0,"final_message":"child completed cleanly","output_tail":"usage tail only"}`
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
