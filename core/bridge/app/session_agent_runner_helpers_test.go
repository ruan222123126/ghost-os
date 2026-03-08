package app

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"ghost-os/bridge/agent"
	"ghost-os/bridge/llm"
	"ghost-os/bridge/memory"
	"ghost-os/bridge/session"
	"ghost-os/bridge/tools"
)

type fakeSelectorEngine struct {
	result           ToolSelectorResult
	calls            int
	lastDecisionHint string
}

func (f *fakeSelectorEngine) SelectTools(_ context.Context, _ string, _ []llm.Message, decisionHint string, _ string) ToolSelectorResult {
	f.calls++
	f.lastDecisionHint = strings.TrimSpace(decisionHint)
	return f.result
}

type runnerDecisionExtractor struct {
	memo memory.DecisionMemo
	err  error
}

func (e runnerDecisionExtractor) Summarize([]llm.Message) (string, error) {
	return "", e.err
}

func (e runnerDecisionExtractor) ExtractDecisionMemo(memory.DecisionCaptureInput) (memory.DecisionMemo, error) {
	if e.err != nil {
		return memory.DecisionMemo{}, e.err
	}
	return e.memo, nil
}

type runnerMockTool struct {
	name string
}

func (m *runnerMockTool) Name() string { return m.name }

func (m *runnerMockTool) Description() string { return "runner mock" }

func (m *runnerMockTool) Parameters() json.RawMessage { return json.RawMessage(`{"type":"object"}`) }

func (m *runnerMockTool) Execute(context.Context, json.RawMessage, string) (string, error) {
	return "", nil
}

func newRunnerTestRegistry() *tools.Registry {
	registry := tools.NewRegistry()
	for _, name := range []string{"list_files", "read_file", "search_files", "bash_exec", "script_exec", "ask_human"} {
		registry.Register(&runnerMockTool{name: name})
	}
	return registry
}

func newRunnerTestDeps(cfg Config) agentRuntimeDependencies {
	registry := newRunnerTestRegistry()
	return agentRuntimeDependencies{
		cfg:      cfg,
		registry: registry,
	}
}

func runnerSelectorEnv() memory.DecisionEnvFingerprint {
	return memory.DecisionEnvFingerprint{
		WorkspaceRoot:    "/workspace/ghost-os",
		Platform:         "linux/amd64",
		GraphNamespace:   "workspace:test",
		Domain:           "coding",
		ToolNames:        []string{"ask_human", "read_file", "search_files", "apply_diff", "bash_exec"},
		ToolsetSignature: "apply_diff,ask_human,bash_exec,read_file,search_files",
	}
}

func newRunnerDecisionHintManager(t *testing.T) *memory.MemoryManager {
	t.Helper()
	baseDir := t.TempDir()
	manager := memory.NewMemoryManager(memory.MemoryConfig{
		WarmCapacity:             32,
		WarmPath:                 filepath.Join(baseDir, "warm.json"),
		ColdBaseDir:              filepath.Join(baseDir, "cold"),
		DecisionEnabled:          true,
		DecisionCaptureOnTurn:    true,
		DecisionCaptureOnTurnSet: true,
		DecisionPath:             filepath.Join(baseDir, "decision"),
		Summarizer: runnerDecisionExtractor{memo: memory.DecisionMemo{
			IntentSummary:   "fix config migration",
			StrategySummary: "inspect target file before patching",
			Confidence:      0.93,
			ReuseScore:      0.91,
		}},
	})
	t.Cleanup(manager.StopDreaming)
	return manager
}

func captureRunnerDecisionHint(t *testing.T, manager *memory.MemoryManager, env memory.DecisionEnvFingerprint, outcome string, askHumanPrompt string) {
	t.Helper()
	now := time.Now().UTC()
	assistant := llm.Message{
		Role: llm.RoleAssistant,
		Text: "Inspecting config before patching.",
		ToolCalls: []llm.ToolCall{
			{ID: "call-read", Name: "read_file", Arguments: json.RawMessage(`{"path":"config.toml"}`)},
			{ID: "call-search", Name: "search_files", Arguments: json.RawMessage(`{"pattern":"migration"}`)},
			{ID: "call-patch", Name: "apply_diff", Arguments: json.RawMessage(`{"path":"config.toml","diff":"@@"}`)},
			{ID: "call-bash", Name: "bash_exec", Arguments: json.RawMessage(`{"command":"go test ./core/bridge/app"}`)},
		},
	}
	if strings.TrimSpace(askHumanPrompt) != "" {
		assistant.ToolCalls = append(assistant.ToolCalls, llm.ToolCall{ID: "call-human", Name: "ask_human", Arguments: json.RawMessage(`{"prompt":"` + askHumanPrompt + `"}`)})
	}
	messages := []llm.Message{
		assistant,
		{Role: llm.RoleTool, ToolCallID: "call-read", Text: agent.FormatToolResult("read_file", "trace-selector-hint", "file content", nil)},
		{Role: llm.RoleTool, ToolCallID: "call-search", Text: agent.FormatToolResult("search_files", "trace-selector-hint", "pattern match", nil)},
		{Role: llm.RoleTool, ToolCallID: "call-patch", Text: agent.FormatToolResult("apply_diff", "trace-selector-hint", "patched config", nil)},
		{Role: llm.RoleTool, ToolCallID: "call-bash", Text: agent.FormatToolResult("bash_exec", "trace-selector-hint", "tests passed", nil)},
	}
	if strings.TrimSpace(askHumanPrompt) != "" {
		messages = append(messages, llm.Message{Role: llm.RoleTool, ToolCallID: "call-human", Text: agent.FormatToolResult("ask_human", "trace-selector-hint", "", nil)})
	}
	if err := manager.CaptureDecisionTurn(memory.DecisionCaptureInput{
		SessionID:      "runner-selector-session",
		TraceID:        "trace-selector-hint",
		TurnID:         "turn-1",
		Namespace:      env.GraphNamespace,
		UserMessage:    "fix config migration",
		Outcome:        outcome,
		Environment:    env,
		TurnStartedAt:  now.Add(-time.Minute),
		TurnFinishedAt: now,
		NewMessages:    messages,
	}); err != nil {
		t.Fatalf("capture decision hint: %v", err)
	}
}

func newDecisionCaptureState(t *testing.T, manager *memory.MemoryManager, sessionID string, traceID string) (*sessionTurnState, *agent.History, *session.Store, *session.Session) {
	t.Helper()
	store, err := session.NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	sess := session.NewSession("system")
	sess.ID = sessionID
	history := agent.NewHistory("system")
	turnAgent := agent.NewAgentWithHistory(nil, nil, history, 2)
	state := &sessionTurnState{
		sessionStore: store,
		persistence:  newSessionTurnCommitter(store, manager, traceID),
		sess:         sess,
		agent:        turnAgent,
		traceID:      traceID,
		userMessage:  "update core/bridge/app/session_agent_runner.go:96",
		preTurnMessages: []llm.Message{
			{Role: llm.RoleSystem, Text: "system"},
			{Role: llm.RoleUser, Text: "Please inspect core/bridge/app/session_agent_runner.go:96 before editing."},
		},
		answeredQuestions: []memory.DecisionAnsweredQuestion{{
			QuestionID: "q-prev-1",
			Prompt:     "Confirm deployment target?",
			Answer:     "yes",
			AskedAt:    time.Date(2026, 3, 7, 9, 0, 0, 0, time.UTC),
			AnsweredAt: time.Date(2026, 3, 7, 9, 5, 0, 0, time.UTC),
			ToolCallID: "call-prev",
			TraceID:    traceID,
		}},
		turnStartedAt: time.Date(2026, 3, 7, 10, 0, 0, 0, time.UTC),
		environment: memory.DecisionEnvFingerprint{
			OS:               "linux",
			Platform:         "linux/amd64",
			WorkspaceRoot:    "/workspace/ghost-os",
			Provider:         "openai",
			Model:            "gpt-4o",
			GraphNamespace:   "workspace:test",
			Domain:           "coding",
			ToolNames:        []string{"ask_human", "read_file"},
			ToolsetSignature: "ask_human,read_file",
			PathHints:        []string{"core/bridge/app/session_agent_runner.go:96"},
			TargetAppOrSite:  "core/bridge/app/session_agent_runner.go:96",
		},
	}
	return state, history, store, sess
}

func readDecisionMemosForRunnerTest(t *testing.T, decisionDir string) []memory.DecisionMemo {
	t.Helper()
	path := filepath.Join(decisionDir, "memos.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read memos file: %v", err)
	}
	var payload struct {
		Memos []memory.DecisionMemo `json:"memos"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		t.Fatalf("decode memos file: %v", err)
	}
	return payload.Memos
}

func runnerHasQuestion(questions []memory.DecisionQuestion, expected string) bool {
	for _, question := range questions {
		if question.Question == expected {
			return true
		}
	}
	return false
}
