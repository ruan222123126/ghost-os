package orchestration

import (
	"context"
	"errors"

	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/llm"
	agentadapter "ghost-os/bridge/orchestration/internal/adapters/agent"
	"ghost-os/bridge/orchestration/internal/app/agentturn"
	"ghost-os/bridge/orchestration/internal/app/agentturn/turnstate"
	internaltrace "ghost-os/bridge/orchestration/internal/trace"
	bridgeruntime "ghost-os/bridge/runtime"
	"ghost-os/bridge/session"
	"ghost-os/bridge/streaming"
	"ghost-os/bridge/tools"
)

// SessionTurnRunner 为 service 层暴露瘦执行接口，隐藏运行时装配与会话编排细节。
type SessionTurnRunner interface {
	RunTurn(ctx context.Context, message string, sessionID string, traceID string) (string, string, error)
	RunTurnStream(ctx context.Context, message string, sessionID string, traceID string, sink streaming.Sink) (string, string, error)
}

type SessionTurnRunnerWithOverrides interface {
	RunTurnWithOverrides(
		ctx context.Context,
		message string,
		sessionID string,
		traceID string,
		runtimeOverrides *TaskRuntimeOverrides,
	) (string, string, error)
}

type SessionTurnStreamRunnerWithOverrides interface {
	RunTurnStreamWithOverrides(
		ctx context.Context,
		message string,
		sessionID string,
		traceID string,
		sink streaming.Sink,
		runtimeOverrides *TaskRuntimeOverrides,
	) (string, string, error)
}

type StructuredSessionTurnRunner interface {
	RunTurnInput(ctx context.Context, input llm.Message, sessionID string, traceID string) (string, string, error)
	RunTurnStreamInput(ctx context.Context, input llm.Message, sessionID string, traceID string, sink streaming.Sink) (string, string, error)
}

type requestRuntimeOptions = agentturn.RequestRuntimeOptions

type requestRuntimeAwareRunner interface {
	WithRequestRuntimeOptions(*requestRuntimeOptions) any
}

func applyRequestRuntimeOptionsToStore(
	store bridgeconfig.Store,
	options *requestRuntimeOptions,
) bridgeconfig.Store {
	if options == nil {
		return store
	}
	return bridgeconfig.WithProjectRootOverride(store, options.ProjectRoot)
}

type sessionTurnSetupError = agentturn.SessionSetupError
type sessionTurnState = turnstate.State
type SessionTurnCommitter = turnstate.Committer

func newSessionTurnCommitter(sessionStore *session.Store) *SessionTurnCommitter {
	return turnstate.NewCommitter(sessionStore)
}

type sessionTurnRuntimeFactory struct {
	inner AgentRuntimeFactory
}

func (f sessionTurnRuntimeFactory) Build(store bridgeconfig.Store) (agentadapter.RuntimeDependencies, error) {
	inner := f.inner
	if inner == nil {
		inner = newAgentRuntimeFactory()
	}
	deps, err := inner.Build(store)
	if err != nil {
		return nil, err
	}
	return deps, nil
}

func newSessionTurnPreparer(
	runtimeFactory AgentRuntimeFactory,
	configStore bridgeconfig.Store,
	sessionStore *session.Store,
	runRegistry *RunRegistry,
	selectorFactory func(bridgeconfig.Config, tools.ToolCatalog) bridgeruntime.SelectorEngine,
) agentadapter.SessionTurnPreparer {
	if runtimeFactory == nil {
		runtimeFactory = newAgentRuntimeFactory()
	}
	config := agentadapter.SessionTurnPreparerConfig{
		RuntimeFactory:  sessionTurnRuntimeFactory{inner: runtimeFactory},
		ConfigStore:     configStore,
		SessionStore:    sessionStore,
		SelectorFactory: selectorFactory,
		Logger:          serviceActionLogger{},
	}
	if runRegistry != nil {
		config.RunRegistry = runRegistry
	}
	return agentadapter.NewSessionTurnPreparer(config)
}

// SessionAgentRunner 负责执行单轮 agent；运行时装配下沉到 sessionTurnPreparer。
type SessionAgentRunner struct {
	runtimeFactory  AgentRuntimeFactory
	configStore     bridgeconfig.Store
	sessionStore    *session.Store
	runRegistry     *RunRegistry
	selectorFactory func(bridgeconfig.Config, tools.ToolCatalog) bridgeruntime.SelectorEngine
}

func NewSessionAgentRunner(
	runtimeFactory AgentRuntimeFactory,
	configStore bridgeconfig.Store,
	sessionStore *session.Store,
	runRegistry *RunRegistry,
) *SessionAgentRunner {
	if runtimeFactory == nil {
		runtimeFactory = newAgentRuntimeFactory()
	}
	return &SessionAgentRunner{
		runtimeFactory: runtimeFactory,
		configStore:    configStore,
		sessionStore:   sessionStore,
		runRegistry:    runRegistry,
	}
}

func (r *SessionAgentRunner) WithRequestRuntimeOptions(options *requestRuntimeOptions) any {
	if r == nil || options == nil {
		return r
	}
	cloned := *r
	cloned.configStore = applyRequestRuntimeOptionsToStore(r.configStore, options)
	return &cloned
}

func (r *SessionAgentRunner) prepareTurnWithRuntimeOverrides(
	ctx context.Context,
	userInput llm.Message,
	sessionID string,
	traceID string,
	runtimeOverrides *TaskRuntimeOverrides,
) (*sessionTurnState, error) {
	return r.turnPreparer().PrepareWithRuntimeOverrides(ctx, userInput, sessionID, traceID, runtimeOverrides)
}

func (r *SessionAgentRunner) turnPreparer() agentadapter.SessionTurnPreparer {
	if r == nil {
		return newSessionTurnPreparer(nil, nil, nil, nil, nil)
	}
	return newSessionTurnPreparer(
		r.runtimeFactory,
		r.configStore,
		r.sessionStore,
		r.runRegistry,
		r.selectorFactory,
	)
}

func (r *SessionAgentRunner) RunTurn(ctx context.Context, userMessage string, sessionID string, traceID string) (string, string, error) {
	return r.RunTurnInput(ctx, llm.Message{
		Role: llm.RoleUser,
		Text: userMessage,
	}, sessionID, traceID)
}

func (r *SessionAgentRunner) RunTurnWithOverrides(
	ctx context.Context,
	userMessage string,
	sessionID string,
	traceID string,
	runtimeOverrides *TaskRuntimeOverrides,
) (string, string, error) {
	return r.RunTurnInputWithOverrides(ctx, llm.Message{
		Role: llm.RoleUser,
		Text: userMessage,
	}, sessionID, traceID, runtimeOverrides)
}

func (r *SessionAgentRunner) RunTurnStreamWithOverrides(
	ctx context.Context,
	userMessage string,
	sessionID string,
	traceID string,
	sink streaming.Sink,
	runtimeOverrides *TaskRuntimeOverrides,
) (string, string, error) {
	return r.RunTurnStreamInputWithOverrides(ctx, llm.Message{
		Role: llm.RoleUser,
		Text: userMessage,
	}, sessionID, traceID, sink, runtimeOverrides)
}

func (r *SessionAgentRunner) RunTurnInput(ctx context.Context, input llm.Message, sessionID string, traceID string) (string, string, error) {
	return r.RunTurnInputWithOverrides(ctx, input, sessionID, traceID, nil)
}

func (r *SessionAgentRunner) RunTurnInputWithOverrides(
	ctx context.Context,
	input llm.Message,
	sessionID string,
	traceID string,
	runtimeOverrides *TaskRuntimeOverrides,
) (string, string, error) {
	turn, err := r.prepareTurnWithRuntimeOverrides(ctx, input, sessionID, traceID, runtimeOverrides)
	if err != nil {
		return "", "", err
	}
	defer turn.Close()

	response, runErr := turn.Agent.RunMessageWithTraceID(turn.ExecCtx, input, turn.TraceID)
	return turn.Complete(response, runErr, nil)
}

func (r *SessionAgentRunner) RunTurnStream(ctx context.Context, userMessage string, sessionID string, traceID string, sink streaming.Sink) (string, string, error) {
	return r.RunTurnStreamInput(ctx, llm.Message{
		Role: llm.RoleUser,
		Text: userMessage,
	}, sessionID, traceID, sink)
}

func (r *SessionAgentRunner) RunTurnStreamInput(ctx context.Context, input llm.Message, sessionID string, traceID string, sink streaming.Sink) (string, string, error) {
	return r.RunTurnStreamInputWithOverrides(ctx, input, sessionID, traceID, sink, nil)
}

func (r *SessionAgentRunner) RunTurnStreamInputWithOverrides(
	ctx context.Context,
	input llm.Message,
	sessionID string,
	traceID string,
	sink streaming.Sink,
	runtimeOverrides *TaskRuntimeOverrides,
) (string, string, error) {
	turn, err := r.prepareTurnWithRuntimeOverrides(ctx, input, sessionID, traceID, runtimeOverrides)
	if err != nil {
		var setupErr *sessionTurnSetupError
		if errors.As(err, &setupErr) {
			if emitErr := internaltrace.EmitStreamErrorEvent(ctx, internaltrace.EnsureEventSink(sink), traceID, 0, "", setupErr.SessionID, setupErr.StatusCode, setupErr); emitErr != nil {
				return "", "", emitErr
			}
		}
		return "", "", err
	}
	defer turn.Close()
	return turnstate.RunPreparedTurnStream(ctx, turn, input, sink, parseSessionEndForStream)
}
