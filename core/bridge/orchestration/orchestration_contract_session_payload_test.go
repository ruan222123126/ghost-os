package orchestration

import (
	"encoding/json"
	"ghost-os/bridge/agent"
	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/llm"
	"ghost-os/bridge/session"
	"strings"
	"testing"
)

func TestConfigResponseFromSnapshotIncludesRuntimeFlags(t *testing.T) {
	response := configResponseFromSnapshot(bridgeconfig.Snapshot{
		MaxTurns:                     9,
		TaskExecutionTimeoutMS:       600000,
		LLMCompletionRetryCount:      0,
		LLMCompletionRetryIntervalMS: 150,
		SessionHumanLogFullEnabled:   true,
		SessionSystemPromptVisible:   false,
		AssistantMarkdownEnabled:     false,
		ToolCallCompactOutputEnabled: true,
		MemoryModeEnabled:            true,
		MicrocompactEnabled:          true,
		SessionTitleMode:             bridgeconfig.SessionTitleModeFirstMessage,
		WebSearchTavilyURL:           "https://proxy.example/tavily",
		WebSearchExaURL:              "https://proxy.example/exa",
		WebSearchTavilyAPIKeySet:     true,
		WebSearchExaAPIKeySet:        true,
		ProjectRoot:                  "/tmp/ghost-os",
	})

	if !response.SessionHumanLogFullEnabled {
		t.Fatal("expected session_human_log_full_enabled to be true")
	}
	if response.MaxTurns != 9 {
		t.Fatalf("unexpected max_turns: got %d want %d", response.MaxTurns, 9)
	}
	if response.TaskExecutionTimeoutMs != 600000 {
		t.Fatalf("unexpected task_execution_timeout_ms: got %d want %d", response.TaskExecutionTimeoutMs, 600000)
	}
	if response.LlmCompletionRetryCount != 0 {
		t.Fatalf("unexpected llm_completion_retry_count: got %d want %d", response.LlmCompletionRetryCount, 0)
	}
	if response.LlmCompletionRetryIntervalMs != 150 {
		t.Fatalf(
			"unexpected llm_completion_retry_interval_ms: got %d want %d",
			response.LlmCompletionRetryIntervalMs,
			150,
		)
	}
	if response.SessionSystemPromptVisibleEnabled {
		t.Fatal("expected session_system_prompt_visible_enabled to be false")
	}
	if response.AssistantMarkdownEnabled {
		t.Fatal("expected assistant_markdown_enabled to be false")
	}
	if !response.ToolCallCompactOutputEnabled {
		t.Fatal("expected tool_call_compact_output_enabled to be true")
	}
	if !response.MemoryModeEnabled {
		t.Fatal("expected memory_mode_enabled to be true")
	}
	if !response.MicrocompactEnabled {
		t.Fatal("expected microcompact_enabled to be true")
	}
	if response.SessionTitleMode != bridgeconfig.SessionTitleModeFirstMessage {
		t.Fatalf("unexpected session_title_mode: got %q", response.SessionTitleMode)
	}
	if response.WebSearchTavilyURL != "https://proxy.example/tavily" {
		t.Fatalf("unexpected web_search_tavily_url: got %q want %q", response.WebSearchTavilyURL, "https://proxy.example/tavily")
	}
	if response.WebSearchExaURL != "https://proxy.example/exa" {
		t.Fatalf("unexpected web_search_exa_url: got %q want %q", response.WebSearchExaURL, "https://proxy.example/exa")
	}
	if !response.WebSearchTavilyAPIKeySet {
		t.Fatal("expected web_search_tavily_api_key_set to be true")
	}
	if !response.WebSearchExaAPIKeySet {
		t.Fatal("expected web_search_exa_api_key_set to be true")
	}
	if response.ProjectRoot != "/tmp/ghost-os" {
		t.Fatalf("unexpected project_root: got %q want %q", response.ProjectRoot, "/tmp/ghost-os")
	}
}

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
		Status:  session.TurnDraftStatusAwaitingHuman,
		PendingQuestions: []session.TurnDraftPendingQuestion{
			{
				QuestionID:    "q-1",
				Prompt:        "Ship it?",
				SelectionMode: session.HumanQuestionSelectionSingle,
				Options: []session.HumanQuestionOption{
					{Label: "Yes"},
				},
			},
		},
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
			"question:q-1",
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
	if payload.TurnDraft.Status != session.TurnDraftStatusAwaitingHuman {
		t.Fatalf("unexpected turn_draft status: %+v", payload.TurnDraft)
	}
	if len(payload.TurnDraft.PendingQuestions) != 1 || payload.TurnDraft.PendingQuestions[0].QuestionID != "q-1" {
		t.Fatalf("unexpected pending questions: %+v", payload.TurnDraft.PendingQuestions)
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
		Status:  session.TurnDraftStatusStreaming,
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

func TestBuildSessionDetailPayloadNormalizesEmptyTurnDraftItemOrder(t *testing.T) {
	sess := session.NewSession("system prompt")
	sess.ID = "session-draft-error"
	sess.TurnDraft = &session.TurnDraft{
		TraceID: "trace-draft-error",
		Turn:    0,
		Status:  session.TurnDraftStatusError,
		Error:   "stream interrupted: context canceled",
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
	if payload.TurnDraft == nil {
		t.Fatal("expected latest window to include turn_draft")
	}
	if payload.TurnDraft.ItemOrder == nil {
		t.Fatal("expected empty item_order slice, got nil")
	}
	if len(payload.TurnDraft.ItemOrder) != 0 {
		t.Fatalf("expected empty item_order slice, got %+v", payload.TurnDraft.ItemOrder)
	}
}

func TestParseSessionEndSignalPassThroughPlainText(t *testing.T) {
	message, signal, err := parseSessionEndSignal("normal answer")
	if err != nil {
		t.Fatalf("parseSessionEndSignal returned error: %v", err)
	}
	if message != "normal answer" {
		t.Fatalf("unexpected message: got %q want %q", message, "normal answer")
	}
	if signal != nil {
		t.Fatalf("signal should be nil for plain text: %+v", signal)
	}
}

func TestParseSessionEndSignalStructured(t *testing.T) {
	message, signal, err := parseSessionEndSignal(`{"signal":"END_SESSION","message":"bye"}`)
	if err != nil {
		t.Fatalf("parseSessionEndSignal returned error: %v", err)
	}
	if message != "bye" {
		t.Fatalf("unexpected message: got %q want %q", message, "bye")
	}
	if signal == nil {
		t.Fatal("signal should not be nil")
	}
	if signal.Signal != busAssistantSessionEndSignal {
		t.Fatalf("unexpected signal: got %q want %q", signal.Signal, busAssistantSessionEndSignal)
	}
}

func TestParseSessionEndSignalRejectsInvalidEndPayload(t *testing.T) {
	_, _, err := parseSessionEndSignal(`{"signal":"END_SESSION","message":"","extra":"x"}`)
	if err == nil {
		t.Fatal("expected error but got nil")
	}
}
