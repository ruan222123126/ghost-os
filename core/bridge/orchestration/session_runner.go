package orchestration

import (
	"context"
	"errors"

	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/llm"
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

type StructuredSessionTurnRunner interface {
	RunTurnInput(ctx context.Context, input llm.Message, sessionID string, traceID string) (string, string, error)
	RunTurnStreamInput(ctx context.Context, input llm.Message, sessionID string, traceID string, sink streaming.Sink) (string, string, error)
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

func (r *SessionAgentRunner) prepareTurn(ctx context.Context, userInput llm.Message, sessionID string, traceID string) (*sessionTurnState, error) {
	return r.prepareTurnWithRuntimeOverrides(ctx, userInput, sessionID, traceID, nil)
}

func (r *SessionAgentRunner) prepareTurnWithRuntimeOverrides(
	ctx context.Context,
	userInput llm.Message,
	sessionID string,
	traceID string,
	runtimeOverrides *TaskRuntimeOverrides,
) (*sessionTurnState, error) {
	return r.turnPreparer().prepareWithRuntimeOverrides(ctx, userInput, sessionID, traceID, runtimeOverrides)
}

func (r *SessionAgentRunner) turnPreparer() *sessionTurnPreparer {
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
	defer turn.close()

	response, runErr := turn.agent.RunMessageWithTraceID(turn.execCtx, input, turn.traceID)
	return turn.complete(response, runErr, nil)
}

func (r *SessionAgentRunner) RunTurnStream(ctx context.Context, userMessage string, sessionID string, traceID string, sink streaming.Sink) (string, string, error) {
	return r.RunTurnStreamInput(ctx, llm.Message{
		Role: llm.RoleUser,
		Text: userMessage,
	}, sessionID, traceID, sink)
}

func (r *SessionAgentRunner) RunTurnStreamInput(ctx context.Context, input llm.Message, sessionID string, traceID string, sink streaming.Sink) (string, string, error) {
	streamSink := newStreamTerminalBuffer(ensureEventSink(sink))
	turn, err := r.prepareTurn(ctx, input, sessionID, traceID)
	if err != nil {
		var setupErr *sessionTurnSetupError
		if errors.As(err, &setupErr) {
			if emitErr := emitStreamErrorEvent(ctx, streamSink, traceID, 0, "", setupErr.sessionID, setupErr.statusCode, setupErr); emitErr != nil {
				return "", "", emitErr
			}
		}
		return "", "", err
	}
	defer turn.close()

	turn.agent.SetStreamLifecyclePayloadBuilder(newSessionStreamLifecyclePayloadBuilder(turn))
	runSink := newSessionDraftCheckpointSink(streamSink, turn.sessionStore, turn.sess)

	response, runErr := turn.agent.RunMessageStreamWithTraceID(turn.execCtx, input, turn.traceID, runSink)
	response, persistedSessionID, err := turn.complete(response, runErr, func(err error, awaitingHuman bool) error {
		turnNumber := turn.agent.LastTurn()
		stepID, stepErr := streaming.AssistantStepID(turnNumber)
		if stepErr != nil {
			return stepErr
		}
		if awaitingHuman {
			stepID = ""
		}
		return emitStreamErrorEvent(ctx, streamSink, turn.traceID, turnNumber, stepID, turn.currentSessionID(), 0, err)
	})
	if err != nil {
		streamSink.Discard()
		return "", persistedSessionID, err
	}
	if emitErr := streamSink.Flush(ctx); emitErr != nil {
		return "", "", emitErr
	}
	return response, persistedSessionID, nil
}
