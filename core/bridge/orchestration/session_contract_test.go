package orchestration

import (
	"encoding/json"
	"strings"
	"testing"

	"ghost-os/bridge/agent"
	"ghost-os/bridge/llm"
	"ghost-os/bridge/session"
)

func TestBuildSessionMessagePayloadProjectsToolResult(t *testing.T) {
	message := llm.Message{
		Role:       llm.RoleTool,
		ToolCallID: "call-1",
		Text:       agent.FormatToolResult("read_file", "trace-1", "README.md contents", nil),
	}

	payload := buildSessionMessagePayload(3, message)
	if payload.Index != 3 {
		t.Fatalf("unexpected index: got %d want %d", payload.Index, 3)
	}
	if payload.Role != string(llm.RoleTool) {
		t.Fatalf("unexpected role: got %q want %q", payload.Role, llm.RoleTool)
	}
	if payload.Text != "README.md contents" {
		t.Fatalf("unexpected projected text: got %q want %q", payload.Text, "README.md contents")
	}
	if strings.Contains(payload.Text, `"status"`) {
		t.Fatalf("tool text should not leak internal envelope: %q", payload.Text)
	}
	if payload.ToolCallID != "call-1" {
		t.Fatalf("unexpected tool_call_id: got %q want %q", payload.ToolCallID, "call-1")
	}
	if payload.ToolResult == nil {
		t.Fatal("expected tool_result projection")
	}
	if payload.ToolResult.Tool != "read_file" {
		t.Fatalf("unexpected tool name: got %q want %q", payload.ToolResult.Tool, "read_file")
	}
	if payload.ToolResult.Status != "success" {
		t.Fatalf("unexpected status: got %q want %q", payload.ToolResult.Status, "success")
	}
	if payload.ToolResult.TraceID != "trace-1" {
		t.Fatalf("unexpected trace id: got %q want %q", payload.ToolResult.TraceID, "trace-1")
	}
	if payload.ToolResult.Output != "README.md contents" {
		t.Fatalf("unexpected output: got %q want %q", payload.ToolResult.Output, "README.md contents")
	}
	if payload.HumanInteraction != nil {
		t.Fatal("unexpected human_interaction for regular tool result")
	}
}

func TestBuildSessionMessagePayloadProjectsAnsweredAskHuman(t *testing.T) {
	toolOutput := `{"question_id":"q-1","prompt":"Which database should I use?","selection_mode":"single","options":[{"label":"PostgreSQL"},{"label":"Other","allow_custom":true}],"answer":"PostgreSQL"}`
	message := llm.Message{
		Role: llm.RoleTool,
		Text: agent.FormatToolResult("ask_human", "trace-2", toolOutput, nil),
	}

	payload := buildSessionMessagePayload(5, message)
	if payload.Index != 5 {
		t.Fatalf("unexpected index: got %d want %d", payload.Index, 5)
	}
	if strings.Contains(payload.Text, `"question_id"`) {
		t.Fatalf("tool text should not leak ask_human payload: %q", payload.Text)
	}
	if !strings.Contains(payload.Text, "Which database should I use?") {
		t.Fatalf("projected text should include prompt: %q", payload.Text)
	}
	if !strings.Contains(payload.Text, "PostgreSQL") {
		t.Fatalf("projected text should include answer: %q", payload.Text)
	}
	if payload.ToolResult == nil {
		t.Fatal("expected tool_result projection")
	}
	if payload.ToolResult.Tool != "ask_human" {
		t.Fatalf("unexpected tool name: got %q want %q", payload.ToolResult.Tool, "ask_human")
	}
	if payload.ToolResult.Output != "" {
		t.Fatalf("ask_human tool_result.output should stay hidden, got %q", payload.ToolResult.Output)
	}
	if payload.HumanInteraction == nil {
		t.Fatal("expected human_interaction projection")
	}
	if payload.HumanInteraction.QuestionID != "q-1" {
		t.Fatalf("unexpected question id: got %q want %q", payload.HumanInteraction.QuestionID, "q-1")
	}
	if payload.HumanInteraction.Prompt != "Which database should I use?" {
		t.Fatalf("unexpected prompt: got %q want %q", payload.HumanInteraction.Prompt, "Which database should I use?")
	}
	if payload.HumanInteraction.SelectionMode != session.HumanQuestionSelectionSingle {
		t.Fatalf("unexpected selection mode: got %q want %q", payload.HumanInteraction.SelectionMode, session.HumanQuestionSelectionSingle)
	}
	if len(payload.HumanInteraction.Options) != 2 || !payload.HumanInteraction.Options[1].AllowCustom {
		t.Fatalf("unexpected options: %+v", payload.HumanInteraction.Options)
	}
	if payload.HumanInteraction.Answer != "PostgreSQL" {
		t.Fatalf("unexpected answer: got %q want %q", payload.HumanInteraction.Answer, "PostgreSQL")
	}
}

func TestBuildSessionMessagePayloadProjectsAssistantThinking(t *testing.T) {
	payload := buildSessionMessagePayload(2, llm.Message{
		Role:             llm.RoleAssistant,
		Text:             "done",
		ReasoningContent: json.RawMessage(`["step 1", {"summary_text":"step 2"}]`),
	})

	if payload.Thinking != "step 1\nstep 2" {
		t.Fatalf("unexpected thinking: %q", payload.Thinking)
	}
}

func TestBuildSessionMessagePayloadProjectsToolCallAssistantThinking(t *testing.T) {
	payload := buildSessionMessagePayload(4, llm.Message{
		Role:             llm.RoleAssistant,
		ReasoningContent: json.RawMessage(`{"summary_text":"before tool"}`),
		ToolCalls: []llm.ToolCall{{
			ID:        "call-1",
			Name:      "read_file",
			Arguments: json.RawMessage(`{"path":"README.md"}`),
		}},
	})

	if payload.Thinking != "before tool" {
		t.Fatalf("unexpected thinking: %q", payload.Thinking)
	}
	if len(payload.ToolCalls) != 1 {
		t.Fatalf("expected tool calls to stay projected, got %+v", payload.ToolCalls)
	}
}

func TestBuildSessionDetailPayloadIncludesTurnDraftForLatestWindow(t *testing.T) {
	sess := session.NewSession("system prompt")
	sess.ID = "session-draft-detail"
	sess.TurnDraft = &session.TurnDraft{
		TraceID: "trace-draft",
		Turn:    1,
		AssistantSegments: []session.TurnDraftSegment{
			{ID: "stream-segment:assistant:1", Content: "partial answer"},
		},
		ThinkingSegments: []session.TurnDraftSegment{
			{ID: "stream-segment:thinking:1", Content: "analyzing"},
		},
		Tools: []session.TurnDraftTool{
			{ID: "stream-tool:trace-draft:call-1", Content: `{"path":"README.md"}`, ToolName: "read_file"},
		},
		ItemOrder: []string{
			"thinking:stream-segment:thinking:1",
			"tool:stream-tool:trace-draft:call-1",
			"assistant:stream-segment:assistant:1",
		},
	}

	page := session.MessagePage{
		Limit: 100,
		Messages: []session.IndexedMessage{
			{
				Index:   0,
				Message: sess.Messages[0],
			},
		},
	}
	payload := buildSessionDetailPayload(sess, page, true)
	if len(payload.Messages) != 1 {
		t.Fatalf("expected messages to stay committed-only, got %d messages", len(payload.Messages))
	}
	if payload.TurnDraft == nil {
		t.Fatal("expected latest window to include turn_draft")
	}
	if payload.TurnDraft.TraceID != "trace-draft" || payload.TurnDraft.Turn != 1 {
		t.Fatalf("unexpected turn_draft header: %+v", payload.TurnDraft)
	}
	if len(payload.TurnDraft.AssistantSegments) != 1 || payload.TurnDraft.AssistantSegments[0].Content != "partial answer" {
		t.Fatalf("unexpected assistant segments: %+v", payload.TurnDraft.AssistantSegments)
	}
	if len(payload.TurnDraft.ThinkingSegments) != 1 || payload.TurnDraft.ThinkingSegments[0].Content != "analyzing" {
		t.Fatalf("unexpected thinking segments: %+v", payload.TurnDraft.ThinkingSegments)
	}
}

func TestBuildSessionDetailPayloadSkipsTurnDraftForOlderWindow(t *testing.T) {
	sess := session.NewSession("system prompt")
	sess.ID = "session-draft-older-window"
	sess.TurnDraft = &session.TurnDraft{
		TraceID: "trace-draft",
		Turn:    1,
	}
	page := session.MessagePage{
		Limit: 100,
		Messages: []session.IndexedMessage{
			{
				Index:   0,
				Message: sess.Messages[0],
			},
		},
	}

	payload := buildSessionDetailPayload(sess, page, false)
	if len(payload.Messages) != 1 {
		t.Fatalf("expected old page to skip draft, got %d messages", len(payload.Messages))
	}
	if payload.TurnDraft != nil {
		t.Fatalf("expected old page to skip turn_draft, got %+v", payload.TurnDraft)
	}
}
