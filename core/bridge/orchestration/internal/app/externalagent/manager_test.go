package externalagent

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"

	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/llm"
	"ghost-os/bridge/orchestration/internal/contracts/api"
	"ghost-os/bridge/session"
	"ghost-os/bridge/streaming"
)

func TestResolveExecutionPolicyMapsPermissionModes(t *testing.T) {
	cases := []struct {
		mode         string
		wantMode     string
		wantSandbox  string
		wantApproval string
	}{
		{
			mode:         bridgeconfig.ExternalCodexPermissionReadOnly,
			wantMode:     bridgeconfig.ExternalCodexPermissionReadOnly,
			wantSandbox:  "read-only",
			wantApproval: "never",
		},
		{
			mode:         bridgeconfig.ExternalCodexPermissionDefault,
			wantMode:     bridgeconfig.ExternalCodexPermissionDefault,
			wantSandbox:  "workspace-write",
			wantApproval: "untrusted",
		},
		{
			mode:         bridgeconfig.ExternalCodexPermissionSafeYolo,
			wantMode:     bridgeconfig.ExternalCodexPermissionSafeYolo,
			wantSandbox:  "workspace-write",
			wantApproval: "on-failure",
		},
		{
			mode:         bridgeconfig.ExternalCodexPermissionYolo,
			wantMode:     bridgeconfig.ExternalCodexPermissionYolo,
			wantSandbox:  "danger-full-access",
			wantApproval: "never",
		},
		{
			mode:         "",
			wantMode:     bridgeconfig.DefaultExternalCodexPermissionMode,
			wantSandbox:  "workspace-write",
			wantApproval: "untrusted",
		},
	}

	for _, tc := range cases {
		t.Run(tc.wantMode, func(t *testing.T) {
			got, err := ResolveExecutionPolicy(tc.mode)
			if err != nil {
				t.Fatalf("ResolveExecutionPolicy: %v", err)
			}
			if got.PermissionMode != tc.wantMode || got.Sandbox != tc.wantSandbox || got.ApprovalPolicy != tc.wantApproval {
				t.Fatalf("unexpected policy: got=%+v", got)
			}
		})
	}

	if _, err := ResolveExecutionPolicy("unrestricted"); err == nil {
		t.Fatal("expected invalid permission mode to fail")
	}
}

func TestManagerExecuteStreamMapsCodexEventsAndHistory(t *testing.T) {
	manager, sessionStore, fake := newExternalAgentTestManager(t)
	fake.startTurn = func(ctx context.Context, client *fakeCodexClient, opts TurnOptions) (string, error) {
		client.emit(CodexEvent{Type: "task_started", Payload: map[string]any{"turn_id": "turn-1"}})
		client.emit(CodexEvent{Type: "agent_message", Payload: map[string]any{"message": "hello"}})
		client.emit(CodexEvent{Type: "exec_command_begin", Payload: map[string]any{
			"call_id": "exec-1",
			"command": "go test ./...",
		}})
		client.emit(CodexEvent{Type: "exec_command_end", Payload: map[string]any{
			"call_id": "exec-1",
			"output":  "ok",
			"status":  "completed",
		}})
		client.emit(CodexEvent{Type: "patch_apply_begin", Payload: map[string]any{
			"call_id": "patch-1",
			"changes": "README.md",
		}})
		client.emit(CodexEvent{Type: "patch_apply_end", Payload: map[string]any{
			"call_id": "patch-1",
			"status":  "completed",
		}})
		client.emit(CodexEvent{Type: "mcp_tool_begin", Payload: map[string]any{
			"call_id": "mcp-1",
			"name":    "github",
		}})
		client.emit(CodexEvent{Type: "mcp_tool_end", Payload: map[string]any{
			"call_id": "mcp-1",
			"output":  "mcp ok",
			"status":  "completed",
		}})
		client.emit(CodexEvent{Type: "task_complete", Payload: map[string]any{"turn_id": "turn-1"}})
		return "turn-1", nil
	}
	sink := &collectingSink{}

	message, sessionID, err := manager.ExecuteStream(context.Background(), api.ExternalAgentRequest{
		Message:        "run tests",
		PermissionMode: bridgeconfig.ExternalCodexPermissionSafeYolo,
		Model:          "gpt-5-codex",
		Effort:         "high",
	}, "trace-codex", sink, true)
	if err != nil {
		t.Fatalf("ExecuteStream: %v", err)
	}
	if message != "hello" {
		t.Fatalf("unexpected final message: got %q want %q", message, "hello")
	}
	if sessionID == "" {
		t.Fatal("expected session id")
	}
	if fake.startThreadOpts.ApprovalPolicy != "on-failure" || fake.startThreadOpts.Sandbox != "workspace-write" {
		t.Fatalf("unexpected thread policy: %+v", fake.startThreadOpts)
	}
	if fake.turnOpts.Model != "gpt-5-codex" || fake.turnOpts.Effort != "high" {
		t.Fatalf("unexpected turn opts: %+v", fake.turnOpts)
	}

	assertEventTypes(t, sink.events(), []streaming.EventType{
		streaming.EventRunStarted,
		streaming.EventCompletionDelta,
		streaming.EventToolCallStarted,
		streaming.EventToolCallFinished,
		streaming.EventToolCallStarted,
		streaming.EventToolCallFinished,
		streaming.EventToolCallStarted,
		streaming.EventToolCallFinished,
		streaming.EventMessage,
		streaming.EventDone,
	})

	loaded, err := sessionStore.Load(sessionID)
	if err != nil {
		t.Fatalf("load session: %v", err)
	}
	if loaded.ExternalRuntime == nil || loaded.ExternalRuntime.Provider != ProviderCodex {
		t.Fatalf("expected external runtime: %+v", loaded.ExternalRuntime)
	}
	if loaded.ExternalRuntime.PermissionMode != bridgeconfig.ExternalCodexPermissionSafeYolo {
		t.Fatalf("unexpected permission mode: %+v", loaded.ExternalRuntime)
	}
	if !hasToolCall(loaded.Messages, "codex_exec", "exec-1") ||
		!hasToolCall(loaded.Messages, "codex_patch", "patch-1") ||
		!hasToolCall(loaded.Messages, "codex_mcp", "mcp-1") {
		t.Fatalf("expected codex tool calls in history: %+v", loaded.Messages)
	}
	if !hasToolResult(loaded.Messages, "exec-1", "ok") ||
		!hasToolResult(loaded.Messages, "mcp-1", "mcp ok") {
		t.Fatalf("expected codex tool results in history: %+v", loaded.Messages)
	}
	if got := loaded.Messages[len(loaded.Messages)-1]; got.Role != llm.RoleAssistant || got.Text != "hello" {
		t.Fatalf("unexpected final history message: %+v", got)
	}
}

func TestManagerExecuteStreamPersistsRunningTurnDraft(t *testing.T) {
	manager, sessionStore, fake := newExternalAgentTestManager(t)
	fake.turnStarted = make(chan struct{})
	emitMore := make(chan struct{})
	finish := make(chan struct{})
	fake.startTurn = func(ctx context.Context, client *fakeCodexClient, opts TurnOptions) (string, error) {
		client.emit(CodexEvent{Type: "task_started", Payload: map[string]any{"turn_id": "turn-draft"}})
		client.emit(CodexEvent{Type: "agent_message", Payload: map[string]any{"message": "partial"}})
		close(client.turnStarted)
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-emitMore:
		}
		client.emit(CodexEvent{Type: "agent_message", Payload: map[string]any{"message": " answer"}})
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-finish:
		}
		client.emit(CodexEvent{Type: "task_complete", Payload: map[string]any{"turn_id": "turn-draft"}})
		return "turn-draft", nil
	}

	sink := newCollectingSink()
	done := make(chan executeResult, 1)
	go func() {
		message, sessionID, err := manager.ExecuteStream(context.Background(), api.ExternalAgentRequest{
			Message: "run in background",
		}, "trace-draft", sink, true)
		done <- executeResult{message: message, sessionID: sessionID, err: err}
	}()

	started := sink.waitForType(t, streaming.EventRunStarted)
	<-fake.turnStarted
	time.Sleep(350 * time.Millisecond)
	close(emitMore)
	var lastDraft *session.TurnDraft
	var lastAssistant string
	var lastLoadErr error
	if err := waitUntil(func() bool {
		loaded, err := sessionStore.Load(started.SessionID)
		if err != nil {
			lastLoadErr = err
			return false
		}
		lastLoadErr = nil
		lastDraft = loaded.TurnDraft
		if loaded.AssistantDraft != nil {
			lastAssistant = loaded.AssistantDraft.Text
		} else {
			lastAssistant = ""
		}
		if loaded.TurnDraft == nil || loaded.AssistantDraft == nil {
			return false
		}
		if loaded.TurnDraft.Status != session.TurnDraftStatusStreaming {
			return false
		}
		if loaded.AssistantDraft.Text != "partialanswer" {
			return false
		}
		return len(loaded.TurnDraft.AssistantSegments) == 1 &&
			loaded.TurnDraft.AssistantSegments[0].Content == "partialanswer"
	}); err != nil {
		t.Fatalf(
			"running turn draft was not persisted: %v loadErr=%v draft=%+v assistant=%q",
			err,
			lastLoadErr,
			lastDraft,
			lastAssistant,
		)
	}

	close(finish)
	result := waitExecuteResult(t, done)
	if result.err != nil {
		t.Fatalf("ExecuteStream: %v", result.err)
	}
	if result.message != "partialanswer" {
		t.Fatalf("unexpected final message: got %q want %q", result.message, "partialanswer")
	}
	loaded, err := sessionStore.Load(result.sessionID)
	if err != nil {
		t.Fatalf("load session: %v", err)
	}
	if loaded.TurnDraft != nil || loaded.AssistantDraft != nil {
		t.Fatalf("expected drafts to clear after completion: turn=%+v assistant=%+v", loaded.TurnDraft, loaded.AssistantDraft)
	}
}

func TestManagerApprovalBlocksUntilApproved(t *testing.T) {
	manager, sessionStore, fake := newExternalAgentTestManager(t)
	fake.startTurn = func(ctx context.Context, client *fakeCodexClient, opts TurnOptions) (string, error) {
		decision, err := client.requestApproval(ctx, ApprovalRequest{
			ID:     "approval-1",
			Kind:   "exec",
			Tool:   "codex_exec",
			CallID: "exec-1",
			Prompt: "Approve command execution",
			Payload: map[string]any{
				"command": "go test ./...",
			},
		})
		client.setApprovalDecision(decision)
		if err != nil {
			return "", err
		}
		client.emit(CodexEvent{Type: "agent_message", Payload: map[string]any{"message": "approved"}})
		client.emit(CodexEvent{Type: "task_complete", Payload: map[string]any{"turn_id": "turn-approval"}})
		return "turn-approval", nil
	}
	sink := newCollectingSink()
	done := make(chan executeResult, 1)
	go func() {
		message, sessionID, err := manager.ExecuteStream(context.Background(), api.ExternalAgentRequest{
			Message: "needs approval",
		}, "trace-approval", sink, true)
		done <- executeResult{message: message, sessionID: sessionID, err: err}
	}()

	awaiting := sink.waitForType(t, streaming.EventAwaitingHuman)
	if awaiting.SessionID == "" {
		t.Fatal("awaiting_human should include session_id")
	}
	if _, err := manager.Approve(api.ExternalAgentApprovalParams{
		SessionID:  awaiting.SessionID,
		ApprovalID: "approval-1",
		Decision:   DecisionApprovedForSession,
	}); err != nil {
		t.Fatalf("Approve: %v", err)
	}

	result := waitExecuteResult(t, done)
	if result.err != nil {
		t.Fatalf("ExecuteStream: %v", result.err)
	}
	if result.message != "approved" {
		t.Fatalf("unexpected message: %q", result.message)
	}
	if fake.approvalDecision() != DecisionApprovedForSession {
		t.Fatalf("unexpected approval decision: %q", fake.approvalDecision())
	}
	loaded, err := sessionStore.Load(result.sessionID)
	if err != nil {
		t.Fatalf("load session: %v", err)
	}
	if loaded.ExternalRuntime == nil || loaded.ExternalRuntime.Status != StatusIdle || len(loaded.ExternalRuntime.PendingApprovals) != 0 {
		t.Fatalf("approval should be cleared from runtime: %+v", loaded.ExternalRuntime)
	}
	if !hasToolCall(loaded.Messages, "codex_approval", "approval:approval-1") ||
		!hasToolResult(loaded.Messages, "approval:approval-1", DecisionApprovedForSession) {
		t.Fatalf("expected approval call/result history: %+v", loaded.Messages)
	}
}

func TestManagerStopInterruptsCurrentTurn(t *testing.T) {
	manager, _, fake := newExternalAgentTestManager(t)
	fake.turnStarted = make(chan struct{})
	fake.interrupted = make(chan struct{})
	fake.startTurn = func(ctx context.Context, client *fakeCodexClient, opts TurnOptions) (string, error) {
		client.emit(CodexEvent{Type: "task_started", Payload: map[string]any{"turn_id": "turn-stop"}})
		close(client.turnStarted)
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-client.interrupted:
			client.emit(CodexEvent{Type: "turn_aborted", Payload: map[string]any{"reason": "stopped"}})
			return "turn-stop", nil
		}
	}
	sink := newCollectingSink()
	done := make(chan executeResult, 1)
	go func() {
		message, sessionID, err := manager.ExecuteStream(context.Background(), api.ExternalAgentRequest{
			Message: "long run",
		}, "trace-stop", sink, true)
		done <- executeResult{message: message, sessionID: sessionID, err: err}
	}()

	<-fake.turnStarted
	started := sink.waitForType(t, streaming.EventRunStarted)
	if err := waitUntil(func() bool {
		ext, err := manager.externalRuntime(started.SessionID)
		return err == nil && ext != nil && ext.TurnID == "turn-stop"
	}); err != nil {
		t.Fatalf("runtime did not record turn id: %v", err)
	}

	response, err := manager.Stop(context.Background(), api.ExternalAgentStopParams{SessionID: started.SessionID})
	if err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if response.Status != StatusIdle || response.ThreadID != "thread-1" {
		t.Fatalf("unexpected stop response: %+v", response)
	}
	if fake.interruptThreadID() != "thread-1" || fake.interruptTurnID() != "turn-stop" {
		t.Fatalf("unexpected interrupt call: thread=%q turn=%q", fake.interruptThreadID(), fake.interruptTurnID())
	}
	if result := waitExecuteResult(t, done); result.err != nil {
		t.Fatalf("ExecuteStream after stop: %v", result.err)
	}
}

func TestManagerStopWaitsForTurnAbortBeforeReturning(t *testing.T) {
	manager, _, fake := newExternalAgentTestManager(t)
	fake.turnStarted = make(chan struct{})
	fake.interrupted = make(chan struct{})
	interruptObserved := make(chan struct{})
	releaseAbort := make(chan struct{})
	fake.startTurn = func(ctx context.Context, client *fakeCodexClient, opts TurnOptions) (string, error) {
		client.emit(CodexEvent{Type: "task_started", Payload: map[string]any{"turn_id": "turn-wait"}})
		close(client.turnStarted)
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-client.interrupted:
			close(interruptObserved)
		}
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-releaseAbort:
		}
		client.emit(CodexEvent{Type: "turn_aborted", Payload: map[string]any{"reason": "stopped"}})
		return "turn-wait", nil
	}
	sink := newCollectingSink()
	done := make(chan executeResult, 1)
	go func() {
		message, sessionID, err := manager.ExecuteStream(context.Background(), api.ExternalAgentRequest{
			Message: "pause me",
		}, "trace-stop-wait", sink, true)
		done <- executeResult{message: message, sessionID: sessionID, err: err}
	}()

	<-fake.turnStarted
	started := sink.waitForType(t, streaming.EventRunStarted)
	if err := waitUntil(func() bool {
		ext, err := manager.externalRuntime(started.SessionID)
		return err == nil && ext != nil && ext.TurnID == "turn-wait"
	}); err != nil {
		t.Fatalf("runtime did not record turn id: %v", err)
	}

	stopDone := make(chan error, 1)
	go func() {
		_, err := manager.Stop(context.Background(), api.ExternalAgentStopParams{SessionID: started.SessionID})
		stopDone <- err
	}()
	<-interruptObserved
	select {
	case err := <-stopDone:
		t.Fatalf("Stop returned before turn_aborted: %v", err)
	case <-time.After(50 * time.Millisecond):
	}
	close(releaseAbort)
	if err := <-stopDone; err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if result := waitExecuteResult(t, done); result.err != nil {
		t.Fatalf("ExecuteStream after stop: %v", result.err)
	}
}

func TestManagerExecuteStreamResumesCodexThreadAfterStop(t *testing.T) {
	manager, _, fake := newExternalAgentTestManager(t)
	fake.turnStarted = make(chan struct{})
	fake.interrupted = make(chan struct{})
	fake.startTurn = func(ctx context.Context, client *fakeCodexClient, opts TurnOptions) (string, error) {
		client.emit(CodexEvent{Type: "task_started", Payload: map[string]any{"turn_id": "turn-stop-resume"}})
		close(client.turnStarted)
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-client.interrupted:
			client.emit(CodexEvent{Type: "turn_aborted", Payload: map[string]any{"reason": "stopped"}})
			return "turn-stop-resume", nil
		}
	}
	sink := newCollectingSink()
	done := make(chan executeResult, 1)
	go func() {
		message, sessionID, err := manager.ExecuteStream(context.Background(), api.ExternalAgentRequest{
			Message: "pause before continuing",
		}, "trace-stop-resume", sink, true)
		done <- executeResult{message: message, sessionID: sessionID, err: err}
	}()

	<-fake.turnStarted
	started := sink.waitForType(t, streaming.EventRunStarted)
	if err := waitUntil(func() bool {
		ext, err := manager.externalRuntime(started.SessionID)
		return err == nil && ext != nil && ext.TurnID == "turn-stop-resume"
	}); err != nil {
		t.Fatalf("runtime did not record turn id: %v", err)
	}
	if _, err := manager.Stop(context.Background(), api.ExternalAgentStopParams{SessionID: started.SessionID}); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if result := waitExecuteResult(t, done); result.err != nil {
		t.Fatalf("ExecuteStream after stop: %v", result.err)
	}

	fake.startTurn = func(ctx context.Context, client *fakeCodexClient, opts TurnOptions) (string, error) {
		client.emit(CodexEvent{Type: "task_started", Payload: map[string]any{"turn_id": "turn-continued"}})
		client.emit(CodexEvent{Type: "agent_message", Payload: map[string]any{"message": "continued"}})
		client.emit(CodexEvent{Type: "task_complete", Payload: map[string]any{"turn_id": "turn-continued"}})
		return "turn-continued", nil
	}
	message, sessionID, err := manager.ExecuteStream(context.Background(), api.ExternalAgentRequest{
		Message:   "continue",
		SessionID: started.SessionID,
	}, "trace-continued", newCollectingSink(), false)
	if err != nil {
		t.Fatalf("ExecuteStream resume: %v", err)
	}
	if sessionID != started.SessionID {
		t.Fatalf("unexpected resumed session id: got %q want %q", sessionID, started.SessionID)
	}
	if message != "continued" {
		t.Fatalf("unexpected resumed message: got %q want %q", message, "continued")
	}
	if fake.resumeThreadID() != "thread-1" {
		t.Fatalf("expected resume thread-1, got %q", fake.resumeThreadID())
	}
}

func TestManagerRuntimeForSessionPassesConfiguredExecutionPaths(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("GHOST_CONFIG_PATH", filepath.Join(tempDir, "config.toml"))
	t.Setenv("GHOST_PROVIDER", "custom")
	t.Setenv("GHOST_BASE_URL", "https://initial.example/v1")
	t.Setenv("GHOST_PROJECT_ROOT", tempDir)
	t.Setenv("GHOST_TASKS_PATH", filepath.Join(tempDir, "tasks"))
	t.Setenv("GHOST_PROMPTS_DIR", filepath.Join(tempDir, "prompts"))
	t.Setenv("GHOST_CODEX_CLI_PATH", "/opt/codex/bin/codex")
	t.Setenv("GHOST_NODE_BIN_PATH", "/opt/node/bin/node")

	configStore, err := bridgeconfig.NewStoreFromEnv()
	if err != nil {
		t.Fatalf("NewStoreFromEnv: %v", err)
	}
	sessionStore, err := session.NewStore(filepath.Join(tempDir, "sessions"))
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}

	var captured ClientConfig
	manager := NewManager(configStore, sessionStore)
	manager.ClientFactory = func(cfg ClientConfig) CodexClient {
		captured = cfg
		return newFakeCodexClient()
	}
	t.Cleanup(manager.Close)

	cfg, err := configStore.Config()
	if err != nil {
		t.Fatalf("Config: %v", err)
	}
	if _, err := manager.runtimeForSession("session-1", cfg, filepath.Join(tempDir, "project")); err != nil {
		t.Fatalf("runtimeForSession: %v", err)
	}
	if captured.CodexPath != "/opt/codex/bin/codex" {
		t.Fatalf("unexpected codex path: got %q", captured.CodexPath)
	}
	if captured.NodePath != "/opt/node/bin/node" {
		t.Fatalf("unexpected node path: got %q", captured.NodePath)
	}
}

func newExternalAgentTestManager(t *testing.T) (*Manager, *session.Store, *fakeCodexClient) {
	t.Helper()

	tempDir := t.TempDir()
	t.Setenv("GHOST_CONFIG_PATH", filepath.Join(tempDir, "config.toml"))
	t.Setenv("GHOST_PROVIDER", "custom")
	t.Setenv("GHOST_BASE_URL", "https://initial.example/v1")
	t.Setenv("GHOST_PROJECT_ROOT", tempDir)
	t.Setenv("GHOST_TASKS_PATH", filepath.Join(tempDir, "tasks"))
	t.Setenv("GHOST_PROMPTS_DIR", filepath.Join(tempDir, "prompts"))

	configStore, err := bridgeconfig.NewStoreFromEnv()
	if err != nil {
		t.Fatalf("NewStoreFromEnv: %v", err)
	}
	sessionStore, err := session.NewStore(filepath.Join(tempDir, "sessions"))
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	fake := newFakeCodexClient()
	manager := NewManager(configStore, sessionStore)
	manager.ClientFactory = func(ClientConfig) CodexClient {
		return fake
	}
	t.Cleanup(manager.Close)
	return manager, sessionStore, fake
}

type fakeCodexClient struct {
	mu sync.Mutex

	eventHandler    func(CodexEvent)
	approvalHandler func(context.Context, ApprovalRequest) (string, error)
	startTurn       func(context.Context, *fakeCodexClient, TurnOptions) (string, error)

	startThreadOpts  ThreadOptions
	resumeThreadOpts ThreadOptions
	turnOpts         TurnOptions
	decision         string

	turnStarted chan struct{}
	interrupted chan struct{}
	threadID    string
	turnID      string
}

func newFakeCodexClient() *fakeCodexClient {
	return &fakeCodexClient{
		threadID: "thread-1",
		turnID:   "turn-1",
	}
}

func (f *fakeCodexClient) Connect(context.Context) error { return nil }

func (f *fakeCodexClient) StartThread(_ context.Context, opts ThreadOptions) (ThreadResult, error) {
	f.mu.Lock()
	f.startThreadOpts = opts
	threadID := f.threadID
	f.mu.Unlock()
	return ThreadResult{ThreadID: threadID, Model: opts.Model}, nil
}

func (f *fakeCodexClient) ResumeThread(_ context.Context, opts ThreadOptions) (ThreadResult, error) {
	f.mu.Lock()
	f.resumeThreadOpts = opts
	threadID := opts.ThreadID
	f.mu.Unlock()
	return ThreadResult{ThreadID: threadID, Model: opts.Model}, nil
}

func (f *fakeCodexClient) StartTurn(ctx context.Context, opts TurnOptions) (string, error) {
	f.mu.Lock()
	f.turnOpts = opts
	startTurn := f.startTurn
	turnID := f.turnID
	f.mu.Unlock()
	if startTurn == nil {
		f.emit(CodexEvent{Type: "task_complete", Payload: map[string]any{"turn_id": turnID}})
		return turnID, nil
	}
	return startTurn(ctx, f, opts)
}

func (f *fakeCodexClient) InterruptTurn(_ context.Context, threadID string, turnID string) error {
	f.mu.Lock()
	f.threadID = threadID
	f.turnID = turnID
	interrupted := f.interrupted
	f.mu.Unlock()
	if interrupted != nil {
		close(interrupted)
	}
	return nil
}

func (f *fakeCodexClient) SetEventHandler(handler func(CodexEvent)) {
	f.mu.Lock()
	f.eventHandler = handler
	f.mu.Unlock()
}

func (f *fakeCodexClient) SetApprovalHandler(handler func(context.Context, ApprovalRequest) (string, error)) {
	f.mu.Lock()
	f.approvalHandler = handler
	f.mu.Unlock()
}

func (f *fakeCodexClient) Close() error { return nil }

func (f *fakeCodexClient) emit(event CodexEvent) {
	f.mu.Lock()
	handler := f.eventHandler
	f.mu.Unlock()
	if handler != nil {
		handler(event)
	}
}

func (f *fakeCodexClient) requestApproval(ctx context.Context, approval ApprovalRequest) (string, error) {
	f.mu.Lock()
	handler := f.approvalHandler
	f.mu.Unlock()
	if handler == nil {
		return "", errors.New("approval handler is not configured")
	}
	return handler(ctx, approval)
}

func (f *fakeCodexClient) setApprovalDecision(decision string) {
	f.mu.Lock()
	f.decision = decision
	f.mu.Unlock()
}

func (f *fakeCodexClient) approvalDecision() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.decision
}

func (f *fakeCodexClient) interruptThreadID() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.threadID
}

func (f *fakeCodexClient) interruptTurnID() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.turnID
}

func (f *fakeCodexClient) resumeThreadID() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.resumeThreadOpts.ThreadID
}

type collectingSink struct {
	mu      sync.Mutex
	emitted []streaming.Event
	waitCh  chan streaming.Event
}

func newCollectingSink() *collectingSink {
	return &collectingSink{waitCh: make(chan streaming.Event, 32)}
}

func (s *collectingSink) Emit(_ context.Context, event streaming.Event) (streaming.Event, error) {
	s.mu.Lock()
	s.emitted = append(s.emitted, event)
	waitCh := s.waitCh
	s.mu.Unlock()
	if waitCh != nil {
		select {
		case waitCh <- event:
		default:
		}
	}
	return event, nil
}

func (s *collectingSink) eventsSnapshot() []streaming.Event {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]streaming.Event(nil), s.emitted...)
}

func (s *collectingSink) events() []streaming.Event {
	return s.eventsSnapshot()
}

func (s *collectingSink) waitForType(t *testing.T, eventType streaming.EventType) streaming.Event {
	t.Helper()
	timeout := time.After(2 * time.Second)
	for {
		for _, event := range s.eventsSnapshot() {
			if event.Type == eventType {
				return event
			}
		}
		select {
		case event := <-s.waitCh:
			if event.Type == eventType {
				return event
			}
		case <-timeout:
			t.Fatalf("timed out waiting for stream event %q; events=%+v", eventType, s.eventsSnapshot())
		}
	}
}

type executeResult struct {
	message   string
	sessionID string
	err       error
}

func waitExecuteResult(t *testing.T, ch <-chan executeResult) executeResult {
	t.Helper()
	select {
	case result := <-ch:
		return result
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for ExecuteStream")
		return executeResult{}
	}
}

func waitUntil(ok func() bool) error {
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if ok() {
			return nil
		}
		time.Sleep(10 * time.Millisecond)
	}
	return errors.New("condition was not satisfied before timeout")
}

func assertEventTypes(t *testing.T, events []streaming.Event, want []streaming.EventType) {
	t.Helper()
	if len(events) != len(want) {
		t.Fatalf("unexpected event count: got=%d want=%d events=%+v", len(events), len(want), events)
	}
	for index, eventType := range want {
		if events[index].Type != eventType {
			t.Fatalf("unexpected event[%d]: got=%q want=%q events=%+v", index, events[index].Type, eventType, events)
		}
	}
}

func hasToolCall(messages []llm.Message, name string, id string) bool {
	for _, msg := range messages {
		for _, call := range msg.ToolCalls {
			if call.Name == name && call.ID == id {
				return true
			}
		}
	}
	return false
}

func hasToolResult(messages []llm.Message, callID string, text string) bool {
	for _, msg := range messages {
		if msg.Role != llm.RoleTool || msg.ToolCallID != callID {
			continue
		}
		result, ok := llm.ParseToolResultEnvelope(msg.Text)
		if !ok {
			continue
		}
		if result.Output == text || result.Error == text {
			return true
		}
	}
	return false
}
