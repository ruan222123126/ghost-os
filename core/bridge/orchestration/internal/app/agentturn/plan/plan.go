package plan

import (
	"context"
	"errors"
	"net/http"

	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/internal/runtimeutil"
	"ghost-os/bridge/llm"
	bridgemode "ghost-os/bridge/mode"
	appsessions "ghost-os/bridge/orchestration/internal/app/sessions"
	"ghost-os/bridge/orchestration/internal/domain/runtimeopts"
	internaltrace "ghost-os/bridge/orchestration/internal/trace"
	"ghost-os/bridge/session"
)

var ErrRuntimeFactoryRequired = errors.New("plan mode runtime factory is not configured")

type RuntimeDependencies struct {
	Config       bridgeconfig.Config
	Client       llm.Completer
	SystemPrompt string
	Cleanup      func()
}

func (d RuntimeDependencies) Close() {
	if d.Cleanup != nil {
		d.Cleanup()
	}
}

type RuntimeFactory interface {
	Build(store bridgeconfig.Store) (RuntimeDependencies, error)
}

type RunRegistry interface {
	Register(sessionID string, traceID string, cancel context.CancelFunc) error
	Unregister(sessionID string)
}

type RequestRuntimeOptions = runtimeopts.RequestOptions

type Request struct {
	UserInput      llm.Message
	SessionID      string
	RequestRuntime *RequestRuntimeOptions
}

type Result struct {
	Message   string
	SessionID string
}

type Runner struct {
	RuntimeFactory RuntimeFactory
	ConfigStore    bridgeconfig.Store
	SessionStore   *session.Store
	RunRegistry    RunRegistry
}

func (r Runner) Execute(
	ctx context.Context,
	req Request,
	traceID string,
) (Result, error) {
	if r.RuntimeFactory == nil {
		return Result{}, ErrRuntimeFactoryRequired
	}
	runtimeStore := applyRequestRuntimeOptionsToStore(r.ConfigStore, req.RequestRuntime)
	deps, err := r.RuntimeFactory.Build(runtimeStore)
	if err != nil {
		return Result{}, err
	}
	defer deps.Close()
	runtimeSelection := runtimeutil.BuildGhostRuntimeSelection(
		runtimeStore,
		deps.Config,
		session.RuntimeSelectionModePlan,
	)

	sess, history, err := r.loadHistory(req.SessionID, deps)
	if err != nil {
		return Result{}, err
	}
	execCtx, cleanup, err := r.register(ctx, sess.ID, traceID)
	if err != nil {
		return Result{}, err
	}
	defer cleanup()

	assistantMessage, conversationState, err := completePlan(execCtx, deps, history, req.UserInput)
	if err != nil {
		return Result{}, err
	}
	if err := r.persistPlanTurn(sess, req.UserInput, assistantMessage, conversationState, runtimeSelection); err != nil {
		return Result{}, err
	}
	return Result{Message: assistantMessage.Text, SessionID: sess.ID}, nil
}

func ErrorStatus(err error) int {
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

func (r Runner) loadHistory(
	sessionID string,
	deps RuntimeDependencies,
) (*session.Session, *llm.CompletionRequest, error) {
	historyBuilder := appsessions.NewHistoryBuilderFromConfig(
		deps.Config,
		deps.SystemPrompt,
		r.SessionStore,
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
	deps RuntimeDependencies,
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

func (r Runner) persistPlanTurn(
	sess *session.Session,
	userInput llm.Message,
	assistantMessage llm.Message,
	conversationState llm.ConversationState,
	runtimeSelection *session.RuntimeSelection,
) error {
	if sess == nil {
		return session.ErrSessionNotFound
	}
	sess.ConversationState = conversationState
	if runtimeSelection != nil {
		sess.SetLastRuntimeSelection(*runtimeSelection)
	}
	sess.AddMessage(userInput)
	sess.AddMessage(assistantMessage)
	if r.SessionStore == nil {
		return nil
	}
	return r.SessionStore.Save(sess)
}

func (r Runner) register(
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

func applyRequestRuntimeOptionsToStore(
	store bridgeconfig.Store,
	options *RequestRuntimeOptions,
) bridgeconfig.Store {
	if options == nil {
		return store
	}
	return bridgeconfig.WithProjectRootOverride(store, options.ProjectRoot)
}
