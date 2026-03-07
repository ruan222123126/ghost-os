package memory

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"ghost-os/bridge/agent"
	"ghost-os/bridge/llm"
)

type decisionTestExtractor struct {
	memo DecisionMemo
	err  error
}

func (d decisionTestExtractor) Summarize(_ []llm.Message) (string, error) {
	return "", nil
}

func (d decisionTestExtractor) ExtractDecisionMemo(_ DecisionCaptureInput) (DecisionMemo, error) {
	if d.err != nil {
		return DecisionMemo{}, d.err
	}
	return d.memo, nil
}

func TestDecisionServiceCaptureTurn_SuccessMemo(t *testing.T) {
	service, _ := newDecisionCaptureTestService(t, nil)
	if err := service.CaptureTurn(decisionCaptureTestInput(DecisionOutcomeSuccess)); err != nil {
		t.Fatalf("CaptureTurn returned error: %v", err)
	}
	memos := service.store.ListMemos("workspace:test")
	if len(memos) != 1 {
		t.Fatalf("expected 1 memo, got %d", len(memos))
	}
	memo := memos[0]
	if memo.Outcome != DecisionOutcomeSuccess {
		t.Fatalf("unexpected outcome: %q", memo.Outcome)
	}
	if len(memo.ToolsUsed) != 1 || memo.ToolsUsed[0].Name != "read_file" {
		t.Fatalf("unexpected tools used: %#v", memo.ToolsUsed)
	}
	if len(memo.QuestionsAsked) != 1 || memo.QuestionsAsked[0].Answer != "yes" {
		t.Fatalf("unexpected questions asked: %#v", memo.QuestionsAsked)
	}
	if memo.Environment.Domain != "coding" {
		t.Fatalf("unexpected domain: %q", memo.Environment.Domain)
	}
	if memo.HumanBlocked {
		t.Fatalf("did not expect human_blocked for success memo")
	}
}

func TestDecisionServiceCaptureTurn_AwaitingHumanMemo(t *testing.T) {
	service, _ := newDecisionCaptureTestService(t, nil)
	input := decisionCaptureTestInput(DecisionOutcomeAwaitingHuman)
	input.NewMessages = []llm.Message{
		{Role: llm.RoleUser, Text: "ship it?"},
		{Role: llm.RoleAssistant, ToolCalls: []llm.ToolCall{{ID: "call-ask", Name: "ask_human", Arguments: json.RawMessage(`{"prompt":"Ship now?"}`)}}},
	}
	if err := service.CaptureTurn(input); err != nil {
		t.Fatalf("CaptureTurn returned error: %v", err)
	}
	memo := service.store.ListMemos("workspace:test")[0]
	if memo.Outcome != DecisionOutcomeAwaitingHuman {
		t.Fatalf("unexpected outcome: %q", memo.Outcome)
	}
	if !memo.HumanBlocked {
		t.Fatalf("expected human_blocked=true")
	}
	if len(memo.QuestionsAsked) == 0 || memo.QuestionsAsked[0].Question != "Ship now?" {
		t.Fatalf("expected ask_human prompt to persist, got %#v", memo.QuestionsAsked)
	}
	if len(memo.NeedsHumanFor) == 0 || memo.NeedsHumanFor[0] != "Ship now?" {
		t.Fatalf("expected needs_human_for to include prompt, got %#v", memo.NeedsHumanFor)
	}
}

func TestDecisionServiceCaptureTurn_PartialMemoWhenToolErrorsExist(t *testing.T) {
	service, _ := newDecisionCaptureTestService(t, nil)
	input := decisionCaptureTestInput(DecisionOutcomeSuccess)
	input.NewMessages = []llm.Message{
		{Role: llm.RoleUser, Text: "run tests"},
		{Role: llm.RoleAssistant, ToolCalls: []llm.ToolCall{{ID: "call-bash", Name: "bash_exec", Arguments: json.RawMessage(`{"command":"go test ./..."}`)}}},
		{Role: llm.RoleTool, ToolCallID: "call-bash", Text: agent.FormatToolResult("bash_exec", "trace-test", "", errors.New("tests failed"))},
		{Role: llm.RoleAssistant, Text: "I stopped after the failing test run."},
	}
	if err := service.CaptureTurn(input); err != nil {
		t.Fatalf("CaptureTurn returned error: %v", err)
	}
	memo := service.store.ListMemos("workspace:test")[0]
	if memo.Outcome != DecisionOutcomePartial {
		t.Fatalf("unexpected outcome: %q", memo.Outcome)
	}
	if len(memo.FailureReasons) == 0 {
		t.Fatalf("expected failure reasons to be captured")
	}
}

func TestDecisionServiceCaptureTurn_MergeWorkerDraftKeepsRuleFields(t *testing.T) {
	extractor := decisionTestExtractor{memo: DecisionMemo{
		Outcome:         DecisionOutcomeSuccess,
		ToolsUsed:       []DecisionToolUse{{Name: "script_exec"}},
		QuestionsAsked:  []DecisionQuestion{{Question: "wrong question"}},
		HumanBlocked:    false,
		IntentKey:       "intent.fix_config",
		StrategySummary: "Patch the file, then verify.",
		Confidence:      0.91,
		ReuseScore:      0.88,
	}}
	service, _ := newDecisionCaptureTestService(t, extractor)
	input := decisionCaptureTestInput(DecisionOutcomeAwaitingHuman)
	input.NewMessages = []llm.Message{
		{Role: llm.RoleAssistant, ToolCalls: []llm.ToolCall{{ID: "call-ask", Name: "ask_human", Arguments: json.RawMessage(`{"prompt":"Ship now?"}`)}}},
	}
	if err := service.CaptureTurn(input); err != nil {
		t.Fatalf("CaptureTurn returned error: %v", err)
	}
	memo := service.store.ListMemos("workspace:test")[0]
	if memo.Outcome != DecisionOutcomeAwaitingHuman {
		t.Fatalf("worker should not override outcome: %q", memo.Outcome)
	}
	if len(memo.ToolsUsed) != 1 || memo.ToolsUsed[0].Name != "ask_human" {
		t.Fatalf("worker should not override tools_used: %#v", memo.ToolsUsed)
	}
	if len(memo.QuestionsAsked) == 0 || memo.QuestionsAsked[0].Question != "Ship now?" {
		t.Fatalf("worker should not override questions_asked: %#v", memo.QuestionsAsked)
	}
	if !memo.HumanBlocked {
		t.Fatalf("worker should not override human_blocked")
	}
	if memo.IntentKey != "intent.fix_config" || memo.StrategySummary == "" {
		t.Fatalf("expected worker summary fields to merge, got %+v", memo)
	}
	if memo.Confidence != 0.91 || memo.ReuseScore != 0.88 {
		t.Fatalf("expected worker scores to win, got confidence=%v reuse=%v", memo.Confidence, memo.ReuseScore)
	}
}

func TestDecisionServiceCaptureTurn_FallsBackToRuleOnlyWhenWorkerFails(t *testing.T) {
	service, _ := newDecisionCaptureTestService(t, decisionTestExtractor{err: errors.New("worker unavailable")})
	if err := service.CaptureTurn(decisionCaptureTestInput(DecisionOutcomeSuccess)); err != nil {
		t.Fatalf("CaptureTurn returned error: %v", err)
	}
	memo := service.store.ListMemos("workspace:test")[0]
	if memo.Outcome != DecisionOutcomeSuccess {
		t.Fatalf("unexpected outcome: %q", memo.Outcome)
	}
	if memo.IntentSummary != "" || memo.StrategySummary != "" {
		t.Fatalf("expected rule-only fallback without worker summaries, got %+v", memo)
	}
}

func TestDecisionServiceCaptureTurn_TruncatesVerboseToolOutput(t *testing.T) {
	service, _ := newDecisionCaptureTestService(t, nil)
	verboseOutput := strings.Repeat("build output ", 40) + "TAIL_SECRET_MARKER"
	input := decisionCaptureTestInput(DecisionOutcomeSuccess)
	input.NewMessages = []llm.Message{
		{Role: llm.RoleAssistant, ToolCalls: []llm.ToolCall{{ID: "call-bash", Name: "bash_exec", Arguments: json.RawMessage(`{"command":"go test ./..."}`)}}},
		{Role: llm.RoleTool, ToolCallID: "call-bash", Text: agent.FormatToolResult("bash_exec", "trace-truncate", verboseOutput, nil)},
	}
	if err := service.CaptureTurn(input); err != nil {
		t.Fatalf("CaptureTurn returned error: %v", err)
	}
	memo := service.store.ListMemos("workspace:test")[0]
	if len(memo.ToolsUsed) != 1 {
		t.Fatalf("expected one tool use, got %#v", memo.ToolsUsed)
	}
	if got := memo.ToolsUsed[0].OutputSummary; len(got) > 220 || strings.Contains(got, "TAIL_SECRET_MARKER") {
		t.Fatalf("expected output summary to be truncated, got %q", got)
	}
}

func TestDecisionServiceCaptureTurn_DoesNotPersistRawMessagesOrCoT(t *testing.T) {
	service, dir := newDecisionCaptureTestService(t, nil)
	input := decisionCaptureTestInput(DecisionOutcomeSuccess)
	input.RecentHistory = []llm.Message{{Role: llm.RoleAssistant, Text: "SECRET_COT_TOKEN should never be persisted"}}
	if err := service.CaptureTurn(input); err != nil {
		t.Fatalf("CaptureTurn returned error: %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(dir, defaultDecisionMemosPathName))
	if err != nil {
		t.Fatalf("read memos file: %v", err)
	}
	text := string(raw)
	if strings.Contains(text, "recent_history") || strings.Contains(text, "new_messages") {
		t.Fatalf("raw history should not be persisted: %s", text)
	}
	if strings.Contains(text, "SECRET_COT_TOKEN") {
		t.Fatalf("recent history content leaked into memo: %s", text)
	}
}

func newDecisionCaptureTestService(t *testing.T, extractor any) (*DecisionService, string) {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "decision")
	config := MemoryConfig{
		DecisionEnabled:          true,
		DecisionCaptureOnTurn:    true,
		DecisionCaptureOnTurnSet: true,
		DecisionPath:             dir,
	}
	if summarizer, ok := extractor.(Summarizer); ok {
		config.Summarizer = summarizer
	}
	service := NewDecisionService(config, nil)
	if !service.Enabled() {
		t.Fatalf("expected decision service to be enabled")
	}
	return service, dir
}

func decisionCaptureTestInput(outcome string) DecisionCaptureInput {
	startedAt := time.Date(2026, 3, 7, 10, 0, 0, 0, time.UTC)
	finishedAt := startedAt.Add(2 * time.Minute)
	return DecisionCaptureInput{
		Namespace:   "workspace:test",
		SessionID:   "session-capture-1",
		TraceID:     "trace-capture-1",
		TurnID:      "turn-1",
		UserMessage: "inspect config.toml",
		RecentHistory: []llm.Message{
			{Role: llm.RoleUser, Text: "check config"},
			{Role: llm.RoleAssistant, Text: "I'll inspect the config."},
		},
		NewMessages: []llm.Message{
			{Role: llm.RoleUser, Text: "inspect config.toml"},
			{Role: llm.RoleAssistant, ToolCalls: []llm.ToolCall{{ID: "call-read", Name: "read_file", Arguments: json.RawMessage(`{"path":"config.toml"}`)}}},
			{Role: llm.RoleTool, ToolCallID: "call-read", Text: agent.FormatToolResult("read_file", "trace-capture-1", "loaded config.toml", nil)},
			{Role: llm.RoleAssistant, Text: "Configuration looks healthy."},
		},
		Outcome: outcome,
		AnsweredQuestions: []DecisionAnsweredQuestion{{
			QuestionID: "q-1",
			Prompt:     "Proceed with config migration?",
			Answer:     "yes",
			ToolCallID: "call-prev",
			TraceID:    "trace-prev",
			AskedAt:    startedAt.Add(-5 * time.Minute),
			AnsweredAt: startedAt.Add(-4 * time.Minute),
		}},
		TurnStartedAt:  startedAt,
		TurnFinishedAt: finishedAt,
		Environment: DecisionEnvFingerprint{
			OS:               "linux",
			Platform:         "linux/amd64",
			Shell:            "/bin/bash",
			WorkspaceRoot:    "/workspace/ghost-os",
			Provider:         "openai",
			Model:            "gpt-4o",
			GraphNamespace:   "workspace:test",
			Domain:           "coding",
			ToolsetSignature: "ask_human,read_file",
			PathHints:        []string{"config.toml", "core/bridge/app/session_agent_runner.go:96"},
			TargetAppOrSite:  "config.toml",
			ToolNames:        []string{"ask_human", "read_file"},
		},
	}
}
