package app

import (
	"context"
	"encoding/json"
	"errors"
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

func TestSessionTurnPreparer_SelectToolsForTurn_BypassesAskHumanContinuation(t *testing.T) {
	selector := &fakeSelectorEngine{result: ToolSelectorResult{Mode: "subset", Tools: []string{"read_file", "ask_human"}}}
	preparer := &sessionTurnPreparer{selectorFactory: func(Config) selectorEngine { return selector }}
	deps := newRunnerTestDeps(Config{ToolSelectorEnabled: true, ToolSelectorMode: "llm", MaxTurns: 6})
	history := agent.NewHistory("system prompt")

	catalog, prompt := preparer.selectToolsForTurn(context.Background(), deps, "runner-ask", history, "resume", true, "trace-ask", runnerSelectorEnv())
	if selector.calls != 0 {
		t.Fatalf("selector should not run for ask_human continuation, got %d calls", selector.calls)
	}
	if len(catalog.ToolDefs()) != len(deps.registry.ToolDefs()) {
		t.Fatal("expected full registry for ask_human continuation")
	}
	if prompt != "" {
		t.Fatalf("expected empty prompt override, got %q", prompt)
	}
}

func TestSessionTurnPreparer_SelectToolsForTurn_UsesFullRegistryWhenDisabled(t *testing.T) {
	selector := &fakeSelectorEngine{result: ToolSelectorResult{Mode: "subset", Tools: []string{"read_file", "ask_human"}}}
	preparer := &sessionTurnPreparer{selectorFactory: func(Config) selectorEngine { return selector }}
	deps := newRunnerTestDeps(Config{ToolSelectorEnabled: false, MaxTurns: 6})
	history := agent.NewHistory("system prompt")

	catalog, prompt := preparer.selectToolsForTurn(context.Background(), deps, "runner-disabled", history, "read config", false, "trace-disabled", runnerSelectorEnv())
	if selector.calls != 0 {
		t.Fatalf("selector should not run when disabled, got %d calls", selector.calls)
	}
	if len(catalog.ToolDefs()) != len(deps.registry.ToolDefs()) {
		t.Fatal("expected full registry when selector disabled")
	}
	if prompt != "" {
		t.Fatalf("expected empty prompt override, got %q", prompt)
	}
}

func TestSessionTurnPreparer_SelectToolsForTurn_ShadowModeKeepsFullRegistry(t *testing.T) {
	selector := &fakeSelectorEngine{result: ToolSelectorResult{Mode: "subset", Tools: []string{"read_file", "ask_human"}, Confidence: 0.92}}
	preparer := &sessionTurnPreparer{selectorFactory: func(Config) selectorEngine { return selector }}
	deps := newRunnerTestDeps(Config{ToolSelectorEnabled: true, ToolSelectorShadow: true, MaxTurns: 6})
	history := agent.NewHistory("system prompt")

	catalog, prompt := preparer.selectToolsForTurn(context.Background(), deps, "runner-shadow", history, "read config", false, "trace-shadow", runnerSelectorEnv())
	if selector.calls != 1 {
		t.Fatalf("expected selector to run in shadow mode, got %d calls", selector.calls)
	}
	if len(catalog.ToolDefs()) != len(deps.registry.ToolDefs()) {
		t.Fatal("expected full registry in shadow mode")
	}
	if prompt != "" {
		t.Fatalf("expected empty prompt override in shadow mode, got %q", prompt)
	}
}

func TestSessionTurnPreparer_SelectToolsForTurn_SubsetUpdatesPromptCount(t *testing.T) {
	selector := &fakeSelectorEngine{result: ToolSelectorResult{Mode: "subset", Tools: []string{"read_file", "ask_human"}, Confidence: 0.96}}
	preparer := &sessionTurnPreparer{selectorFactory: func(Config) selectorEngine { return selector }}
	deps := newRunnerTestDeps(Config{ToolSelectorEnabled: true, ToolSelectorMode: "llm", MaxTurns: 6})
	history := agent.NewHistory("system prompt")
	history.Append(llm.Message{Role: llm.RoleUser, Text: "please read config"})
	history.Append(llm.Message{Role: llm.RoleAssistant, Text: "ok"})

	catalog, prompt := preparer.selectToolsForTurn(context.Background(), deps, "runner-subset", history, "read config.go", false, "trace-subset", runnerSelectorEnv())
	if selector.calls != 1 {
		t.Fatalf("expected selector to run once, got %d calls", selector.calls)
	}
	if len(catalog.ToolDefs()) != 2 {
		t.Fatalf("expected 2 tool defs, got %d", len(catalog.ToolDefs()))
	}
	if catalog.Get("read_file") == nil {
		t.Fatal("expected scoped catalog to expose read_file")
	}
	if catalog.Get("list_files") != nil {
		t.Fatal("expected scoped catalog to hide list_files")
	}
	if !strings.Contains(prompt, "Available tools: 2") {
		t.Fatalf("expected prompt to reflect filtered tool count, got %q", prompt)
	}
}

func TestSessionTurnPreparer_SelectToolsForTurn_PassesDecisionHintToSelector(t *testing.T) {
	env := runnerSelectorEnv()
	manager := newRunnerDecisionHintManager(t)
	captureRunnerDecisionHint(t, manager, env, memory.DecisionOutcomeSuccess, "")
	selector := &fakeSelectorEngine{result: ToolSelectorResult{Mode: "subset", Tools: []string{"read_file", "ask_human"}, Confidence: 0.94}}
	preparer := &sessionTurnPreparer{
		sharedMemoryManager: manager,
		selectorFactory:     func(Config) selectorEngine { return selector },
	}
	deps := newRunnerTestDeps(Config{ToolSelectorEnabled: true, ToolSelectorMode: "llm", MemoryDecisionSelectorHintEnabled: true, MaxTurns: 6})
	history := agent.NewHistory("system prompt")

	preparer.selectToolsForTurn(context.Background(), deps, "runner-selector-hint", history, "fix config migration", false, "trace-selector-hint", env)
	if selector.calls != 1 {
		t.Fatalf("expected selector to run once, got %d", selector.calls)
	}
	if !strings.Contains(selector.lastDecisionHint, "Similar successful cases used:") {
		t.Fatalf("expected selector to receive decision hint, got %q", selector.lastDecisionHint)
	}
}

func TestSessionTurnPreparer_SelectToolsForTurn_NoHitsKeepsExistingBehavior(t *testing.T) {
	selector := &fakeSelectorEngine{result: ToolSelectorResult{Mode: "subset", Tools: []string{"read_file", "ask_human"}, Confidence: 0.96}}
	preparer := &sessionTurnPreparer{
		sharedMemoryManager: newRunnerDecisionHintManager(t),
		selectorFactory:     func(Config) selectorEngine { return selector },
	}
	deps := newRunnerTestDeps(Config{ToolSelectorEnabled: true, ToolSelectorMode: "llm", MemoryDecisionSelectorHintEnabled: true, MaxTurns: 6})
	history := agent.NewHistory("system prompt")

	catalog, prompt := preparer.selectToolsForTurn(context.Background(), deps, "runner-no-hits", history, "brand new task", false, "trace-no-hits", runnerSelectorEnv())
	if selector.lastDecisionHint != "" {
		t.Fatalf("expected empty decision hint when no hits, got %q", selector.lastDecisionHint)
	}
	if len(catalog.ToolDefs()) != 2 || !strings.Contains(prompt, "Available tools: 2") {
		t.Fatalf("expected subset behavior to stay intact, tools=%d prompt=%q", len(catalog.ToolDefs()), prompt)
	}
}

func TestSessionTurnPreparer_SelectToolsForTurn_HintFailureFallsBackToEmptyHint(t *testing.T) {
	selector := &fakeSelectorEngine{result: ToolSelectorResult{Mode: "subset", Tools: []string{"read_file", "ask_human"}, Confidence: 0.9}}
	preparer := &sessionTurnPreparer{
		sharedMemoryManager: newRunnerDecisionHintManager(t),
		selectorFactory:     func(Config) selectorEngine { return selector },
		decisionHintBuilder: func(*memory.MemoryManager, memory.SessionScope, string) (string, []memory.DecisionHit, error) {
			return "", nil, errors.New("boom")
		},
	}
	deps := newRunnerTestDeps(Config{ToolSelectorEnabled: true, ToolSelectorMode: "llm", MemoryDecisionSelectorHintEnabled: true, MaxTurns: 6})

	preparer.selectToolsForTurn(context.Background(), deps, "runner-hint-error", agent.NewHistory("system prompt"), "fix config migration", false, "trace-hint-error", runnerSelectorEnv())
	if selector.lastDecisionHint != "" {
		t.Fatalf("expected empty decision hint on failure, got %q", selector.lastDecisionHint)
	}
}

func TestSessionTurnPreparer_SelectToolsForTurn_ShadowModeStillUsesHintInput(t *testing.T) {
	env := runnerSelectorEnv()
	manager := newRunnerDecisionHintManager(t)
	captureRunnerDecisionHint(t, manager, env, memory.DecisionOutcomeSuccess, "")
	selector := &fakeSelectorEngine{result: ToolSelectorResult{Mode: "subset", Tools: []string{"read_file", "ask_human"}, Confidence: 0.92}}
	preparer := &sessionTurnPreparer{
		sharedMemoryManager: manager,
		selectorFactory:     func(Config) selectorEngine { return selector },
	}
	deps := newRunnerTestDeps(Config{ToolSelectorEnabled: true, ToolSelectorShadow: true, ToolSelectorMode: "llm", MemoryDecisionSelectorHintEnabled: true, MaxTurns: 6})

	catalog, prompt := preparer.selectToolsForTurn(context.Background(), deps, "runner-shadow-hint", agent.NewHistory("system prompt"), "fix config migration", false, "trace-shadow-hint", env)
	if selector.lastDecisionHint == "" {
		t.Fatal("expected shadow mode to still pass decision hint to selector")
	}
	if len(catalog.ToolDefs()) != len(deps.registry.ToolDefs()) || prompt != "" {
		t.Fatalf("expected shadow mode to keep full registry, tools=%d prompt=%q", len(catalog.ToolDefs()), prompt)
	}
}

func TestSessionTurnPreparer_SelectToolsForTurn_AskHumanContinuationSkipsHintLookup(t *testing.T) {
	hintCalls := 0
	preparer := &sessionTurnPreparer{
		sharedMemoryManager: newRunnerDecisionHintManager(t),
		selectorFactory:     func(Config) selectorEngine { return &fakeSelectorEngine{} },
		decisionHintBuilder: func(*memory.MemoryManager, memory.SessionScope, string) (string, []memory.DecisionHit, error) {
			hintCalls++
			return "should not run", nil, nil
		},
	}
	deps := newRunnerTestDeps(Config{ToolSelectorEnabled: true, ToolSelectorMode: "llm", MemoryDecisionSelectorHintEnabled: true, MaxTurns: 6})

	preparer.selectToolsForTurn(context.Background(), deps, "runner-ask-skip", agent.NewHistory("system prompt"), "resume", true, "trace-ask-skip", runnerSelectorEnv())
	if hintCalls != 0 {
		t.Fatalf("expected ask_human continuation to skip hint lookup, got %d calls", hintCalls)
	}
}

func TestSessionTurnStateComplete_PersistsAwaitingHumanMessages(t *testing.T) {
	store, err := session.NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	sess := session.NewSession("system")
	sess.ID = "runner-awaiting-1"
	history := agent.NewHistory("system")
	turnAgent := agent.NewAgentWithHistory(nil, nil, history, 1)
	history.Append(llm.Message{Role: llm.RoleUser, Text: "need approval"})

	state := &sessionTurnState{
		sessionStore: store,
		persistence:  newSessionTurnCommitter(store, nil, "trace-awaiting"),
		sess:         sess,
		agent:        turnAgent,
	}

	response, sessionID, err := state.complete("", &agent.ErrAwaitingHuman{QuestionID: "q-1", Prompt: "Ship now?"}, nil)
	if response != "" {
		t.Fatalf("unexpected response: got %q want empty", response)
	}
	if sessionID != sess.ID {
		t.Fatalf("unexpected session_id: got %q want %q", sessionID, sess.ID)
	}
	var awaitingErr *agent.ErrAwaitingHuman
	if !errors.As(err, &awaitingErr) {
		t.Fatalf("expected awaiting human error, got %v", err)
	}

	loaded, err := store.Load(sess.ID)
	if err != nil {
		t.Fatalf("load persisted session: %v", err)
	}
	if len(loaded.Messages) != 2 {
		t.Fatalf("unexpected persisted message count: got %d want %d", len(loaded.Messages), 2)
	}
	if loaded.Messages[1].Text != "need approval" {
		t.Fatalf("unexpected persisted message: got %q want %q", loaded.Messages[1].Text, "need approval")
	}
}

func TestSessionTurnStateComplete_ReportsPersistFailureViaCallback(t *testing.T) {
	store, err := session.NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("new store: %v", err)
	}

	sess := session.NewSession("system")
	sess.ID = "invalid session id"
	history := agent.NewHistory("system")
	turnAgent := agent.NewAgentWithHistory(nil, nil, history, 1)
	history.Append(llm.Message{Role: llm.RoleAssistant, Text: "done"})

	state := &sessionTurnState{
		sessionStore: store,
		persistence:  newSessionTurnCommitter(store, nil, "trace-persist-error"),
		sess:         sess,
		agent:        turnAgent,
	}

	callbackCalled := false
	callbackAwaiting := true
	_, _, err = state.complete("done", nil, func(err error, awaitingHuman bool) error {
		callbackCalled = true
		callbackAwaiting = awaitingHuman
		return nil
	})
	if !callbackCalled {
		t.Fatal("expected persist callback to run")
	}
	if callbackAwaiting {
		t.Fatal("persist callback should receive awaitingHuman=false for success path")
	}
	if !errors.Is(err, session.ErrInvalidSessionID) {
		t.Fatalf("expected invalid session id error, got %v", err)
	}
}

func TestSessionTurnStateComplete_CapturesDecisionMemoOnSuccess(t *testing.T) {
	decisionDir := filepath.Join(t.TempDir(), "decision")
	manager := memory.NewMemoryManager(memory.MemoryConfig{
		DecisionEnabled:          true,
		DecisionCaptureOnTurn:    true,
		DecisionCaptureOnTurnSet: true,
		DecisionPath:             decisionDir,
	})
	state, history, _, sess := newDecisionCaptureState(t, manager, "runner-success-1", "trace-success")
	history.Append(llm.Message{Role: llm.RoleUser, Text: "update config"})
	history.Append(llm.Message{Role: llm.RoleAssistant, ToolCalls: []llm.ToolCall{{ID: "call-read", Name: "read_file", Arguments: json.RawMessage(`{"path":"config.toml"}`)}}})
	history.Append(llm.Message{Role: llm.RoleTool, ToolCallID: "call-read", Text: agent.FormatToolResult("read_file", "trace-success", "loaded config.toml", nil)})
	history.Append(llm.Message{Role: llm.RoleAssistant, Text: "Updated config and verified the result."})

	response, sessionID, err := state.complete("Updated config and verified the result.", nil, nil)
	if err != nil {
		t.Fatalf("complete returned error: %v", err)
	}
	if response == "" || sessionID != sess.ID {
		t.Fatalf("unexpected response/session: response=%q session=%q", response, sessionID)
	}

	memos := readDecisionMemosForRunnerTest(t, decisionDir)
	if len(memos) != 1 {
		t.Fatalf("expected 1 memo, got %d", len(memos))
	}
	memo := memos[0]
	if memo.Outcome != memory.DecisionOutcomeSuccess {
		t.Fatalf("unexpected outcome: %q", memo.Outcome)
	}
	if len(memo.ToolsUsed) != 1 || memo.ToolsUsed[0].Name != "read_file" {
		t.Fatalf("unexpected tools used: %#v", memo.ToolsUsed)
	}
	if len(memo.QuestionsAsked) != 1 || memo.QuestionsAsked[0].Answer != "yes" {
		t.Fatalf("expected answered question to persist, got %#v", memo.QuestionsAsked)
	}
	if memo.Environment.Domain != "coding" {
		t.Fatalf("unexpected environment domain: %q", memo.Environment.Domain)
	}
	if memo.TurnID == "" {
		t.Fatalf("expected turn id to be populated")
	}
	if memo.OutcomeSummary == "" {
		t.Fatalf("expected outcome summary to be populated")
	}
	if memo.Environment.ToolsetSignature == "" {
		t.Fatalf("expected toolset signature to be populated")
	}
	if memo.CreatedAt.IsZero() {
		t.Fatalf("expected created_at to be populated")
	}
	if len(memo.Environment.ToolNames) == 0 {
		t.Fatalf("expected environment tool names")
	}
	if len(memo.Environment.PathHints) == 0 {
		t.Fatalf("expected path hints to be populated")
	}
	if memo.Namespace != "workspace:test" {
		t.Fatalf("unexpected namespace: %q", memo.Namespace)
	}
	if memo.TraceID != "trace-success" {
		t.Fatalf("unexpected trace id: %q", memo.TraceID)
	}
	if memo.SessionID != sess.ID {
		t.Fatalf("unexpected session id in memo: %q", memo.SessionID)
	}
	if len(memo.QuestionsAsked) != 1 {
		t.Fatalf("unexpected questions: %#v", memo.QuestionsAsked)
	}
	if memo.QuestionsAsked[0].Question != "Confirm deployment target?" {
		t.Fatalf("unexpected persisted answered question: %#v", memo.QuestionsAsked)
	}
	if memo.Environment.TargetAppOrSite != "core/bridge/app/session_agent_runner.go:96" {
		t.Fatalf("unexpected target app/site: %q", memo.Environment.TargetAppOrSite)
	}
	if memo.Environment.WorkspaceRoot == "" {
		t.Fatalf("expected workspace root")
	}
	if memo.Environment.Platform == "" {
		t.Fatalf("expected platform")
	}
	if memo.Environment.Provider == "" {
		t.Fatalf("expected provider")
	}
	if memo.Environment.Model == "" {
		t.Fatalf("expected model")
	}
	if memo.Environment.GraphNamespace != "workspace:test" {
		t.Fatalf("unexpected graph namespace: %q", memo.Environment.GraphNamespace)
	}
	if memo.Environment.OS == "" {
		t.Fatalf("expected os fingerprint")
	}
	if memo.ReuseScore == 0 || memo.Confidence == 0 {
		t.Fatalf("expected rule defaults for confidence/reuse: %+v", memo)
	}
	if memo.HumanBlocked {
		t.Fatalf("did not expect human blocked for success memo")
	}
	if len(memo.NeedsHumanFor) != 0 {
		t.Fatalf("did not expect needs_human_for for success memo: %#v", memo.NeedsHumanFor)
	}
	if memo.ProblemSummary != "" || memo.ContextSummary != "" {
		t.Fatalf("did not expect worker-only summaries by default: %+v", memo)
	}
	if memo.IntentSummary != "" {
		t.Fatalf("did not expect intent summary without worker: %+v", memo)
	}
	if memo.Environment.TargetAppOrSite == "" {
		t.Fatalf("expected target app/site hint")
	}
	t.Cleanup(manager.StopDreaming)
}

func TestSessionTurnStateComplete_CapturesDecisionMemoOnAwaitingHuman(t *testing.T) {
	decisionDir := filepath.Join(t.TempDir(), "decision")
	manager := memory.NewMemoryManager(memory.MemoryConfig{
		DecisionEnabled:          true,
		DecisionCaptureOnTurn:    true,
		DecisionCaptureOnTurnSet: true,
		DecisionPath:             decisionDir,
	})
	state, history, _, _ := newDecisionCaptureState(t, manager, "runner-awaiting-capture", "trace-awaiting-capture")
	history.Append(llm.Message{Role: llm.RoleUser, Text: "ship it?"})
	history.Append(llm.Message{Role: llm.RoleAssistant, ToolCalls: []llm.ToolCall{{ID: "call-ask", Name: "ask_human", Arguments: json.RawMessage(`{"prompt":"Ship now?"}`)}}})

	_, _, err := state.complete("", &agent.ErrAwaitingHuman{QuestionID: "q-1", Prompt: "Ship now?"}, nil)
	if err == nil {
		t.Fatalf("expected awaiting human error")
	}
	memos := readDecisionMemosForRunnerTest(t, decisionDir)
	if len(memos) != 1 {
		t.Fatalf("expected 1 memo, got %d", len(memos))
	}
	memo := memos[0]
	if memo.Outcome != memory.DecisionOutcomeAwaitingHuman {
		t.Fatalf("unexpected outcome: %q", memo.Outcome)
	}
	if !memo.HumanBlocked {
		t.Fatalf("expected human_blocked=true")
	}
	if !runnerHasQuestion(memo.QuestionsAsked, "Ship now?") {
		t.Fatalf("expected ask_human question in memo, got %#v", memo.QuestionsAsked)
	}
	if len(memo.NeedsHumanFor) == 0 || memo.NeedsHumanFor[0] != "Ship now?" {
		t.Fatalf("expected needs_human_for to include prompt, got %#v", memo.NeedsHumanFor)
	}
	t.Cleanup(manager.StopDreaming)
}

func TestSessionTurnStateComplete_DoesNotCaptureWhenPersistFails(t *testing.T) {
	decisionDir := filepath.Join(t.TempDir(), "decision")
	manager := memory.NewMemoryManager(memory.MemoryConfig{
		DecisionEnabled:          true,
		DecisionCaptureOnTurn:    true,
		DecisionCaptureOnTurnSet: true,
		DecisionPath:             decisionDir,
	})
	state, history, _, _ := newDecisionCaptureState(t, manager, "invalid session id", "trace-persist-no-capture")
	history.Append(llm.Message{Role: llm.RoleAssistant, Text: "done"})

	_, _, err := state.complete("done", nil, nil)
	if !errors.Is(err, session.ErrInvalidSessionID) {
		t.Fatalf("expected invalid session id error, got %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(decisionDir, "memos.json")); !os.IsNotExist(statErr) {
		t.Fatalf("expected no memo file on persist failure, stat err=%v", statErr)
	}
	t.Cleanup(manager.StopDreaming)
}

func TestSessionTurnStateComplete_CaptureErrorDoesNotAffectReturn(t *testing.T) {
	baseDir := t.TempDir()
	decisionPath := filepath.Join(baseDir, "decision-file")
	if err := os.WriteFile(decisionPath, []byte("not-a-dir"), 0o600); err != nil {
		t.Fatalf("write blocker file: %v", err)
	}
	manager := memory.NewMemoryManager(memory.MemoryConfig{
		DecisionEnabled:          true,
		DecisionCaptureOnTurn:    true,
		DecisionCaptureOnTurnSet: true,
		DecisionPath:             decisionPath,
	})
	state, history, _, sess := newDecisionCaptureState(t, manager, "runner-capture-error", "trace-capture-error")
	history.Append(llm.Message{Role: llm.RoleAssistant, Text: "done"})

	response, sessionID, err := state.complete("done", nil, nil)
	if err != nil {
		t.Fatalf("capture failure should not leak to caller: %v", err)
	}
	if response != "done" || sessionID != sess.ID {
		t.Fatalf("unexpected response/session after capture failure: response=%q session=%q", response, sessionID)
	}
	t.Cleanup(manager.StopDreaming)
}

func TestSessionTurnStateComplete_DoesNotCaptureWhenToggleDisabled(t *testing.T) {
	decisionDir := filepath.Join(t.TempDir(), "decision")
	manager := memory.NewMemoryManager(memory.MemoryConfig{
		DecisionEnabled:          true,
		DecisionCaptureOnTurn:    false,
		DecisionCaptureOnTurnSet: true,
		DecisionPath:             decisionDir,
	})
	state, history, _, _ := newDecisionCaptureState(t, manager, "runner-capture-off", "trace-capture-off")
	history.Append(llm.Message{Role: llm.RoleAssistant, Text: "done"})

	if _, _, err := state.complete("done", nil, nil); err != nil {
		t.Fatalf("complete returned error: %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(decisionDir, "memos.json")); !os.IsNotExist(statErr) {
		t.Fatalf("expected no memo file when capture toggle is off, stat err=%v", statErr)
	}
	t.Cleanup(manager.StopDreaming)
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
