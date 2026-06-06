package agentturn

import (
	"context"
	"errors"
	"net/http"

	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/llm"
	bridgemode "ghost-os/bridge/mode"
	"ghost-os/bridge/orchestration/internal/domain/sessionturn"
	internaltrace "ghost-os/bridge/orchestration/internal/trace"
	"ghost-os/bridge/session"
)

var ErrPlanModeRuntimeFactoryRequired = errors.New("plan mode runtime factory is not configured")

type PlanRuntimeDependencies struct {
	Config       bridgeconfig.Config
	Client       llm.Completer
	SystemPrompt string
	Cleanup      func()
}

func (d PlanRuntimeDependencies) Close() {
	if d.Cleanup != nil {
		d.Cleanup()
	}
}

type PlanRuntimeFactory interface {
	Build(store bridgeconfig.Store) (PlanRuntimeDependencies, error)
}

type PlanRunRegistry interface {
	Register(sessionID string, traceID string, cancel context.CancelFunc) error
	Unregister(sessionID string)
}

type PlanResult struct {
	Message   string
	SessionID string
}

type PlanRunner struct {
	RuntimeFactory PlanRuntimeFactory
	ConfigStore    bridgeconfig.Store
	SessionStore   *session.Store
	RunRegistry    PlanRunRegistry
}

func (r PlanRunner) Execute(
	ctx context.Context,
	prepared PreparedRequest,
	traceID string,
) (PlanResult, error) {
	if r.RuntimeFactory == nil {
		return PlanResult{}, ErrPlanModeRuntimeFactoryRequired
	}
	deps, err := r.RuntimeFactory.Build(applyPlanRequestRuntimeOptionsToStore(r.ConfigStore, prepared.RequestRuntime))
	if err != nil {
		return PlanResult{}, err
	}
	defer deps.Close()

	sess, history, err := r.loadHistory(prepared.SessionID, deps)
	if err != nil {
		return PlanResult{}, err
	}
	execCtx, cleanup, err := r.register(ctx, sess.ID, traceID)
	if err != nil {
		return PlanResult{}, err
	}
	defer cleanup()

	assistantMessage, conversationState, err := completePlan(execCtx, deps, history, prepared.UserInput)
	if err != nil {
		return PlanResult{}, err
	}
	if err := r.persistPlanTurn(sess, prepared.UserInput, assistantMessage, conversationState); err != nil {
		return PlanResult{}, err
	}
	return PlanResult{Message: assistantMessage.Text, SessionID: sess.ID}, nil
}

func PlanErrorStatus(err error) int {
	switch {
	case errors.Is(err, session.ErrInvalidSessionID):
		return http.StatusBadRequest
	case errors.Is(err, session.ErrSessionNotFound):
		return http.StatusNotFound
	case errors.Is(err, internaltrace.ErrSessionInflight), errors.Is(err, context.Canceled):
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}

func (r PlanRunner) loadHistory(
	sessionID string,
	deps PlanRuntimeDependencies,
) (*session.Session, *llm.CompletionRequest, error) {
	historyBuilder := sessionturn.NewSessionHistoryBuilder(
		deps.Config.Provider,
		deps.SystemPrompt,
		r.SessionStore,
		deps.Config.ToolSearch.IdleTurns,
		deps.Config.MicrocompactEnabled,
		"",
	)
	sess, created, err := historyBuilder.LoadOrCreateSession(sessionID)
	if err != nil {
		return nil, nil, err
	}
	if created && r.SessionStore != nil {
		if err := r.SessionStore.Save(sess); err != nil {
			return nil, nil, err
		}
	}
	history := historyBuilder.BuildHistory(sess)
	history.UpdateSystemPrompt(bridgemode.BuildSystemPrompt(deps.SystemPrompt))
	request := &llm.CompletionRequest{
		Messages:          history.Messages(),
		ConversationState: history.ConversationState(),
		ResponseOptions:   llm.CloneResponseOptions(deps.Config.ResponseOptions),
	}
	return sess, request, nil
}

func completePlan(
	ctx context.Context,
	deps PlanRuntimeDependencies,
	request *llm.CompletionRequest,
	userInput llm.Message,
) (llm.Message, llm.ConversationState, error) {
	completionRequest := llm.CompletionRequest{
		Messages:          appendPlanCompletionMessages(request.Messages, userInput),
		ConversationState: request.ConversationState,
		ResponseOptions:   request.ResponseOptions,
	}
	response, err := deps.Client.Complete(ctx, completionRequest)
	if err != nil {
		return llm.Message{}, llm.ConversationState{}, err
	}
	return bridgemode.ValidateResponse(response)
}

func appendPlanCompletionMessages(messages []llm.Message, userInput llm.Message) []llm.Message {
	appended := llm.CloneMessages(messages)
	appended = append(appended, llm.CloneMessages([]llm.Message{userInput})...)
	return appended
}

func (r PlanRunner) persistPlanTurn(
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
	if r.SessionStore == nil {
		return nil
	}
	return r.SessionStore.Save(sess)
}

func (r PlanRunner) register(
	ctx context.Context,
	sessionID string,
	traceID string,
) (context.Context, func(), error) {
	if r.RunRegistry == nil {
		return ctx, func() {}, nil
	}
	execCtx, cancel := context.WithCancel(ctx)
	if err := r.RunRegistry.Register(sessionID, traceID, cancel); err != nil {
		cancel()
		return ctx, func() {}, err
	}
	return execCtx, func() {
		cancel()
		r.RunRegistry.Unregister(sessionID)
	}, nil
}

func applyPlanRequestRuntimeOptionsToStore(
	store bridgeconfig.Store,
	options *RequestRuntimeOptions,
) bridgeconfig.Store {
	if options == nil {
		return store
	}
	return bridgeconfig.WithProjectRootOverride(store, options.ProjectRoot)
}
