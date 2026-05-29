package orchestration

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"ghost-os/bridge/agent"
	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/llm"
	bridgeruntime "ghost-os/bridge/runtime"
	"ghost-os/bridge/session"
	"ghost-os/bridge/streaming"
	"ghost-os/bridge/tools"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestNewAgentResponsePayloadBuildsNormalSessionResponse(t *testing.T) {
	payload, err := newAgentResponsePayload("final answer", "session-1", nil, agentResponseMeta{})
	if err != nil {
		t.Fatalf("newAgentResponsePayload returned error: %v", err)
	}
	if payload.Message != "final answer" {
		t.Fatalf("unexpected message: got %q want %q", payload.Message, "final answer")
	}
	if payload.SessionID != "session-1" {
		t.Fatalf("unexpected session id: got %q want %q", payload.SessionID, "session-1")
	}
	if payload.SessionEnded {
		t.Fatal("session_ended should be false for normal response")
	}
	if payload.SessionEnd != nil {
		t.Fatalf("session_end should be nil for normal response: %+v", payload.SessionEnd)
	}
}

func TestValidateAgentResponsePayloadRejectsInconsistentSessionEnd(t *testing.T) {
	err := validateAgentResponsePayload(agentResponse{
		Message:      "final answer",
		SessionID:    "session-1",
		SessionEnded: true,
		SessionEnd: &assistantSessionEndSignalPayload{
			Signal:  busAssistantSessionEndSignal,
			Message: "different",
		},
	})
	if err == nil {
		t.Fatal("expected error but got nil")
	}
}

func TestValidateAgentResponsePayloadAcceptsPlanModeWithoutIterationFields(t *testing.T) {
	err := validateAgentResponsePayload(agentResponse{
		Message:      "plan output",
		SessionID:    "session-1",
		SessionEnded: false,
		Mode:         agentModePlan,
	})
	if err != nil {
		t.Fatalf("validateAgentResponsePayload returned error: %v", err)
	}
}

func TestPrepareAgentTurnRequestAcceptsImageOnlyInput(t *testing.T) {
	prepared, err := prepareAgentTurnRequest(agentParams{
		Images: []sessionImageContent{{
			URL:      "data:image/png;base64,ZmFrZS1pbWFnZQ==",
			MimeType: "image/png",
		}},
	})
	if err != nil {
		t.Fatalf("prepareAgentTurnRequest returned error: %v", err)
	}
	if prepared.UserInput.Text != "" || len(prepared.UserInput.Content) != 1 || prepared.UserInput.Content[0].Image == nil {
		t.Fatalf("unexpected prepared input: %+v", prepared.UserInput)
	}
}

func TestPrepareAgentTurnRequestRejectsImageWithoutSource(t *testing.T) {
	_, err := prepareAgentTurnRequest(agentParams{
		Images: []sessionImageContent{{MimeType: "image/png"}},
	})
	if err == nil {
		t.Fatal("expected error but got nil")
	}
	if err.Error() != "images[0] requires path or url" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPrepareAgentTurnRequestNormalizesPlanMode(t *testing.T) {
	prepared, err := prepareAgentTurnRequest(agentParams{
		Mode:    " PLAN ",
		Message: "plan this task",
	})
	if err != nil {
		t.Fatalf("prepareAgentTurnRequest returned error: %v", err)
	}
	if prepared.Mode != agentModePlan {
		t.Fatalf("unexpected mode: got %q want %q", prepared.Mode, agentModePlan)
	}
}

func TestPrepareAgentTurnRequestRejectsUnknownMode(t *testing.T) {
	_, err := prepareAgentTurnRequest(agentParams{
		Mode:    "execute",
		Message: "hello",
	})
	if err == nil {
		t.Fatal("expected error but got nil")
	}
	if err.Error() != `unsupported agent mode: "execute"` {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMessagesWithSystemPromptReplacesFirstSystemMessage(t *testing.T) {
	input := []llm.Message{
		{
			Role: llm.RoleSystem,
			Text: "old prompt",
		},
		{
			Role: llm.RoleUser,
			Text: "hello",
		},
	}

	got := messagesWithSystemPrompt(input, "new prompt")

	if len(got) != 2 {
		t.Fatalf("unexpected message count: got %d want %d", len(got), 2)
	}
	if got[0].Role != llm.RoleSystem || got[0].Text != "new prompt" {
		t.Fatalf("unexpected first message: %+v", got[0])
	}
	if got[1].Role != llm.RoleUser || got[1].Text != "hello" {
		t.Fatalf("unexpected second message: %+v", got[1])
	}
	if input[0].Text != "old prompt" {
		t.Fatalf("input should stay unchanged: got %q want %q", input[0].Text, "old prompt")
	}
}

func TestMessagesWithSystemPromptPrependsWhenMissing(t *testing.T) {
	input := []llm.Message{
		{
			Role: llm.RoleUser,
			Text: "hello",
		},
	}

	got := messagesWithSystemPrompt(input, "system prompt")

	if len(got) != 2 {
		t.Fatalf("unexpected message count: got %d want %d", len(got), 2)
	}
	if got[0].Role != llm.RoleSystem || got[0].Text != "system prompt" {
		t.Fatalf("unexpected first message: %+v", got[0])
	}
	if got[1].Role != llm.RoleUser || got[1].Text != "hello" {
		t.Fatalf("unexpected second message: %+v", got[1])
	}
	if len(input) != 1 {
		t.Fatalf("input should stay unchanged: got len=%d want %d", len(input), 1)
	}
}

type testRuntimeFactory struct {
	deps agentRuntimeDependencies
	err  error
}

type proTestRuntimeFactory = testRuntimeFactory

func (f testRuntimeFactory) Build(bridgeconfig.Store) (agentRuntimeDependencies, error) {
	if f.err != nil {
		return agentRuntimeDependencies{}, f.err
	}
	if strings.TrimSpace(f.deps.cfg.PromptsDir) == "" {
		f.deps.cfg.PromptsDir = os.Getenv("GHOST_PROMPTS_DIR")
	}
	return f.deps, nil
}

type testCompleter struct {
	responses []*llm.CompletionResponse
	requests  []llm.CompletionRequest
}

type proTestCompleter = testCompleter

func (f *testCompleter) Complete(_ context.Context, request llm.CompletionRequest) (*llm.CompletionResponse, error) {
	f.requests = append(f.requests, request)
	if len(f.responses) == 0 {
		return nil, errors.New("unexpected complete call")
	}
	response := f.responses[0]
	f.responses = f.responses[1:]
	return response, nil
}

func TestExecuteAgentActionTreatsProPrefixAsStandardMessage(t *testing.T) {
	_, service, sessionStore := newTestHandlerWithService(t, nil, nil)

	completer := &testCompleter{
		responses: []*llm.CompletionResponse{
			{
				Message:      llm.Message{Role: llm.RoleAssistant, Text: "standard response"},
				FinishReason: llm.FinishStop,
			},
		},
	}
	service.runtimeFactory = testRuntimeFactory{
		deps: agentRuntimeDependencies{
			cfg: bridgeconfig.Config{
				MaxTurns:         4,
				ProMaxIterations: 1,
				Provider:         bridgeconfig.ProviderConfig{Model: "gpt-4o"},
			},
			client:       completer,
			registry:     tools.NewRegistry(),
			systemPrompt: "system prompt",
		},
	}
	service.agentRunner = NewSessionAgentRunner(
		service.runtimeFactory,
		service.configStore,
		sessionStore,
		service.runRegistry,
	)

	payloadResult, err := service.executeAgentAction(context.Background(), agentParams{Message: "pro 1 fix config"}, "trace-pro-limit")
	if err != nil {
		t.Fatalf("executeAgentAction returned error: %v", err)
	}
	if payloadResult.Outcome != ServiceOutcomeSuccess {
		t.Fatalf("unexpected outcome: %s", payloadResult.Outcome)
	}
	payload := payloadResult.Payload.(agentResponse)
	if payload.Mode != "" || payload.StoppedBy != "" || payload.IterationCount != 0 {
		t.Fatalf("expected standard response without pro fields, got %+v", payload)
	}
	if payload.Message != "standard response" {
		t.Fatalf("unexpected message: %q", payload.Message)
	}
	if len(completer.requests) != 1 {
		t.Fatalf("unexpected complete request count: %d", len(completer.requests))
	}
	last := completer.requests[0].Messages[len(completer.requests[0].Messages)-1]
	if last.Text != "pro 1 fix config" {
		t.Fatalf("unexpected user message: %q", last.Text)
	}
}

func TestExecuteAgentActionRunsPlanModeWithExplicitModePriority(t *testing.T) {
	_, service, _ := newTestHandlerWithService(t, nil, nil)
	completer := &proTestCompleter{
		responses: []*llm.CompletionResponse{
			{
				Message: llm.Message{
					Role: llm.RoleAssistant,
					Text: "【用户意图】\n- 拆解任务\n【任务编排】\n1. task_id=T1; objective=分析需求; inputs=用户消息; depends_on=none; executor=main_ai\n【执行顺序】\n1. 先分析再执行\n【完成判定】\n1. 任务清单完整",
				},
				FinishReason: llm.FinishStop,
			},
		},
	}
	service.runtimeFactory = proTestRuntimeFactory{
		deps: agentRuntimeDependencies{
			cfg: bridgeconfig.Config{
				MaxTurns: 4,
				Provider: bridgeconfig.ProviderConfig{Model: "gpt-4o"},
			},
			client:       completer,
			registry:     tools.NewRegistry(),
			systemPrompt: "system prompt",
		},
	}

	payloadAny, err := service.executeAgentAction(context.Background(), agentParams{
		Mode:    agentModePlan,
		Message: "pro fix config",
	}, "trace-plan")
	if err != nil {
		t.Fatalf("executeAgentAction returned error: %v", err)
	}
	if payloadAny.Outcome != ServiceOutcomeSuccess {
		t.Fatalf("unexpected outcome: %s", payloadAny.Outcome)
	}

	payload, ok := payloadAny.Payload.(agentResponse)
	if !ok {
		t.Fatalf("unexpected payload type: %T", payloadAny)
	}
	if payload.Mode != agentModePlan {
		t.Fatalf("unexpected mode: got %q want %q", payload.Mode, agentModePlan)
	}
	if payload.IterationCount != 0 || payload.StoppedBy != "" {
		t.Fatalf("plan mode should not carry pro fields: %+v", payload)
	}
	if len(completer.requests) != 1 {
		t.Fatalf("unexpected complete request count: %d", len(completer.requests))
	}
	req := completer.requests[0]
	if len(req.Tools) != 0 {
		t.Fatalf("plan mode must not expose tools, got %+v", req.Tools)
	}
	if got := req.Messages[len(req.Messages)-1].Text; got != "pro fix config" {
		t.Fatalf("unexpected user message: got %q want %q", got, "pro fix config")
	}
	if !strings.Contains(req.Messages[0].Text, "planning orchestrator") {
		t.Fatalf("unexpected system prompt: %q", req.Messages[0].Text)
	}
}

func TestExecuteAgentActionRejectsPlanModeToolCalls(t *testing.T) {
	_, service, _ := newTestHandlerWithService(t, nil, nil)
	completer := &proTestCompleter{
		responses: []*llm.CompletionResponse{
			{
				Message: llm.Message{
					Role: llm.RoleAssistant,
					ToolCalls: []llm.ToolCall{{
						ID:        "call-1",
						Name:      "script_exec",
						Arguments: []byte(`{"script":"echo hi"}`),
					}},
				},
				FinishReason: llm.FinishToolCalls,
			},
		},
	}
	service.runtimeFactory = proTestRuntimeFactory{
		deps: agentRuntimeDependencies{
			cfg: bridgeconfig.Config{
				MaxTurns: 4,
				Provider: bridgeconfig.ProviderConfig{Model: "gpt-4o"},
			},
			client:       completer,
			registry:     tools.NewRegistry(),
			systemPrompt: "system prompt",
		},
	}

	_, err := service.executeAgentAction(context.Background(), agentParams{
		Mode:    agentModePlan,
		Message: "make a plan",
	}, "trace-plan-toolcall")
	if err == nil {
		t.Fatal("expected error but got nil")
	}
	if code := legacyStatusFromServiceError(err); code != 500 {
		t.Fatalf("unexpected status code: got %d want %d", code, 500)
	}
	if !strings.Contains(err.Error(), "cannot contain tool calls") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRequestProjectRootOverrideAppliesToStandardTurn(t *testing.T) {
	projectRoot := t.TempDir()
	factory := &projectRootCaptureFactory{
		client: &proTestCompleter{
			responses: []*llm.CompletionResponse{{
				Message:      llm.Message{Role: llm.RoleAssistant, Text: "ok"},
				FinishReason: llm.FinishStop,
			}},
		},
	}
	service := newRequestRuntimeTestService(t, factory)

	result, err := service.executeAgentAction(context.Background(), agentParams{
		Message:     "hello",
		ProjectRoot: projectRoot,
	}, "trace-standard-project-root")
	if err != nil {
		t.Fatalf("executeAgentAction: %v", err)
	}
	if result.Outcome != ServiceOutcomeSuccess {
		t.Fatalf("unexpected outcome: %s", result.Outcome)
	}
	assertCapturedProjectRoot(t, factory, projectRoot)
}

func TestRequestProjectRootOverrideAppliesToPlanMode(t *testing.T) {
	projectRoot := t.TempDir()
	factory := &projectRootCaptureFactory{
		client: &proTestCompleter{
			responses: []*llm.CompletionResponse{{
				Message: llm.Message{
					Role: llm.RoleAssistant,
					Text: "【任务编排】\n1. task_id=T1; objective=分析请求; inputs=用户消息; depends_on=none; executor=main_ai",
				},
				FinishReason: llm.FinishStop,
			}},
		},
	}
	service := newRequestRuntimeTestService(t, factory)

	result, err := service.executeAgentAction(context.Background(), agentParams{
		Mode:        "plan",
		Message:     "plan this task",
		ProjectRoot: projectRoot,
	}, "trace-plan-project-root")
	if err != nil {
		t.Fatalf("executeAgentAction: %v", err)
	}
	if result.Outcome != ServiceOutcomeSuccess {
		t.Fatalf("unexpected outcome: %s", result.Outcome)
	}
	assertCapturedProjectRoot(t, factory, projectRoot)
}

func TestRequestProjectRootOverrideAppliesToProPrefixedStandardTurn(t *testing.T) {
	projectRoot := t.TempDir()
	factory := &projectRootCaptureFactory{
		client: &proTestCompleter{
			responses: []*llm.CompletionResponse{{
				Message:      llm.Message{Role: llm.RoleAssistant, Text: "ok"},
				FinishReason: llm.FinishStop,
			}},
		},
	}
	service := newRequestRuntimeTestService(t, factory)

	result, err := service.executeAgentAction(context.Background(), agentParams{
		Message:     "pro fix config",
		ProjectRoot: projectRoot,
	}, "trace-pro-project-root")
	if err != nil {
		t.Fatalf("executeAgentAction: %v", err)
	}
	if result.Outcome != ServiceOutcomeSuccess {
		t.Fatalf("unexpected outcome: %s", result.Outcome)
	}
	assertCapturedProjectRoot(t, factory, projectRoot)
}

type projectRootCaptureFactory struct {
	client       agent.Completer
	projectRoots []string
}

func (f *projectRootCaptureFactory) Build(store bridgeconfig.Store) (agentRuntimeDependencies, error) {
	cfg, err := store.Config()
	if err != nil {
		return agentRuntimeDependencies{}, err
	}
	cfg.MaxTurns = 4
	cfg.ProMaxIterations = 2
	f.projectRoots = append(f.projectRoots, cfg.ProjectRoot)
	return agentRuntimeDependencies{
		cfg:          cfg,
		client:       f.client,
		registry:     tools.NewRegistry(),
		systemPrompt: "system prompt",
	}, nil
}

func newRequestRuntimeTestService(t *testing.T, factory AgentRuntimeFactory) *bridgeService {
	t.Helper()

	tempDir := t.TempDir()
	t.Setenv("GHOST_CONFIG_PATH", filepath.Join(tempDir, "config.toml"))
	t.Setenv("GHOST_PROVIDER", "custom")
	t.Setenv("GHOST_BASE_URL", "https://initial.example/v1")
	t.Setenv("GHOST_API_KEY", "test-key")
	t.Setenv("GHOST_PROMPTS_DIR", filepath.Join(tempDir, "prompts"))

	store, err := bridgeconfig.NewStoreFromEnv()
	if err != nil {
		t.Fatalf("NewStoreFromEnv: %v", err)
	}
	sessionStore, err := session.NewStore(filepath.Join(tempDir, "sessions"))
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	service := newBridgeServiceWithStreamExecutor(store, sessionStore, nil, nil)
	service.runtimeFactory = factory
	if runner, ok := service.agentRunner.(*SessionAgentRunner); ok {
		runner.runtimeFactory = factory
	}
	t.Cleanup(service.Close)
	return service
}

func assertCapturedProjectRoot(t *testing.T, factory *projectRootCaptureFactory, want string) {
	t.Helper()

	if len(factory.projectRoots) != 1 {
		t.Fatalf("expected one captured project_root, got %#v", factory.projectRoots)
	}
	if factory.projectRoots[0] != want {
		t.Fatalf("unexpected captured project_root: got %q want %q", factory.projectRoots[0], want)
	}
}

func TestSchemaActionEnumMatchesRegisteredBusActions(t *testing.T) {
	schemaActions := loadSchemaActionEnum(t)
	runtimeActions := registeredBusActions()

	missingInRuntime := actionDiff(schemaActions, runtimeActions)
	if len(missingInRuntime) != 0 {
		t.Fatalf("schema declares unsupported bus actions: %v", missingInRuntime)
	}

	missingInSchema := actionDiff(runtimeActions, schemaActions)
	if len(missingInSchema) != 0 {
		t.Fatalf("runtime bus actions missing from schema enum: %v", missingInSchema)
	}
}

func loadSchemaActionEnum(t *testing.T) map[string]struct{} {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve caller path")
	}
	path := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "shared", "schema", "defs", "base.json"))

	type actionSchema struct {
		Defs struct {
			Action struct {
				Enum []string `json:"enum"`
			} `json:"action"`
		} `json:"$defs"`
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read base schema: %v", err)
	}

	var parsed actionSchema
	if err := json.Unmarshal(raw, &parsed); err != nil {
		t.Fatalf("decode base schema: %v", err)
	}

	actions := make(map[string]struct{}, len(parsed.Defs.Action.Enum))
	for _, action := range parsed.Defs.Action.Enum {
		normalized := strings.TrimSpace(action)
		if normalized != "" {
			actions[normalized] = struct{}{}
		}
	}
	return actions
}

func registeredBusActions() map[string]struct{} {
	service := newBridgeServiceState(nil, nil)
	registerDefaultActions(service)

	names := service.registeredActionNames()
	actions := make(map[string]struct{}, len(names))
	for _, action := range names {
		actions[action] = struct{}{}
	}
	return actions
}

func actionDiff(left map[string]struct{}, right map[string]struct{}) []string {
	missing := make([]string, 0, len(left))
	for action := range left {
		if _, ok := right[action]; !ok {
			missing = append(missing, action)
		}
	}
	sort.Strings(missing)
	return missing
}

func TestConfigUpdatePropagatesRuntimeFlagsThroughOrchestration(t *testing.T) {
	_, service, _ := newTestHandlerWithService(t, nil, nil)

	raw := json.RawMessage(`{
		"max_turns": 9,
		"task_execution_timeout_ms": 600000,
		"llm_completion_retry_count": 0,
		"llm_completion_retry_interval_ms": 150,
		"session_human_log_full_enabled": true,
		"session_system_prompt_visible_enabled": false,
		"assistant_markdown_enabled": false,
		"tool_call_compact_output_enabled": true,
		"memory_mode_enabled": true,
		"microcompact_enabled": true,
		"session_title_mode": "first_message"
	}`)

	result, err := service.dispatchAction(
		context.Background(),
		busActionConfigUpdate,
		raw,
		"trace-web-rooter-update",
	)
	if err != nil {
		t.Fatalf("config update failed: %v", err)
	}
	if result.Outcome != ServiceOutcomeSuccess {
		t.Fatalf("unexpected outcome: %s", result.Outcome)
	}

	snapshot, ok := result.Payload.(configResponse)
	if !ok {
		t.Fatalf("unexpected payload type: %T", result.Payload)
	}
	if !snapshot.SessionHumanLogFullEnabled {
		t.Fatal("expected session_human_log_full_enabled in snapshot")
	}
	if snapshot.MaxTurns != 9 {
		t.Fatalf("expected max_turns to be 9 in snapshot, got %d", snapshot.MaxTurns)
	}
	if snapshot.TaskExecutionTimeoutMs != 600000 {
		t.Fatalf(
			"expected task_execution_timeout_ms to be 600000 in snapshot, got %d",
			snapshot.TaskExecutionTimeoutMs,
		)
	}
	if snapshot.LlmCompletionRetryCount != 0 {
		t.Fatalf(
			"expected llm_completion_retry_count to be 0 in snapshot, got %d",
			snapshot.LlmCompletionRetryCount,
		)
	}
	if snapshot.LlmCompletionRetryIntervalMs != 150 {
		t.Fatalf(
			"expected llm_completion_retry_interval_ms to be 150 in snapshot, got %d",
			snapshot.LlmCompletionRetryIntervalMs,
		)
	}
	if snapshot.SessionSystemPromptVisibleEnabled {
		t.Fatal("expected session_system_prompt_visible_enabled to be false in snapshot")
	}
	if snapshot.AssistantMarkdownEnabled {
		t.Fatal("expected assistant_markdown_enabled to be false in snapshot")
	}
	if !snapshot.ToolCallCompactOutputEnabled {
		t.Fatal("expected tool_call_compact_output_enabled to be true in snapshot")
	}
	if !snapshot.MemoryModeEnabled {
		t.Fatal("expected memory_mode_enabled to be true in snapshot")
	}
	if !snapshot.MicrocompactEnabled {
		t.Fatal("expected microcompact_enabled to be true in snapshot")
	}
	if snapshot.SessionTitleMode != "first_message" {
		t.Fatalf("expected session_title_mode in snapshot, got %q", snapshot.SessionTitleMode)
	}

	cfg, err := service.configStore.Config()
	if err != nil {
		t.Fatalf("load runtime config: %v", err)
	}
	if !cfg.SessionHumanLogFullEnabled {
		t.Fatal("expected session_human_log_full_enabled in runtime config")
	}
	if cfg.MaxTurns != 9 {
		t.Fatalf("expected max_turns to be 9 in runtime config, got %d", cfg.MaxTurns)
	}
	if cfg.TaskExecutionTimeoutMS != 600000 {
		t.Fatalf(
			"expected task_execution_timeout_ms to be 600000 in runtime config, got %d",
			cfg.TaskExecutionTimeoutMS,
		)
	}
	if cfg.LLMCompletionRetryCount != 0 {
		t.Fatalf(
			"expected llm_completion_retry_count to be 0 in runtime config, got %d",
			cfg.LLMCompletionRetryCount,
		)
	}
	if cfg.LLMCompletionRetryIntervalMS != 150 {
		t.Fatalf(
			"expected llm_completion_retry_interval_ms to be 150 in runtime config, got %d",
			cfg.LLMCompletionRetryIntervalMS,
		)
	}
	if cfg.SessionSystemPromptVisible {
		t.Fatal("expected session_system_prompt_visible_enabled to be false in runtime config")
	}
	if cfg.AssistantMarkdownEnabled {
		t.Fatal("expected assistant_markdown_enabled to be false in runtime config")
	}
	if !cfg.ToolCallCompactOutputEnabled {
		t.Fatal("expected tool_call_compact_output_enabled to be true in runtime config")
	}
	if !cfg.MemoryModeEnabled {
		t.Fatal("expected memory_mode_enabled to be true in runtime config")
	}
	if !cfg.MicrocompactEnabled {
		t.Fatal("expected microcompact_enabled to be true in runtime config")
	}
	if cfg.SessionTitleMode != "first_message" {
		t.Fatalf("expected session_title_mode in runtime config, got %q", cfg.SessionTitleMode)
	}
	scheduler := service.taskScheduler()
	if scheduler == nil {
		t.Fatal("expected task scheduler to be initialized")
	}
	if got := scheduler.ExecutionTimeout(); got != 10*time.Minute {
		t.Fatalf("expected scheduler timeout to be 10m, got %s", got)
	}
}

func TestEnsureSessionActiveRequiresSessionStoreForExistingSession(t *testing.T) {
	service := &bridgeService{}

	err := service.ensureSessionActive("session-1")
	if err == nil {
		t.Fatal("expected error when session store is missing")
	}
	if kind := ServiceErrorKindOf(err); kind != ServiceErrorInternal {
		t.Fatalf("unexpected error kind: got=%s want=%s", kind, ServiceErrorInternal)
	}
	if code := legacyStatusFromServiceError(err); code != http.StatusInternalServerError {
		t.Fatalf("unexpected status code: got=%d want=%d", code, http.StatusInternalServerError)
	}
	if !strings.Contains(err.Error(), "session store is not configured") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestEnsureSessionActiveSkipsValidationWhenSessionIDEmpty(t *testing.T) {
	service := &bridgeService{}

	err := service.ensureSessionActive("   ")
	if err != nil {
		t.Fatalf("expected nil error for empty session id, got: %v", err)
	}
}

func TestEnsureSessionNotInflightRequiresRunRegistryForExistingSession(t *testing.T) {
	service := &bridgeService{}

	err := service.ensureSessionNotInflight("session-1")
	if err == nil {
		t.Fatal("expected error when run registry is missing")
	}
	if kind := ServiceErrorKindOf(err); kind != ServiceErrorInternal {
		t.Fatalf("unexpected error kind: got=%s want=%s", kind, ServiceErrorInternal)
	}
	if code := legacyStatusFromServiceError(err); code != http.StatusInternalServerError {
		t.Fatalf("unexpected status code: got=%d want=%d", code, http.StatusInternalServerError)
	}
	if !strings.Contains(err.Error(), "run registry is not configured") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestEnsureSessionNotInflightSkipsValidationWhenSessionIDEmpty(t *testing.T) {
	service := &bridgeService{}

	err := service.ensureSessionNotInflight("")
	if err != nil {
		t.Fatalf("expected nil error for empty session id, got: %v", err)
	}
}

func newTestHandlerWithService(t *testing.T, executor agentExecutorFunc, streamExecutor agentStreamExecutorFunc) (http.Handler, *bridgeService, *session.Store) {
	t.Helper()

	tempDir := t.TempDir()
	t.Setenv("GHOST_CONFIG_PATH", tempDir+"/config.toml")
	t.Setenv("GHOST_API_KEY", "test-key")
	t.Setenv("GHOST_TASKS_PATH", tempDir+"/tasks")
	t.Setenv("GHOST_PROMPTS_DIR", tempDir+"/prompts")
	t.Setenv("GHOST_RSS_POLL_ENABLED", "false")
	t.Setenv("GHOST_RSS_BRIEFING_ENABLED", "false")
	t.Setenv("GHOST_RSS_INBOX_PATH", tempDir+"/rss/inbox.json")
	t.Setenv("GHOST_RSS_FEEDS_PATH", tempDir+"/rss/feeds.json")

	if executor == nil {
		executor = func(_ context.Context, _ string, _ string, _ string, _ bridgeconfig.Store, _ *session.Store) (string, string, error) {
			return "ok", "session-test", nil
		}
	}

	store, err := bridgeconfig.NewStoreFromEnv()
	if err != nil {
		t.Fatalf("new config store: %v", err)
	}

	sessionStore, err := session.NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("new session store: %v", err)
	}

	service := newBridgeServiceWithStreamExecutor(store, sessionStore, executor, streamExecutor)
	if err := service.StartBackgroundRuntimes(); err != nil {
		t.Fatalf("start background runtimes: %v", err)
	}
	t.Cleanup(service.Close)
	return nil, service, sessionStore
}

func newTempSessionStore(t *testing.T) *session.Store {
	t.Helper()

	t.Setenv("GHOST_PROMPTS_DIR", filepath.Join(t.TempDir(), "prompts"))
	store, err := session.NewStore(filepath.Join(t.TempDir(), "sessions"))
	if err != nil {
		t.Fatalf("new session store: %v", err)
	}
	return store
}

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

const preCreateForcedCompletionError = "forced completion failure"

type preCreateAssertCompleter struct {
	t            *testing.T
	sessionStore *session.Store
	systemPrompt string
	calls        int
}

func (c *preCreateAssertCompleter) Complete(
	_ context.Context,
	_ llm.CompletionRequest,
) (*llm.CompletionResponse, error) {
	c.calls++
	c.assertSessionExistsBeforeLoop()
	return nil, errors.New(preCreateForcedCompletionError)
}

func (c *preCreateAssertCompleter) assertSessionExistsBeforeLoop() {
	c.t.Helper()

	metadata, err := c.sessionStore.ListMetadata()
	if err != nil {
		c.t.Fatalf("list metadata before completion: %v", err)
	}
	if len(metadata) != 1 {
		c.t.Fatalf("expected pre-created session before completion, got %d", len(metadata))
	}

	sess, err := c.sessionStore.Load(metadata[0].ID)
	if err != nil {
		c.t.Fatalf("load pre-created session before completion: %v", err)
	}
	if sess.MessageCount != 1 || len(sess.Messages) != 1 {
		c.t.Fatalf("expected pre-created session shell with only system prompt, got %+v", sess.Messages)
	}
	if sess.Messages[0].Role != llm.RoleSystem || sess.Messages[0].Text != c.systemPrompt {
		c.t.Fatalf("unexpected pre-created system prompt: %+v", sess.Messages[0])
	}
}

func TestSessionAgentRunnerPreCreatesSessionBeforeLoopRunTurn(t *testing.T) {
	sessionStore := newTempSessionStore(t)
	const systemPrompt = "base system prompt"
	completer := &preCreateAssertCompleter{
		t:            t,
		sessionStore: sessionStore,
		systemPrompt: systemPrompt,
	}
	runner := newPreCreateSessionRunner(completer, sessionStore, systemPrompt)

	_, sessionID, err := runner.RunTurn(context.Background(), "hello precreate", "", "trace-precreate")
	if err == nil || !strings.Contains(err.Error(), preCreateForcedCompletionError) {
		t.Fatalf("expected forced completion error, got: %v", err)
	}
	if completer.calls != 1 {
		t.Fatalf("expected one completion call, got %d", completer.calls)
	}
	assertPreCreatedSessionShellAfterFailure(t, sessionStore, sessionID, systemPrompt)
}

func TestSessionAgentRunnerPreCreatesSessionBeforeLoopRunTurnStream(t *testing.T) {
	sessionStore := newTempSessionStore(t)
	const systemPrompt = "base system prompt"
	completer := &preCreateAssertCompleter{
		t:            t,
		sessionStore: sessionStore,
		systemPrompt: systemPrompt,
	}
	runner := newPreCreateSessionRunner(completer, sessionStore, systemPrompt)

	_, sessionID, err := runner.RunTurnStream(
		context.Background(),
		"hello precreate stream",
		"",
		"trace-precreate-stream",
		streaming.NopSink{},
	)
	if err == nil || !strings.Contains(err.Error(), preCreateForcedCompletionError) {
		t.Fatalf("expected forced completion error, got: %v", err)
	}
	if completer.calls != 1 {
		t.Fatalf("expected one completion call, got %d", completer.calls)
	}
	assertPreCreatedSessionShellAfterFailure(t, sessionStore, sessionID, systemPrompt)
}

func TestSessionAgentRunnerRejectsResumeLikeTurnWithoutPendingHumanAnswer(t *testing.T) {
	sessionStore := newTempSessionStore(t)
	sess := session.NewSession("base system prompt")
	if err := sessionStore.Save(sess); err != nil {
		t.Fatalf("save session: %v", err)
	}

	runner := newPreCreateSessionRunner(&preCreateAssertCompleter{
		t:            t,
		sessionStore: sessionStore,
		systemPrompt: "base system prompt",
	}, sessionStore, "base system prompt")

	_, _, err := runner.RunTurn(context.Background(), "   ", sess.ID, "trace-empty-resume")
	if err == nil || !strings.Contains(err.Error(), "resume requires pending human answers") {
		t.Fatalf("expected resume guard error, got %v", err)
	}
}

func TestSessionAgentRunnerRunTurnInputPersistsUserImages(t *testing.T) {
	sessionStore := newTempSessionStore(t)
	completer := &proTestCompleter{
		responses: []*llm.CompletionResponse{{
			Message:      llm.Message{Role: llm.RoleAssistant, Text: "done"},
			FinishReason: llm.FinishStop,
		}},
	}
	runner := NewSessionAgentRunner(proTestRuntimeFactory{
		deps: agentRuntimeDependencies{
			cfg:          bridgeconfig.Config{MaxTurns: 3, PromptsPath: "", Provider: bridgeconfig.ProviderConfig{Type: llm.ProviderOpenAI, Model: "gpt-4o"}},
			client:       completer,
			registry:     tools.NewRegistry(),
			systemPrompt: "base system prompt",
		},
	}, nil, sessionStore, nil)

	input := llm.Message{Role: llm.RoleUser, Text: "describe this", Content: []llm.ContentPart{{Type: llm.ContentTypeImage, Image: &llm.ImageContent{URL: "data:image/png;base64,ZmFrZS1pbWFnZQ==", MimeType: "image/png"}}}}
	if _, sessionID, err := runner.RunTurnInput(context.Background(), input, "", "trace-image-input"); err != nil {
		t.Fatalf("run turn input: %v", err)
	} else if len(completer.requests) != 1 || len(completer.requests[0].Messages) < 2 {
		t.Fatalf("unexpected completer requests: %+v", completer.requests)
	} else if got := completer.requests[0].Messages[1]; got.Role != llm.RoleUser || len(got.Content) != 1 || got.Content[0].Image == nil {
		t.Fatalf("expected user image content in model request, got %+v", got)
	} else if loaded, err := sessionStore.Load(sessionID); err != nil {
		t.Fatalf("load session: %v", err)
	} else if len(loaded.Messages) < 2 || len(loaded.Messages[1].Content) != 1 || loaded.Messages[1].Content[0].Image == nil {
		t.Fatalf("expected persisted user image content, got %+v", loaded.Messages)
	} else if loaded.Messages[1].Content[0].Image.URL != "data:image/png;base64,ZmFrZS1pbWFnZQ==" {
		t.Fatalf("unexpected persisted image url: %q", loaded.Messages[1].Content[0].Image.URL)
	}
}

func newPreCreateSessionRunner(
	completer llm.Completer,
	sessionStore *session.Store,
	systemPrompt string,
) *SessionAgentRunner {
	return NewSessionAgentRunner(proTestRuntimeFactory{
		deps: agentRuntimeDependencies{
			cfg: bridgeconfig.Config{
				MaxTurns:    3,
				PromptsPath: "",
				Provider:    bridgeconfig.ProviderConfig{Type: llm.ProviderOpenAI, Model: "gpt-4o"},
			},
			client:       completer,
			registry:     tools.NewRegistry(),
			systemPrompt: systemPrompt,
		},
	}, nil, sessionStore, nil)
}

func assertPreCreatedSessionShellAfterFailure(
	t *testing.T,
	sessionStore *session.Store,
	sessionID string,
	systemPrompt string,
) {
	t.Helper()

	resolvedSessionID := strings.TrimSpace(sessionID)
	if resolvedSessionID == "" {
		metadata, err := sessionStore.ListMetadata()
		if err != nil {
			t.Fatalf("list metadata after failure: %v", err)
		}
		if len(metadata) != 1 {
			t.Fatalf("expected one persisted session after failure, got %d", len(metadata))
		}
		resolvedSessionID = strings.TrimSpace(metadata[0].ID)
	}
	loaded, err := sessionStore.Load(resolvedSessionID)
	if err != nil {
		t.Fatalf("load persisted session: %v", err)
	}
	if loaded.MessageCount != 1 || len(loaded.Messages) != 1 {
		t.Fatalf("expected only pre-created session shell after failure, got %+v", loaded.Messages)
	}
	if loaded.Messages[0].Role != llm.RoleSystem || loaded.Messages[0].Text != systemPrompt {
		t.Fatalf("unexpected persisted system prompt: %+v", loaded.Messages[0])
	}
}

type draftStreamingCompleter struct {
	deltas   []llm.LLMDelta
	response *llm.CompletionResponse
	runErr   error
}

func (c *draftStreamingCompleter) Complete(_ context.Context, _ llm.CompletionRequest) (*llm.CompletionResponse, error) {
	if c.runErr != nil {
		return nil, c.runErr
	}
	if c.response == nil {
		return nil, errors.New("response is nil")
	}
	return c.response, nil
}

func (c *draftStreamingCompleter) CompleteStream(
	ctx context.Context,
	_ llm.CompletionRequest,
	sink llm.LLMStreamSink,
) (*llm.CompletionResponse, error) {
	for _, delta := range c.deltas {
		if err := sink.OnDelta(ctx, delta); err != nil {
			return nil, err
		}
	}
	if c.runErr != nil {
		return nil, c.runErr
	}
	if c.response == nil {
		return nil, errors.New("response is nil")
	}
	return c.response, nil
}

func TestRunTurnStreamInputPersistsAssistantDraftOnError(t *testing.T) {
	sessionStore := newTempSessionStore(t)
	sess := newPersistedSessionForDraftTests(t, sessionStore, "session-stream-draft")

	completer := &draftStreamingCompleter{
		deltas: []llm.LLMDelta{
			{Kind: llm.DeltaKindText, Text: "partial "},
			{Kind: llm.DeltaKindText, Text: "answer"},
		},
		runErr: errors.New("stream interrupted"),
	}
	runner := newDraftTestRunner(sessionStore, completer)

	_, returnedSessionID, err := runner.RunTurnStreamInput(
		context.Background(),
		llm.Message{Role: llm.RoleUser, Text: "continue"},
		sess.ID,
		"trace-stream-draft",
		streaming.NopSink{},
	)
	if err == nil {
		t.Fatal("expected stream run to fail")
	}
	if returnedSessionID != sess.ID {
		t.Fatalf("expected persisted session id on failed turn, got %q want %q", returnedSessionID, sess.ID)
	}

	loaded, loadErr := sessionStore.Load(sess.ID)
	if loadErr != nil {
		t.Fatalf("load session: %v", loadErr)
	}
	if len(loaded.Messages) != 2 {
		t.Fatalf("expected system + user messages, got %+v", loaded.Messages)
	}
	if loaded.Messages[1].Role != llm.RoleUser || loaded.Messages[1].Text != "continue" {
		t.Fatalf("unexpected persisted user message: %+v", loaded.Messages[1])
	}
	if loaded.AssistantDraft == nil {
		t.Fatal("expected assistant draft to persist on stream error")
	}
	if loaded.AssistantDraft.Text != "partial answer" {
		t.Fatalf("unexpected assistant draft: %q", loaded.AssistantDraft.Text)
	}
	if loaded.TurnDraft != nil {
		t.Fatalf("expected turn_draft to be cleared after error, got %+v", loaded.TurnDraft)
	}
}

func TestRunTurnStreamInputPersistsAssistantDraftOnCancel(t *testing.T) {
	sessionStore := newTempSessionStore(t)
	sess := newPersistedSessionForDraftTests(t, sessionStore, "session-stream-cancel")

	completer := &draftStreamingCompleter{
		deltas: []llm.LLMDelta{
			{Kind: llm.DeltaKindText, Text: "partial "},
			{Kind: llm.DeltaKindText, Text: "answer"},
		},
		runErr: context.Canceled,
	}
	runner := newDraftTestRunner(sessionStore, completer)

	_, returnedSessionID, err := runner.RunTurnStreamInput(
		context.Background(),
		llm.Message{Role: llm.RoleUser, Text: "continue"},
		sess.ID,
		"trace-stream-cancel",
		streaming.NopSink{},
	)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation, got %v", err)
	}
	if returnedSessionID != sess.ID {
		t.Fatalf("expected persisted session id on cancellation, got %q want %q", returnedSessionID, sess.ID)
	}

	loaded, loadErr := sessionStore.Load(sess.ID)
	if loadErr != nil {
		t.Fatalf("load session: %v", loadErr)
	}
	if len(loaded.Messages) != 2 {
		t.Fatalf("expected system + user messages, got %+v", loaded.Messages)
	}
	if loaded.Messages[1].Role != llm.RoleUser || loaded.Messages[1].Text != "continue" {
		t.Fatalf("unexpected persisted user message: %+v", loaded.Messages[1])
	}
	if loaded.AssistantDraft == nil {
		t.Fatal("expected assistant draft to persist on cancellation")
	}
	if loaded.AssistantDraft.Text != "partial answer" {
		t.Fatalf("unexpected assistant draft: %q", loaded.AssistantDraft.Text)
	}
	if loaded.TurnDraft != nil {
		t.Fatalf("expected turn_draft to be cleared after cancellation, got %+v", loaded.TurnDraft)
	}
}

func TestRunTurnInputClearsAssistantDraftOnSuccess(t *testing.T) {
	sessionStore := newTempSessionStore(t)
	sess := newPersistedSessionForDraftTests(t, sessionStore, "session-success-clear-draft")
	sess.AssistantDraft = &session.AssistantDraft{
		Text:    "stale draft",
		TraceID: "trace-old",
		Turn:    1,
	}
	sess.TurnDraft = &session.TurnDraft{
		TraceID: "trace-old",
		Turn:    1,
	}
	if err := sessionStore.Save(sess); err != nil {
		t.Fatalf("save stale draft session: %v", err)
	}

	completer := &proTestCompleter{
		responses: []*llm.CompletionResponse{{
			Message:      llm.Message{Role: llm.RoleAssistant, Text: "done"},
			FinishReason: llm.FinishStop,
		}},
	}
	runner := newDraftTestRunner(sessionStore, completer)

	if _, _, err := runner.RunTurnInput(
		context.Background(),
		llm.Message{Role: llm.RoleUser, Text: "hello"},
		sess.ID,
		"trace-clear-draft",
	); err != nil {
		t.Fatalf("run turn input: %v", err)
	}

	loaded, loadErr := sessionStore.Load(sess.ID)
	if loadErr != nil {
		t.Fatalf("load session: %v", loadErr)
	}
	if loaded.AssistantDraft != nil {
		t.Fatalf("expected assistant draft to be cleared, got %+v", loaded.AssistantDraft)
	}
	if loaded.TurnDraft != nil {
		t.Fatalf("expected turn_draft to be cleared, got %+v", loaded.TurnDraft)
	}
}

func newDraftTestRunner(sessionStore *session.Store, completer llm.Completer) *SessionAgentRunner {
	return NewSessionAgentRunner(proTestRuntimeFactory{
		deps: agentRuntimeDependencies{
			cfg: bridgeconfig.Config{
				MaxTurns:    3,
				PromptsPath: "",
				PromptsDir:  os.Getenv("GHOST_PROMPTS_DIR"),
				Provider:    bridgeconfig.ProviderConfig{Type: llm.ProviderOpenAI, Model: "gpt-4o"},
			},
			client:       completer,
			registry:     tools.NewRegistry(),
			systemPrompt: "base system prompt",
		},
	}, nil, sessionStore, nil)
}

func newPersistedSessionForDraftTests(t *testing.T, store *session.Store, sessionID string) *session.Session {
	t.Helper()
	sess := session.NewSession("base system prompt")
	sess.ID = sessionID
	if err := store.Save(sess); err != nil {
		t.Fatalf("save session: %v", err)
	}
	return sess
}

const (
	titleRequestWaitTimeout = 2 * time.Second
	titleTestMaxTurns       = 3
)

func TestTitleFromFirstMessageUsesFirstSentenceAndTruncates(t *testing.T) {
	got := titleFromFirstMessage("   Build a session title. Then continue with details.\nnext paragraph")
	if got != "Build a session title." {
		t.Fatalf("unexpected title: %q", got)
	}

	longTitle := titleFromFirstMessage("abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyz")
	if len([]rune(longTitle)) != sessionTitleRuneLimit || !strings.HasSuffix(longTitle, "...") {
		t.Fatalf("unexpected truncated title: %q", longTitle)
	}
}

func TestSessionAgentRunnerFirstMessageTitleOnlyForNewSession(t *testing.T) {
	sessionStore := newTempSessionStore(t)
	runner := newTitleSessionRunner(newTitleTestCompleter(nil), sessionStore, bridgeconfig.SessionTitleModeFirstMessage)

	_, sessionID, err := runner.RunTurn(context.Background(), "Plan the release. Include checks.", "", "trace-title")
	if err != nil {
		t.Fatalf("run turn: %v", err)
	}
	loaded, err := sessionStore.Load(sessionID)
	if err != nil {
		t.Fatalf("load new session: %v", err)
	}
	if loaded.Title != "Plan the release." {
		t.Fatalf("unexpected new session title: %q", loaded.Title)
	}

	if _, _, err := runner.RunTurn(context.Background(), "Rename attempt", sessionID, "trace-title-2"); err != nil {
		t.Fatalf("run existing turn: %v", err)
	}
	loaded, err = sessionStore.Load(sessionID)
	if err != nil {
		t.Fatalf("reload session: %v", err)
	}
	if loaded.Title != "Plan the release." {
		t.Fatalf("existing session title changed: %q", loaded.Title)
	}
}

func TestSessionAgentRunnerAITitleFailureKeepsEmptyTitle(t *testing.T) {
	sessionStore := newTempSessionStore(t)
	completer := newTitleTestCompleter(errors.New("title failed"))
	runner := newTitleSessionRunner(completer, sessionStore, bridgeconfig.SessionTitleModeAIGenerated)

	_, sessionID, err := runner.RunTurn(context.Background(), "Generate background title", "", "trace-ai-title")
	if err != nil {
		t.Fatalf("run turn: %v", err)
	}
	completer.waitForTitleRequest(t)
	loaded, err := sessionStore.Load(sessionID)
	if err != nil {
		t.Fatalf("load session: %v", err)
	}
	if loaded.Title != "" {
		t.Fatalf("expected empty title after failure, got %q", loaded.Title)
	}
}

type titleTestCompleter struct {
	titleErr error
	done     chan struct{}
	once     sync.Once
}

func newTitleTestCompleter(titleErr error) *titleTestCompleter {
	return &titleTestCompleter{titleErr: titleErr, done: make(chan struct{})}
}

func (c *titleTestCompleter) Complete(
	_ context.Context,
	request llm.CompletionRequest,
) (*llm.CompletionResponse, error) {
	if isTitleCompletionRequest(request) {
		c.once.Do(func() { close(c.done) })
		if c.titleErr != nil {
			return nil, c.titleErr
		}
		return &llm.CompletionResponse{
			Message:      llm.Message{Role: llm.RoleAssistant, Text: "Generated Title"},
			FinishReason: llm.FinishStop,
		}, nil
	}
	return &llm.CompletionResponse{
		Message:      llm.Message{Role: llm.RoleAssistant, Text: "done"},
		FinishReason: llm.FinishStop,
	}, nil
}

func (c *titleTestCompleter) waitForTitleRequest(t *testing.T) {
	t.Helper()
	select {
	case <-c.done:
	case <-time.After(titleRequestWaitTimeout):
		t.Fatal("timed out waiting for title request")
	}
}

func isTitleCompletionRequest(request llm.CompletionRequest) bool {
	return len(request.Messages) > 0 && strings.Contains(request.Messages[0].Text, "会话标题生成器")
}

func newTitleSessionRunner(
	completer llm.Completer,
	sessionStore *session.Store,
	mode string,
) *SessionAgentRunner {
	return NewSessionAgentRunner(proTestRuntimeFactory{
		deps: agentRuntimeDependencies{
			cfg: bridgeconfig.Config{
				MaxTurns:         titleTestMaxTurns,
				SessionTitleMode: mode,
				Provider: bridgeconfig.ProviderConfig{
					Type:  llm.ProviderOpenAI,
					Model: "gpt-4o",
				},
			},
			client:       completer,
			registry:     tools.NewRegistry(),
			systemPrompt: "base system prompt",
		},
	}, nil, sessionStore, nil)
}

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
	runAgent := newSessionTurnPreparer(nil, nil, nil, nil, nil).buildTurnAgent(
		deps,
		catalog,
		sess,
		agent.NewHistory(prompt),
	)

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
		sessionStore: sessionStore,
		persistence:  newSessionTurnCommitter(sessionStore),
		sess:         sess,
		agent:        runAgent,
		execCtx:      execCtx,
		traceID:      "trace-tool-turn-transaction",
	}

	response, runErr := runAgent.RunWithTraceID(execCtx, "hello", turn.traceID)
	if runErr == nil {
		t.Fatal("expected completion failure after tool turn")
	}

	_, persistedSessionID, err := turn.complete(response, runErr, nil)
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

func TestTaskSchemaDefinesKindSpecificContracts(t *testing.T) {
	defs := loadTaskSchemaDefs(t)
	assertSchemaOneOfRefs(t, defs, "taskCreateRequest",
		"agentMessageTaskCreateRequest",
		"workflowTaskCreateRequest",
		"orchestrationTaskCreateRequest",
	)
	assertSchemaOneOfRefs(t, defs, "taskPayload",
		"agentMessageTaskPayload",
		"workflowTaskPayload",
		"orchestrationTaskPayload",
	)
	assertSchemaRequired(t, defs, "agentMessageTaskCreateRequest", "message")
	assertSchemaRequired(t, defs, "workflowTaskCreateRequest", "workflow")
	assertSchemaRequired(t, defs, "orchestrationTaskCreateRequest", "name")
	assertSchemaRequired(t, defs, "orchestrationTaskCreateRequest", "orchestration")
	assertSchemaRequired(t, defs, "taskUpdateRequest", "id")
	assertSchemaProperty(t, defs, "taskUpdateRequest", "name")
	assertSchemaProperty(t, defs, "taskRuntimeOverrides", "provider_name")
	assertSchemaProperty(t, defs, "taskRuntimeOverrides", "model")
	assertSchemaProperty(t, defs, "taskRuntimeOverrides", "system_prompt")
	assertSchemaProperty(t, defs, "taskRuntimeOverrides", "preset_id")
	assertSchemaProperty(t, defs, "taskRuntimeOverrides", "tool_allowlist")
	assertSchemaProperty(t, defs, "taskRuntimeOverrides", "tool_allowlist_only")
	assertSchemaProperty(t, defs, "taskRuntimeOverrides", "max_turns")
	assertSchemaProperty(t, defs, "agentMessageTaskCreateRequest", "runtime_overrides")
	assertSchemaProperty(t, defs, "taskUpdateRequest", "runtime_overrides")
	assertSchemaProperty(t, defs, "agentMessageTaskPayload", "runtime_overrides")
	assertSchemaProperty(t, defs, "workflowNode", "start")
	assertSchemaProperty(t, defs, "workflowNode", "tool")
	assertSchemaProperty(t, defs, "workflowNode", "llm")
	assertSchemaProperty(t, defs, "workflowNode", "agent")
	assertSchemaProperty(t, defs, "workflowNode", "if")
	assertSchemaProperty(t, defs, "workflowNode", "loop")
	assertSchemaProperty(t, defs, "orchestrationNode", "group")
	assertSchemaProperty(t, defs, "orchestrationNode", "agent")
	assertSchemaProperty(t, defs, "workflowStartNode", "inputs")
	assertSchemaRequired(t, defs, "workflowInputVariable", "name")
	assertSchemaRequired(t, defs, "workflowInputVariable", "type")
	assertSchemaProperty(t, defs, "workflowInputVariable", "default")
	assertSchemaRequired(t, defs, "workflowToolNode", "tool_name")
	assertSchemaRequired(t, defs, "workflowLLMNode", "prompt")
	assertSchemaRequired(t, defs, "workflowAgentNode", "message")
	assertSchemaProperty(t, defs, "workflowAgentNode", "runtime_overrides")
	assertSchemaRequired(t, defs, "workflowIfNode", "operator")
	assertSchemaRequired(t, defs, "workflowIfNode", "true_node_id")
	assertSchemaRequired(t, defs, "workflowIfNode", "false_node_id")
	assertSchemaRequired(t, defs, "workflowLoopNode", "max_iterations")
	assertSchemaRequired(t, defs, "workflowLoopNode", "body_node_id")
	assertSchemaRequired(t, defs, "workflowLoopNode", "exit_node_id")
	assertSchemaRequired(t, defs, "taskRelayConfig", "max_rounds")
	assertSchemaRequired(t, defs, "orchestrationGroupNode", "title")
	assertSchemaRequired(t, defs, "orchestrationGroupNode", "shared_context")
	assertSchemaRequired(t, defs, "orchestrationGroupNode", "speaking_mode")
	assertSchemaRequired(t, defs, "orchestrationGroupNode", "max_rounds")
	assertSchemaProperty(t, defs, "orchestrationGroupNode", "owner_agent_id")
	assertSchemaRequired(t, defs, "orchestrationAgentNode", "title")
	assertSchemaRequired(t, defs, "orchestrationAgentNode", "message")
	assertSchemaRequired(t, defs, "orchestrationEdge", "kind")
}

func loadTaskSchemaDefs(t *testing.T) map[string]any {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve caller path")
	}
	path := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "shared", "schema", "defs", "tasks.json"))
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read tasks schema: %v", err)
	}
	var document map[string]any
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatalf("decode tasks schema: %v", err)
	}
	defs, ok := document["$defs"].(map[string]any)
	if !ok {
		t.Fatal("tasks schema defs missing")
	}
	return defs
}

func assertSchemaOneOfRefs(t *testing.T, defs map[string]any, name string, want ...string) {
	t.Helper()
	definition := schemaDefinition(t, defs, name)
	items, ok := definition["oneOf"].([]any)
	if !ok || len(items) != len(want) {
		t.Fatalf("unexpected oneOf in %s: %#v", name, definition["oneOf"])
	}
	for index, ref := range want {
		item, ok := items[index].(map[string]any)
		if !ok || item["$ref"] != "#/$defs/"+ref {
			t.Fatalf("unexpected oneOf[%d] in %s: %#v", index, name, items[index])
		}
	}
}

func assertSchemaRequired(t *testing.T, defs map[string]any, name string, field string) {
	t.Helper()
	required, ok := schemaDefinition(t, defs, name)["required"].([]any)
	if !ok {
		t.Fatalf("required missing in %s", name)
	}
	for _, item := range required {
		if item == field {
			return
		}
	}
	t.Fatalf("field %q is not required in %s: %#v", field, name, required)
}

func assertSchemaProperty(t *testing.T, defs map[string]any, name string, field string) {
	t.Helper()
	properties := schemaProperties(t, defs, name)
	if _, ok := properties[field]; !ok {
		t.Fatalf("field %q missing in %s", field, name)
	}
}

func schemaDefinition(t *testing.T, defs map[string]any, name string) map[string]any {
	t.Helper()
	definition, ok := defs[name].(map[string]any)
	if !ok {
		t.Fatalf("schema definition %q missing", name)
	}
	return definition
}

func schemaProperties(t *testing.T, defs map[string]any, name string) map[string]any {
	t.Helper()
	properties, ok := schemaDefinition(t, defs, name)["properties"].(map[string]any)
	if !ok {
		t.Fatalf("properties missing in %s", name)
	}
	return properties
}

func TestValidateTaskDefinitionRejectsUnsupportedTaskKind(t *testing.T) {
	task := ScheduledTask{
		TaskKind: "unexpected_kind",
		Message:  "hello",
	}

	err := validateTaskDefinition(&task)
	if err == nil || !strings.Contains(err.Error(), `unsupported task_kind "unexpected_kind"`) {
		t.Fatalf("expected unsupported task_kind error, got %v", err)
	}
}

func TestTaskCreateRejectsUnsupportedTaskKind(t *testing.T) {
	_, service, _ := newTestHandlerWithService(t, nil, nil)

	_, code, err := service.executeTaskCreateAction(taskCreateParams{
		TaskKind:        "unexpected_kind",
		Message:         "hello",
		IntervalSeconds: 60,
	}, "trace-invalid-kind")
	if err == nil {
		t.Fatal("expected task create to fail")
	}
	if code != http.StatusBadRequest {
		t.Fatalf("unexpected status code: got %d want %d", code, http.StatusBadRequest)
	}
	if !strings.Contains(err.Error(), `unsupported task_kind "unexpected_kind"`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTaskExecutorAdapterRejectsUnsupportedTaskKind(t *testing.T) {
	result := taskExecutorAdapter{}.Execute(context.Background(), ScheduledTask{
		TaskKind: "unexpected_kind",
		Message:  "hello",
	}, "trace-invalid-kind")
	if result.Status != taskRunStatusError {
		t.Fatalf("unexpected execution status: got %q want %q", result.Status, taskRunStatusError)
	}
	if result.Error != `unsupported task_kind "unexpected_kind"` {
		t.Fatalf("unexpected execution error: %q", result.Error)
	}
}

func TestTaskMutationRunnerUpdateRollsBackRegistrationWhenSaveFails(t *testing.T) {
	original := mutationTestTask()
	store := &taskMutationStoreStub{
		loadedTask: original,
		saveErrs:   []error{errors.New("save failed")},
	}
	scheduler := newTaskMutationSchedulerStub(original)
	runner := taskMutationRunner{store: store, scheduler: scheduler}

	message := "updated message"
	_, err := runner.Update(taskUpdateParams{ID: original.ID, Message: &message})
	if err == nil || !strings.Contains(err.Error(), "save failed") {
		t.Fatalf("expected save failure, got %v", err)
	}
	assertRegisteredTaskMessage(t, scheduler, original.ID, original.Message)
	if store.loadedTask.Message != original.Message {
		t.Fatalf("expected store to keep original task, got %#v", store.loadedTask)
	}
	if len(store.savedTasks) != 1 || store.savedTasks[0].Message != message {
		t.Fatalf("expected one attempted save with new message, got %#v", store.savedTasks)
	}
	if len(scheduler.upserts) != 1 || scheduler.upserts[0].Message != original.Message {
		t.Fatalf("expected rollback upsert with original task, got %#v", scheduler.upserts)
	}
}

func TestTaskMutationRunnerUpdateRollsBackPersistedTaskWhenUpsertFails(t *testing.T) {
	original := mutationTestTask()
	store := &taskMutationStoreStub{loadedTask: original}
	scheduler := newTaskMutationSchedulerStub(original)
	scheduler.upsertErrs = []error{errors.New("upsert failed")}
	runner := taskMutationRunner{store: store, scheduler: scheduler}

	message := "updated message"
	_, err := runner.Update(taskUpdateParams{ID: original.ID, Message: &message})
	if err == nil || !strings.Contains(err.Error(), "upsert failed") {
		t.Fatalf("expected upsert failure, got %v", err)
	}
	assertRegisteredTaskMessage(t, scheduler, original.ID, original.Message)
	if store.loadedTask.Message != original.Message {
		t.Fatalf("expected store rollback to restore original task, got %#v", store.loadedTask)
	}
	if len(store.savedTasks) != 2 {
		t.Fatalf("expected update save and rollback save, got %#v", store.savedTasks)
	}
	if store.savedTasks[0].Message != message || store.savedTasks[1].Message != original.Message {
		t.Fatalf("unexpected saved task sequence: %#v", store.savedTasks)
	}
	if len(scheduler.upserts) != 2 {
		t.Fatalf("expected failed upsert and rollback upsert, got %#v", scheduler.upserts)
	}
	if scheduler.upserts[0].Message != message || scheduler.upserts[1].Message != original.Message {
		t.Fatalf("unexpected upsert sequence: %#v", scheduler.upserts)
	}
}

func TestTaskMutationRunnerDeleteRollsBackRegistrationWhenDeleteFails(t *testing.T) {
	original := mutationTestTask()
	store := &taskMutationStoreStub{
		loadedTask: original,
		deleteErrs: []error{errors.New("delete failed")},
	}
	scheduler := newTaskMutationSchedulerStub(original)
	runner := taskMutationRunner{store: store, scheduler: scheduler}

	_, err := runner.Delete(taskIDParams{ID: original.ID})
	if err == nil || !strings.Contains(err.Error(), "delete failed") {
		t.Fatalf("expected delete failure, got %v", err)
	}
	assertRegisteredTaskMessage(t, scheduler, original.ID, original.Message)
	if store.loadedTask.Message != original.Message {
		t.Fatalf("expected store rollback to restore original task, got %#v", store.loadedTask)
	}
	if len(store.savedTasks) != 1 || store.savedTasks[0].Message != original.Message {
		t.Fatalf("expected one rollback save with original task, got %#v", store.savedTasks)
	}
	if len(store.deletedIDs) != 1 || store.deletedIDs[0] != original.ID {
		t.Fatalf("expected delete attempt for %q, got %#v", original.ID, store.deletedIDs)
	}
	if len(scheduler.upserts) != 1 || scheduler.upserts[0].Message != original.Message {
		t.Fatalf("expected rollback upsert with original task, got %#v", scheduler.upserts)
	}
}

type taskMutationStoreStub struct {
	loadedTask ScheduledTask
	saveErrs   []error
	deleteErrs []error
	deletedIDs []string
	savedTasks []ScheduledTask
}

func (s *taskMutationStoreStub) LoadTask(_ string) (*ScheduledTask, error) {
	task := cloneScheduledTask(s.loadedTask)
	return &task, nil
}

func (s *taskMutationStoreStub) SaveTask(task *ScheduledTask) error {
	s.savedTasks = append(s.savedTasks, cloneScheduledTask(*task))
	if len(s.saveErrs) > 0 {
		err := s.saveErrs[0]
		s.saveErrs = s.saveErrs[1:]
		return err
	}
	s.loadedTask = cloneScheduledTask(*task)
	return nil
}

func (s *taskMutationStoreStub) DeleteTask(taskID string) error {
	s.deletedIDs = append(s.deletedIDs, taskID)
	if len(s.deleteErrs) > 0 {
		err := s.deleteErrs[0]
		s.deleteErrs = s.deleteErrs[1:]
		if err != nil {
			return err
		}
	}
	s.loadedTask = ScheduledTask{}
	return nil
}

type taskMutationSchedulerStub struct {
	registered    map[string]ScheduledTask
	upsertErrs    []error
	unregisterIDs []string
	upserts       []ScheduledTask
}

func newTaskMutationSchedulerStub(task ScheduledTask) *taskMutationSchedulerStub {
	return &taskMutationSchedulerStub{
		registered: map[string]ScheduledTask{task.ID: cloneScheduledTask(task)},
	}
}

func (s *taskMutationSchedulerStub) Upsert(task ScheduledTask) error {
	s.upserts = append(s.upserts, cloneScheduledTask(task))
	if len(s.upsertErrs) > 0 {
		err := s.upsertErrs[0]
		s.upsertErrs = s.upsertErrs[1:]
		if err != nil {
			return err
		}
	}
	s.registered[task.ID] = cloneScheduledTask(task)
	return nil
}

func (s *taskMutationSchedulerStub) Unregister(taskID string) error {
	s.unregisterIDs = append(s.unregisterIDs, taskID)
	delete(s.registered, taskID)
	return nil
}

func (s *taskMutationSchedulerStub) RunNow(task ScheduledTask, _ string) (TaskRunLog, error) {
	return TaskRunLog{TaskID: task.ID}, nil
}

func mutationTestTask() ScheduledTask {
	return ScheduledTask{
		ID:              "task-update-test",
		Message:         "original message",
		TaskKind:        taskKindAgentMessage,
		ScheduleType:    taskScheduleTypeInterval,
		IntervalSeconds: 60,
		Enabled:         true,
		CreatedAt:       time.Unix(100, 0).UTC(),
		UpdatedAt:       time.Unix(200, 0).UTC(),
		NextRunAt:       time.Unix(300, 0).UTC(),
	}
}

func assertRegisteredTaskMessage(
	t *testing.T,
	scheduler *taskMutationSchedulerStub,
	taskID string,
	want string,
) {
	t.Helper()
	task, ok := scheduler.registered[taskID]
	if !ok {
		t.Fatalf("expected task %q to stay registered", taskID)
	}
	if task.Message != want {
		t.Fatalf("unexpected registered task: %#v", task)
	}
}

func TestValidateWorkflowTaskRuntimeAllowsAllToolsWhenAllowlistEmpty(t *testing.T) {
	definition := workflowWithToolNode("web_search")

	err := validateWorkflowTaskRuntime(definition, bridgeconfig.TaskConfig{})
	if err != nil {
		t.Fatalf("expected empty allowlist to allow all tools, got %v", err)
	}
}

func TestValidateWorkflowTaskRuntimeRejectsToolOutsideExplicitAllowlist(t *testing.T) {
	definition := workflowWithToolNode("web_search")
	cfg := bridgeconfig.TaskConfig{WorkflowToolAllowlist: []string{"script_exec"}}

	err := validateWorkflowTaskRuntime(definition, cfg)
	if err == nil {
		t.Fatal("expected workflow tool validation to fail")
	}
	if !strings.Contains(err.Error(), `workflow tool "web_search" is not allowed`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestWorkflowTaskCreateRejectsInvalidAgentRuntimeOverrides(t *testing.T) {
	_, service, _ := newTestHandlerWithService(t, nil, nil)
	configureRuntimeOverrideProviders(t, service)

	workflow := &WorkflowDefinition{
		Nodes: []WorkflowNode{
			{ID: "start-node", Type: workflowNodeTypeStart},
			{
				ID:   "agent-node",
				Type: workflowNodeTypeAgent,
				Agent: &WorkflowAgentNode{
					Message: "run agent",
					RuntimeOverrides: &TaskRuntimeOverrides{
						ProviderName: "openai-main",
					},
				},
			},
			{ID: "end-node", Type: workflowNodeTypeEnd},
		},
		Edges: []WorkflowEdge{
			{FromNodeID: "start-node", ToNodeID: "agent-node"},
			{FromNodeID: "agent-node", ToNodeID: "end-node"},
		},
	}

	_, code, err := service.executeTaskCreateAction(taskCreateParams{
		TaskKind:        taskKindWorkflow,
		Workflow:        workflow,
		IntervalSeconds: 60,
	}, "trace-workflow-agent-runtime-invalid")
	if err == nil {
		t.Fatal("expected workflow agent runtime validation to fail")
	}
	if code != http.StatusBadRequest {
		t.Fatalf("unexpected status code: got %d want %d", code, http.StatusBadRequest)
	}
	if !strings.Contains(err.Error(), "provider_name requires model") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRelayTaskCreateAppliesConfiguredDefaults(t *testing.T) {
	_, service, _ := newTestHandlerWithService(t, nil, nil)
	_, err := service.executeConfigUpdateAction(configUpdateRequest{
		RelayDefaultStopPolicy:         stringPointer(taskRelayStopPolicyMaxRounds),
		RelayDefaultMaxRounds:          intPointer(7),
		RelayDefaultExecutionTimeoutMs: intPointer(0),
	}, "trace-relay-defaults-config")
	if err != nil {
		t.Fatalf("config update failed: %v", err)
	}

	createdRaw, code, err := service.executeTaskCreateAction(taskCreateParams{
		TaskKind:        taskKindAgentMessage,
		Message:         "run relay task",
		AgentMode:       taskAgentModeRelay,
		IntervalSeconds: 60,
	}, "trace-relay-defaults-task")
	if err != nil {
		t.Fatalf("task create failed: %v", err)
	}
	if code != http.StatusCreated {
		t.Fatalf("unexpected status code: got %d want %d", code, http.StatusCreated)
	}
	created := createdRaw.(taskPayload)
	if created.Relay == nil {
		t.Fatal("expected relay defaults")
	}
	if created.Relay.StopPolicy != taskRelayStopPolicyMaxRounds || created.Relay.MaxRounds != 7 {
		t.Fatalf("unexpected relay defaults: %+v", created.Relay)
	}
	if created.Relay.ExecutionTimeoutMS == nil || *created.Relay.ExecutionTimeoutMS != 0 {
		t.Fatalf("unexpected relay execution timeout: %+v", created.Relay.ExecutionTimeoutMS)
	}
}

func TestRelayTaskCreateAppliesAIDecidesMaxRoundDefault(t *testing.T) {
	_, service, _ := newTestHandlerWithService(t, nil, nil)
	_, err := service.executeConfigUpdateAction(configUpdateRequest{
		RelayDefaultStopPolicy:         stringPointer(taskRelayStopPolicyAIDecides),
		RelayDefaultMaxRounds:          intPointer(6),
		RelayDefaultExecutionTimeoutMs: intPointer(0),
	}, "trace-relay-ai-defaults-config")
	if err != nil {
		t.Fatalf("config update failed: %v", err)
	}

	createdRaw, code, err := service.executeTaskCreateAction(taskCreateParams{
		TaskKind:        taskKindAgentMessage,
		Message:         "run relay task",
		AgentMode:       taskAgentModeRelay,
		IntervalSeconds: 60,
	}, "trace-relay-ai-defaults-task")
	if err != nil {
		t.Fatalf("task create failed: %v", err)
	}
	if code != http.StatusCreated {
		t.Fatalf("unexpected status code: got %d want %d", code, http.StatusCreated)
	}
	created := createdRaw.(taskPayload)
	if created.Relay == nil {
		t.Fatal("expected relay defaults")
	}
	if created.Relay.StopPolicy != taskRelayStopPolicyAIDecides || created.Relay.MaxRounds != 6 {
		t.Fatalf("unexpected relay defaults: %+v", created.Relay)
	}
}

func TestNormalizeTaskRuntimeOverrides(t *testing.T) {
	normalized, err := normalizeTaskRuntimeOverrides(nil)
	if err != nil {
		t.Fatalf("normalize nil overrides: %v", err)
	}
	if normalized != nil {
		t.Fatalf("expected nil overrides, got %#v", normalized)
	}

	normalized, err = normalizeTaskRuntimeOverrides(&TaskRuntimeOverrides{
		Model:         " ",
		ToolAllowlist: []string{"  ", "\n"},
	})
	if err != nil {
		t.Fatalf("normalize empty overrides: %v", err)
	}
	if normalized != nil {
		t.Fatalf("expected empty overrides to collapse to nil, got %#v", normalized)
	}

	normalized, err = normalizeTaskRuntimeOverrides(&TaskRuntimeOverrides{
		ProviderName:      " openai-main ",
		Model:             " gpt-5.4 ",
		SystemPrompt:      " be concise ",
		PresetID:          " preset-a ",
		ToolAllowlist:     []string{"web_search", "script_exec", "web_search"},
		ToolAllowlistOnly: boolPointer(true),
		MaxTurns:          intPointer(3),
	})
	if err != nil {
		t.Fatalf("normalize valid overrides: %v", err)
	}
	if normalized.ProviderName != "openai-main" {
		t.Fatalf("unexpected provider_name: %#v", normalized)
	}
	if normalized.Model != "gpt-5.4" {
		t.Fatalf("unexpected model: %#v", normalized)
	}
	if normalized.SystemPrompt != "be concise" {
		t.Fatalf("unexpected system_prompt: %#v", normalized)
	}
	if normalized.PresetID != "preset-a" {
		t.Fatalf("unexpected preset_id: %#v", normalized)
	}
	if normalized.ToolAllowlistOnly == nil || !*normalized.ToolAllowlistOnly {
		t.Fatalf("expected tool_allowlist_only=true, got %#v", normalized)
	}
	if normalized.MaxTurns == nil || *normalized.MaxTurns != 3 {
		t.Fatalf("unexpected max_turns: %#v", normalized)
	}
	if strings.Join(normalized.ToolAllowlist, ",") != "script_exec,web_search" {
		t.Fatalf("unexpected allowlist normalization: %#v", normalized.ToolAllowlist)
	}
}

func TestTaskCreateRejectsUnknownRuntimeOverrideTool(t *testing.T) {
	_, service, _ := newTestHandlerWithService(t, nil, nil)

	_, code, err := service.executeTaskCreateAction(taskCreateParams{
		Message:          "run with bad override",
		TaskKind:         taskKindAgentMessage,
		RuntimeOverrides: &TaskRuntimeOverrides{ToolAllowlist: []string{"ghost_tool"}},
		IntervalSeconds:  60,
	}, "trace-task-runtime-override-invalid-tool")
	if err == nil {
		t.Fatal("expected runtime override tool validation to fail")
	}
	if code != http.StatusBadRequest {
		t.Fatalf("unexpected status code: got %d want %d", code, http.StatusBadRequest)
	}
	if !strings.Contains(err.Error(), "unknown tool in tool_allowlist: ghost_tool") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTaskRunNowAppliesRuntimeOverrideToolAllowlist(t *testing.T) {
	_, service, _ := newTestHandlerWithService(t, nil, nil)
	if err := service.configStore.UpdateTool(bridgeconfig.ToolUpdateRequest{
		Name:    "script_exec",
		Enabled: boolPointer(false),
	}); err != nil {
		t.Fatalf("disable script_exec: %v", err)
	}

	completer := &workflowTestCompleter{
		response: &llm.CompletionResponse{
			Message:      llm.Message{Role: llm.RoleAssistant, Text: "ok"},
			FinishReason: llm.FinishStop,
		},
	}
	registry := tools.NewRegistry()
	registry.Register(&workflowTestTool{name: "script_exec", output: `{"status":"ok"}`})
	registry.Register(&workflowTestTool{name: "web_search", output: `{"status":"ok"}`})
	runtimeFactory := &captureRuntimeOverrideFactory{
		baseConfig: bridgeconfig.Config{
			MaxTurns: 4,
			ToolSelector: bridgeconfig.ToolSelectorConfig{
				Allowlist: []string{"script_exec", "web_search"},
			},
			ToolSearch: bridgeconfig.ToolSearchConfig{
				IdleTurns: 3,
			},
		},
		completer: completer,
		registry:  registry,
	}
	service.runtimeFactory = runtimeFactory
	service.agentRunner = NewSessionAgentRunner(
		runtimeFactory,
		service.configStore,
		service.sessionStore,
		service.runRegistry,
	)

	createdRaw, code, err := service.executeTaskCreateAction(taskCreateParams{
		Message:  "use narrowed tool set",
		TaskKind: taskKindAgentMessage,
		RuntimeOverrides: &TaskRuntimeOverrides{
			ToolAllowlist: []string{"script_exec"},
		},
		IntervalSeconds: 60,
	}, "trace-task-runtime-override-create")
	if err != nil || code != http.StatusCreated {
		t.Fatalf("create task: code=%d err=%v", code, err)
	}
	created := createdRaw.(taskPayload)

	runRaw, code, err := service.executeTaskRunNowAction(taskIDParams{ID: created.ID}, "trace-task-runtime-override-run")
	if err != nil || code != http.StatusOK {
		t.Fatalf("run task: code=%d err=%v", code, err)
	}
	run := runRaw.(taskRunPayload)
	if run.Run.Status != taskRunStatusSuccess {
		t.Fatalf("unexpected run status: %#v", run.Run)
	}
	if len(run.Run.NodeResults) != 1 {
		t.Fatalf("expected one agent_message node result, got %#v", run.Run.NodeResults)
	}
	node := run.Run.NodeResults[0]
	if node.NodeID != taskKindAgentMessage || node.NodeType != taskKindAgentMessage || node.Status != taskRunStatusSuccess {
		t.Fatalf("unexpected agent node identity: %#v", node)
	}
	input, ok := node.Input.(map[string]any)
	message, hasMessage := input["message"].(string)
	if !ok || !hasMessage || strings.TrimSpace(message) != "use narrowed tool set" {
		t.Fatalf("unexpected agent node input: %#v", node.Input)
	}
	output, ok := node.Output.(map[string]any)
	if !ok {
		t.Fatalf("unexpected agent node output: %#v", node.Output)
	}
	if _, exists := output["response_preview"]; !exists {
		t.Fatalf("expected response_preview in node output: %#v", node.Output)
	}
	if len(completer.requests) != 1 {
		t.Fatalf("expected one completion request, got %d", len(completer.requests))
	}
	names := make([]string, 0, len(completer.requests[0].Tools))
	for _, def := range completer.requests[0].Tools {
		names = append(names, def.Name)
	}
	if strings.Join(names, ",") != "script_exec" {
		t.Fatalf("unexpected tool scope: %v", names)
	}
}

func TestTaskRunNowAppliesRuntimeOverrideProviderPromptAndMaxTurns(t *testing.T) {
	_, service, _ := newTestHandlerWithService(t, nil, nil)
	configureRuntimeOverrideProviders(t, service)

	completer := &workflowTestCompleter{
		response: &llm.CompletionResponse{
			Message:      llm.Message{Role: llm.RoleAssistant, Text: "ok"},
			FinishReason: llm.FinishStop,
		},
	}
	factory := &captureRuntimeOverrideFactory{
		completer: completer,
		registry:  tools.NewRegistry(),
	}
	service.runtimeFactory = factory
	service.agentRunner = NewSessionAgentRunner(
		factory,
		service.configStore,
		service.sessionStore,
		service.runRegistry,
	)

	createdRaw, code, err := service.executeTaskCreateAction(taskCreateParams{
		Message:  "run with full overrides",
		TaskKind: taskKindAgentMessage,
		RuntimeOverrides: &TaskRuntimeOverrides{
			ProviderName:      "anthropic-main",
			Model:             "claude-3.7",
			SystemPrompt:      "override prompt only",
			ToolAllowlistOnly: boolPointer(true),
			ToolAllowlist:     []string{},
			MaxTurns:          intPointer(1),
		},
		IntervalSeconds: 60,
	}, "trace-task-runtime-override-full-create")
	if err != nil || code != http.StatusCreated {
		t.Fatalf("create task: code=%d err=%v", code, err)
	}
	created := createdRaw.(taskPayload)

	runRaw, code, err := service.executeTaskRunNowAction(taskIDParams{ID: created.ID}, "trace-task-runtime-override-full-run")
	if err != nil || code != http.StatusOK {
		t.Fatalf("run task: code=%d err=%v", code, err)
	}
	run := runRaw.(taskRunPayload)
	if run.Run.Status != taskRunStatusSuccess {
		t.Fatalf("unexpected run status: %#v", run.Run)
	}
	if len(factory.configs) != 1 {
		t.Fatalf("expected one runtime config build, got %d", len(factory.configs))
	}
	cfg := factory.configs[0]
	if cfg.Provider.Type != llm.ProviderAnthropic || cfg.Provider.Model != "claude-3.7" {
		t.Fatalf("unexpected provider override config: %+v", cfg.Provider)
	}
	if cfg.MaxTurns != 1 {
		t.Fatalf("unexpected max_turns override: %d", cfg.MaxTurns)
	}
	if !cfg.ToolSelector.AllowlistOnly || len(cfg.ToolSelector.Allowlist) != 0 {
		t.Fatalf("unexpected tool selector override: %+v", cfg.ToolSelector)
	}
	if len(completer.requests) != 1 {
		t.Fatalf("expected one completion request, got %d", len(completer.requests))
	}
	if len(completer.requests[0].Messages) == 0 || completer.requests[0].Messages[0].Text != "override prompt only" {
		t.Fatalf("unexpected system prompt override request: %#v", completer.requests[0].Messages)
	}
	if len(completer.requests[0].Tools) != 0 {
		t.Fatalf("expected explicit no-tools request, got %#v", completer.requests[0].Tools)
	}
}

func TestTaskCreateRejectsUnknownRuntimeOverrideProviderModelAndMaxTurns(t *testing.T) {
	_, service, _ := newTestHandlerWithService(t, nil, nil)
	configureRuntimeOverrideProviders(t, service)

	tests := []struct {
		name      string
		overrides *TaskRuntimeOverrides
		want      string
	}{
		{
			name:      "unknown provider",
			overrides: &TaskRuntimeOverrides{ProviderName: "ghost", Model: "gpt-5.4"},
			want:      `provider_name "ghost" is not configured`,
		},
		{
			name:      "model outside provider",
			overrides: &TaskRuntimeOverrides{Model: "claude-3.7"},
			want:      `model "claude-3.7" is not configured for provider "openai-main"`,
		},
		{
			name:      "invalid max turns",
			overrides: &TaskRuntimeOverrides{MaxTurns: intPointer(0)},
			want:      "max_turns must be > 0",
		},
		{
			name:      "missing preset",
			overrides: &TaskRuntimeOverrides{PresetID: "ghost-preset"},
			want:      `preset_id "ghost-preset" is not configured`,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			_, code, err := service.executeTaskCreateAction(taskCreateParams{
				Message:          "bad override",
				TaskKind:         taskKindAgentMessage,
				RuntimeOverrides: test.overrides,
				IntervalSeconds:  60,
			}, "trace-task-runtime-override-invalid")
			if err == nil {
				t.Fatal("expected runtime override validation to fail")
			}
			if code != http.StatusBadRequest {
				t.Fatalf("unexpected status code: got %d want %d", code, http.StatusBadRequest)
			}
			if !strings.Contains(err.Error(), test.want) {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestTaskRunNowBuildsSystemPromptFromPresetWithoutMutatingStoredPromptFiles(t *testing.T) {
	_, service, _ := newTestHandlerWithService(t, nil, nil)
	configureRuntimeOverrideProviders(t, service)
	library := []bridgeconfig.SystemPromptLibraryItem{
		{ID: "rule-global", Name: "Rule Global", InsertPoint: bridgeconfig.SystemPromptInsertPointRule, Content: "global rule", Active: true},
		{ID: "rule-preset", Name: "Rule Preset", InsertPoint: bridgeconfig.SystemPromptInsertPointRule, Content: "preset rule", Active: false},
		{ID: "core-global", Name: "Core Global", InsertPoint: bridgeconfig.SystemPromptInsertPointCoreJob, Content: "global core", Active: true},
		{ID: "core-preset", Name: "Core Preset", InsertPoint: bridgeconfig.SystemPromptInsertPointCoreJob, Content: "preset core", Active: false},
		{ID: "context-global", Name: "Context Global", InsertPoint: bridgeconfig.SystemPromptInsertPointContext, Content: "global context", Active: true},
		{ID: "context-preset", Name: "Context Preset", InsertPoint: bridgeconfig.SystemPromptInsertPointContext, Content: "preset context", Active: false},
	}
	if _, err := service.configStore.UpdateSystemPrompts(bridgeconfig.SystemPromptUpdateRequest{
		PromptLibrary: &library,
	}); err != nil {
		t.Fatalf("seed prompt library: %v", err)
	}
	preset, err := service.configStore.CreatePreset(bridgeconfig.PresetCreateRequest{
		Name:          "Research",
		ToolAllowlist: []string{},
		PromptRefs: bridgeconfig.PresetPromptRefs{
			Rule:    "rule-preset",
			CoreJob: "core-preset",
			Context: []string{"context-preset"},
		},
	})
	if err != nil {
		t.Fatalf("create preset: %v", err)
	}

	completer := &workflowTestCompleter{
		response: &llm.CompletionResponse{
			Message:      llm.Message{Role: llm.RoleAssistant, Text: "ok"},
			FinishReason: llm.FinishStop,
		},
	}
	factory := &captureRuntimeOverrideFactory{
		completer: completer,
		registry:  tools.NewRegistry(),
	}
	service.runtimeFactory = factory
	service.agentRunner = NewSessionAgentRunner(
		factory,
		service.configStore,
		service.sessionStore,
		service.runRegistry,
	)

	createdRaw, code, err := service.executeTaskCreateAction(taskCreateParams{
		Message:  "run with preset",
		TaskKind: taskKindAgentMessage,
		RuntimeOverrides: &TaskRuntimeOverrides{
			PresetID:          preset.ID,
			ToolAllowlistOnly: boolPointer(true),
			ToolAllowlist:     []string{},
		},
		IntervalSeconds: 60,
	}, "trace-task-runtime-override-preset-create")
	if err != nil || code != http.StatusCreated {
		t.Fatalf("create task: code=%d err=%v", code, err)
	}
	created := createdRaw.(taskPayload)

	runRaw, code, err := service.executeTaskRunNowAction(taskIDParams{ID: created.ID}, "trace-task-runtime-override-preset-run")
	if err != nil || code != http.StatusOK {
		t.Fatalf("run task: code=%d err=%v", code, err)
	}
	run := runRaw.(taskRunPayload)
	if run.Run.Status != taskRunStatusSuccess {
		t.Fatalf("unexpected run status: %#v", run.Run)
	}
	if len(completer.requests) != 1 {
		t.Fatalf("expected one completion request, got %d", len(completer.requests))
	}
	systemPrompt := completer.requests[0].Messages[0].Text
	if !strings.Contains(systemPrompt, "preset rule") ||
		!strings.Contains(systemPrompt, "preset core") ||
		!strings.Contains(systemPrompt, "preset context") {
		t.Fatalf("unexpected preset system prompt: %q", systemPrompt)
	}
	if strings.Contains(systemPrompt, "global rule") || strings.Contains(systemPrompt, "global context") {
		t.Fatalf("expected preset runtime prompt to replace active prompt refs, got %q", systemPrompt)
	}

	files, err := service.configStore.SystemPrompts()
	if err != nil {
		t.Fatalf("reload prompt library: %v", err)
	}
	if activePromptID(files.PromptLibrary, bridgeconfig.SystemPromptInsertPointRule) != "rule-global" {
		t.Fatalf("expected stored rule prompt to remain global, got %+v", files.PromptLibrary)
	}
	if activePromptID(files.PromptLibrary, bridgeconfig.SystemPromptInsertPointCoreJob) != "core-global" {
		t.Fatalf("expected stored core prompt to remain global, got %+v", files.PromptLibrary)
	}
	if activePromptIDs(files.PromptLibrary, bridgeconfig.SystemPromptInsertPointContext)[0] != "context-global" {
		t.Fatalf("expected stored context prompt to remain global, got %+v", files.PromptLibrary)
	}
}

func TestExecuteAgentActionWithRuntimeOverridesRejectsMissingPresetAtRunTime(t *testing.T) {
	_, service, _ := newTestHandlerWithService(t, nil, nil)
	factory := &captureRuntimeOverrideFactory{
		completer: &workflowTestCompleter{
			response: &llm.CompletionResponse{
				Message:      llm.Message{Role: llm.RoleAssistant, Text: "ok"},
				FinishReason: llm.FinishStop,
			},
		},
		registry: tools.NewRegistry(),
	}
	service.runtimeFactory = factory
	service.agentRunner = NewSessionAgentRunner(
		factory,
		service.configStore,
		service.sessionStore,
		service.runRegistry,
	)

	_, err := service.executeAgentActionWithRuntimeOverrides(
		context.Background(),
		agentParams{Message: "run with missing preset"},
		&TaskRuntimeOverrides{PresetID: "ghost-preset"},
		"trace-missing-preset-run",
	)
	if err == nil || !strings.Contains(err.Error(), `preset_id "ghost-preset" is not configured`) {
		t.Fatalf("expected missing preset runtime error, got %v", err)
	}
}

func configureRuntimeOverrideProviders(t *testing.T, service *bridgeService) {
	t.Helper()
	if err := service.configStore.AddProvider(bridgeconfig.ProviderRecord{
		Name:    "openai-main",
		Type:    llm.ProviderOpenAI,
		BaseURL: "https://api.openai.com/v1",
		APIKey:  stringPointer("openai-key"),
		Models:  []string{"gpt-5.4"},
	}); err != nil {
		t.Fatalf("add openai provider: %v", err)
	}
	if err := service.configStore.AddProvider(bridgeconfig.ProviderRecord{
		Name:    "anthropic-main",
		Type:    llm.ProviderAnthropic,
		BaseURL: "https://api.anthropic.com",
		APIKey:  stringPointer("anthropic-key"),
		Models:  []string{"claude-3.7"},
	}); err != nil {
		t.Fatalf("add anthropic provider: %v", err)
	}
	if err := service.configStore.SetActiveProvider("openai-main"); err != nil {
		t.Fatalf("set active provider: %v", err)
	}
}

type captureRuntimeOverrideFactory struct {
	baseConfig bridgeconfig.Config
	configs    []bridgeconfig.Config
	completer  *workflowTestCompleter
	registry   *tools.Registry
	buildError error
}

func (f *captureRuntimeOverrideFactory) Build(store bridgeconfig.Store) (agentRuntimeDependencies, error) {
	if f.buildError != nil {
		return agentRuntimeDependencies{}, f.buildError
	}
	cfg, err := store.Config()
	if err != nil {
		return agentRuntimeDependencies{}, err
	}
	cfg = applyRuntimeOverrideBaseConfig(cfg, f.baseConfig)
	f.configs = append(f.configs, cfg)
	return NewRuntimeDependencies(cfg, f.completer, f.registry, "global prompt", nil), nil
}

func applyRuntimeOverrideBaseConfig(
	cfg bridgeconfig.Config,
	base bridgeconfig.Config,
) bridgeconfig.Config {
	if cfg.MaxTurns == 0 && base.MaxTurns > 0 {
		cfg.MaxTurns = base.MaxTurns
	}
	if len(cfg.ToolSelector.Allowlist) == 0 && len(base.ToolSelector.Allowlist) > 0 {
		cfg.ToolSelector.Allowlist = append([]string(nil), base.ToolSelector.Allowlist...)
	}
	if !cfg.ToolSelector.AllowlistOnly && base.ToolSelector.AllowlistOnly {
		cfg.ToolSelector.AllowlistOnly = true
	}
	if len(cfg.ToolSelector.Blocklist) == 0 && len(base.ToolSelector.Blocklist) > 0 {
		cfg.ToolSelector.Blocklist = append([]string(nil), base.ToolSelector.Blocklist...)
	}
	if cfg.ToolSearch.IdleTurns == 0 && base.ToolSearch.IdleTurns > 0 {
		cfg.ToolSearch.IdleTurns = base.ToolSearch.IdleTurns
	}
	return cfg
}

func activePromptID(
	library []bridgeconfig.SystemPromptLibraryItem,
	insertPoint bridgeconfig.SystemPromptInsertPoint,
) string {
	for _, item := range library {
		if item.InsertPoint == insertPoint && item.Active {
			return item.ID
		}
	}
	return ""
}

func activePromptIDs(
	library []bridgeconfig.SystemPromptLibraryItem,
	insertPoint bridgeconfig.SystemPromptInsertPoint,
) []string {
	ids := make([]string, 0, len(library))
	for _, item := range library {
		if item.InsertPoint == insertPoint && item.Active {
			ids = append(ids, item.ID)
		}
	}
	return ids
}

func boolPointer(value bool) *bool {
	return &value
}

func intPointer(value int) *int {
	return &value
}

func stringPointer(value string) *string {
	return &value
}

func TestOrchestrationOwnerPublicDispatchAppendsSharedTranscript(t *testing.T) {
	_, service, _ := newTestHandlerWithService(t, nil, nil)
	installOwnerDispatchRuntime(
		service,
		dispatchToolResponse("dispatch-1", `{"action":"public_once","participant_ids":["agent-2"],"instruction":"speak now"}`),
		dispatchToolResponse("dispatch-2", `{"action":"end_group"}`),
	)

	run := runOrchestrationTaskNow(t, service, buildOwnerDefinition("agent-1", 3))
	groupOutput := findNodeOutput(t, run.Run.NodeResults, "group-1")
	sharedTranscript := outputSlice(t, groupOutput, "shared_transcript")
	if len(sharedTranscript) != 1 {
		t.Fatalf("expected one public transcript entry, got %#v", sharedTranscript)
	}
	entry := sharedTranscript[0].(map[string]any)
	if entry["agent_id"] != "agent-2" || entry["content"] != "ok" {
		t.Fatalf("expected public member result in shared transcript, got %#v", entry)
	}
}

func TestOrchestrationOwnerPrivateDispatchDoesNotAppendSharedTranscript(t *testing.T) {
	_, service, _ := newTestHandlerWithService(t, nil, nil)
	installOwnerDispatchRuntime(
		service,
		dispatchToolResponse("dispatch-1", `{"action":"private_once","participant_ids":["agent-1","agent-2"],"instruction":"private chat"}`),
		dispatchToolResponse("dispatch-2", `{"action":"end_group"}`),
	)

	run := runOrchestrationTaskNow(t, service, buildOwnerDefinition("agent-1", 3))
	groupOutput := findNodeOutput(t, run.Run.NodeResults, "group-1")
	if sharedTranscript := outputSlice(t, groupOutput, "shared_transcript"); len(sharedTranscript) != 0 {
		t.Fatalf("expected private dispatch to leave shared transcript empty, got %#v", sharedTranscript)
	}
	dispatchResults := outputSlice(t, groupOutput, "dispatch_results")
	privateTranscript := dispatchResults[0].(map[string]any)["private_transcript"].([]any)
	if len(privateTranscript) == 0 {
		t.Fatalf("expected owner-visible private transcript in dispatch log, got %#v", dispatchResults[0])
	}
}

func TestOrchestrationOwnerEndGroupStopsBeforeMemberDispatch(t *testing.T) {
	agentCalls := 0
	executor := func(_ context.Context, _ string, _ string, _ string, _ bridgeconfig.Store, _ *session.Store) (string, string, error) {
		agentCalls++
		return "unexpected member dispatch", "member-session", nil
	}
	_, service, _ := newTestHandlerWithService(t, executor, nil)
	installOwnerDispatchRuntime(service, dispatchToolResponse("dispatch-1", `{"action":"end_group"}`))

	run := runOrchestrationTaskNow(t, service, buildOwnerDefinition("agent-1", 3))
	groupOutput := findNodeOutput(t, run.Run.NodeResults, "group-1")
	if agentCalls != 0 {
		t.Fatalf("expected end_group to skip member dispatch, got %d calls", agentCalls)
	}
	if completed := outputInt(t, groupOutput, "completed_rounds"); completed != 0 {
		t.Fatalf("expected completed_rounds=0 after immediate end_group, got %d", completed)
	}
	if memberResults := outputSlice(t, groupOutput, "member_results"); len(memberResults) != 0 {
		t.Fatalf("expected no member results after immediate end_group, got %#v", memberResults)
	}
	dispatchResults := outputSlice(t, groupOutput, "dispatch_results")
	if len(dispatchResults) != 1 || dispatchResults[0].(map[string]any)["action"] != "end_group" {
		t.Fatalf("expected single end_group dispatch result, got %#v", dispatchResults)
	}
}

func TestOrchestrationMemberRuntimeOverridesApplyToAgentRuntime(t *testing.T) {
	_, service, _ := newTestHandlerWithService(t, nil, nil)
	configureRuntimeOverrideProviders(t, service)
	preset := createRuntimeOverridePreset(t, service)
	completer := &workflowTestCompleter{
		response: &llm.CompletionResponse{
			Message:      llm.Message{Role: llm.RoleAssistant, Text: "member ok"},
			FinishReason: llm.FinishStop,
		},
	}
	registry := tools.NewRegistry()
	registry.Register(&workflowTestTool{name: "script_exec", output: `{"status":"ok"}`})
	registry.Register(&workflowTestTool{name: "web_search", output: `{"status":"ok"}`})
	factory := &captureRuntimeOverrideFactory{completer: completer, registry: registry}
	service.runtimeFactory = factory
	service.agentRunner = NewSessionAgentRunner(factory, service.configStore, service.sessionStore, service.runRegistry)

	run := runOrchestrationTaskNow(t, service, buildSingleMemberDefinitionWithOverrides(&TaskRuntimeOverrides{
		ProviderName:      "anthropic-main",
		Model:             "claude-3.7",
		PresetID:          preset.ID,
		ToolAllowlistOnly: boolPointer(true),
		ToolAllowlist:     []string{"script_exec"},
		MaxTurns:          intPointer(9),
	}))

	assertMemberRuntimeRun(t, run, factory, completer, preset.ID)
}

func assertMemberRuntimeRun(
	t *testing.T,
	run taskRunPayload,
	factory *captureRuntimeOverrideFactory,
	completer *workflowTestCompleter,
	presetID string,
) {
	t.Helper()
	if run.Run.Status != taskRunStatusSuccess {
		t.Fatalf("unexpected run status: %#v", run.Run)
	}
	assertCapturedMemberRuntimeConfig(t, factory)
	assertMemberCompletionRequest(t, completer)
	groupOutput := findNodeOutput(t, run.Run.NodeResults, "group-1")
	memberResult := outputSlice(t, groupOutput, "member_results")[0].(map[string]any)
	runtimeOverrides := memberResult["runtime_overrides"].(map[string]any)
	if runtimeOverrides["preset_id"] != presetID {
		t.Fatalf("expected member preset_id snapshot, got %#v", runtimeOverrides)
	}
	if _, exists := runtimeOverrides["max_turns"]; exists {
		t.Fatalf("expected orchestration member max_turns to be stripped, got %#v", runtimeOverrides)
	}
}

func assertCapturedMemberRuntimeConfig(t *testing.T, factory *captureRuntimeOverrideFactory) {
	t.Helper()
	if len(factory.configs) != 1 {
		t.Fatalf("expected one runtime config build, got %#v", factory.configs)
	}
	cfg := factory.configs[0]
	if cfg.Provider.Type != llm.ProviderAnthropic || cfg.Provider.Model != "claude-3.7" {
		t.Fatalf("unexpected provider/model override config: %+v", cfg.Provider)
	}
	if !cfg.ToolSelector.AllowlistOnly || strings.Join(cfg.ToolSelector.Allowlist, ",") != "script_exec" {
		t.Fatalf("unexpected member tool allowlist config: %+v", cfg.ToolSelector)
	}
}

func assertMemberCompletionRequest(t *testing.T, completer *workflowTestCompleter) {
	t.Helper()
	if len(completer.requests) != 1 {
		t.Fatalf("expected one completion request, got %d", len(completer.requests))
	}
	request := completer.requests[0]
	if strings.Join(completionToolNames(request.Tools), ",") != "script_exec" {
		t.Fatalf("unexpected completion tool scope: %#v", request.Tools)
	}
	systemPrompt := request.Messages[0].Text
	if !strings.Contains(systemPrompt, "preset rule") ||
		!strings.Contains(systemPrompt, "preset core") ||
		!strings.Contains(systemPrompt, "preset context") {
		t.Fatalf("expected preset prompt in member runtime request, got %q", systemPrompt)
	}
}

func installOwnerDispatchRuntime(service *bridgeService, responses ...*llm.CompletionResponse) *proTestCompleter {
	completer := &proTestCompleter{responses: responses}
	service.runtimeFactory = proTestRuntimeFactory{
		deps: agentRuntimeDependencies{
			cfg:          bridgeconfig.Config{MaxTurns: 4, Provider: bridgeconfig.ProviderConfig{Model: "gpt-5.4"}},
			client:       completer,
			registry:     tools.NewRegistry(),
			systemPrompt: "system prompt",
		},
	}
	return completer
}

func dispatchToolResponse(id string, args string) *llm.CompletionResponse {
	return &llm.CompletionResponse{
		Message: llm.Message{
			Role: llm.RoleAssistant,
			ToolCalls: []llm.ToolCall{{
				ID:        id,
				Name:      orchestrationDispatchToolName,
				Arguments: json.RawMessage(args),
			}},
		},
		FinishReason: llm.FinishToolCalls,
	}
}

func createRuntimeOverridePreset(t *testing.T, service *bridgeService) bridgeconfig.Preset {
	t.Helper()
	library := []bridgeconfig.SystemPromptLibraryItem{
		{ID: "rule-preset", Name: "Rule Preset", InsertPoint: bridgeconfig.SystemPromptInsertPointRule, Content: "preset rule", Active: false},
		{ID: "core-preset", Name: "Core Preset", InsertPoint: bridgeconfig.SystemPromptInsertPointCoreJob, Content: "preset core", Active: false},
		{ID: "context-preset", Name: "Context Preset", InsertPoint: bridgeconfig.SystemPromptInsertPointContext, Content: "preset context", Active: false},
	}
	if _, err := service.configStore.UpdateSystemPrompts(bridgeconfig.SystemPromptUpdateRequest{PromptLibrary: &library}); err != nil {
		t.Fatalf("seed prompt library: %v", err)
	}
	preset, err := service.configStore.CreatePreset(bridgeconfig.PresetCreateRequest{
		Name:          "Research",
		ToolAllowlist: []string{},
		PromptRefs: bridgeconfig.PresetPromptRefs{
			Rule:    "rule-preset",
			CoreJob: "core-preset",
			Context: []string{"context-preset"},
		},
	})
	if err != nil {
		t.Fatalf("create preset: %v", err)
	}
	return preset
}

func completionToolNames(defs []llm.ToolDef) []string {
	names := make([]string, 0, len(defs))
	for _, def := range defs {
		names = append(names, def.Name)
	}
	return names
}

func outputSlice(t *testing.T, output map[string]any, key string) []any {
	t.Helper()
	if output[key] == nil {
		return nil
	}
	items, ok := output[key].([]any)
	if !ok {
		t.Fatalf("expected %s slice in output, got %#v", key, output[key])
	}
	return items
}

func outputInt(t *testing.T, output map[string]any, key string) int {
	t.Helper()
	switch value := output[key].(type) {
	case int:
		return value
	case float64:
		return int(value)
	default:
		t.Fatalf("expected %s int in output, got %#v", key, output[key])
		return 0
	}
}

func TestOrchestrationOwnerRepairsEmptyResponseIntoDispatch(t *testing.T) {
	_, service, _ := newTestHandlerWithService(t, nil, nil)
	completer := installOwnerDispatchRuntime(
		service,
		ownerAssistantResponse(""),
		dispatchToolResponse("dispatch-1", `{"action":"end_group"}`),
	)

	run := runOrchestrationTaskNow(t, service, buildOwnerDefinition("agent-1", 1))
	if run.Run.Status != taskRunStatusSuccess {
		t.Fatalf("expected repaired owner run to succeed, got %#v", run.Run)
	}
	groupOutput := findNodeOutput(t, run.Run.NodeResults, "group-1")
	dispatchResults := groupOutput["dispatch_results"].([]any)
	if len(dispatchResults) != 1 || dispatchResults[0].(map[string]any)["action"] != "end_group" {
		t.Fatalf("expected repaired end_group dispatch, got %#v", dispatchResults)
	}
	if len(completer.requests) != 2 {
		t.Fatalf("expected owner repair round, got %#v", completer.requests)
	}
	repairRequest := completer.requests[1]
	if repairRequest.ToolChoice != "" {
		t.Fatalf("expected repair request to keep free tool choice, got %#v", repairRequest)
	}
	lastMessage := repairRequest.Messages[len(repairRequest.Messages)-1].Text
	if !strings.Contains(lastMessage, "(empty response)") || !strings.Contains(lastMessage, "did not advance the owner-led orchestration yet") {
		t.Fatalf("unexpected owner repair prompt: %q", lastMessage)
	}
}

func TestOrchestrationOwnerFailureRetainsOwnerSessionIDAfterRepairAttempt(t *testing.T) {
	_, service, _ := newTestHandlerWithService(t, nil, nil)
	completer := installOwnerDispatchRuntime(
		service,
		ownerAssistantResponse(""),
		ownerAssistantResponse("我还是不调用工具。"),
	)

	run := runOrchestrationTaskNow(t, service, buildOwnerDefinition("agent-1", 1))
	if run.Run.Status != taskRunStatusError {
		t.Fatalf("expected owner run to fail after invalid repair, got %#v", run.Run)
	}
	groupOutput := findNodeOutput(t, run.Run.NodeResults, "group-1")
	if strings.TrimSpace(groupOutput["owner_session_id"].(string)) == "" {
		t.Fatalf("expected failed owner run to retain owner_session_id, got %#v", groupOutput)
	}
	if len(completer.requests) != 2 || completer.requests[1].ToolChoice != "" {
		t.Fatalf("expected repair request before failure, got %#v", completer.requests)
	}
}

func ownerAssistantResponse(text string) *llm.CompletionResponse {
	return &llm.CompletionResponse{
		Message: llm.Message{
			Role: llm.RoleAssistant,
			Text: text,
		},
		FinishReason: llm.FinishStop,
	}
}

func TestOrchestrationOwnerPrivateSendReachesOnlyRecipient(t *testing.T) {
	executor := privateSendRecipientExecutor()
	_, service, _ := newTestHandlerWithService(t, executor, nil)
	installPrivateSendOwnerRuntime(service)

	run := runOrchestrationTaskNow(t, service, buildOwnerDefinition("agent-1", 3))
	groupOutput := findNodeOutput(t, run.Run.NodeResults, "group-1")
	dispatches := groupOutput["dispatch_results"].([]any)
	first := dispatches[0].(map[string]any)
	if first["action"] != "private_send" {
		t.Fatalf("expected private_send first, got %#v", first)
	}
	deliveries := first["private_deliveries"].([]any)
	if len(deliveries) != 1 || deliveries[0].(map[string]any)["content"] != "secret-role" {
		t.Fatalf("expected visible private delivery content, got %#v", first)
	}
	member := dispatches[1].(map[string]any)["member_results"].([]any)[0].(map[string]any)
	if member["content"] != "saw-secret" {
		t.Fatalf("expected recipient to see private content, got %#v", member)
	}
	assertRunTranscriptSession(t, service, run.Run.SessionIDOutput, nil, []string{"私聊投递: agent-2 <- secret-role"})
	loaded, err := service.sessionStore.Load(run.Run.SessionIDOutput)
	if err != nil {
		t.Fatalf("load run transcript session %q: %v", run.Run.SessionIDOutput, err)
	}
	transcript := sessionMessagesText(loaded.Messages)
	required := []string{
		"tool_call: orchestration_dispatch {\"action\":\"private_send\",\"private_messages\":[{\"content\":\"secret-role\",\"participant_id\":\"agent-2\"}]}",
		"tool: {\"status\":\"success\",\"tool\":\"orchestration_dispatch\"",
		"secret-role",
	}
	for _, expected := range required {
		if !strings.Contains(transcript, expected) {
			t.Fatalf("run transcript missing %q in:\n%s", expected, transcript)
		}
	}
}

func privateSendRecipientExecutor() agentExecutorFunc {
	return func(_ context.Context, message string, _ string, _ string, _ bridgeconfig.Store, _ *session.Store) (string, string, error) {
		if strings.Contains(message, "secret-role") {
			return "saw-secret", "member-session", nil
		}
		return "missing-secret", "member-session", nil
	}
}

func installPrivateSendOwnerRuntime(service *bridgeService) {
	completer := &proTestCompleter{responses: []*llm.CompletionResponse{
		dispatchToolResponse("dispatch-1", `{"action":"private_send","private_messages":[{"participant_id":"agent-2","content":"secret-role"}]}`),
		dispatchToolResponse("dispatch-2", `{"action":"public_once","participant_ids":["agent-2"],"instruction":"speak now"}`),
		dispatchToolResponse("dispatch-3", `{"action":"end_group"}`),
	}}
	service.runtimeFactory = proTestRuntimeFactory{
		deps: agentRuntimeDependencies{
			cfg:          bridgeconfig.Config{MaxTurns: 4, Provider: bridgeconfig.ProviderConfig{Model: "gpt-5.4"}},
			client:       completer,
			registry:     tools.NewRegistry(),
			systemPrompt: "system prompt",
		},
	}
}

func TestOrchestrationSequentialRoundSeesPreviousMemberOutput(t *testing.T) {
	executor := func(_ context.Context, message string, sessionID string, _ string, _ bridgeconfig.Store, _ *session.Store) (string, string, error) {
		switch {
		case strings.Contains(message, "agent-a"):
			return "alpha", "session-a", nil
		case strings.Contains(message, "agent-b"):
			if strings.Contains(message, "A: alpha") {
				return "saw-alpha", "session-b", nil
			}
			return "missing-alpha", "session-b", nil
		default:
			return "unknown", "session-x", nil
		}
	}
	_, service, _ := newTestHandlerWithService(t, executor, nil)
	run := runOrchestrationTaskNow(t, service, buildTestOrchestrationDefinition(orchestrationModeSequential, 1))
	groupOutput := findNodeOutput(t, run.Run.NodeResults, "group-1")
	memberResults := groupOutput["member_results"].([]any)
	lastResult := memberResults[len(memberResults)-1].(map[string]any)
	if lastResult["content"] != "saw-alpha" {
		t.Fatalf("expected sequential member to see previous transcript, got %#v", lastResult)
	}
	assertRunTranscriptSession(t, service, run.Run.SessionIDOutput, nil, []string{
		taskRunTranscriptEventMarker,
		"编排进入群组：Group 1（group-1）",
		"编排进入第 1 轮",
		"A（agent-1） · 第 1 轮",
		"B（agent-2） · 第 1 轮",
	})
}

func TestOrchestrationParallelRoundUsesSharedSnapshot(t *testing.T) {
	executor := func(_ context.Context, message string, sessionID string, _ string, _ bridgeconfig.Store, _ *session.Store) (string, string, error) {
		switch {
		case strings.Contains(message, "agent-a"):
			return "alpha", "session-a", nil
		case strings.Contains(message, "agent-b"):
			if strings.Contains(message, "A: alpha") {
				return "snapshot-leaked", "session-b", nil
			}
			return "snapshot-clean", "session-b", nil
		default:
			return "unknown", "session-x", nil
		}
	}
	_, service, _ := newTestHandlerWithService(t, executor, nil)
	run := runOrchestrationTaskNow(t, service, buildTestOrchestrationDefinition(orchestrationModeParallel, 1))
	groupOutput := findNodeOutput(t, run.Run.NodeResults, "group-1")
	memberResults := groupOutput["member_results"].([]any)
	lastResult := memberResults[len(memberResults)-1].(map[string]any)
	if lastResult["content"] != "snapshot-clean" {
		t.Fatalf("expected parallel member snapshot isolation, got %#v", lastResult)
	}
}

func TestOrchestrationReusesMemberSessionWithinRun(t *testing.T) {
	sessionInputs := make([]string, 0, 2)
	executor := func(_ context.Context, _ string, sessionID string, _ string, _ bridgeconfig.Store, sessionStore *session.Store) (string, string, error) {
		sessionInputs = append(sessionInputs, sessionID)
		if sessionID == "" {
			sess := session.NewSession("")
			if err := sessionStore.Save(sess); err != nil {
				return "", "", err
			}
			return "loop", sess.ID, nil
		}
		return "loop", sessionID, nil
	}
	_, service, _ := newTestHandlerWithService(t, executor, nil)
	definition := buildSingleMemberDefinition(2)
	run := runOrchestrationTaskNow(t, service, definition)
	if len(sessionInputs) != 2 || sessionInputs[0] != "" || sessionInputs[1] == "" {
		t.Fatalf("unexpected session reuse inputs: %#v", sessionInputs)
	}
	groupOutput := findNodeOutput(t, run.Run.NodeResults, "group-1")
	sessions := groupOutput["member_session_ids"].(map[string]any)
	if sessions["agent-1"] != sessionInputs[1] {
		t.Fatalf("unexpected member sessions: %#v", sessions)
	}
}

func TestOrchestrationMemberFailureRunTranscriptReplaysPersistedToolMessages(t *testing.T) {
	_, service, _ := newTestHandlerWithService(t, nil, nil)
	seedFailedMemberToolSession(t, service.sessionStore, "failed-member-session")
	service.agentRunner = &workflowTestRunner{
		sessionID: "failed-member-session",
		err:       errors.New("trace_id=trace-member turn=0 complete_once: read response body: context deadline exceeded"),
	}

	run := runOrchestrationTaskNow(t, service, buildSingleMemberDefinition(1))
	groupOutput := findNodeOutput(t, run.Run.NodeResults, "group-1")
	memberResults := groupOutput["member_results"].([]any)
	member := memberResults[0].(map[string]any)
	if member["session_id"] != "failed-member-session" {
		t.Fatalf("expected failed member session id to be preserved, got %#v", member)
	}

	assertRunTranscriptSession(t, service, run.Run.SessionIDOutput, nil, []string{
		"成员状态：Looper（agent-1） · 第 1 轮 -> error",
		"tool_call: read_file {\"path\":\"README.md\"}",
		"tool: {\"status\":\"success\",\"tool\":\"read_file\"",
	})
}

func TestOrchestrationLegacyBoundaryNodesStillRun(t *testing.T) {
	_, service, _ := newTestHandlerWithService(t, nil, nil)
	run := runOrchestrationTaskNow(t, service, buildLegacyBoundaryDefinition(orchestrationModeSequential, 1))
	if run.Run.Status != taskRunStatusSuccess {
		t.Fatalf("unexpected legacy run status: %#v", run.Run)
	}
	for _, result := range run.Run.NodeResults {
		if result.NodeType == orchestrationNodeTypeStart || result.NodeType == orchestrationNodeTypeEnd {
			t.Fatalf("legacy boundary nodes should not appear in run results: %#v", run.Run.NodeResults)
		}
	}
}

func TestOrchestrationMemberResultsIncludePresetIDAndOmitMaxTurns(t *testing.T) {
	_, service, _ := newTestHandlerWithService(t, nil, nil)
	preset, err := service.configStore.CreatePreset(bridgeconfig.PresetCreateRequest{
		Name:          "Research",
		ToolAllowlist: []string{},
	})
	if err != nil {
		t.Fatalf("create preset: %v", err)
	}

	run := runOrchestrationTaskNow(t, service, buildSingleMemberDefinitionWithOverrides(&TaskRuntimeOverrides{
		PresetID: preset.ID,
		MaxTurns: intPointer(3),
	}))
	groupOutput := findNodeOutput(t, run.Run.NodeResults, "group-1")
	memberResults := groupOutput["member_results"].([]any)
	runtimeOverrides := memberResults[0].(map[string]any)["runtime_overrides"].(map[string]any)
	if runtimeOverrides["preset_id"] != preset.ID {
		t.Fatalf("expected preset_id in member runtime snapshot, got %#v", runtimeOverrides)
	}
	if _, exists := runtimeOverrides["max_turns"]; exists {
		t.Fatalf("expected max_turns to be omitted from orchestration member snapshot, got %#v", runtimeOverrides)
	}
}

func TestOrchestrationOwnerPublicOnceReturnsToOwnerAndLogsDispatch(t *testing.T) {
	_, service, _ := newTestHandlerWithService(t, nil, nil)
	completer := &proTestCompleter{
		responses: []*llm.CompletionResponse{
			{
				Message: llm.Message{
					Role: llm.RoleAssistant,
					ToolCalls: []llm.ToolCall{{
						ID:        "dispatch-1",
						Name:      orchestrationDispatchToolName,
						Arguments: json.RawMessage(`{"action":"public_once","participant_ids":["agent-2"],"instruction":"speak now"}`),
					}},
				},
				FinishReason: llm.FinishToolCalls,
			},
			{
				Message: llm.Message{
					Role: llm.RoleAssistant,
					ToolCalls: []llm.ToolCall{{
						ID:        "dispatch-2",
						Name:      orchestrationDispatchToolName,
						Arguments: json.RawMessage(`{"action":"end_group"}`),
					}},
				},
				FinishReason: llm.FinishToolCalls,
			},
		},
	}
	service.runtimeFactory = proTestRuntimeFactory{
		deps: agentRuntimeDependencies{
			cfg:          bridgeconfig.Config{MaxTurns: 4, Provider: bridgeconfig.ProviderConfig{Model: "gpt-5.4"}},
			client:       completer,
			registry:     tools.NewRegistry(),
			systemPrompt: "system prompt",
		},
	}
	run := runOrchestrationTaskNow(t, service, buildOwnerDefinition("agent-1", 3))
	groupOutput := findNodeOutput(t, run.Run.NodeResults, "group-1")
	dispatchResults := groupOutput["dispatch_results"].([]any)
	if len(dispatchResults) != 2 {
		t.Fatalf("expected public dispatch and end_group to be logged, got %#v output=%#v", dispatchResults, groupOutput)
	}
	dispatch := dispatchResults[0].(map[string]any)
	if dispatch["action"] != "public_once" {
		t.Fatalf("unexpected dispatch action: %#v", dispatch)
	}
	memberResults := dispatch["member_results"].([]any)
	if len(memberResults) != 1 {
		t.Fatalf("unexpected member results: %#v", dispatch)
	}
	member := memberResults[0].(map[string]any)
	if member["agent_id"] != "agent-2" {
		t.Fatalf("unexpected public participant: %#v", member)
	}
	if groupOutput["owner_agent_id"] != "agent-1" {
		t.Fatalf("missing owner_agent_id in output: %#v", groupOutput)
	}
	if strings.TrimSpace(groupOutput["owner_session_id"].(string)) == "" {
		t.Fatalf("expected owner_session_id in output: %#v", groupOutput)
	}
	lastDispatch := dispatchResults[1].(map[string]any)
	if lastDispatch["action"] != "end_group" {
		t.Fatalf("expected end_group to be logged, got %#v", lastDispatch)
	}
	assertRunTranscriptSession(t, service, run.Run.SessionIDOutput, nil, []string{
		"编排第 1 轮调度：public_once",
		"参与者: agent-2",
		"指令: speak now",
		"tool_call: orchestration_dispatch {\"action\":\"public_once\",\"instruction\":\"speak now\",\"participant_ids\":[\"agent-2\"]}",
		"tool: {\"status\":\"success\",\"tool\":\"orchestration_dispatch\"",
		"Member（agent-2） · 第 1 轮",
	})
}

func TestOrchestrationOwnerPrivateDispatchStaysOutOfSharedTranscript(t *testing.T) {
	_, service, _ := newTestHandlerWithService(t, nil, nil)
	completer := &proTestCompleter{
		responses: []*llm.CompletionResponse{
			{
				Message: llm.Message{
					Role: llm.RoleAssistant,
					ToolCalls: []llm.ToolCall{{
						ID:        "dispatch-1",
						Name:      orchestrationDispatchToolName,
						Arguments: json.RawMessage(`{"action":"private_once","participant_ids":["agent-1","agent-2"],"instruction":"private chat"}`),
					}},
				},
				FinishReason: llm.FinishToolCalls,
			},
			{
				Message: llm.Message{
					Role: llm.RoleAssistant,
					ToolCalls: []llm.ToolCall{{
						ID:        "dispatch-2",
						Name:      orchestrationDispatchToolName,
						Arguments: json.RawMessage(`{"action":"end_group"}`),
					}},
				},
				FinishReason: llm.FinishToolCalls,
			},
		},
	}
	service.runtimeFactory = proTestRuntimeFactory{
		deps: agentRuntimeDependencies{
			cfg:          bridgeconfig.Config{MaxTurns: 4, Provider: bridgeconfig.ProviderConfig{Model: "gpt-5.4"}},
			client:       completer,
			registry:     tools.NewRegistry(),
			systemPrompt: "system prompt",
		},
	}
	run := runOrchestrationTaskNow(t, service, buildOwnerDefinition("agent-1", 3))
	groupOutput := findNodeOutput(t, run.Run.NodeResults, "group-1")
	dispatchResults := groupOutput["dispatch_results"].([]any)
	if len(dispatchResults) != 2 {
		t.Fatalf("expected private dispatch and end_group to be logged, got %#v output=%#v", dispatchResults, groupOutput)
	}
	dispatch := dispatchResults[0].(map[string]any)
	if dispatch["action"] != "private_once" {
		t.Fatalf("unexpected dispatch action: %#v", dispatch)
	}
	privateTranscript, ok := dispatch["private_transcript"].([]any)
	if !ok {
		t.Fatalf("expected private_transcript in dispatch log: %#v", dispatch)
	}
	if len(privateTranscript) == 0 {
		t.Fatalf("expected non-empty private_transcript in dispatch log: %#v", dispatch)
	}
	sharedTranscript, ok := groupOutput["shared_transcript"].([]any)
	if !ok {
		sharedTranscript = []any{}
	}
	for _, item := range sharedTranscript {
		entry := item.(map[string]any)
		if strings.Contains(entry["content"].(string), "private") {
			t.Fatalf("shared transcript leaked private content: %#v", sharedTranscript)
		}
	}
}

func TestOrchestrationOwnerPrivateDispatchRetainsTranscriptWhenOwnerNotIncluded(t *testing.T) {
	_, service, _ := newTestHandlerWithService(t, nil, nil)
	completer := &proTestCompleter{
		responses: []*llm.CompletionResponse{
			{
				Message: llm.Message{
					Role: llm.RoleAssistant,
					ToolCalls: []llm.ToolCall{{
						ID:        "dispatch-1",
						Name:      orchestrationDispatchToolName,
						Arguments: json.RawMessage(`{"action":"private_once","participant_ids":["agent-2","agent-3"],"instruction":"private chat"}`),
					}},
				},
				FinishReason: llm.FinishToolCalls,
			},
		},
	}
	service.runtimeFactory = proTestRuntimeFactory{
		deps: agentRuntimeDependencies{
			cfg:          bridgeconfig.Config{MaxTurns: 4, Provider: bridgeconfig.ProviderConfig{Model: "gpt-5.4"}},
			client:       completer,
			registry:     tools.NewRegistry(),
			systemPrompt: "system prompt",
		},
	}
	run := runOrchestrationTaskNow(t, service, buildOwnerDefinitionWithThirdMember("agent-1", 1))
	groupOutput := findNodeOutput(t, run.Run.NodeResults, "group-1")
	dispatchResults := groupOutput["dispatch_results"].([]any)
	dispatch := dispatchResults[0].(map[string]any)
	privateTranscript, ok := dispatch["private_transcript"].([]any)
	if !ok || len(privateTranscript) == 0 {
		t.Fatalf("expected private_transcript to stay available for user-visible logs: %#v", dispatch)
	}
	if dispatch["owner_visible"] != false {
		t.Fatalf("expected owner_visible=false when owner is absent: %#v", dispatch)
	}
}

func TestOrchestrationOwnerDispatchAppliesRuntimeOverrides(t *testing.T) {
	_, service, _ := newTestHandlerWithService(t, nil, nil)
	configureRuntimeOverrideProviders(t, service)
	completer := &workflowTestCompleter{
		response: &llm.CompletionResponse{
			Message: llm.Message{
				Role: llm.RoleAssistant,
				ToolCalls: []llm.ToolCall{{
					ID:        "dispatch-1",
					Name:      orchestrationDispatchToolName,
					Arguments: json.RawMessage(`{"action":"end_group"}`),
				}},
			},
			FinishReason: llm.FinishToolCalls,
		},
	}
	factory := &captureRuntimeOverrideFactory{
		baseConfig: bridgeconfig.Config{
			MaxTurns: 9,
			Provider: bridgeconfig.ProviderConfig{Model: "baseline-model"},
		},
		completer: completer,
		registry:  tools.NewRegistry(),
	}
	factory.registry.Register(&workflowTestTool{name: "script_exec", output: `{"status":"ok"}`})
	service.runtimeFactory = factory

	ownerDefinition := buildOwnerDefinition("agent-1", 1)
	ownerDefinition.Nodes[1].Agent.RuntimeOverrides = &TaskRuntimeOverrides{
		ProviderName:      "anthropic-main",
		Model:             "claude-3.7",
		SystemPrompt:      "judge override",
		ToolAllowlistOnly: boolPointer(true),
		ToolAllowlist:     []string{"script_exec"},
	}

	run := runOrchestrationTaskNow(t, service, ownerDefinition)
	if run.Run.Status != taskRunStatusSuccess {
		t.Fatalf("unexpected run status: %#v", run.Run)
	}
	if len(factory.configs) != 1 {
		t.Fatalf("expected one runtime build, got %#v", factory.configs)
	}
	cfg := factory.configs[0]
	if cfg.Provider.Model != "claude-3.7" {
		t.Fatalf("expected owner override model, got %#v", cfg.Provider)
	}
	if cfg.Provider.Type != llm.ProviderAnthropic {
		t.Fatalf("expected owner override provider type, got %#v", cfg.Provider)
	}
	if !cfg.ToolSelector.AllowlistOnly {
		t.Fatalf("expected allowlist_only for owner dispatch runtime, got %#v", cfg.ToolSelector)
	}
	if len(cfg.ToolSelector.Allowlist) != 1 || cfg.ToolSelector.Allowlist[0] != "script_exec" {
		t.Fatalf("unexpected owner allowlist: %#v", cfg.ToolSelector)
	}
	if len(completer.requests) != 1 {
		t.Fatalf("expected one owner completion request, got %#v", completer.requests)
	}
	request := completer.requests[0]
	if strings.Join(completionToolNames(request.Tools), ",") != "orchestration_dispatch,script_exec" {
		t.Fatalf("expected owner to see dispatch plus allowlisted tools, got %#v", request.Tools)
	}
	if len(request.Messages) == 0 || request.Messages[0].Role != llm.RoleSystem {
		t.Fatalf("expected system prompt in owner request, got %#v", request.Messages)
	}
	systemPrompt := request.Messages[0].Text
	if !strings.Contains(systemPrompt, "judge override") {
		t.Fatalf("expected owner override prompt prefix, got %q", systemPrompt)
	}
	if !strings.Contains(systemPrompt, "你是当前群组的群主") ||
		!strings.Contains(systemPrompt, "你可以像普通 agent 一样自由分析") {
		t.Fatalf("expected owner control prompt, got %q", systemPrompt)
	}
}

func runOrchestrationTaskNow(t *testing.T, service *bridgeService, definition *OrchestrationDefinition) taskRunPayload {
	t.Helper()
	createdRaw, _, err := service.executeTaskCreateAction(taskCreateParams{
		TaskKind:        taskKindOrchestration,
		Name:            "test-orchestration",
		Orchestration:   definition,
		IntervalSeconds: 60,
		Scope:           taskListScopeOrchestration,
	}, "trace-orchestration-create")
	if err != nil {
		t.Fatalf("create orchestration task: %v", err)
	}
	created := createdRaw.(taskPayload)
	runRaw, _, err := service.executeTaskRunNowAction(taskIDParams{ID: created.ID, Scope: taskListScopeOrchestration}, "trace-orchestration-run")
	if err != nil {
		t.Fatalf("run orchestration task: %v", err)
	}
	return runRaw.(taskRunPayload)
}

func findNodeOutput(t *testing.T, results []RunNodeResult, nodeID string) map[string]any {
	t.Helper()
	for _, result := range results {
		if result.NodeID == nodeID {
			output, ok := result.Output.(map[string]any)
			if !ok {
				t.Fatalf("node %s output type: %#v", nodeID, result.Output)
			}
			return output
		}
	}
	t.Fatalf("node result %s not found", nodeID)
	return nil
}

func buildTestOrchestrationDefinition(mode string, maxRounds int) *OrchestrationDefinition {
	return &OrchestrationDefinition{
		Nodes: []OrchestrationNode{
			{ID: "group-1", Type: orchestrationNodeTypeGroup, Group: &OrchestrationGroupNode{Title: "Group 1", SharedContext: "shared", SpeakingMode: mode, MaxRounds: maxRounds}},
			{ID: "agent-1", Type: orchestrationNodeTypeAgent, Agent: &OrchestrationAgentNode{Title: "A", Message: "agent-a"}},
			{ID: "agent-2", Type: orchestrationNodeTypeAgent, Agent: &OrchestrationAgentNode{Title: "B", Message: "agent-b"}},
		},
		Edges: []OrchestrationEdge{
			{FromNodeID: "agent-1", ToNodeID: "group-1", Kind: orchestrationEdgeKindMember},
			{FromNodeID: "agent-2", ToNodeID: "group-1", Kind: orchestrationEdgeKindMember},
		},
	}
}

func buildSingleMemberDefinition(maxRounds int) *OrchestrationDefinition {
	return &OrchestrationDefinition{
		Nodes: []OrchestrationNode{
			{ID: "group-1", Type: orchestrationNodeTypeGroup, Group: &OrchestrationGroupNode{Title: "Group 1", SharedContext: "", SpeakingMode: orchestrationModeSequential, MaxRounds: maxRounds}},
			{ID: "agent-1", Type: orchestrationNodeTypeAgent, Agent: &OrchestrationAgentNode{Title: "Looper", Message: "loop-agent"}},
		},
		Edges: []OrchestrationEdge{
			{FromNodeID: "agent-1", ToNodeID: "group-1", Kind: orchestrationEdgeKindMember},
		},
	}
}

func buildLegacyBoundaryDefinition(mode string, maxRounds int) *OrchestrationDefinition {
	return &OrchestrationDefinition{
		Nodes: []OrchestrationNode{
			{ID: "start-node", Type: orchestrationNodeTypeStart},
			{ID: "group-1", Type: orchestrationNodeTypeGroup, Group: &OrchestrationGroupNode{Title: "Group 1", SharedContext: "", SpeakingMode: mode, MaxRounds: maxRounds}},
			{ID: "agent-1", Type: orchestrationNodeTypeAgent, Agent: &OrchestrationAgentNode{Title: "Looper", Message: "loop-agent"}},
			{ID: "end-node", Type: orchestrationNodeTypeEnd},
		},
		Edges: []OrchestrationEdge{
			{FromNodeID: "start-node", ToNodeID: "group-1", Kind: orchestrationEdgeKindControl},
			{FromNodeID: "group-1", ToNodeID: "end-node", Kind: orchestrationEdgeKindControl},
			{FromNodeID: "agent-1", ToNodeID: "group-1", Kind: orchestrationEdgeKindMember},
		},
	}
}

func buildOwnerDefinitionWithThirdMember(ownerAgentID string, maxRounds int) *OrchestrationDefinition {
	definition := buildOwnerDefinition(ownerAgentID, maxRounds)
	definition.Nodes = append(definition.Nodes,
		OrchestrationNode{ID: "agent-3", Type: orchestrationNodeTypeAgent, Agent: &OrchestrationAgentNode{Title: "Member 3", Message: "member-3"}},
	)
	definition.Edges = append(definition.Edges,
		OrchestrationEdge{FromNodeID: "agent-3", ToNodeID: "group-1", Kind: orchestrationEdgeKindMember},
	)
	return definition
}

func seedFailedMemberToolSession(t *testing.T, store *session.Store, sessionID string) {
	t.Helper()
	sess := session.NewSession("")
	sess.ID = sessionID
	sess.AddMessage(llm.Message{
		Role: llm.RoleAssistant,
		ToolCalls: []llm.ToolCall{{
			ID:        "call-1",
			Name:      "read_file",
			Arguments: json.RawMessage(`{"path":"README.md"}`),
		}},
	})
	sess.AddMessage(llm.Message{
		Role:       llm.RoleTool,
		ToolCallID: "call-1",
		Text:       agent.FormatToolResult("read_file", "trace-member", "file body", nil),
	})
	if err := store.Save(sess); err != nil {
		t.Fatalf("save failed member tool session: %v", err)
	}
}

func TestValidateTaskDefinitionOrchestrationAcceptsEmptyDraft(t *testing.T) {
	task := ScheduledTask{
		TaskKind:      taskKindOrchestration,
		Name:          "empty-orchestration",
		Orchestration: &OrchestrationDefinition{},
	}
	if err := validateTaskDefinition(&task); err != nil {
		t.Fatalf("validate empty orchestration: %v", err)
	}
}

func TestValidateTaskDefinitionOrchestrationAcceptsLegacyBoundaries(t *testing.T) {
	task := ScheduledTask{
		TaskKind:      taskKindOrchestration,
		Name:          "legacy-orchestration",
		Orchestration: buildLegacyBoundaryDefinition(orchestrationModeSequential, 1),
	}
	if err := validateTaskDefinition(&task); err != nil {
		t.Fatalf("validate legacy orchestration: %v", err)
	}
}

func TestValidateTaskDefinitionOrchestrationRejectsAgentOnlyGraph(t *testing.T) {
	task := ScheduledTask{
		TaskKind: taskKindOrchestration,
		Name:     "broken-orchestration",
		Orchestration: &OrchestrationDefinition{
			Nodes: []OrchestrationNode{
				{ID: "agent-1", Type: orchestrationNodeTypeAgent, Agent: &OrchestrationAgentNode{Title: "Solo", Message: "hello"}},
			},
		},
	}
	err := validateTaskDefinition(&task)
	if err == nil || !strings.Contains(err.Error(), "requires at least 1 group node") {
		t.Fatalf("expected missing-group error, got %v", err)
	}
}

func TestValidateTaskDefinitionOrchestrationOwnerModeRequiresOwnerAgentID(t *testing.T) {
	task := ScheduledTask{
		TaskKind: taskKindOrchestration,
		Name:     "owner-missing",
		Orchestration: &OrchestrationDefinition{
			Nodes: []OrchestrationNode{
				{
					ID:   "group-1",
					Type: orchestrationNodeTypeGroup,
					Group: &OrchestrationGroupNode{
						Title:        "Group 1",
						SpeakingMode: orchestrationModeOwner,
						MaxRounds:    1,
					},
				},
				{ID: "agent-1", Type: orchestrationNodeTypeAgent, Agent: &OrchestrationAgentNode{Title: "Owner", Message: "owner"}},
			},
			Edges: []OrchestrationEdge{
				{FromNodeID: "agent-1", ToNodeID: "group-1", Kind: orchestrationEdgeKindMember},
			},
		},
	}
	err := validateTaskDefinition(&task)
	if err == nil || !strings.Contains(err.Error(), `requires owner_agent_id in owner mode`) {
		t.Fatalf("expected owner_agent_id validation error, got %v", err)
	}
}

func TestValidateTaskDefinitionOrchestrationOwnerModeRequiresOwnerToBeMember(t *testing.T) {
	task := ScheduledTask{
		TaskKind:      taskKindOrchestration,
		Name:          "owner-not-member",
		Orchestration: buildOwnerDefinition("ghost-owner", 1),
	}
	err := validateTaskDefinition(&task)
	if err == nil || !strings.Contains(err.Error(), `owner_agent_id "ghost-owner" must be an existing member`) {
		t.Fatalf("expected owner membership validation error, got %v", err)
	}
}

func TestOrchestrationTaskCreateStripsMemberMaxTurns(t *testing.T) {
	_, service, _ := newTestHandlerWithService(t, nil, nil)

	createdRaw, code, err := service.executeTaskCreateAction(taskCreateParams{
		TaskKind:        taskKindOrchestration,
		Name:            "strip-member-max-turns",
		Orchestration:   buildSingleMemberDefinitionWithOverrides(&TaskRuntimeOverrides{MaxTurns: intPointer(3)}),
		IntervalSeconds: 60,
		Scope:           taskListScopeOrchestration,
	}, "trace-orchestration-strip-max-turns")
	if err != nil || code != 201 {
		t.Fatalf("create orchestration task: code=%d err=%v", code, err)
	}
	created := createdRaw.(taskPayload)

	task, err := service.taskStore().LoadTask(created.ID)
	if err != nil {
		t.Fatalf("load saved orchestration task: %v", err)
	}
	agentNode := task.Orchestration.Nodes[1]
	if agentNode.Agent == nil {
		t.Fatalf("expected agent node payload, got %#v", task.Orchestration.Nodes)
	}
	if agentNode.Agent.RuntimeOverrides != nil && agentNode.Agent.RuntimeOverrides.MaxTurns != nil {
		t.Fatalf("expected orchestration member max_turns to be stripped, got %#v", agentNode.Agent.RuntimeOverrides)
	}
}

func buildSingleMemberDefinitionWithOverrides(overrides *TaskRuntimeOverrides) *OrchestrationDefinition {
	definition := buildSingleMemberDefinition(1)
	definition.Nodes[1].Agent.RuntimeOverrides = overrides
	return definition
}

func buildOwnerDefinition(ownerAgentID string, maxRounds int) *OrchestrationDefinition {
	return &OrchestrationDefinition{
		Nodes: []OrchestrationNode{
			{
				ID:   "group-1",
				Type: orchestrationNodeTypeGroup,
				Group: &OrchestrationGroupNode{
					Title:         "Group 1",
					SharedContext: "",
					SpeakingMode:  orchestrationModeOwner,
					OwnerAgentID:  ownerAgentID,
					MaxRounds:     maxRounds,
				},
			},
			{ID: "agent-1", Type: orchestrationNodeTypeAgent, Agent: &OrchestrationAgentNode{Title: "Owner", Message: "owner-agent"}},
			{ID: "agent-2", Type: orchestrationNodeTypeAgent, Agent: &OrchestrationAgentNode{Title: "Member", Message: "member-agent"}},
		},
		Edges: []OrchestrationEdge{
			{FromNodeID: "agent-1", ToNodeID: "group-1", Kind: orchestrationEdgeKindMember},
			{FromNodeID: "agent-2", ToNodeID: "group-1", Kind: orchestrationEdgeKindMember},
		},
	}
}

func TestRelayAIDecidesStopsAtMaxRounds(t *testing.T) {
	completer := &proTestCompleter{
		responses: []*llm.CompletionResponse{
			relayUpdateResponse("call-relay-1", "round 1", "more"),
			relayUpdateResponse("call-relay-2", "round 2", "more"),
		},
	}
	result, err := runRelayModeTest(t, completer, TaskRelayConfig{
		StopPolicy: taskRelayStopPolicyAIDecides,
		MaxRounds:  2,
	})
	if err != nil {
		t.Fatalf("relay run failed: %v", err)
	}
	if result.StoppedBy != relayModeStopMaxRounds {
		t.Fatalf("unexpected stop reason: %s", result.StoppedBy)
	}
	if len(result.Records) != 2 || len(completer.requests) != 2 {
		t.Fatalf("unexpected rounds: records=%d requests=%d", len(result.Records), len(completer.requests))
	}
	if !strings.Contains(result.Message, "Reached relay max_rounds=2") {
		t.Fatalf("unexpected message: %q", result.Message)
	}
}

func TestRelayAIDecidesCanComplete(t *testing.T) {
	completer := &proTestCompleter{
		responses: []*llm.CompletionResponse{
			relayCompleteResponse("call-relay-complete"),
		},
	}
	result, err := runRelayModeTest(t, completer, TaskRelayConfig{
		StopPolicy: taskRelayStopPolicyAIDecides,
		MaxRounds:  5,
	})
	if err != nil {
		t.Fatalf("relay run failed: %v", err)
	}
	if result.StoppedBy != relayModeStopCompleted || result.Message != "done" {
		t.Fatalf("unexpected result: %+v", result)
	}
	if len(result.Records) != 1 || len(completer.requests) != 1 {
		t.Fatalf("unexpected rounds: records=%d requests=%d", len(result.Records), len(completer.requests))
	}
}

func TestRelayRepairsPlainTextRoundIntoHandoff(t *testing.T) {
	completer := &proTestCompleter{
		responses: []*llm.CompletionResponse{
			relayTextResponse("我已经查到了广州天气，接下来整理结果。"),
			relayUpdateResponse("call-relay-1", "查到了广州天气", "整理成最终答复"),
		},
	}
	result, err := runRelayModeTest(t, completer, TaskRelayConfig{
		StopPolicy: taskRelayStopPolicyMaxRounds,
		MaxRounds:  1,
	})
	if err != nil {
		t.Fatalf("relay run failed: %v", err)
	}
	if result.StoppedBy != relayModeStopMaxRounds {
		t.Fatalf("unexpected stop reason: %s", result.StoppedBy)
	}
	if len(result.Records) != 1 || len(completer.requests) != 2 {
		t.Fatalf("unexpected rounds: records=%d requests=%d", len(result.Records), len(completer.requests))
	}
	repairRequest := completer.requests[1]
	if repairRequest.ToolChoice != "required" {
		t.Fatalf("unexpected repair tool_choice: %+v", repairRequest)
	}
	lastMessage := repairRequest.Messages[len(repairRequest.Messages)-1]
	if !strings.Contains(lastMessage.Text, "invalid for relay mode") {
		t.Fatalf("unexpected repair prompt: %q", lastMessage.Text)
	}
}

func TestRelayReturnsErrorWhenRepairStillDoesNotHandoff(t *testing.T) {
	completer := &proTestCompleter{
		responses: []*llm.CompletionResponse{
			relayTextResponse("先给你一个普通答复。"),
			relayTextResponse("我还是直接回答。"),
		},
	}
	_, err := runRelayModeTest(t, completer, TaskRelayConfig{
		StopPolicy: taskRelayStopPolicyMaxRounds,
		MaxRounds:  1,
	})
	if err == nil {
		t.Fatal("expected relay run to fail")
	}
	if !strings.Contains(err.Error(), "ended without relay handoff tool") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func runRelayModeTest(
	t *testing.T,
	completer *proTestCompleter,
	relay TaskRelayConfig,
) (relayModeResult, error) {
	t.Helper()

	store := newTempSessionStore(t)
	sess := session.NewSession("system")
	runner := relayModeRunner{sessionStore: store}
	deps := agentRuntimeDependencies{
		cfg: bridgeconfig.Config{
			MaxTurns: 4,
		},
		client:               completer,
		registry:             tools.NewRegistry(),
		systemPrompt:         "system",
		systemPromptOverride: true,
	}
	return runner.run(context.Background(), relayRunDeps{
		deps:    deps,
		relay:   relay,
		task:    ScheduledTask{Message: "finish task"},
		session: sess,
		traceID: "trace-relay-test",
	})
}

func relayUpdateResponse(callID string, did string, remaining string) *llm.CompletionResponse {
	return relayToolResponse(callID, "relay_update_record", map[string]any{
		"did":             did,
		"remaining":       remaining,
		"failed_attempts": []string{},
		"next_step":       "continue",
	})
}

func relayCompleteResponse(callID string) *llm.CompletionResponse {
	return relayToolResponse(callID, "relay_complete", map[string]any{
		"did":              "finished",
		"remaining":        "none",
		"failed_attempts":  []string{},
		"next_step":        "none",
		"final_message":    "done",
		"final_change_log": "finished task",
	})
}

func relayToolResponse(callID string, name string, args map[string]any) *llm.CompletionResponse {
	encoded, err := json.Marshal(args)
	if err != nil {
		panic(err)
	}
	return &llm.CompletionResponse{
		Message: llm.Message{
			Role: llm.RoleAssistant,
			ToolCalls: []llm.ToolCall{{
				ID:        callID,
				Name:      name,
				Arguments: encoded,
			}},
		},
		FinishReason: llm.FinishToolCalls,
	}
}

func relayTextResponse(text string) *llm.CompletionResponse {
	return &llm.CompletionResponse{
		Message: llm.Message{
			Role: llm.RoleAssistant,
			Text: text,
		},
		FinishReason: llm.FinishStop,
	}
}

func TestSchemaByToolNameReturnsMissingWhenToolHasNoSchema(t *testing.T) {
	schema, ok := schemaByToolName(map[string]map[string]any{
		"script_exec": {"type": "object"},
	}, "web_search")
	if ok {
		t.Fatal("expected missing schema flag for tool without schema")
	}
	if schema != nil {
		t.Fatalf("expected nil schema for tool without schema, got %#v", schema)
	}
}

func TestSchemaByToolNameReturnsSchemaWhenPresent(t *testing.T) {
	schema, ok := schemaByToolName(map[string]map[string]any{
		"script_exec": {"type": "object"},
	}, "script_exec")
	if !ok {
		t.Fatal("expected schema to exist")
	}
	if got, ok := schema["type"].(string); !ok || got != "object" {
		t.Fatalf("expected object schema type, got %#v", schema["type"])
	}
}

func TestExecuteFindIconTemplateUploadStoresTemplate(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	payload, err := executeFindIconTemplateUpload(findIconTemplateUploadRequest{
		Filename: "icon.png",
		MimeType: "image/png",
		DataURL:  "data:image/png;base64,R2hvc3Q=",
	})
	if err != nil {
		t.Fatalf("executeFindIconTemplateUpload returned error: %v", err)
	}
	if !strings.HasPrefix(payload.TemplatePath, filepath.Join(homeDir, ".ghost-os")) {
		t.Fatalf("unexpected template path: %q", payload.TemplatePath)
	}
	if len(payload.SHA256) != 64 {
		t.Fatalf("unexpected sha256 length: %q", payload.SHA256)
	}
	if _, err := os.Stat(payload.TemplatePath); err != nil {
		t.Fatalf("template path should exist: %v", err)
	}
}

func TestNormalizeFindIconPreviewRequestRejectsInvalidThreshold(t *testing.T) {
	_, err := normalizeFindIconPreviewRequest(findIconPreviewRequest{
		TemplatePath: "/tmp/icon.png",
		Threshold:    floatPtr(1.1),
	})
	if err == nil || !strings.Contains(err.Error(), "threshold must be between 0 and 1") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDecodeFindIconPreviewPayloadParsesMatches(t *testing.T) {
	payload, err := decodeFindIconPreviewPayload(`{"display_id":1,"matches":[{"score":0.95}]}`)
	if err != nil {
		t.Fatalf("decodeFindIconPreviewPayload returned error: %v", err)
	}
	if !payload.Exists || payload.MatchCount != 1 {
		t.Fatalf("unexpected payload: %+v", payload)
	}
	if payload.DisplayID == nil || *payload.DisplayID != 1 {
		t.Fatalf("unexpected display id: %+v", payload.DisplayID)
	}
}

func floatPtr(value float64) *float64 {
	return &value
}

func TestValidateTaskDefinitionWorkflowAcceptsLinearNodeChain(t *testing.T) {
	task := ScheduledTask{
		TaskKind: taskKindWorkflow,
		Workflow: &WorkflowDefinition{
			Nodes: []WorkflowNode{
				{ID: "start-node", Type: workflowNodeTypeStart},
				{ID: "tool-node", Type: workflowNodeTypeTool, Tool: &WorkflowToolNode{ToolName: "script_exec"}},
				{ID: "llm-node", Type: workflowNodeTypeLLM, LLM: &WorkflowLLMNode{Prompt: "summarize"}},
				{ID: "agent-node", Type: workflowNodeTypeAgent, Agent: &WorkflowAgentNode{Message: "reply"}},
				{ID: "end-node", Type: workflowNodeTypeEnd},
			},
			Edges: []WorkflowEdge{
				{FromNodeID: "start-node", ToNodeID: "tool-node"},
				{FromNodeID: "tool-node", ToNodeID: "llm-node"},
				{FromNodeID: "llm-node", ToNodeID: "agent-node"},
				{FromNodeID: "agent-node", ToNodeID: "end-node"},
			},
		},
	}
	if err := validateTaskDefinition(&task); err != nil {
		t.Fatalf("validate workflow chain: %v", err)
	}
}

func TestValidateTaskDefinitionWorkflowAcceptsStartParallelBranches(t *testing.T) {
	task := ScheduledTask{
		TaskKind: taskKindWorkflow,
		Workflow: &WorkflowDefinition{
			Nodes: []WorkflowNode{
				{ID: "start-node", Type: workflowNodeTypeStart},
				{ID: "agent-a", Type: workflowNodeTypeAgent, Agent: &WorkflowAgentNode{Message: "a"}},
				{ID: "agent-b", Type: workflowNodeTypeAgent, Agent: &WorkflowAgentNode{Message: "b"}},
				{ID: "end-node", Type: workflowNodeTypeEnd},
			},
			Edges: []WorkflowEdge{
				{FromNodeID: "start-node", ToNodeID: "agent-a"},
				{FromNodeID: "start-node", ToNodeID: "agent-b"},
				{FromNodeID: "agent-a", ToNodeID: "end-node"},
				{FromNodeID: "agent-b", ToNodeID: "end-node"},
			},
		},
	}
	if err := validateTaskDefinition(&task); err != nil {
		t.Fatalf("validate workflow start parallel branches: %v", err)
	}
}

func TestValidateTaskDefinitionWorkflowRejectsNodePayloadMismatch(t *testing.T) {
	task := ScheduledTask{
		TaskKind: taskKindWorkflow,
		Workflow: &WorkflowDefinition{
			Nodes: []WorkflowNode{
				{ID: "start-node", Type: workflowNodeTypeStart},
				{ID: "tool-node", Type: workflowNodeTypeTool, LLM: &WorkflowLLMNode{Prompt: "bad"}},
				{ID: "end-node", Type: workflowNodeTypeEnd},
			},
			Edges: []WorkflowEdge{
				{FromNodeID: "start-node", ToNodeID: "tool-node"},
				{FromNodeID: "tool-node", ToNodeID: "end-node"},
			},
		},
	}
	err := validateTaskDefinition(&task)
	if err == nil || !strings.Contains(err.Error(), "payload does not match type") {
		t.Fatalf("expected payload mismatch, got %v", err)
	}
}

func TestValidateTaskDefinitionWorkflowRejectsBranchingFromNonControlNode(t *testing.T) {
	task := ScheduledTask{
		TaskKind: taskKindWorkflow,
		Workflow: &WorkflowDefinition{
			Nodes: []WorkflowNode{
				{ID: "start-node", Type: workflowNodeTypeStart},
				{ID: "tool-node", Type: workflowNodeTypeTool, Tool: &WorkflowToolNode{ToolName: "script_exec"}},
				{ID: "llm-node", Type: workflowNodeTypeLLM, LLM: &WorkflowLLMNode{Prompt: "branch"}},
				{ID: "end-node", Type: workflowNodeTypeEnd},
			},
			Edges: []WorkflowEdge{
				{FromNodeID: "start-node", ToNodeID: "tool-node"},
				{FromNodeID: "tool-node", ToNodeID: "llm-node"},
				{FromNodeID: "tool-node", ToNodeID: "end-node"},
				{FromNodeID: "llm-node", ToNodeID: "end-node"},
			},
		},
	}
	err := validateTaskDefinition(&task)
	if err == nil || !strings.Contains(err.Error(), `must have in>=1 and out=1`) {
		t.Fatalf("expected non-control branching failure, got %v", err)
	}
}

func TestValidateTaskDefinitionWorkflowAcceptsIfAndLoopNodes(t *testing.T) {
	task := ScheduledTask{
		TaskKind: taskKindWorkflow,
		Workflow: &WorkflowDefinition{
			Nodes: []WorkflowNode{
				{ID: "start-node", Type: workflowNodeTypeStart},
				{
					ID:   "if-node",
					Type: workflowNodeTypeIf,
					If: &WorkflowIfNode{
						Operator:    workflowIfOperatorIsEmpty,
						TrueNodeID:  "loop-node",
						FalseNodeID: "end-node",
					},
				},
				{
					ID:   "loop-node",
					Type: workflowNodeTypeLoop,
					Loop: &WorkflowLoopNode{
						MaxIterations: 2,
						BodyNodeID:    "tool-node",
						ExitNodeID:    "end-node",
					},
				},
				{ID: "tool-node", Type: workflowNodeTypeTool, Tool: &WorkflowToolNode{ToolName: "script_exec"}},
				{ID: "end-node", Type: workflowNodeTypeEnd},
			},
			Edges: []WorkflowEdge{
				{FromNodeID: "start-node", ToNodeID: "if-node"},
				{FromNodeID: "if-node", ToNodeID: "loop-node"},
				{FromNodeID: "if-node", ToNodeID: "end-node"},
				{FromNodeID: "loop-node", ToNodeID: "tool-node"},
				{FromNodeID: "loop-node", ToNodeID: "end-node"},
				{FromNodeID: "tool-node", ToNodeID: "loop-node"},
			},
		},
	}
	if err := validateTaskDefinition(&task); err != nil {
		t.Fatalf("validate workflow with if/loop: %v", err)
	}
}

func TestValidateTaskDefinitionWorkflowRejectsLoopWithoutCycle(t *testing.T) {
	task := ScheduledTask{
		TaskKind: taskKindWorkflow,
		Workflow: &WorkflowDefinition{
			Nodes: []WorkflowNode{
				{ID: "start-node", Type: workflowNodeTypeStart},
				{
					ID:   "loop-node",
					Type: workflowNodeTypeLoop,
					Loop: &WorkflowLoopNode{
						MaxIterations: 3,
						BodyNodeID:    "tool-node",
						ExitNodeID:    "end-node",
					},
				},
				{ID: "tool-node", Type: workflowNodeTypeTool, Tool: &WorkflowToolNode{ToolName: "script_exec"}},
				{ID: "end-node", Type: workflowNodeTypeEnd},
			},
			Edges: []WorkflowEdge{
				{FromNodeID: "start-node", ToNodeID: "loop-node"},
				{FromNodeID: "loop-node", ToNodeID: "tool-node"},
				{FromNodeID: "loop-node", ToNodeID: "end-node"},
				{FromNodeID: "tool-node", ToNodeID: "end-node"},
			},
		},
	}
	err := validateTaskDefinition(&task)
	if err == nil || !strings.Contains(err.Error(), "body path must return to the loop node") {
		t.Fatalf("expected loop cycle validation failure, got %v", err)
	}
}

func TestTaskWorkflowCreateRejectsToolOutsideAllowlist(t *testing.T) {
	t.Setenv("GHOST_WORKFLOW_TOOL_ALLOWLIST", "script_exec")
	_, service, _ := newTestHandlerWithService(t, nil, nil)

	_, code, err := service.executeTaskCreateAction(taskCreateParams{
		TaskKind:        taskKindWorkflow,
		Workflow:        workflowWithToolNode("web_search"),
		IntervalSeconds: 60,
	}, "trace-workflow-tool-reject")
	if err == nil {
		t.Fatal("expected workflow tool allowlist validation to fail")
	}
	if code != http.StatusBadRequest {
		t.Fatalf("unexpected status code: got %d want %d", code, http.StatusBadRequest)
	}
	if !strings.Contains(err.Error(), `workflow tool "web_search" is not allowed`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateTaskDefinitionWorkflowAcceptsStartInputs(t *testing.T) {
	task := ScheduledTask{
		TaskKind: taskKindWorkflow,
		Workflow: workflowWithStartInputs([]WorkflowInputVariable{
			{
				Name:        "topic",
				Type:        workflowInputTypeString,
				Required:    true,
				Default:     []byte(`"release notes"`),
				Description: "summary topic",
			},
			{
				Name:    "threshold",
				Type:    workflowInputTypeNumber,
				Default: []byte(`0.7`),
			},
			{
				Name:    "flags",
				Type:    workflowInputTypeObject,
				Default: []byte(`{"urgent":true}`),
			},
			{
				Name:    "tags",
				Type:    workflowInputTypeArray,
				Default: []byte(`["ops","weekly"]`),
			},
		}),
	}
	if err := validateTaskDefinition(&task); err != nil {
		t.Fatalf("validate workflow with start inputs: %v", err)
	}
}

func TestValidateTaskDefinitionWorkflowRejectsInvalidStartInputs(t *testing.T) {
	tests := []struct {
		name   string
		inputs []WorkflowInputVariable
		want   string
	}{
		{
			name: "invalid variable name",
			inputs: []WorkflowInputVariable{
				{Name: "1topic", Type: workflowInputTypeString},
			},
			want: "must match ^[A-Za-z_][A-Za-z0-9_]*$",
		},
		{
			name: "duplicate variable names",
			inputs: []WorkflowInputVariable{
				{Name: "topic", Type: workflowInputTypeString},
				{Name: "topic", Type: workflowInputTypeNumber},
			},
			want: "duplicate name",
		},
		{
			name: "default type mismatch",
			inputs: []WorkflowInputVariable{
				{Name: "retry_count", Type: workflowInputTypeNumber, Default: []byte(`"3"`)},
			},
			want: "must match declared type",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			task := ScheduledTask{
				TaskKind: taskKindWorkflow,
				Workflow: workflowWithStartInputs(test.inputs),
			}
			err := validateTaskDefinition(&task)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("expected error containing %q, got %v", test.want, err)
			}
		})
	}
}

func workflowWithStartInputs(inputs []WorkflowInputVariable) *WorkflowDefinition {
	return &WorkflowDefinition{
		Nodes: []WorkflowNode{
			{
				ID:    "start-node",
				Type:  workflowNodeTypeStart,
				Start: &WorkflowStartNode{Inputs: inputs},
			},
			{ID: "end-node", Type: workflowNodeTypeEnd},
		},
		Edges: []WorkflowEdge{{FromNodeID: "start-node", ToNodeID: "end-node"}},
	}
}

func TestPrepareWorkflowToolArgumentsUploadsFindIconTemplate(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)

	input := map[string]any{
		"mode":   "atomic",
		"action": "find_icon",
		"params": map[string]any{
			"workflow_template_data_url": "data:image/png;base64,R2hvc3Q=",
			"template_filename":          "icon.png",
			"template_mime_type":         "image/png",
			"threshold":                  0.91,
		},
	}
	original := cloneTaskActionParams(input)

	prepared, err := prepareWorkflowToolArguments(screenControlToolID, input)
	if err != nil {
		t.Fatalf("prepareWorkflowToolArguments returned error: %v", err)
	}

	params := decodeWorkflowToolParams(t, prepared)
	templatePath := strings.TrimSpace(workflowMapString(params, "template_path"))
	if templatePath == "" {
		t.Fatalf("expected template_path to be generated, got: %+v", params)
	}
	if !strings.HasPrefix(templatePath, filepath.Join(homeDir, ".ghost-os")) {
		t.Fatalf("unexpected template path: %q", templatePath)
	}
	if _, err := os.Stat(templatePath); err != nil {
		t.Fatalf("template path should exist: %v", err)
	}
	if _, exists := params["workflow_template_data_url"]; exists {
		t.Fatalf("workflow_template_data_url should be removed after upload: %+v", params)
	}
	if params["threshold"] != 0.91 {
		t.Fatalf("expected threshold to be preserved, got: %+v", params)
	}
	if !reflect.DeepEqual(input, original) {
		t.Fatalf("input arguments should remain immutable: before=%+v after=%+v", original, input)
	}
}

func TestPrepareWorkflowToolArgumentsRejectsInvalidFindIconUploadData(t *testing.T) {
	_, err := prepareWorkflowToolArguments(screenControlToolID, map[string]any{
		"mode":   "atomic",
		"action": "find_icon",
		"params": map[string]any{
			"workflow_template_data_url": "invalid-data-url",
			"template_mime_type":         "image/png",
		},
	})
	if err == nil || !strings.Contains(err.Error(), "data_url must start with data:") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPrepareWorkflowToolArgumentsRejectsLegacyFindIconUploadKeys(t *testing.T) {
	_, err := prepareWorkflowToolArguments(screenControlToolID, map[string]any{
		"mode":   "atomic",
		"action": "find_icon",
		"params": map[string]any{
			"template_data_url": "data:image/png;base64,R2hvc3Q=",
		},
	})
	if err == nil || !strings.Contains(err.Error(), "legacy find_icon data_url keys are not supported") {
		t.Fatalf("unexpected error for template_data_url: %v", err)
	}

	_, err = prepareWorkflowToolArguments(screenControlToolID, map[string]any{
		"mode":   "atomic",
		"action": "find_icon",
		"params": map[string]any{
			"data_url": "data:image/png;base64,R2hvc3Q=",
		},
	})
	if err == nil || !strings.Contains(err.Error(), "legacy find_icon data_url keys are not supported") {
		t.Fatalf("unexpected error for data_url: %v", err)
	}
}

func TestPrepareWorkflowToolArgumentsSkipsNonScreenControlTools(t *testing.T) {
	input := map[string]any{
		"command": "pwd",
	}
	prepared, err := prepareWorkflowToolArguments("script_exec", input)
	if err != nil {
		t.Fatalf("prepareWorkflowToolArguments returned error: %v", err)
	}
	if !reflect.DeepEqual(prepared, input) {
		t.Fatalf("unexpected prepared arguments: got=%+v want=%+v", prepared, input)
	}
}

func decodeWorkflowToolParams(t *testing.T, arguments map[string]any) map[string]any {
	t.Helper()
	params, ok := arguments["params"].(map[string]any)
	if !ok {
		t.Fatalf("params should be an object, got: %T", arguments["params"])
	}
	return params
}

func TestExecuteWorkflowToolNodeScreenControlWorkflowStepsResolvesFindIconCoordinateRef(t *testing.T) {
	tool := &workflowScreenControlSequenceTool{
		name: screenControlToolID,
		outputs: []string{
			`{"matches":[{"center":{"x":321,"y":654}}],"display_id":5}`,
			`{"status":"clicked"}`,
		},
	}
	registry := tools.NewRegistry()
	registry.Register(tool)

	outcome := executeWorkflowToolNode(
		context.Background(),
		agentRuntimeDependencies{registry: registry},
		buildWorkflowScreenControlNode(map[string]any{
			"workflow_steps": []any{
				map[string]any{"action": "find_icon", "params": map[string]any{"template_path": "/tmp/icon.png"}},
				map[string]any{"action": "click", "params": map[string]any{"coordinate_ref": workflowScreenControlFindIconCoordinateRef}},
			},
		}),
		"trace-workflow-screen-coordinate-ref",
	)

	if outcome.err != nil {
		t.Fatalf("expected success, got error: %v", outcome.err)
	}
	if len(tool.calls) != 2 {
		t.Fatalf("unexpected tool call count: %d", len(tool.calls))
	}
	params, ok := tool.calls[1]["params"].(map[string]any)
	if !ok {
		t.Fatalf("expected second call params, got: %#v", tool.calls[1])
	}
	if params["x"] != float64(321) || params["y"] != float64(654) {
		t.Fatalf("unexpected resolved click coordinates: %#v", params)
	}
	if params["display_id"] != float64(5) {
		t.Fatalf("unexpected resolved display_id: %#v", params)
	}
	if _, exists := params[workflowScreenControlCoordinateRefKey]; exists {
		t.Fatalf("coordinate_ref should be removed before execution: %#v", params)
	}
}

func TestExecuteWorkflowToolNodeScreenControlWorkflowStepsRejectsMissingFindIconReference(t *testing.T) {
	tool := &workflowScreenControlSequenceTool{name: screenControlToolID}
	registry := tools.NewRegistry()
	registry.Register(tool)

	outcome := executeWorkflowToolNode(
		context.Background(),
		agentRuntimeDependencies{registry: registry},
		buildWorkflowScreenControlNode(map[string]any{
			"workflow_steps": []any{
				map[string]any{"action": "click", "params": map[string]any{"coordinate_ref": workflowScreenControlFindIconCoordinateRef}},
			},
		}),
		"trace-workflow-screen-coordinate-ref-missing",
	)

	if outcome.err == nil {
		t.Fatal("expected missing find_icon reference error")
	}
	if !strings.Contains(outcome.err.Error(), "requires a previous find_icon result") {
		t.Fatalf("unexpected error: %v", outcome.err)
	}
	if len(tool.calls) != 0 {
		t.Fatalf("tool should not execute when coordinate_ref is unresolved: %d", len(tool.calls))
	}
}

func TestExecuteWorkflowToolNodeScreenControlWorkflowStepsSequential(t *testing.T) {
	tool := &workflowScreenControlSequenceTool{
		name: screenControlToolID,
		outputs: []string{
			`{"step":"one"}`,
			`{"step":"two"}`,
		},
	}
	registry := tools.NewRegistry()
	registry.Register(tool)

	outcome := executeWorkflowToolNode(
		context.Background(),
		agentRuntimeDependencies{registry: registry},
		buildWorkflowScreenControlNode(map[string]any{
			"mode": "atomic",
			"workflow_steps": []any{
				map[string]any{"action": "screenshot"},
				map[string]any{"action": "click", "params": map[string]any{"x": 10, "y": 20}},
			},
		}),
		"trace-workflow-screen-steps",
	)

	if outcome.err != nil {
		t.Fatalf("expected success, got error: %v", outcome.err)
	}
	if outcome.status != taskRunStatusSuccess {
		t.Fatalf("unexpected status: %q", outcome.status)
	}
	if len(tool.calls) != 2 {
		t.Fatalf("unexpected tool call count: %d", len(tool.calls))
	}
	if tool.calls[0]["action"] != "screenshot" || tool.calls[1]["action"] != "click_icon" {
		t.Fatalf("unexpected call sequence: %#v", tool.calls)
	}
	if len(tool.callTimes) != 2 {
		t.Fatalf("unexpected call time count: %d", len(tool.callTimes))
	}
	const minimumObservedDelay = 180 * time.Millisecond
	if gap := tool.callTimes[1].Sub(tool.callTimes[0]); gap < minimumObservedDelay {
		t.Fatalf("expected inter-step gap >= %s, got %s", minimumObservedDelay, gap)
	}

	payload, ok := outcome.outputValue.(map[string]any)
	if !ok {
		t.Fatalf("expected map output, got %T", outcome.outputValue)
	}
	if payload["step_count"] != 2 {
		t.Fatalf("unexpected step_count: %#v", payload["step_count"])
	}
	steps, ok := payload["steps"].([]map[string]any)
	if !ok || len(steps) != 2 {
		t.Fatalf("unexpected steps payload: %#v", payload["steps"])
	}
	finalOutput, ok := payload["final_output"].(map[string]any)
	if !ok || finalOutput["step"] != "two" {
		t.Fatalf("unexpected final_output: %#v", payload["final_output"])
	}
	if !strings.Contains(outcome.outputText, `"step_count":2`) {
		t.Fatalf("expected structured output_text, got %q", outcome.outputText)
	}
}

func TestExecuteWorkflowToolNodeScreenControlWorkflowStepsFailFast(t *testing.T) {
	tool := &workflowScreenControlSequenceTool{
		name:    screenControlToolID,
		outputs: []string{`{"step":"one"}`},
		failAt:  2,
	}
	registry := tools.NewRegistry()
	registry.Register(tool)

	outcome := executeWorkflowToolNode(
		context.Background(),
		agentRuntimeDependencies{registry: registry},
		buildWorkflowScreenControlNode(map[string]any{
			"workflow_steps": []any{
				map[string]any{"action": "screenshot"},
				map[string]any{"action": "find_text", "params": map[string]any{"text": "Submit"}},
				map[string]any{"action": "click", "params": map[string]any{"x": 8, "y": 9}},
			},
		}),
		"trace-workflow-screen-fail-fast",
	)

	if outcome.err == nil {
		t.Fatal("expected fail-fast error")
	}
	if !strings.Contains(outcome.err.Error(), "step 2 (find_text)") {
		t.Fatalf("unexpected fail-fast error: %v", outcome.err)
	}
	if len(tool.calls) != 2 {
		t.Fatalf("expected stop after second step, got calls=%d", len(tool.calls))
	}
}

func TestExecuteWorkflowToolNodeScreenControlWorkflowStepsRejectsConflict(t *testing.T) {
	tool := &workflowScreenControlSequenceTool{name: screenControlToolID}
	registry := tools.NewRegistry()
	registry.Register(tool)

	outcome := executeWorkflowToolNode(
		context.Background(),
		agentRuntimeDependencies{registry: registry},
		buildWorkflowScreenControlNode(map[string]any{
			"action": "find_text",
			"params": map[string]any{"text": "Submit"},
			"workflow_steps": []any{
				map[string]any{"action": "screenshot"},
			},
		}),
		"trace-workflow-screen-conflict",
	)

	if outcome.err == nil {
		t.Fatal("expected conflict error")
	}
	if !strings.Contains(outcome.err.Error(), "conflicts with action/params") {
		t.Fatalf("unexpected conflict error: %v", outcome.err)
	}
	if len(tool.calls) != 0 {
		t.Fatalf("tool should not execute on conflict, calls=%d", len(tool.calls))
	}
}

func TestExecuteWorkflowToolNodeScreenControlSingleActionStillWorks(t *testing.T) {
	tool := &workflowScreenControlSequenceTool{
		name:    screenControlToolID,
		outputs: []string{`{"status":"ok"}`},
	}
	registry := tools.NewRegistry()
	registry.Register(tool)

	outcome := executeWorkflowToolNode(
		context.Background(),
		agentRuntimeDependencies{registry: registry},
		buildWorkflowScreenControlNode(map[string]any{
			"action": "find_text",
			"params": map[string]any{"text": "Submit"},
		}),
		"trace-workflow-screen-single",
	)

	if outcome.err != nil {
		t.Fatalf("expected single-action success, got error: %v", outcome.err)
	}
	if outcome.status != taskRunStatusSuccess {
		t.Fatalf("unexpected status: %q", outcome.status)
	}
	if len(tool.calls) != 1 || tool.calls[0]["action"] != "find_text" {
		t.Fatalf("unexpected single-action call: %#v", tool.calls)
	}
}

func TestRandomWorkflowScreenControlStepDelayWithinDefaultRange(t *testing.T) {
	minimum := time.Duration(workflowScreenControlStepDelayMinMS) * time.Millisecond
	maximum := time.Duration(workflowScreenControlStepDelayMaxMS) * time.Millisecond
	for range 64 {
		delay := randomWorkflowScreenControlStepDelay()
		if delay < minimum || delay > maximum {
			t.Fatalf("expected delay in [%s, %s], got %s", minimum, maximum, delay)
		}
	}
}

func buildWorkflowScreenControlNode(arguments map[string]any) WorkflowNode {
	return WorkflowNode{
		ID:   "tool-node",
		Type: workflowNodeTypeTool,
		Tool: &WorkflowToolNode{
			ToolName:  screenControlToolID,
			Arguments: arguments,
		},
	}
}

type workflowScreenControlSequenceTool struct {
	name      string
	outputs   []string
	failAt    int
	calls     []map[string]any
	callTimes []time.Time
}

func (t *workflowScreenControlSequenceTool) Name() string { return t.name }

func (t *workflowScreenControlSequenceTool) Description() string {
	return "workflow screen control sequence test tool"
}

func (t *workflowScreenControlSequenceTool) Parameters() json.RawMessage {
	return json.RawMessage(`{"type":"object"}`)
}

func (t *workflowScreenControlSequenceTool) Execute(_ context.Context, args json.RawMessage, _ string) (string, error) {
	decoded := map[string]any{}
	if err := json.Unmarshal(args, &decoded); err != nil {
		return "", err
	}
	t.calls = append(t.calls, decoded)
	t.callTimes = append(t.callTimes, time.Now())
	if t.failAt > 0 && len(t.calls) == t.failAt {
		return "", errors.New("forced step failure")
	}
	callIndex := len(t.calls) - 1
	if callIndex >= 0 && callIndex < len(t.outputs) {
		return t.outputs[callIndex], nil
	}
	return `{"status":"ok"}`, nil
}

func TestTaskWorkflowRunNowExecutesToolLLMAndAgentNodes(t *testing.T) {
	t.Setenv("GHOST_WORKFLOW_TOOL_ALLOWLIST", "script_exec")
	_, service, _ := newTestHandlerWithService(t, nil, nil)

	tool := &workflowTestTool{name: "script_exec", output: `{"status":"ok"}`}
	completer := &workflowTestCompleter{
		response: &llm.CompletionResponse{Message: llm.Message{Role: llm.RoleAssistant, Text: "llm summary"}},
	}
	registry := tools.NewRegistry()
	registry.Register(tool)
	service.runtimeFactory = proTestRuntimeFactory{
		deps: NewRuntimeDependencies(bridgeconfig.Config{Task: bridgeconfig.TaskConfig{WorkflowToolAllowlist: []string{"script_exec"}}}, completer, registry, "", nil),
	}
	agentRunner := &workflowTestRunner{message: "agent done", sessionID: "workflow-session"}
	service.agentRunner = agentRunner

	createdRaw, code, err := service.executeTaskCreateAction(taskCreateParams{
		TaskKind:        taskKindWorkflow,
		Workflow:        workflowWithToolLLMAndAgentNodes("script_exec"),
		IntervalSeconds: 60,
	}, "trace-workflow-node-create")
	if err != nil || code != http.StatusCreated {
		t.Fatalf("create workflow task: code=%d err=%v", code, err)
	}
	created := createdRaw.(taskPayload)

	runRaw, code, err := service.executeTaskRunNowAction(taskIDParams{ID: created.ID}, "trace-workflow-node-run")
	if err != nil || code != http.StatusOK {
		t.Fatalf("run workflow task: code=%d err=%v", code, err)
	}
	run := runRaw.(taskRunPayload)
	if run.Run.Status != taskRunStatusSuccess {
		t.Fatalf("unexpected run status: %#v", run.Run)
	}
	if run.Run.ResponsePreview != "workflow completed at agent-node: agent done" {
		t.Fatalf("unexpected run preview: %#v", run.Run)
	}
	assertRunTranscriptSession(t, service, run.Run.SessionIDOutput, []string{"workflow-session"}, []string{
		taskRunTranscriptEventMarker,
		"节点 tool-node 使用工具：script_exec",
		"节点 llm-node（LLM）",
		"节点 agent-node（agent）",
		"agent done",
	})
	assertWorkflowNodeResultsSequence(
		t,
		run.Run.NodeResults,
		[]string{"start-node", "tool-node", "llm-node", "agent-node", "end-node"},
	)
	if len(tool.calls) != 1 || tool.calls[0]["command"] != "pwd" {
		t.Fatalf("unexpected tool calls: %#v", tool.calls)
	}
	if len(completer.requests) != 1 {
		t.Fatalf("unexpected llm requests: %#v", completer.requests)
	}
	request := completer.requests[0]
	if len(request.Tools) != 0 || !request.ConversationState.IsZero() {
		t.Fatalf("expected direct llm request without tools/state, got %#v", request)
	}
	if len(request.Messages) != 2 || request.Messages[0].Role != llm.RoleSystem || request.Messages[1].Role != llm.RoleUser {
		t.Fatalf("unexpected llm messages: %#v", request.Messages)
	}
	if len(agentRunner.calls) != 1 || agentRunner.calls[0].sessionID != "" || agentRunner.calls[0].message != "fixed agent message" {
		t.Fatalf("unexpected agent runner calls: %#v", agentRunner.calls)
	}
}

func TestTaskWorkflowRunNowPassesAgentRuntimeOverrides(t *testing.T) {
	_, service, _ := newTestHandlerWithService(t, nil, nil)
	configureRuntimeOverrideProviders(t, service)
	agentRunner := &workflowTestRunner{message: "agent done", sessionID: "workflow-session"}
	service.agentRunner = agentRunner

	createdRaw, code, err := service.executeTaskCreateAction(taskCreateParams{
		TaskKind: taskKindWorkflow,
		Workflow: &WorkflowDefinition{
			Nodes: []WorkflowNode{
				{ID: "start-node", Type: workflowNodeTypeStart},
				{
					ID:   "agent-node",
					Type: workflowNodeTypeAgent,
					Agent: &WorkflowAgentNode{
						Message: "run agent",
						RuntimeOverrides: &TaskRuntimeOverrides{
							ProviderName:      "openai-main",
							Model:             "gpt-5.4",
							SystemPrompt:      "override prompt",
							ToolAllowlistOnly: boolPointer(true),
							ToolAllowlist:     []string{"script_exec"},
							MaxTurns:          intPointer(2),
						},
					},
				},
				{ID: "end-node", Type: workflowNodeTypeEnd},
			},
			Edges: []WorkflowEdge{
				{FromNodeID: "start-node", ToNodeID: "agent-node"},
				{FromNodeID: "agent-node", ToNodeID: "end-node"},
			},
		},
		IntervalSeconds: 60,
	}, "trace-workflow-agent-override-create")
	if err != nil || code != http.StatusCreated {
		t.Fatalf("create workflow task: code=%d err=%v", code, err)
	}
	created := createdRaw.(taskPayload)

	runRaw, code, err := service.executeTaskRunNowAction(taskIDParams{ID: created.ID}, "trace-workflow-agent-override-run")
	if err != nil || code != http.StatusOK {
		t.Fatalf("run workflow task: code=%d err=%v", code, err)
	}
	run := runRaw.(taskRunPayload)
	if run.Run.Status != taskRunStatusSuccess {
		t.Fatalf("unexpected run status: %#v", run.Run)
	}
	if len(agentRunner.calls) != 1 {
		t.Fatalf("expected one agent runner call, got %#v", agentRunner.calls)
	}
	if agentRunner.calls[0].runtimeOverrides == nil {
		t.Fatalf("expected runtime overrides to be forwarded, got %#v", agentRunner.calls[0])
	}
	if agentRunner.calls[0].runtimeOverrides.ProviderName != "openai-main" ||
		agentRunner.calls[0].runtimeOverrides.Model != "gpt-5.4" {
		t.Fatalf("unexpected provider/model overrides: %#v", agentRunner.calls[0].runtimeOverrides)
	}
	if agentRunner.calls[0].runtimeOverrides.MaxTurns == nil || *agentRunner.calls[0].runtimeOverrides.MaxTurns != 2 {
		t.Fatalf("unexpected max_turns override: %#v", agentRunner.calls[0].runtimeOverrides)
	}
}

func TestTaskWorkflowRunNowRoutesIfNodeByToolOutput(t *testing.T) {
	t.Setenv("GHOST_WORKFLOW_TOOL_ALLOWLIST", "script_exec")
	_, service, _ := newTestHandlerWithService(t, nil, nil)

	tool := &workflowTestTool{name: "script_exec", output: "status=ok"}
	registry := tools.NewRegistry()
	registry.Register(tool)
	service.runtimeFactory = proTestRuntimeFactory{
		deps: NewRuntimeDependencies(bridgeconfig.Config{Task: bridgeconfig.TaskConfig{WorkflowToolAllowlist: []string{"script_exec"}}}, &workflowTestCompleter{}, registry, "", nil),
	}
	agentRunner := &workflowTestRunner{message: "if branch done", sessionID: "if-session"}
	service.agentRunner = agentRunner

	createdRaw, code, err := service.executeTaskCreateAction(taskCreateParams{
		TaskKind:        taskKindWorkflow,
		Workflow:        workflowWithIfNode("script_exec"),
		IntervalSeconds: 60,
	}, "trace-workflow-if-create")
	if err != nil || code != http.StatusCreated {
		t.Fatalf("create workflow task: code=%d err=%v", code, err)
	}
	created := createdRaw.(taskPayload)

	runRaw, code, err := service.executeTaskRunNowAction(taskIDParams{ID: created.ID}, "trace-workflow-if-run")
	if err != nil || code != http.StatusOK {
		t.Fatalf("run workflow task: code=%d err=%v", code, err)
	}
	run := runRaw.(taskRunPayload)
	if run.Run.Status != taskRunStatusSuccess {
		t.Fatalf("unexpected run status: %#v", run.Run)
	}
	assertRunTranscriptSession(t, service, run.Run.SessionIDOutput, []string{"if-session"}, []string{
		"工作流 if 判断：if-node branch=true",
		"节点 agent-true（agent）",
		"if branch done",
	})
	if len(agentRunner.calls) != 1 || agentRunner.calls[0].message != "true branch message" {
		t.Fatalf("unexpected agent branch: %#v", agentRunner.calls)
	}
}

func TestTaskWorkflowRunNowKeepsIfValueLiteral(t *testing.T) {
	t.Setenv("GHOST_WORKFLOW_TOOL_ALLOWLIST", "script_exec")
	_, service, _ := newTestHandlerWithService(t, nil, nil)

	tool := &workflowTestTool{name: "script_exec", output: "status=ok"}
	registry := tools.NewRegistry()
	registry.Register(tool)
	service.runtimeFactory = proTestRuntimeFactory{
		deps: NewRuntimeDependencies(
			bridgeconfig.Config{Task: bridgeconfig.TaskConfig{WorkflowToolAllowlist: []string{"script_exec"}}},
			&workflowTestCompleter{},
			registry,
			"",
			nil,
		),
	}
	agentRunner := &workflowTestRunner{message: "if template branch done", sessionID: "if-template-session"}
	service.agentRunner = agentRunner

	createdRaw, code, err := service.executeTaskCreateAction(taskCreateParams{
		TaskKind:        taskKindWorkflow,
		Workflow:        workflowWithTemplatedIfNode("script_exec"),
		IntervalSeconds: 60,
	}, "trace-workflow-if-template-create")
	if err != nil || code != http.StatusCreated {
		t.Fatalf("create workflow task: code=%d err=%v", code, err)
	}
	created := createdRaw.(taskPayload)

	runRaw, code, err := service.executeTaskRunNowAction(taskIDParams{ID: created.ID}, "trace-workflow-if-template-run")
	if err != nil || code != http.StatusOK {
		t.Fatalf("run workflow task: code=%d err=%v", code, err)
	}
	run := runRaw.(taskRunPayload)
	if run.Run.Status != taskRunStatusSuccess {
		t.Fatalf("unexpected run status: %#v", run.Run)
	}
	assertRunTranscriptSession(t, service, run.Run.SessionIDOutput, []string{"if-template-session"}, []string{
		"工作流 if 判断：if-node branch=false",
		"value=${inputs.token}",
		"节点 agent-false（agent）",
	})
	if len(agentRunner.calls) != 1 || agentRunner.calls[0].message != "false branch message" {
		t.Fatalf("unexpected agent branch: %#v", agentRunner.calls)
	}
}

func TestTaskWorkflowRunNowExecutesLoopBodyByMaxIterations(t *testing.T) {
	t.Setenv("GHOST_WORKFLOW_TOOL_ALLOWLIST", "script_exec")
	_, service, _ := newTestHandlerWithService(t, nil, nil)

	tool := &workflowTestTool{name: "script_exec", output: "loop body"}
	registry := tools.NewRegistry()
	registry.Register(tool)
	service.runtimeFactory = proTestRuntimeFactory{
		deps: NewRuntimeDependencies(bridgeconfig.Config{Task: bridgeconfig.TaskConfig{WorkflowToolAllowlist: []string{"script_exec"}}}, &workflowTestCompleter{}, registry, "", nil),
	}

	createdRaw, code, err := service.executeTaskCreateAction(taskCreateParams{
		TaskKind:        taskKindWorkflow,
		Workflow:        workflowWithLoopNode("script_exec", 3),
		IntervalSeconds: 60,
	}, "trace-workflow-loop-create")
	if err != nil || code != http.StatusCreated {
		t.Fatalf("create workflow task: code=%d err=%v", code, err)
	}
	created := createdRaw.(taskPayload)

	runRaw, code, err := service.executeTaskRunNowAction(taskIDParams{ID: created.ID}, "trace-workflow-loop-run")
	if err != nil || code != http.StatusOK {
		t.Fatalf("run workflow task: code=%d err=%v", code, err)
	}
	run := runRaw.(taskRunPayload)
	if run.Run.Status != taskRunStatusSuccess {
		t.Fatalf("unexpected run status: %#v", run.Run)
	}
	if len(tool.calls) != 3 {
		t.Fatalf("unexpected loop execution count: got %d want 3", len(tool.calls))
	}
	assertRunTranscriptSession(t, service, run.Run.SessionIDOutput, nil, []string{
		"工作流进入循环：loop-node 第 1/3 轮",
		"工作流退出循环：loop-node",
	})
}

func TestTaskWorkflowRunNowExecutesStartBranchesInParallel(t *testing.T) {
	t.Setenv("GHOST_WORKFLOW_TOOL_ALLOWLIST", "script_exec")
	_, service, _ := newTestHandlerWithService(t, nil, nil)

	tool := newWorkflowParallelProbeTool("script_exec")
	registry := tools.NewRegistry()
	registry.Register(tool)
	service.runtimeFactory = proTestRuntimeFactory{
		deps: NewRuntimeDependencies(
			bridgeconfig.Config{Task: bridgeconfig.TaskConfig{WorkflowToolAllowlist: []string{"script_exec"}}},
			&workflowTestCompleter{},
			registry,
			"",
			nil,
		),
	}

	createdRaw, code, err := service.executeTaskCreateAction(taskCreateParams{
		TaskKind:        taskKindWorkflow,
		Workflow:        workflowWithParallelStartToolBranches("script_exec"),
		IntervalSeconds: 60,
	}, "trace-workflow-parallel-create")
	if err != nil || code != http.StatusCreated {
		t.Fatalf("create workflow task: code=%d err=%v", code, err)
	}
	created := createdRaw.(taskPayload)

	type runNowOutcome struct {
		raw  any
		code int
		err  error
	}
	outcomeCh := make(chan runNowOutcome, 1)
	go func() {
		raw, runCode, runErr := service.executeTaskRunNowAction(taskIDParams{ID: created.ID}, "trace-workflow-parallel-run")
		outcomeCh <- runNowOutcome{raw: raw, code: runCode, err: runErr}
	}()

	if !tool.waitStarted(2, 2*time.Second) {
		tool.release()
		t.Fatal("expected both start branches to run before release")
	}
	tool.release()
	outcome := <-outcomeCh
	if outcome.err != nil || outcome.code != http.StatusOK {
		t.Fatalf("run workflow task: code=%d err=%v", outcome.code, outcome.err)
	}
	run := outcome.raw.(taskRunPayload)
	if run.Run.Status != taskRunStatusSuccess {
		t.Fatalf("unexpected run status: %#v", run.Run)
	}
	if !strings.Contains(run.Run.ResponsePreview, "workflow completed in parallel") {
		t.Fatalf("unexpected run preview: %#v", run.Run)
	}
	assertWorkflowNodeResultsMonotonic(t, run.Run.NodeResults)
	for _, node := range run.Run.NodeResults {
		if !strings.HasPrefix(node.NodeID, "tool-") {
			continue
		}
		if strings.TrimSpace(node.BranchID) == "" {
			t.Fatalf("expected tool branch node to include branch_id: %#v", node)
		}
	}
	if tool.maxConcurrency() < 2 {
		t.Fatalf("expected concurrent execution, got max=%d", tool.maxConcurrency())
	}
}

func TestTaskWorkflowRunNowStopsOnAgentAwaitingHuman(t *testing.T) {
	_, service, _ := newTestHandlerWithService(t, nil, nil)
	service.agentRunner = &workflowTestRunner{
		sessionID: "await-session",
		err: &agent.ErrAwaitingHuman{
			QuestionID: "q-1",
			Prompt:     "Need approval",
		},
	}

	createdRaw, _, err := service.executeTaskCreateAction(taskCreateParams{
		TaskKind:        taskKindWorkflow,
		Workflow:        workflowWithAgentNode(),
		IntervalSeconds: 60,
	}, "trace-workflow-await-create")
	if err != nil {
		t.Fatalf("create workflow task: %v", err)
	}
	created := createdRaw.(taskPayload)

	runRaw, code, err := service.executeTaskRunNowAction(taskIDParams{ID: created.ID}, "trace-workflow-await-run")
	if err != nil || code != http.StatusOK {
		t.Fatalf("run workflow task: code=%d err=%v", code, err)
	}
	run := runRaw.(taskRunPayload)
	if run.Run.Status != taskRunStatusAwaitingHuman {
		t.Fatalf("unexpected run status: %#v", run.Run)
	}
	assertRunTranscriptSession(t, service, run.Run.SessionIDOutput, []string{"await-session"}, []string{
		taskRunTranscriptEventMarker,
		"任务运行结束：awaiting_human",
		"节点 agent-node（agent）",
		"Need approval",
	})
	if !strings.Contains(run.Run.ResponsePreview, "Need approval") {
		t.Fatalf("unexpected awaiting preview: %#v", run.Run)
	}
}

func TestTaskWorkflowRunNowAllowsAllToolsWhenAllowlistCleared(t *testing.T) {
	t.Setenv("GHOST_WORKFLOW_TOOL_ALLOWLIST", "script_exec")
	_, service, _ := newTestHandlerWithService(t, nil, nil)

	tool := &workflowTestTool{name: "script_exec", output: `{"status":"ok"}`}
	registry := tools.NewRegistry()
	registry.Register(tool)
	service.runtimeFactory = proTestRuntimeFactory{
		deps: NewRuntimeDependencies(bridgeconfig.Config{Task: bridgeconfig.TaskConfig{WorkflowToolAllowlist: []string{"script_exec"}}}, &workflowTestCompleter{}, registry, "", nil),
	}

	createdRaw, _, err := service.executeTaskCreateAction(taskCreateParams{
		TaskKind:        taskKindWorkflow,
		Workflow:        workflowWithToolNode("script_exec"),
		IntervalSeconds: 60,
	}, "trace-workflow-config-shift-create")
	if err != nil {
		t.Fatalf("create workflow task: %v", err)
	}
	t.Setenv("GHOST_WORKFLOW_TOOL_ALLOWLIST", "")
	created := createdRaw.(taskPayload)

	runRaw, code, err := service.executeTaskRunNowAction(taskIDParams{ID: created.ID}, "trace-workflow-config-shift-run")
	if err != nil || code != http.StatusOK {
		t.Fatalf("run workflow task: code=%d err=%v", code, err)
	}
	run := runRaw.(taskRunPayload)
	if run.Run.Status != taskRunStatusSuccess {
		t.Fatalf("unexpected run status: %#v", run.Run)
	}
	if strings.TrimSpace(run.Run.Error) != "" {
		t.Fatalf("unexpected run error: %#v", run.Run)
	}
}

func TestTaskWorkflowRunNowKeepsDeprecatedTemplateTokensLiteral(t *testing.T) {
	t.Setenv("GHOST_WORKFLOW_TOOL_ALLOWLIST", "script_exec")
	_, service, _ := newTestHandlerWithService(t, nil, nil)

	tool := &workflowTestTool{
		name:    "script_exec",
		outputs: []string{`{"cwd":"/workspace","ok":true}`},
		output:  `{"status":"done"}`,
	}
	registry := tools.NewRegistry()
	registry.Register(tool)
	service.runtimeFactory = proTestRuntimeFactory{
		deps: NewRuntimeDependencies(
			bridgeconfig.Config{Task: bridgeconfig.TaskConfig{WorkflowToolAllowlist: []string{"script_exec"}}},
			&workflowTestCompleter{},
			registry,
			"",
			nil,
		),
	}

	createdRaw, code, err := service.executeTaskCreateAction(taskCreateParams{
		TaskKind:        taskKindWorkflow,
		Workflow:        workflowWithTemplatedToolChain("script_exec"),
		IntervalSeconds: 60,
	}, "trace-workflow-template-create")
	if err != nil || code != http.StatusCreated {
		t.Fatalf("create workflow task: code=%d err=%v", code, err)
	}
	created := createdRaw.(taskPayload)

	runRaw, code, err := service.executeTaskRunNowAction(taskIDParams{ID: created.ID}, "trace-workflow-template-run")
	if err != nil || code != http.StatusOK {
		t.Fatalf("run workflow task: code=%d err=%v", code, err)
	}
	run := runRaw.(taskRunPayload)
	if run.Run.Status != taskRunStatusSuccess {
		t.Fatalf("unexpected run status: %#v", run.Run)
	}
	if len(tool.calls) != 2 {
		t.Fatalf("unexpected tool call count: %#v", tool.calls)
	}
	if tool.calls[0]["command"] != "${inputs.command}" {
		t.Fatalf("unexpected first tool args: %#v", tool.calls[0])
	}
	if tool.calls[1]["command"] != "echo ${outputs.tool-read.cwd}" {
		t.Fatalf("unexpected second tool command: %#v", tool.calls[1])
	}
	if tool.calls[1]["ok"] != "${outputs.tool-read.ok}" {
		t.Fatalf("unexpected second tool ok arg: %#v", tool.calls[1])
	}
	if tool.calls[1]["text"] != "run ${inputs.command}" {
		t.Fatalf("unexpected second tool text arg: %#v", tool.calls[1])
	}
}

func TestTaskWorkflowRunNowResolvesFindIconVariableForDownstreamTool(t *testing.T) {
	t.Setenv("GHOST_WORKFLOW_TOOL_ALLOWLIST", "screen_control,script_exec")
	_, service, _ := newTestHandlerWithService(t, nil, nil)

	screenTool := &workflowTestTool{
		name:   "screen_control",
		output: `{"matches":[{"center":{"x":321,"y":654}}],"display_id":7}`,
	}
	scriptTool := &workflowTestTool{name: "script_exec", output: `{"status":"ok"}`}
	registry := tools.NewRegistry()
	registry.Register(screenTool)
	registry.Register(scriptTool)
	service.runtimeFactory = proTestRuntimeFactory{
		deps: NewRuntimeDependencies(
			bridgeconfig.Config{Task: bridgeconfig.TaskConfig{WorkflowToolAllowlist: []string{"screen_control", "script_exec"}}},
			&workflowTestCompleter{},
			registry,
			"",
			nil,
		),
	}

	createdRaw, code, err := service.executeTaskCreateAction(taskCreateParams{
		TaskKind:        taskKindWorkflow,
		Workflow:        workflowWithFindIconVariableConsumer("script_exec"),
		IntervalSeconds: 60,
	}, "trace-workflow-find-icon-variable-create")
	if err != nil || code != http.StatusCreated {
		t.Fatalf("create workflow task: code=%d err=%v", code, err)
	}
	created := createdRaw.(taskPayload)

	runRaw, code, err := service.executeTaskRunNowAction(taskIDParams{ID: created.ID}, "trace-workflow-find-icon-variable-run")
	if err != nil || code != http.StatusOK {
		t.Fatalf("run workflow task: code=%d err=%v", code, err)
	}
	run := runRaw.(taskRunPayload)
	if run.Run.Status != taskRunStatusSuccess {
		t.Fatalf("unexpected run status: %#v", run.Run)
	}
	if len(scriptTool.calls) != 1 {
		t.Fatalf("unexpected script_exec call count: %#v", scriptTool.calls)
	}
	if scriptTool.calls[0]["x"] != float64(321) || scriptTool.calls[0]["y"] != float64(654) {
		t.Fatalf("unexpected resolved coordinates: %#v", scriptTool.calls[0])
	}
	point, ok := scriptTool.calls[0]["point"].(map[string]any)
	if !ok || point["display_id"] != float64(7) {
		t.Fatalf("unexpected resolved point payload: %#v", scriptTool.calls[0]["point"])
	}
	if scriptTool.calls[0]["label"] != "prefix 321" {
		t.Fatalf("unexpected resolved label: %#v", scriptTool.calls[0]["label"])
	}
}

func TestTaskWorkflowRunNowFailsWhenFindIconVariableUsedBeforeDefinition(t *testing.T) {
	t.Setenv("GHOST_WORKFLOW_TOOL_ALLOWLIST", "script_exec")
	_, service, _ := newTestHandlerWithService(t, nil, nil)

	tool := &workflowTestTool{name: "script_exec", output: `{"status":"ok"}`}
	registry := tools.NewRegistry()
	registry.Register(tool)
	service.runtimeFactory = proTestRuntimeFactory{
		deps: NewRuntimeDependencies(
			bridgeconfig.Config{Task: bridgeconfig.TaskConfig{WorkflowToolAllowlist: []string{"script_exec"}}},
			&workflowTestCompleter{},
			registry,
			"",
			nil,
		),
	}

	createdRaw, code, err := service.executeTaskCreateAction(taskCreateParams{
		TaskKind:        taskKindWorkflow,
		Workflow:        workflowWithUndefinedFindIconVariable("script_exec"),
		IntervalSeconds: 60,
	}, "trace-workflow-find-icon-missing-create")
	if err != nil || code != http.StatusCreated {
		t.Fatalf("create workflow task: code=%d err=%v", code, err)
	}
	created := createdRaw.(taskPayload)

	runRaw, code, err := service.executeTaskRunNowAction(taskIDParams{ID: created.ID}, "trace-workflow-find-icon-missing-run")
	if err != nil || code != http.StatusOK {
		t.Fatalf("run workflow task: code=%d err=%v", code, err)
	}
	run := runRaw.(taskRunPayload)
	if run.Run.Status != taskRunStatusError {
		t.Fatalf("unexpected run status: %#v", run.Run)
	}
	if !strings.Contains(run.Run.Error, `workflow variable "find_icon" is not defined`) {
		t.Fatalf("unexpected run error: %#v", run.Run)
	}
}

func TestTaskWorkflowRunNowKeepsUndefinedTemplateLiteral(t *testing.T) {
	t.Setenv("GHOST_WORKFLOW_TOOL_ALLOWLIST", "script_exec")
	_, service, _ := newTestHandlerWithService(t, nil, nil)

	tool := &workflowTestTool{name: "script_exec", output: `{"status":"ok"}`}
	registry := tools.NewRegistry()
	registry.Register(tool)
	service.runtimeFactory = proTestRuntimeFactory{
		deps: NewRuntimeDependencies(
			bridgeconfig.Config{Task: bridgeconfig.TaskConfig{WorkflowToolAllowlist: []string{"script_exec"}}},
			&workflowTestCompleter{},
			registry,
			"",
			nil,
		),
	}

	createdRaw, code, err := service.executeTaskCreateAction(taskCreateParams{
		TaskKind:        taskKindWorkflow,
		Workflow:        workflowWithUndefinedTemplate("script_exec"),
		IntervalSeconds: 60,
	}, "trace-workflow-template-missing-create")
	if err != nil || code != http.StatusCreated {
		t.Fatalf("create workflow task: code=%d err=%v", code, err)
	}
	created := createdRaw.(taskPayload)

	runRaw, code, err := service.executeTaskRunNowAction(taskIDParams{ID: created.ID}, "trace-workflow-template-missing-run")
	if err != nil || code != http.StatusOK {
		t.Fatalf("run workflow task: code=%d err=%v", code, err)
	}
	run := runRaw.(taskRunPayload)
	if run.Run.Status != taskRunStatusSuccess {
		t.Fatalf("unexpected run status: %#v", run.Run)
	}
	if len(tool.calls) != 1 || tool.calls[0]["command"] != "${outputs.missing.status}" {
		t.Fatalf("unexpected literal template argument: %#v", tool.calls)
	}
}

type workflowTestTool struct {
	name    string
	output  string
	outputs []string
	calls   []map[string]any
}

func (t *workflowTestTool) Name() string { return t.name }

func (t *workflowTestTool) Description() string { return "workflow test tool" }

func (t *workflowTestTool) Parameters() json.RawMessage { return json.RawMessage(`{"type":"object"}`) }

func (t *workflowTestTool) Execute(_ context.Context, args json.RawMessage, _ string) (string, error) {
	decoded := make(map[string]any)
	if err := json.Unmarshal(args, &decoded); err != nil {
		return "", err
	}
	t.calls = append(t.calls, decoded)
	callIndex := len(t.calls) - 1
	if callIndex >= 0 && callIndex < len(t.outputs) {
		return t.outputs[callIndex], nil
	}
	return t.output, nil
}

type workflowParallelProbeTool struct {
	name        string
	startedCh   chan struct{}
	releaseOnce sync.Once
	releaseCh   chan struct{}
	mu          sync.Mutex
	active      int
	maxActive   int
}

func newWorkflowParallelProbeTool(name string) *workflowParallelProbeTool {
	return &workflowParallelProbeTool{
		name:      name,
		startedCh: make(chan struct{}, 8),
		releaseCh: make(chan struct{}),
	}
}

func (t *workflowParallelProbeTool) Name() string { return t.name }

func (t *workflowParallelProbeTool) Description() string { return "workflow parallel probe tool" }

func (t *workflowParallelProbeTool) Parameters() json.RawMessage {
	return json.RawMessage(`{"type":"object"}`)
}

func (t *workflowParallelProbeTool) Execute(_ context.Context, _ json.RawMessage, _ string) (string, error) {
	t.mu.Lock()
	t.active++
	if t.active > t.maxActive {
		t.maxActive = t.active
	}
	t.mu.Unlock()
	t.startedCh <- struct{}{}
	<-t.releaseCh
	t.mu.Lock()
	t.active--
	t.mu.Unlock()
	return `{"status":"ok"}`, nil
}

func (t *workflowParallelProbeTool) waitStarted(count int, timeout time.Duration) bool {
	for i := 0; i < count; i++ {
		select {
		case <-t.startedCh:
		case <-time.After(timeout):
			return false
		}
	}
	return true
}

func (t *workflowParallelProbeTool) release() {
	t.releaseOnce.Do(func() {
		close(t.releaseCh)
	})
}

func (t *workflowParallelProbeTool) maxConcurrency() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.maxActive
}

type workflowTestCompleter struct {
	response *llm.CompletionResponse
	err      error
	requests []llm.CompletionRequest
}

func (c *workflowTestCompleter) Complete(_ context.Context, request llm.CompletionRequest) (*llm.CompletionResponse, error) {
	c.requests = append(c.requests, request)
	if c.err != nil {
		return nil, c.err
	}
	if c.response == nil {
		return nil, errors.New("unexpected llm completion")
	}
	return c.response, nil
}

type workflowRunnerCall struct {
	message          string
	sessionID        string
	runtimeOverrides *TaskRuntimeOverrides
}

type workflowTestRunner struct {
	message   string
	sessionID string
	err       error
	calls     []workflowRunnerCall
}

func (r *workflowTestRunner) RunTurn(_ context.Context, message string, sessionID string, _ string) (string, string, error) {
	r.calls = append(r.calls, workflowRunnerCall{message: message, sessionID: sessionID})
	return r.message, r.sessionID, r.err
}

func (r *workflowTestRunner) RunTurnWithOverrides(
	_ context.Context,
	message string,
	sessionID string,
	_ string,
	runtimeOverrides *TaskRuntimeOverrides,
) (string, string, error) {
	r.calls = append(r.calls, workflowRunnerCall{
		message:          message,
		sessionID:        sessionID,
		runtimeOverrides: cloneTaskRuntimeOverrides(runtimeOverrides),
	})
	return r.message, r.sessionID, r.err
}

func (r *workflowTestRunner) RunTurnStream(
	ctx context.Context,
	message string,
	sessionID string,
	traceID string,
	_ streaming.Sink,
) (string, string, error) {
	return r.RunTurn(ctx, message, sessionID, traceID)
}

func workflowWithToolNode(toolName string) *WorkflowDefinition {
	return &WorkflowDefinition{
		Nodes: []WorkflowNode{
			{ID: "start-node", Type: workflowNodeTypeStart},
			{ID: "tool-node", Type: workflowNodeTypeTool, Tool: &WorkflowToolNode{ToolName: toolName, Arguments: map[string]any{"command": "pwd"}}},
			{ID: "end-node", Type: workflowNodeTypeEnd},
		},
		Edges: []WorkflowEdge{
			{FromNodeID: "start-node", ToNodeID: "tool-node"},
			{FromNodeID: "tool-node", ToNodeID: "end-node"},
		},
	}
}

func assertWorkflowNodeResultsSequence(t *testing.T, nodeResults []RunNodeResult, expectedNodeIDs []string) {
	t.Helper()
	if len(nodeResults) != len(expectedNodeIDs) {
		t.Fatalf("unexpected node result count: got %d want %d", len(nodeResults), len(expectedNodeIDs))
	}
	for index, node := range nodeResults {
		if node.CompletedSeq != index+1 {
			t.Fatalf("unexpected completed_seq at index=%d: %#v", index, node)
		}
		if node.NodeID != expectedNodeIDs[index] {
			t.Fatalf("unexpected node order at index=%d: got %q want %q", index, node.NodeID, expectedNodeIDs[index])
		}
		if strings.TrimSpace(node.Status) == "" || node.StartedAt.IsZero() || node.FinishedAt.IsZero() {
			t.Fatalf("expected non-empty status/timestamps: %#v", node)
		}
	}
}

func assertWorkflowNodeResultsMonotonic(t *testing.T, nodeResults []RunNodeResult) {
	t.Helper()
	if len(nodeResults) == 0 {
		t.Fatal("expected workflow node results")
	}
	lastSeq := 0
	for index, node := range nodeResults {
		if node.CompletedSeq <= lastSeq {
			t.Fatalf("completed_seq must be strictly increasing at index=%d: %#v", index, node)
		}
		lastSeq = node.CompletedSeq
	}
}

func assertRunTranscriptSession(
	t *testing.T,
	service *bridgeService,
	sessionID string,
	disallowed []string,
	required []string,
) {
	t.Helper()
	if strings.TrimSpace(sessionID) == "" {
		t.Fatal("expected run transcript session id")
	}
	for _, blocked := range disallowed {
		if sessionID == blocked {
			t.Fatalf("expected display transcript session, got execution session %q", sessionID)
		}
	}
	if service == nil || service.sessionStore == nil {
		t.Fatal("session store is not configured")
	}
	loaded, err := service.sessionStore.Load(sessionID)
	if err != nil {
		t.Fatalf("load run transcript session %q: %v", sessionID, err)
	}
	transcript := sessionMessagesText(loaded.Messages)
	for _, expected := range required {
		if !strings.Contains(transcript, expected) {
			t.Fatalf("run transcript missing %q in:\n%s", expected, transcript)
		}
	}
}

func sessionMessagesText(messages []llm.Message) string {
	parts := make([]string, 0, len(messages)*2)
	for _, message := range messages {
		parts = append(parts, string(message.Role)+": "+message.Text)
		for _, call := range message.ToolCalls {
			parts = append(parts, "tool_call: "+call.Name+" "+string(call.Arguments))
		}
	}
	return strings.Join(parts, "\n")
}

func workflowWithParallelStartToolBranches(toolName string) *WorkflowDefinition {
	return &WorkflowDefinition{
		Nodes: []WorkflowNode{
			{ID: "start-node", Type: workflowNodeTypeStart},
			{ID: "tool-a", Type: workflowNodeTypeTool, Tool: &WorkflowToolNode{ToolName: toolName, Arguments: map[string]any{"command": "echo a"}}},
			{ID: "tool-b", Type: workflowNodeTypeTool, Tool: &WorkflowToolNode{ToolName: toolName, Arguments: map[string]any{"command": "echo b"}}},
			{ID: "end-node", Type: workflowNodeTypeEnd},
		},
		Edges: []WorkflowEdge{
			{FromNodeID: "start-node", ToNodeID: "tool-a"},
			{FromNodeID: "start-node", ToNodeID: "tool-b"},
			{FromNodeID: "tool-a", ToNodeID: "end-node"},
			{FromNodeID: "tool-b", ToNodeID: "end-node"},
		},
	}
}

func workflowWithAgentNode() *WorkflowDefinition {
	return &WorkflowDefinition{
		Nodes: []WorkflowNode{
			{ID: "start-node", Type: workflowNodeTypeStart},
			{ID: "agent-node", Type: workflowNodeTypeAgent, Agent: &WorkflowAgentNode{Message: "fixed agent message"}},
			{ID: "end-node", Type: workflowNodeTypeEnd},
		},
		Edges: []WorkflowEdge{
			{FromNodeID: "start-node", ToNodeID: "agent-node"},
			{FromNodeID: "agent-node", ToNodeID: "end-node"},
		},
	}
}

func workflowWithIfNode(toolName string) *WorkflowDefinition {
	return &WorkflowDefinition{
		Nodes: []WorkflowNode{
			{ID: "start-node", Type: workflowNodeTypeStart},
			{ID: "tool-node", Type: workflowNodeTypeTool, Tool: &WorkflowToolNode{ToolName: toolName, Arguments: map[string]any{"command": "pwd"}}},
			{
				ID:   "if-node",
				Type: workflowNodeTypeIf,
				If: &WorkflowIfNode{
					SourceNodeID: "tool-node",
					Operator:     workflowIfOperatorContains,
					Value:        "ok",
					TrueNodeID:   "agent-true",
					FalseNodeID:  "agent-false",
				},
			},
			{ID: "agent-true", Type: workflowNodeTypeAgent, Agent: &WorkflowAgentNode{Message: "true branch message"}},
			{ID: "agent-false", Type: workflowNodeTypeAgent, Agent: &WorkflowAgentNode{Message: "false branch message"}},
			{ID: "end-node", Type: workflowNodeTypeEnd},
		},
		Edges: []WorkflowEdge{
			{FromNodeID: "start-node", ToNodeID: "tool-node"},
			{FromNodeID: "tool-node", ToNodeID: "if-node"},
			{FromNodeID: "if-node", ToNodeID: "agent-true"},
			{FromNodeID: "if-node", ToNodeID: "agent-false"},
			{FromNodeID: "agent-true", ToNodeID: "end-node"},
			{FromNodeID: "agent-false", ToNodeID: "end-node"},
		},
	}
}

func workflowWithTemplatedIfNode(toolName string) *WorkflowDefinition {
	return &WorkflowDefinition{
		Nodes: []WorkflowNode{
			{
				ID:   "start-node",
				Type: workflowNodeTypeStart,
				Start: &WorkflowStartNode{Inputs: []WorkflowInputVariable{
					{Name: "token", Type: workflowInputTypeString, Default: []byte(`"ok"`)},
				}},
			},
			{ID: "tool-node", Type: workflowNodeTypeTool, Tool: &WorkflowToolNode{ToolName: toolName, Arguments: map[string]any{"command": "pwd"}}},
			{
				ID:   "if-node",
				Type: workflowNodeTypeIf,
				If: &WorkflowIfNode{
					SourceNodeID: "tool-node",
					Operator:     workflowIfOperatorContains,
					Value:        "${inputs.token}",
					TrueNodeID:   "agent-true",
					FalseNodeID:  "agent-false",
				},
			},
			{ID: "agent-true", Type: workflowNodeTypeAgent, Agent: &WorkflowAgentNode{Message: "true branch message"}},
			{ID: "agent-false", Type: workflowNodeTypeAgent, Agent: &WorkflowAgentNode{Message: "false branch message"}},
			{ID: "end-node", Type: workflowNodeTypeEnd},
		},
		Edges: []WorkflowEdge{
			{FromNodeID: "start-node", ToNodeID: "tool-node"},
			{FromNodeID: "tool-node", ToNodeID: "if-node"},
			{FromNodeID: "if-node", ToNodeID: "agent-true"},
			{FromNodeID: "if-node", ToNodeID: "agent-false"},
			{FromNodeID: "agent-true", ToNodeID: "end-node"},
			{FromNodeID: "agent-false", ToNodeID: "end-node"},
		},
	}
}

func workflowWithLoopNode(toolName string, maxIterations int) *WorkflowDefinition {
	return &WorkflowDefinition{
		Nodes: []WorkflowNode{
			{ID: "start-node", Type: workflowNodeTypeStart},
			{
				ID:   "loop-node",
				Type: workflowNodeTypeLoop,
				Loop: &WorkflowLoopNode{
					MaxIterations: maxIterations,
					BodyNodeID:    "tool-node",
					ExitNodeID:    "end-node",
				},
			},
			{ID: "tool-node", Type: workflowNodeTypeTool, Tool: &WorkflowToolNode{ToolName: toolName, Arguments: map[string]any{"command": "pwd"}}},
			{ID: "end-node", Type: workflowNodeTypeEnd},
		},
		Edges: []WorkflowEdge{
			{FromNodeID: "start-node", ToNodeID: "loop-node"},
			{FromNodeID: "loop-node", ToNodeID: "tool-node"},
			{FromNodeID: "loop-node", ToNodeID: "end-node"},
			{FromNodeID: "tool-node", ToNodeID: "loop-node"},
		},
	}
}

func workflowWithToolLLMAndAgentNodes(toolName string) *WorkflowDefinition {
	return &WorkflowDefinition{
		Nodes: []WorkflowNode{
			{ID: "start-node", Type: workflowNodeTypeStart},
			{ID: "tool-node", Type: workflowNodeTypeTool, Tool: &WorkflowToolNode{ToolName: toolName, Arguments: map[string]any{"command": "pwd"}}},
			{ID: "llm-node", Type: workflowNodeTypeLLM, LLM: &WorkflowLLMNode{Prompt: "Summarize the previous result", SystemPrompt: "Be concise"}},
			{ID: "agent-node", Type: workflowNodeTypeAgent, Agent: &WorkflowAgentNode{Message: "fixed agent message"}},
			{ID: "end-node", Type: workflowNodeTypeEnd},
		},
		Edges: []WorkflowEdge{
			{FromNodeID: "start-node", ToNodeID: "tool-node"},
			{FromNodeID: "tool-node", ToNodeID: "llm-node"},
			{FromNodeID: "llm-node", ToNodeID: "agent-node"},
			{FromNodeID: "agent-node", ToNodeID: "end-node"},
		},
	}
}

func workflowWithTemplatedToolChain(toolName string) *WorkflowDefinition {
	return &WorkflowDefinition{
		Nodes: []WorkflowNode{
			{
				ID:   "start-node",
				Type: workflowNodeTypeStart,
				Start: &WorkflowStartNode{Inputs: []WorkflowInputVariable{
					{
						Name:    "command",
						Type:    workflowInputTypeString,
						Default: []byte(`"pwd"`),
					},
				}},
			},
			{
				ID:   "tool-read",
				Type: workflowNodeTypeTool,
				Tool: &WorkflowToolNode{
					ToolName:  toolName,
					Arguments: map[string]any{"command": "${inputs.command}"},
				},
			},
			{
				ID:   "tool-use",
				Type: workflowNodeTypeTool,
				Tool: &WorkflowToolNode{
					ToolName: toolName,
					Arguments: map[string]any{
						"command": "echo ${outputs.tool-read.cwd}",
						"ok":      "${outputs.tool-read.ok}",
						"text":    "run ${inputs.command}",
					},
				},
			},
			{ID: "end-node", Type: workflowNodeTypeEnd},
		},
		Edges: []WorkflowEdge{
			{FromNodeID: "start-node", ToNodeID: "tool-read"},
			{FromNodeID: "tool-read", ToNodeID: "tool-use"},
			{FromNodeID: "tool-use", ToNodeID: "end-node"},
		},
	}
}

func workflowWithUndefinedTemplate(toolName string) *WorkflowDefinition {
	return &WorkflowDefinition{
		Nodes: []WorkflowNode{
			{ID: "start-node", Type: workflowNodeTypeStart},
			{
				ID:   "tool-node",
				Type: workflowNodeTypeTool,
				Tool: &WorkflowToolNode{
					ToolName:  toolName,
					Arguments: map[string]any{"command": "${outputs.missing.status}"},
				},
			},
			{ID: "end-node", Type: workflowNodeTypeEnd},
		},
		Edges: []WorkflowEdge{
			{FromNodeID: "start-node", ToNodeID: "tool-node"},
			{FromNodeID: "tool-node", ToNodeID: "end-node"},
		},
	}
}

func workflowWithFindIconVariableConsumer(toolName string) *WorkflowDefinition {
	return &WorkflowDefinition{
		Nodes: []WorkflowNode{
			{ID: "start-node", Type: workflowNodeTypeStart},
			{
				ID:   "screen-node",
				Type: workflowNodeTypeTool,
				Tool: &WorkflowToolNode{
					ToolName: "screen_control",
					Arguments: map[string]any{
						"mode":   "atomic",
						"action": "find_icon",
						"params": map[string]any{"template_path": "/tmp/icon.png"},
					},
				},
			},
			{
				ID:   "tool-use",
				Type: workflowNodeTypeTool,
				Tool: &WorkflowToolNode{
					ToolName: toolName,
					Arguments: map[string]any{
						"x":     "${find_icon.x}",
						"y":     "${find_icon.y}",
						"point": "${find_icon}",
						"label": "prefix ${find_icon.x}",
					},
				},
			},
			{ID: "end-node", Type: workflowNodeTypeEnd},
		},
		Edges: []WorkflowEdge{
			{FromNodeID: "start-node", ToNodeID: "screen-node"},
			{FromNodeID: "screen-node", ToNodeID: "tool-use"},
			{FromNodeID: "tool-use", ToNodeID: "end-node"},
		},
	}
}

func workflowWithUndefinedFindIconVariable(toolName string) *WorkflowDefinition {
	return &WorkflowDefinition{
		Nodes: []WorkflowNode{
			{ID: "start-node", Type: workflowNodeTypeStart},
			{
				ID:   "tool-node",
				Type: workflowNodeTypeTool,
				Tool: &WorkflowToolNode{
					ToolName:  toolName,
					Arguments: map[string]any{"point": "${find_icon}"},
				},
			},
			{ID: "end-node", Type: workflowNodeTypeEnd},
		},
		Edges: []WorkflowEdge{
			{FromNodeID: "start-node", ToNodeID: "tool-node"},
			{FromNodeID: "tool-node", ToNodeID: "end-node"},
		},
	}
}

func TestValidateTaskDefinitionWorkflowAcceptsStartToEnd(t *testing.T) {
	task := ScheduledTask{
		TaskKind: taskKindWorkflow,
		Workflow: validWorkflowDefinition(),
	}
	if err := validateTaskDefinition(&task); err != nil {
		t.Fatalf("validate workflow task: %v", err)
	}
}

func TestValidateTaskDefinitionWorkflowRejectsInvalidCases(t *testing.T) {
	tests := []struct {
		name string
		task ScheduledTask
		want string
	}{
		{name: "missing workflow", task: ScheduledTask{TaskKind: taskKindWorkflow}, want: "workflow is required"},
		{name: "mixed message", task: ScheduledTask{TaskKind: taskKindWorkflow, Message: "x", Workflow: validWorkflowDefinition()}, want: "does not allow message"},
		{name: "unknown node type", task: ScheduledTask{TaskKind: taskKindWorkflow, Workflow: &WorkflowDefinition{Nodes: []WorkflowNode{{ID: "start-node", Type: "start"}, {ID: "end-node", Type: "noop"}}, Edges: []WorkflowEdge{{FromNodeID: "start-node", ToNodeID: "end-node"}}}}, want: "unsupported workflow node type"},
		{name: "extra edge", task: ScheduledTask{TaskKind: taskKindWorkflow, Workflow: &WorkflowDefinition{Nodes: []WorkflowNode{{ID: "start-node", Type: "start"}, {ID: "end-node", Type: "end"}}, Edges: []WorkflowEdge{{FromNodeID: "start-node", ToNodeID: "end-node"}, {FromNodeID: "start-node", ToNodeID: "end-node"}}}}, want: "duplicate workflow edge"},
		{name: "dangling edge", task: ScheduledTask{TaskKind: taskKindWorkflow, Workflow: &WorkflowDefinition{Nodes: []WorkflowNode{{ID: "start-node", Type: "start"}, {ID: "end-node", Type: "end"}}, Edges: []WorkflowEdge{{FromNodeID: "start-node", ToNodeID: "missing-node"}}}}, want: "unknown to_node_id"},
		{name: "self loop", task: ScheduledTask{TaskKind: taskKindWorkflow, Workflow: &WorkflowDefinition{Nodes: []WorkflowNode{{ID: "start-node", Type: "start"}, {ID: "end-node", Type: "end"}}, Edges: []WorkflowEdge{{FromNodeID: "start-node", ToNodeID: "start-node"}}}}, want: "self-loop"},
		{name: "wrong edge direction", task: ScheduledTask{TaskKind: taskKindWorkflow, Workflow: &WorkflowDefinition{Nodes: []WorkflowNode{{ID: "start-node", Type: "start"}, {ID: "end-node", Type: "end"}}, Edges: []WorkflowEdge{{FromNodeID: "end-node", ToNodeID: "start-node"}}}}, want: "must have in"},
		{name: "start payload on non-start node", task: ScheduledTask{TaskKind: taskKindWorkflow, Workflow: &WorkflowDefinition{Nodes: []WorkflowNode{{ID: "start-node", Type: "start"}, {ID: "tool-node", Type: "tool", Start: &WorkflowStartNode{}, Tool: &WorkflowToolNode{ToolName: "script_exec"}}, {ID: "end-node", Type: "end"}}, Edges: []WorkflowEdge{{FromNodeID: "start-node", ToNodeID: "tool-node"}, {FromNodeID: "tool-node", ToNodeID: "end-node"}}}}, want: "payload does not match type"},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			err := validateTaskDefinition(&test.task)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("expected error containing %q, got %v", test.want, err)
			}
		})
	}
}

func TestTaskWorkflowCreateUpdateListAndRunNow(t *testing.T) {
	_, service, _ := newTestHandlerWithService(t, nil, nil)

	createdRaw, code, err := service.executeTaskCreateAction(taskCreateParams{
		TaskKind:        taskKindWorkflow,
		Workflow:        validWorkflowDefinition(),
		IntervalSeconds: 60,
	}, "trace-workflow-create")
	if err != nil || code != http.StatusCreated {
		t.Fatalf("create workflow task: code=%d err=%v", code, err)
	}
	created := createdRaw.(taskPayload)
	if created.TaskKind != taskKindWorkflow || created.Workflow == nil {
		t.Fatalf("unexpected created payload: %#v", created)
	}

	updatedWorkflow := &WorkflowDefinition{
		Nodes: []WorkflowNode{{ID: "entry", Type: "start"}, {ID: "finish", Type: "end"}},
		Edges: []WorkflowEdge{{FromNodeID: "entry", ToNodeID: "finish"}},
	}
	updatedRaw, code, err := service.executeTaskUpdateAction(taskUpdateParams{
		ID:       created.ID,
		Workflow: updatedWorkflow,
	}, "trace-workflow-update")
	if err != nil || code != http.StatusOK {
		t.Fatalf("update workflow task: code=%d err=%v", code, err)
	}
	updated := updatedRaw.(taskPayload)
	if updated.Workflow == nil || updated.Workflow.Edges[0].FromNodeID != "entry" {
		t.Fatalf("unexpected updated payload: %#v", updated)
	}

	listRaw, code, err := service.executeTaskListAction(taskListScopeUser, "trace-workflow-list")
	if err != nil || code != http.StatusOK {
		t.Fatalf("list workflow tasks: code=%d err=%v", code, err)
	}
	items := listRaw.([]taskPayload)
	if len(items) != 1 || items[0].TaskKind != taskKindWorkflow {
		t.Fatalf("unexpected list payload: %#v", items)
	}

	runRaw, code, err := service.executeTaskRunNowAction(taskIDParams{ID: created.ID}, "trace-workflow-run")
	if err != nil || code != http.StatusOK {
		t.Fatalf("run workflow task: code=%d err=%v", code, err)
	}
	run := runRaw.(taskRunPayload)
	if run.Run.Status != taskRunStatusSuccess {
		t.Fatalf("unexpected run status: %#v", run.Run)
	}
	if run.Run.ResponsePreview != "workflow completed: entry -> finish" {
		t.Fatalf("unexpected run preview: %#v", run.Run)
	}
}

func TestTaskWorkflowCreateRejectsMixedFields(t *testing.T) {
	_, service, _ := newTestHandlerWithService(t, nil, nil)

	_, code, err := service.executeTaskCreateAction(taskCreateParams{
		TaskKind:        taskKindWorkflow,
		Message:         "should-fail",
		Workflow:        validWorkflowDefinition(),
		IntervalSeconds: 60,
	}, "trace-workflow-invalid")
	if err == nil {
		t.Fatal("expected mixed-field workflow create to fail")
	}
	if code != http.StatusBadRequest {
		t.Fatalf("unexpected status code: got %d want %d", code, http.StatusBadRequest)
	}
	if !strings.Contains(err.Error(), "does not allow message") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTaskWorkflowGetReturnsStoredDefinition(t *testing.T) {
	_, service, _ := newTestHandlerWithService(t, nil, nil)

	createdRaw, _, err := service.executeTaskCreateAction(taskCreateParams{
		TaskKind: taskKindWorkflow,
		Workflow: validWorkflowDefinition(),
		CronExpr: "*/5 * * * *",
	}, "trace-workflow-get")
	if err != nil {
		t.Fatalf("create workflow task: %v", err)
	}
	created := createdRaw.(taskPayload)
	gotRaw, code, err := service.executeTaskGetAction(taskIDParams{ID: created.ID}, "trace-workflow-get")
	if err != nil || code != http.StatusOK {
		t.Fatalf("get workflow task: code=%d err=%v", code, err)
	}
	got := gotRaw.(taskPayload)
	if got.Workflow == nil {
		t.Fatalf("expected workflow in get payload: %#v", got)
	}
	encoded, err := json.Marshal(got.Workflow)
	if err != nil {
		t.Fatalf("marshal workflow payload: %v", err)
	}
	if !strings.Contains(string(encoded), `"start-node"`) || !strings.Contains(string(encoded), `"end-node"`) {
		t.Fatalf("unexpected workflow payload: %s", string(encoded))
	}
}

func TestTaskWorkflowSystemScopeExcludesWorkflow(t *testing.T) {
	_, service, _ := newTestHandlerWithService(t, nil, nil)

	if _, _, err := service.executeTaskCreateAction(taskCreateParams{
		TaskKind:        taskKindWorkflow,
		Workflow:        validWorkflowDefinition(),
		IntervalSeconds: 60,
	}, "trace-workflow-system-scope"); err != nil {
		t.Fatalf("create workflow task: %v", err)
	}
	raw, code, err := service.executeTaskListAction(taskListScopeSystem, "trace-system-scope")
	if err != nil || code != http.StatusOK {
		t.Fatalf("list system tasks: code=%d err=%v", code, err)
	}
	if items := raw.([]taskPayload); len(items) != 0 {
		t.Fatalf("expected empty system scope, got %#v", items)
	}
}

func validWorkflowDefinition() *WorkflowDefinition {
	return &WorkflowDefinition{
		Nodes: []WorkflowNode{
			{ID: "start-node", Type: workflowNodeTypeStart},
			{ID: "end-node", Type: workflowNodeTypeEnd},
		},
		Edges: []WorkflowEdge{{FromNodeID: "start-node", ToNodeID: "end-node"}},
	}
}

func TestTaskWorkflowRunnerDoesNotNeedAgentExecution(t *testing.T) {
	result := taskExecutorAdapter{}.executeWorkflowTask(context.Background(), ScheduledTask{
		TaskKind: taskKindWorkflow,
		Workflow: validWorkflowDefinition(),
	}, "trace-workflow-direct")
	if result.Status != taskRunStatusSuccess {
		t.Fatalf("unexpected workflow execution result: %#v", result)
	}
}

func TestTaskWorkflowRunNowUsesExistingScheduler(t *testing.T) {
	_, service, _ := newTestHandlerWithService(t, nil, nil)
	if _, _, err := service.executeTaskCreateAction(taskCreateParams{
		TaskKind:        taskKindWorkflow,
		Workflow:        validWorkflowDefinition(),
		IntervalSeconds: 60,
	}, "trace-workflow-scheduler"); err != nil {
		t.Fatalf("create workflow task: %v", err)
	}
	runner, code, err := service.requireTaskMutationRunner()
	if err != nil || code != http.StatusOK {
		t.Fatalf("require runner: code=%d err=%v", code, err)
	}
	taskStore, code, err := service.requireTaskStore()
	if err != nil || code != http.StatusOK {
		t.Fatalf("require task store: code=%d err=%v", code, err)
	}
	tasks, err := taskStore.ListTasks()
	if err != nil || len(tasks) != 1 {
		t.Fatalf("list tasks: tasks=%d err=%v", len(tasks), err)
	}
	run, err := runner.RunNow(taskIDParams{ID: tasks[0].ID}, "trace-workflow-runner")
	if err != nil {
		t.Fatalf("run now through runner: %v", err)
	}
	if run.Run.Status != taskRunStatusSuccess {
		t.Fatalf("unexpected runner status: %#v", run.Run)
	}
}

func TestNormalizeConfiguredToolLists_RejectsOverlap(t *testing.T) {
	_, _, err := normalizeConfiguredToolLists([]string{"script_exec"}, []string{"script_exec"})
	if err == nil {
		t.Fatal("expected overlap error")
	}
}

func TestNormalizeConfiguredToolLists_RejectsUnknownTool(t *testing.T) {
	_, _, err := normalizeConfiguredToolLists([]string{"ghost_tool"}, nil)
	if err == nil {
		t.Fatal("expected unknown tool error")
	}
}

func TestNormalizeConfiguredToolLists_AllowsCodexCLI(t *testing.T) {
	allowlist, blocklist, err := normalizeConfiguredToolLists([]string{"codex_cli"}, nil)
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
	allowlist, blocklist, err := normalizeConfiguredToolLists(nil, []string{"ask_human", "script_exec", "sfind"})
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

func TestSessionTurnPreparer_SelectToolsForTurn_HasNoResidentToolsWithoutAllowlist(t *testing.T) {
	preparer := &sessionTurnPreparer{}
	deps := newRunnerTestDeps(bridgeconfig.Config{ToolSelector: bridgeconfig.ToolSelectorConfig{Blocklist: []string{"script_exec"}}, MaxTurns: 6})

	catalog, prompt, err := preparer.selectToolsForTurn(context.Background(), deps, nil, agent.NewHistory("system prompt"), "read config", false, "trace-policy-disabled")
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
	preparer := &sessionTurnPreparer{}
	deps := newRunnerTestDeps(bridgeconfig.Config{
		ToolSelector: bridgeconfig.ToolSelectorConfig{
			Allowlist: []string{"script_exec"},
		},
		MaxTurns: 6,
	})

	catalog, _, err := preparer.selectToolsForTurn(context.Background(), deps, nil, agent.NewHistory("system prompt"), "read config", false, "trace-policy-allowlist")
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
	preparer := &sessionTurnPreparer{selectorFactory: func(bridgeconfig.Config, tools.ToolCatalog) bridgeruntime.SelectorEngine { return selector }}
	deps := newRunnerTestDeps(bridgeconfig.Config{
		ToolSelector: bridgeconfig.ToolSelectorConfig{
			Enabled:   true,
			Mode:      "llm",
			Allowlist: []string{"ask_human"},
		},
		MaxTurns:   6,
		PromptsDir: filepath.Join(t.TempDir(), "prompts"),
	})

	catalog, prompt, err := preparer.selectToolsForTurn(context.Background(), deps, nil, agent.NewHistory("system prompt"), "read config", false, "trace-policy-subset")
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
	preparer := &sessionTurnPreparer{
		selectorFactory: func(_ bridgeconfig.Config, catalog tools.ToolCatalog) bridgeruntime.SelectorEngine {
			available = toolCatalogNames(catalog)
			return &fakeSelectorEngine{result: bridgeruntime.ToolSelectorResult{Mode: "all"}}
		},
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

	_, _, err := preparer.selectToolsForTurn(context.Background(), deps, nil, agent.NewHistory("system prompt"), "read config", false, "trace-policy-visible")
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
	preparer := &sessionTurnPreparer{
		selectorFactory: func(_ bridgeconfig.Config, catalog tools.ToolCatalog) bridgeruntime.SelectorEngine {
			available = toolCatalogNames(catalog)
			return &fakeSelectorEngine{result: bridgeruntime.ToolSelectorResult{Mode: "all"}}
		},
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

	catalog, _, err := preparer.selectToolsForTurn(context.Background(), deps, nil, agent.NewHistory("system prompt"), "read config", false, "trace-policy-allowlist-only")
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
	preparer := &sessionTurnPreparer{}
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

	catalog, _, err := preparer.selectToolsForTurn(context.Background(), deps, nil, agent.NewHistory("system prompt"), "find tools", false, "trace-policy-tool-search")
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
	preparer := &sessionTurnPreparer{selectorFactory: func(bridgeconfig.Config, tools.ToolCatalog) bridgeruntime.SelectorEngine { return selector }}
	deps := newRunnerTestDeps(bridgeconfig.Config{
		ToolSelector: bridgeconfig.ToolSelectorConfig{
			Enabled: true,
			Mode:    "llm",
		},
		MaxTurns:   6,
		PromptsDir: filepath.Join(t.TempDir(), "prompts"),
	})

	catalog, prompt, err := preparer.selectToolsForTurn(context.Background(), deps, nil, agent.NewHistory("system prompt"), "read config", false, "trace-policy-empty-resident-subset")
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

func containsToolName(names []string, target string) bool {
	for _, name := range names {
		if name == target {
			return true
		}
	}
	return false
}

type fakeSelectorEngine struct {
	result bridgeruntime.ToolSelectorResult
}

func (f *fakeSelectorEngine) SelectTools(_ context.Context, _ string, _ []llm.Message, _ string, _ string) bridgeruntime.ToolSelectorResult {
	return f.result
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
	for _, name := range []string{"ask_human", "script_exec", "codex_cli", "web_search", "sfind"} {
		registry.Register(&runnerMockTool{name: name})
	}
	return registry
}

func newRunnerTestDeps(cfg bridgeconfig.Config) agentRuntimeDependencies {
	if strings.TrimSpace(cfg.PromptsDir) == "" {
		cfg.PromptsDir = os.Getenv("GHOST_PROMPTS_DIR")
	}
	return agentRuntimeDependencies{
		cfg:      cfg,
		registry: newRunnerTestRegistry(),
	}
}
