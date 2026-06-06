package orchestration

import (
	"context"
	"encoding/json"
	"ghost-os/bridge/agent"
	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/llm"
	appagentturn "ghost-os/bridge/orchestration/internal/app/agentturn"
	"ghost-os/bridge/orchestration/internal/app/agentturn/toolselect"
	apptools "ghost-os/bridge/orchestration/internal/app/tools"
	bridgeruntime "ghost-os/bridge/runtime"
	"ghost-os/bridge/session"
	"ghost-os/bridge/streaming"
	"ghost-os/bridge/tools"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCompletionPromptRefreshLoadsSkillContextSameRun(t *testing.T) {
	projectRoot := t.TempDir()
	skillDir := filepath.Join(projectRoot, ".agents", "skills", "release")
	writePromptRefreshSkillFile(t, skillDir, "release_flow", "release skill", "Release body.")

	sess := session.NewSession("")
	sess.AdvanceToolTurn(3)
	registry := tools.NewRegistry()
	registry.Register(tools.NewToolSearchTool(
		registry,
		tools.VisibilityOptions{ToolSearchEnabled: true},
		3,
		tools.ToolSearchOptions{ProjectRoot: projectRoot},
	))
	catalog := tools.NewScopedCatalog(registry, []string{"sfind"})
	cfg := bridgeconfig.Config{
		MaxTurns:    3,
		ProjectRoot: projectRoot,
		PromptsDir:  filepath.Join(t.TempDir(), "prompts"),
		ToolSearch:  bridgeconfig.ToolSearchConfig{Enabled: true, IdleTurns: 3},
	}
	prompt, err := bridgeruntime.BuildSystemPromptForSession(cfg, catalog, sess, cfg.ToolSearch.IdleTurns)
	if err != nil {
		t.Fatalf("BuildSystemPromptForSession: %v", err)
	}

	completer := &promptRefreshCompleter{
		responses: []*llm.CompletionResponse{
			promptRefreshToolCall("call-1", "sfind", `{"action":"load","skill_names":["release_flow"]}`),
			promptRefreshStop("done"),
		},
	}
	deps := agentRuntimeDependencies{cfg: cfg, client: completer, registry: registry}
	runAgent, err := newSessionTurnPreparer(nil, nil, nil, nil, nil).BuildTurnAgent(
		deps,
		catalog,
		sess,
		agent.NewHistory(prompt),
	)
	if err != nil {
		t.Fatalf("buildTurnAgent: %v", err)
	}

	ctx := tools.WithSession(context.Background(), sess)
	if _, err := runAgent.RunMessageWithTraceID(ctx, llm.Message{Role: llm.RoleUser, Text: "load it"}, "trace"); err != nil {
		t.Fatalf("RunMessageWithTraceID: %v", err)
	}
	if len(completer.requests) != 2 {
		t.Fatalf("expected two completion requests, got %d", len(completer.requests))
	}
	if !strings.Contains(completer.requests[1].Messages[0].Text, "Release body.") {
		t.Fatalf("expected refreshed prompt to include loaded skill body, got %q", completer.requests[1].Messages[0].Text)
	}
}

type promptRefreshCompleter struct {
	requests  []llm.CompletionRequest
	responses []*llm.CompletionResponse
}

func (f *promptRefreshCompleter) Complete(
	_ context.Context,
	request llm.CompletionRequest,
) (*llm.CompletionResponse, error) {
	f.requests = append(f.requests, request)
	index := len(f.requests) - 1
	return f.responses[index], nil
}

func promptRefreshToolCall(id string, name string, arguments string) *llm.CompletionResponse {
	return &llm.CompletionResponse{
		FinishReason: llm.FinishToolCalls,
		Message: llm.Message{
			Role: llm.RoleAssistant,
			ToolCalls: []llm.ToolCall{
				{ID: id, Name: name, Arguments: json.RawMessage(arguments)},
			},
		},
	}
}

func promptRefreshStop(text string) *llm.CompletionResponse {
	return &llm.CompletionResponse{
		FinishReason: llm.FinishStop,
		Message: llm.Message{
			Role: llm.RoleAssistant,
			Text: text,
		},
	}
}

func writePromptRefreshSkillFile(
	t *testing.T,
	dir string,
	name string,
	description string,
	body string,
) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll(%s): %v", dir, err)
	}
	content := "---\nname: " + name + "\ndescription: " + description + "\n---\n" + body + "\n"
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(SKILL.md): %v", err)
	}
}

type persistingToolTurnEchoTool struct{}

func (persistingToolTurnEchoTool) Name() string {
	return "echo"
}

func (persistingToolTurnEchoTool) Description() string {
	return "persisting echo"
}

func (persistingToolTurnEchoTool) Parameters() json.RawMessage {
	return json.RawMessage(`{"type":"object"}`)
}

func (persistingToolTurnEchoTool) Execute(
	ctx context.Context,
	_ json.RawMessage,
	_ string,
) (string, error) {
	sess := tools.SessionFromContext(ctx)
	if sess == nil {
		return "", context.Canceled
	}
	checkpoint := tools.SessionCheckpointFromContext(ctx)
	if checkpoint == nil {
		return "", context.Canceled
	}
	sess.EnsureDynamicToolLoaded("echo", "tool-turn-test")
	if err := checkpoint.Save(sess); err != nil {
		return "", err
	}
	return "tool-ok", nil
}

func TestSessionTurnStatePersistsCommittedToolTurnOnLaterError(t *testing.T) {
	sessionStore := newTempSessionStore(t)
	sess := session.NewSession("base system prompt")
	execCtx := tools.WithSession(context.Background(), sess)
	execCtx = tools.WithSessionCheckpoint(execCtx, sessionStore)

	completer := &proTestCompleter{
		responses: []*llm.CompletionResponse{{
			Message: llm.Message{
				Role: llm.RoleAssistant,
				ToolCalls: []llm.ToolCall{{
					ID:        "call-echo-1",
					Name:      "echo",
					Arguments: []byte(`{"input":"hi"}`),
				}},
			},
			FinishReason: llm.FinishToolCalls,
		}},
	}
	registry := tools.NewRegistry()
	registry.Register(persistingToolTurnEchoTool{})
	runAgent := agent.NewAgentWithHistory(completer, registry, agent.NewHistory("base system prompt"), 3)

	turn := &sessionTurnState{
		SessionStore: sessionStore,
		Persistence:  newSessionTurnCommitter(sessionStore),
		Session:      sess,
		Agent:        runAgent,
		ExecCtx:      execCtx,
		TraceID:      "trace-tool-turn-transaction",
	}

	response, runErr := runAgent.RunWithTraceID(execCtx, "hello", turn.TraceID)
	if runErr == nil {
		t.Fatal("expected completion failure after tool turn")
	}

	_, persistedSessionID, err := turn.Complete(response, runErr, nil)
	if err == nil {
		t.Fatal("expected turn completion to return the run error")
	}
	if persistedSessionID != sess.ID {
		t.Fatalf("unexpected persisted session id: got %q want %q", persistedSessionID, sess.ID)
	}

	loaded, loadErr := sessionStore.Load(sess.ID)
	if loadErr != nil {
		t.Fatalf("load session: %v", loadErr)
	}
	if len(loaded.Messages) != 4 {
		t.Fatalf("expected system + committed tool turn messages, got %+v", loaded.Messages)
	}
	if loaded.Messages[1].Role != llm.RoleUser || loaded.Messages[1].Text != "hello" {
		t.Fatalf("unexpected persisted user message: %+v", loaded.Messages[1])
	}
	if loaded.Messages[2].Role != llm.RoleAssistant || len(loaded.Messages[2].ToolCalls) != 1 {
		t.Fatalf("unexpected persisted assistant tool call: %+v", loaded.Messages[2])
	}
	if loaded.Messages[3].Role != llm.RoleTool || loaded.Messages[3].ToolCallID != "call-echo-1" {
		t.Fatalf("unexpected persisted tool result: %+v", loaded.Messages[3])
	}
	if !strings.Contains(loaded.Messages[3].Text, `"status":"success"`) || !strings.Contains(loaded.Messages[3].Text, `"tool":"echo"`) {
		t.Fatalf("unexpected tool result payload: %q", loaded.Messages[3].Text)
	}

	loads := loaded.DynamicToolLoadsSnapshot()
	if len(loads) != 1 || loads[0].ToolName != "echo" {
		t.Fatalf("expected persisted tool-side effect, got %+v", loads)
	}
}

type recordingAppEventSink struct {
	events []streaming.Event
}

func (s *recordingAppEventSink) Emit(_ context.Context, event streaming.Event) (streaming.Event, error) {
	s.events = append(s.events, event)
	return event, nil
}

func mustAppEvent(
	t *testing.T,
	traceID string,
	sessionID string,
	turn int,
	stepID string,
	eventType streaming.EventType,
	payload any,
) streaming.Event {
	t.Helper()

	event, err := streaming.NewEvent(traceID, sessionID, turn, stepID, eventType, payload)
	if err != nil {
		t.Fatalf("NewEvent returned error: %v", err)
	}
	return event
}

func mustAppAssistantStepID(t *testing.T, turn int) string {
	t.Helper()

	stepID, err := streaming.AssistantStepID(turn)
	if err != nil {
		t.Fatalf("AssistantStepID returned error: %v", err)
	}
	return stepID
}

func TestStreamingEventMatchesSharedContract(t *testing.T) {
	event := streaming.Event{
		ID:        "trace-123:000001",
		StepID:    "turn-0001-assistant",
		TraceID:   "trace-123",
		SessionID: "session-123",
		Turn:      1,
		Type:      streaming.EventMessage,
		Payload: map[string]any{
			"text":       "done",
			"session_id": "session-123",
		},
		At: time.Unix(42, 0).UTC(),
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal streaming event: %v", err)
	}

	var contract agentStreamEventContract
	if err := json.Unmarshal(data, &contract); err != nil {
		t.Fatalf("unmarshal shared contract: %v", err)
	}
	if contract.Type != string(streaming.EventMessage) || contract.TraceID != "trace-123" {
		t.Fatalf("unexpected contract envelope: %+v", contract)
	}
	if contract.Payload["text"] != "done" || contract.At == "" {
		t.Fatalf("unexpected contract payload: %+v", contract)
	}
}

func TestSessionPushEventMatchesSharedContract(t *testing.T) {
	event := sessionPushEvent{
		ID:        "session-123:000001",
		Type:      sessionPushAssistantMessage,
		TraceID:   "trace-123",
		SessionID: "session-123",
		Payload: assistantMessagePushPayload{
			Message:      "sent to phone",
			SessionEnded: false,
		},
		At: time.Unix(52, 0).UTC(),
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal session push event: %v", err)
	}

	var contract sessionPushEventContract
	if err := json.Unmarshal(data, &contract); err != nil {
		t.Fatalf("unmarshal shared contract: %v", err)
	}
	if contract.Type != string(sessionPushAssistantMessage) || contract.SessionID != "session-123" {
		t.Fatalf("unexpected contract envelope: %+v", contract)
	}
	if contract.Payload["message"] != "sent to phone" || contract.At == "" {
		t.Fatalf("unexpected contract payload: %+v", contract)
	}
}

func TestNormalizeConfiguredToolLists_RejectsOverlap(t *testing.T) {
	_, _, err := apptools.NormalizeConfiguredToolLists([]string{"script_exec"}, []string{"script_exec"})
	if err == nil {
		t.Fatal("expected overlap error")
	}
}

func TestNormalizeConfiguredToolLists_RejectsUnknownTool(t *testing.T) {
	_, _, err := apptools.NormalizeConfiguredToolLists([]string{"ghost_tool"}, nil)
	if err == nil {
		t.Fatal("expected unknown tool error")
	}
}

func TestNormalizeConfiguredToolLists_AllowsCodexCLI(t *testing.T) {
	allowlist, blocklist, err := apptools.NormalizeConfiguredToolLists([]string{"codex_cli"}, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(allowlist) != 1 || allowlist[0] != "codex_cli" {
		t.Fatalf("unexpected allowlist: %v", allowlist)
	}
	if len(blocklist) != 0 {
		t.Fatalf("unexpected blocklist: %v", blocklist)
	}
}

func TestNormalizeConfiguredToolLists_AllowsBlockingAskHumanAndToolSearch(t *testing.T) {
	allowlist, blocklist, err := apptools.NormalizeConfiguredToolLists(nil, []string{"ask_human", "script_exec", "sfind"})
	if err != nil {
		t.Fatalf("normalizeConfiguredToolLists: %v", err)
	}
	if len(allowlist) != 0 {
		t.Fatalf("unexpected allowlist: %v", allowlist)
	}
	expected := []string{"ask_human", "script_exec", "sfind"}
	if len(blocklist) != len(expected) {
		t.Fatalf("unexpected blocklist: %v", blocklist)
	}
	for index, name := range expected {
		if blocklist[index] != name {
			t.Fatalf("unexpected blocklist at %d: got %v want %v", index, blocklist, expected)
		}
	}
}

func TestToolSelectionPolicy_ApplyAddsAllowlistAndHonorsBlocklist(t *testing.T) {
	policy := bridgeruntime.NewToolSelectionPolicy(bridgeconfig.Config{
		ToolSelector: bridgeconfig.ToolSelectorConfig{
			Allowlist: []string{"codex_cli"},
			Blocklist: []string{"script_exec"},
		},
	})

	selected := policy.Apply([]string{"ask_human", "script_exec", "codex_cli", "web_search"}, []string{"script_exec", "web_search"})
	expected := []string{"codex_cli", "web_search"}
	if len(selected) != len(expected) {
		t.Fatalf("unexpected tool count: got %v want %v", selected, expected)
	}
	for index, name := range expected {
		if selected[index] != name {
			t.Fatalf("unexpected selection at %d: got %v want %v", index, selected, expected)
		}
	}
}

func selectToolsForTest(
	ctx context.Context,
	deps agentRuntimeDependencies,
	history *agent.History,
	userMessage string,
	traceID string,
	selectorFactory toolselect.SelectorFactory,
) (tools.ToolCatalog, string, error) {
	return toolselect.SelectForTurn(ctx, toolselect.Request{
		Config:          deps.cfg,
		Registry:        deps.registry,
		History:         history,
		UserMessage:     userMessage,
		TraceID:         traceID,
		SelectorFactory: selectorFactory,
	})
}

func TestSessionTurnPreparer_SelectToolsForTurn_HasNoResidentToolsWithoutAllowlist(t *testing.T) {
	deps := newRunnerTestDeps(bridgeconfig.Config{ToolSelector: bridgeconfig.ToolSelectorConfig{Blocklist: []string{"script_exec"}}, MaxTurns: 6})

	catalog, prompt, err := selectToolsForTest(context.Background(), deps, agent.NewHistory("system prompt"), "read config", "trace-policy-disabled", nil)
	if err != nil {
		t.Fatalf("selectToolsForTurn returned error: %v", err)
	}
	for _, name := range []string{"script_exec", "ask_human", "codex_cli", "web_search"} {
		if catalog.Get(name) != nil {
			t.Fatalf("expected %q to stay hidden without allowlist, got visible catalog", name)
		}
	}
	if prompt != "" {
		t.Fatalf("expected empty prompt override, got %q", prompt)
	}
}

func TestSessionTurnPreparer_SelectToolsForTurn_AllowlistDefinesResidentToolsWithoutAllowlistOnly(t *testing.T) {
	deps := newRunnerTestDeps(bridgeconfig.Config{
		ToolSelector: bridgeconfig.ToolSelectorConfig{
			Allowlist: []string{"script_exec"},
		},
		MaxTurns: 6,
	})

	catalog, _, err := selectToolsForTest(context.Background(), deps, agent.NewHistory("system prompt"), "read config", "trace-policy-allowlist", nil)
	if err != nil {
		t.Fatalf("selectToolsForTurn returned error: %v", err)
	}
	if catalog.Get("script_exec") == nil {
		t.Fatal("expected allowlisted tool to remain available")
	}
	if catalog.Get("ask_human") != nil || catalog.Get("web_search") != nil || catalog.Get("codex_cli") != nil {
		t.Fatalf("expected non-allowlisted tools to stay hidden")
	}
}

func TestSessionTurnPreparer_SelectToolsForTurn_AppliesAllowlistToSubset(t *testing.T) {
	selector := &fakeSelectorEngine{result: bridgeruntime.ToolSelectorResult{Mode: "subset", Tools: []string{"codex_cli"}, Confidence: 0.9}}
	deps := newRunnerTestDeps(bridgeconfig.Config{
		ToolSelector: bridgeconfig.ToolSelectorConfig{
			Enabled:   true,
			Mode:      "llm",
			Allowlist: []string{"ask_human"},
		},
		MaxTurns:   6,
		PromptsDir: filepath.Join(t.TempDir(), "prompts"),
	})

	catalog, prompt, err := selectToolsForTest(
		context.Background(),
		deps,
		agent.NewHistory("system prompt"),
		"read config",
		"trace-policy-subset",
		func(bridgeconfig.Config, tools.ToolCatalog) bridgeruntime.SelectorEngine { return selector },
	)
	if err != nil {
		t.Fatalf("selectToolsForTurn returned error: %v", err)
	}
	if catalog.Get("ask_human") == nil || catalog.Get("codex_cli") == nil {
		t.Fatal("expected resident and selector-selected tools to remain in scoped subset")
	}
	for _, name := range []string{"web_search", "script_exec", "sfind"} {
		if catalog.Get(name) != nil {
			t.Fatalf("expected %q to stay hidden outside scoped subset", name)
		}
	}
	if prompt == "" {
		t.Fatal("expected prompt override for scoped subset")
	}
}

func TestSessionTurnPreparer_SelectToolsForTurn_PassesSelectorVisibleCatalogOutsideStrictMode(t *testing.T) {
	var available []string
	selectorFactory := func(_ bridgeconfig.Config, catalog tools.ToolCatalog) bridgeruntime.SelectorEngine {
		available = appagentturn.ToolCatalogNames(catalog)
		return &fakeSelectorEngine{result: bridgeruntime.ToolSelectorResult{Mode: "all"}}
	}
	deps := newRunnerTestDeps(bridgeconfig.Config{
		ToolSelector: bridgeconfig.ToolSelectorConfig{
			Enabled:   true,
			Mode:      "llm",
			Allowlist: []string{"ask_human"},
			Blocklist: []string{"script_exec"},
		},
		MaxTurns: 6,
	})

	_, _, err := selectToolsForTest(context.Background(), deps, agent.NewHistory("system prompt"), "read config", "trace-policy-visible", selectorFactory)
	if err != nil {
		t.Fatalf("selectToolsForTurn returned error: %v", err)
	}
	if containsToolName(available, "script_exec") {
		t.Fatalf("expected selector-visible catalog to exclude blocked tool, got %v", available)
	}
	for _, name := range []string{"ask_human", "codex_cli", "web_search"} {
		if !containsToolName(available, name) {
			t.Fatalf("expected selector-visible catalog to include %q outside strict mode, got %v", name, available)
		}
	}
}

func TestSessionTurnPreparer_SelectToolsForTurn_AllowlistOnlyScopesVisibleTools(t *testing.T) {
	var available []string
	selectorFactory := func(_ bridgeconfig.Config, catalog tools.ToolCatalog) bridgeruntime.SelectorEngine {
		available = appagentturn.ToolCatalogNames(catalog)
		return &fakeSelectorEngine{result: bridgeruntime.ToolSelectorResult{Mode: "all"}}
	}
	deps := newRunnerTestDeps(bridgeconfig.Config{
		ToolSelector: bridgeconfig.ToolSelectorConfig{
			Enabled:       true,
			Mode:          "llm",
			AllowlistOnly: true,
			Allowlist:     []string{"script_exec"},
		},
		MaxTurns: 6,
	})

	catalog, _, err := selectToolsForTest(context.Background(), deps, agent.NewHistory("system prompt"), "read config", "trace-policy-allowlist-only", selectorFactory)
	if err != nil {
		t.Fatalf("selectToolsForTurn returned error: %v", err)
	}
	if catalog.Get("script_exec") == nil {
		t.Fatal("expected allowlisted tool to remain available")
	}
	if catalog.Get("web_search") != nil || catalog.Get("codex_cli") != nil {
		t.Fatal("expected non-allowlisted tools to be hidden")
	}
	if catalog.Get("ask_human") != nil {
		t.Fatal("expected ask_human to stay hidden when not allowlisted")
	}
	if len(available) != 1 || available[0] != "script_exec" {
		t.Fatalf("expected selector-visible catalog to stay strict, got %v", available)
	}
}

func TestSessionTurnPreparer_SelectToolsForTurn_ToolSearchScopesVisibleTools(t *testing.T) {
	deps := newRunnerTestDeps(bridgeconfig.Config{
		ToolSelector: bridgeconfig.ToolSelectorConfig{
			Allowlist: []string{"codex_cli", "sfind"},
		},
		ToolSearch: bridgeconfig.ToolSearchConfig{
			Enabled:   true,
			IdleTurns: 3,
		},
		MaxTurns: 6,
	})

	catalog, _, err := selectToolsForTest(context.Background(), deps, agent.NewHistory("system prompt"), "find tools", "trace-policy-tool-search", nil)
	if err != nil {
		t.Fatalf("selectToolsForTurn returned error: %v", err)
	}
	for _, name := range []string{"codex_cli", "sfind"} {
		if catalog.Get(name) == nil {
			t.Fatalf("expected %q to remain visible", name)
		}
	}
	for _, name := range []string{"ask_human", "script_exec", "web_search"} {
		if catalog.Get(name) != nil {
			t.Fatalf("expected %q to stay hidden until dynamically loaded", name)
		}
	}
}

func TestSessionTurnPreparer_SelectToolsForTurn_CanSelectNonResidentToolsWithoutAllowlist(t *testing.T) {
	selector := &fakeSelectorEngine{result: bridgeruntime.ToolSelectorResult{Mode: "subset", Tools: []string{"script_exec"}, Confidence: 0.9}}
	deps := newRunnerTestDeps(bridgeconfig.Config{
		ToolSelector: bridgeconfig.ToolSelectorConfig{
			Enabled: true,
			Mode:    "llm",
		},
		MaxTurns:   6,
		PromptsDir: filepath.Join(t.TempDir(), "prompts"),
	})

	catalog, prompt, err := selectToolsForTest(
		context.Background(),
		deps,
		agent.NewHistory("system prompt"),
		"read config",
		"trace-policy-empty-resident-subset",
		func(bridgeconfig.Config, tools.ToolCatalog) bridgeruntime.SelectorEngine { return selector },
	)
	if err != nil {
		t.Fatalf("selectToolsForTurn returned error: %v", err)
	}
	if catalog.Get("script_exec") == nil {
		t.Fatal("expected selector to enable non-resident tool")
	}
	for _, name := range []string{"ask_human", "codex_cli", "web_search"} {
		if catalog.Get(name) != nil {
			t.Fatalf("expected %q to stay hidden outside scoped subset", name)
		}
	}
	if prompt == "" {
		t.Fatal("expected prompt override for scoped subset")
	}
}
