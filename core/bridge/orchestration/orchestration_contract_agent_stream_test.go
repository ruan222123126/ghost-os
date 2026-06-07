package orchestration

import (
	"context"
	"errors"
	"ghost-os/bridge/agent"
	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/llm"
	"ghost-os/bridge/orchestration/internal/domain/sessionturn"
	"ghost-os/bridge/tools"
	"os"
	"strings"
	"testing"
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

func TestValidateAgentResponsePayloadAcceptsPlanModeResponse(t *testing.T) {
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

	got := sessionturn.MessagesWithSystemPrompt(input, "new prompt")

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

	got := sessionturn.MessagesWithSystemPrompt(input, "system prompt")

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
				MaxTurns: 4,
				Provider: bridgeconfig.ProviderConfig{Model: "gpt-4o"},
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
	if payload.Mode != "" {
		t.Fatalf("expected standard response mode, got %+v", payload)
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
	f.projectRoots = append(f.projectRoots, cfg.ProjectRoot)
	return agentRuntimeDependencies{
		cfg:          cfg,
		client:       f.client,
		registry:     tools.NewRegistry(),
		systemPrompt: "system prompt",
	}, nil
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
