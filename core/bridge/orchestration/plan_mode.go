package orchestration

import (
	"context"
	"errors"
	"net/http"

	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/llm"
	bridgemode "ghost-os/bridge/mode"
	"ghost-os/bridge/session"
)

var (
	errPlanModeRuntimeFactoryRequired = errors.New("plan mode runtime factory is not configured")
	errPlanModeToolCallsForbidden     = bridgemode.ErrToolCallsForbidden
	errPlanModeToolTagForbidden       = bridgemode.ErrToolTagForbidden
	errPlanModeEmptyResponse          = bridgemode.ErrEmptyResponse
)

type planModeRunner struct {
	runtimeFactory AgentRuntimeFactory
	configStore    bridgeconfig.Store
	sessionStore   *session.Store
	runRegistry    *RunRegistry
}

func newPlanModeRunner(service *bridgeService) planModeRunner {
	if service == nil {
		return planModeRunner{}
	}
	factory := service.runtimeFactory
	if factory == nil {
		factory = newAgentRuntimeFactoryWithTaskManager(service.taskToolManager())
	}
	return planModeRunner{
		runtimeFactory: factory,
		configStore:    service.configStore,
		sessionStore:   service.sessionStore,
		runRegistry:    service.runRegistry,
	}
}

func (s *bridgeService) executePlanModeAction(
	ctx context.Context,
	prepared preparedAgentTurnRequest,
	traceID string,
) (agentResponse, int, error) {
	payload, err := newPlanModeRunner(s).Execute(ctx, prepared, traceID)
	if err != nil {
		return agentResponse{}, mapPlanModeError(err), err
	}
	return payload, http.StatusOK, nil
}

func mapPlanModeError(err error) int {
	switch {
	case errors.Is(err, session.ErrInvalidSessionID):
		return http.StatusBadRequest
	case errors.Is(err, session.ErrSessionNotFound):
		return http.StatusNotFound
	case errors.Is(err, ErrSessionInflight), errors.Is(err, context.Canceled):
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}

func (r planModeRunner) Execute(
	ctx context.Context,
	prepared preparedAgentTurnRequest,
	traceID string,
) (agentResponse, error) {
	if r.runtimeFactory == nil {
		return agentResponse{}, errPlanModeRuntimeFactoryRequired
	}
	deps, err := r.runtimeFactory.Build(r.configStore)
	if err != nil {
		return agentResponse{}, err
	}
	defer deps.Close()

	sess, history, err := r.loadHistory(prepared.sessionID, deps)
	if err != nil {
		return agentResponse{}, err
	}
	execCtx, cleanup, err := r.register(ctx, sess.ID, traceID)
	if err != nil {
		return agentResponse{}, err
	}
	defer cleanup()

	assistantMessage, conversationState, err := r.completePlan(execCtx, deps, history, prepared.userInput)
	if err != nil {
		return agentResponse{}, err
	}
	if err := r.persistPlanTurn(sess, prepared.userInput, assistantMessage, conversationState); err != nil {
		return agentResponse{}, err
	}
	return newAgentResponsePayload(assistantMessage.Text, sess.ID, nil, agentResponseMeta{Mode: agentModePlan})
}

func (r planModeRunner) loadHistory(
	sessionID string,
	deps agentRuntimeDependencies,
) (*session.Session, *llm.CompletionRequest, error) {
	historyBuilder := newSessionHistoryBuilder(
		deps.cfg.Provider,
		deps.systemPrompt,
		r.sessionStore,
		deps.cfg.ToolSearch.IdleTurns,
	)
	sess, created, err := historyBuilder.LoadOrCreateSession(sessionID)
	if err != nil {
		return nil, nil, err
	}
	if created && r.sessionStore != nil {
		if err := r.sessionStore.Save(sess); err != nil {
			return nil, nil, err
		}
	}
	history := historyBuilder.BuildHistory(sess)
	history.UpdateSystemPrompt(buildPlanModeSystemPrompt(deps.systemPrompt))
	request := &llm.CompletionRequest{
		Messages:          history.Messages(),
		ConversationState: history.ConversationState(),
		ResponseOptions:   llm.CloneResponseOptions(deps.cfg.ResponseOptions),
	}
	return sess, request, nil
}

func (r planModeRunner) completePlan(
	ctx context.Context,
	deps agentRuntimeDependencies,
	request *llm.CompletionRequest,
	userInput llm.Message,
) (llm.Message, llm.ConversationState, error) {
	completionRequest := llm.CompletionRequest{
		Messages:          appendCompletionMessages(request.Messages, userInput),
		ConversationState: request.ConversationState,
		ResponseOptions:   request.ResponseOptions,
	}
	response, err := deps.client.Complete(ctx, completionRequest)
	if err != nil {
		return llm.Message{}, llm.ConversationState{}, err
	}
	return validatePlanModeResponse(response)
}

func appendCompletionMessages(messages []llm.Message, userInput llm.Message) []llm.Message {
	appended := llm.CloneMessages(messages)
	appended = append(appended, llm.CloneMessages([]llm.Message{userInput})...)
	return appended
}

func validatePlanModeResponse(
	response *llm.CompletionResponse,
) (llm.Message, llm.ConversationState, error) {
	return bridgemode.ValidateResponse(response)
}

func containsPlanModeToolSyntax(text string) bool {
	return bridgemode.ContainsToolSyntax(text)
}

func (r planModeRunner) persistPlanTurn(
	sess *session.Session,
	userInput llm.Message,
	assistantMessage llm.Message,
	conversationState llm.ConversationState,
) error {
	if sess == nil {
		return session.ErrSessionNotFound
	}
	sess.ConversationState = conversationState
	sess.AddMessage(userInput)
	sess.AddMessage(assistantMessage)
	if r.sessionStore == nil {
		return nil
	}
	return r.sessionStore.Save(sess)
}

func (r planModeRunner) register(
	ctx context.Context,
	sessionID string,
	traceID string,
) (context.Context, func(), error) {
	if r.runRegistry == nil {
		return ctx, func() {}, nil
	}
	execCtx, cancel := context.WithCancel(ctx)
	if err := r.runRegistry.Register(sessionID, traceID, cancel); err != nil {
		cancel()
		return ctx, func() {}, err
	}
	return execCtx, func() {
		cancel()
		r.runRegistry.Unregister(sessionID)
	}, nil
}

func buildPlanModeSystemPrompt(basePrompt string) string {
	return bridgemode.BuildSystemPrompt(basePrompt)
}
