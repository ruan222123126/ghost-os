package orchestration

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"ghost-os/bridge/agent"
	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/llm"
	bridgeruntime "ghost-os/bridge/runtime"
	"ghost-os/bridge/session"
	"ghost-os/bridge/tools"
)

// sessionTurnPreparer 只负责单轮运行前的装配，避免 runner 继续吸收 selector / memory / inflight 细节。
type sessionTurnPreparer struct {
	runtimeFactory  AgentRuntimeFactory
	configStore     bridgeconfig.Store
	sessionStore    *session.Store
	runRegistry     *RunRegistry
	selectorFactory func(bridgeconfig.Config, tools.ToolCatalog) bridgeruntime.SelectorEngine
}

type turnPreparationInput struct {
	startedAt      time.Time
	rawUserMessage string
	userMessage    string
	traceID        string
}

func newSessionTurnPreparer(
	runtimeFactory AgentRuntimeFactory,
	configStore bridgeconfig.Store,
	sessionStore *session.Store,
	runRegistry *RunRegistry,
	selectorFactory func(bridgeconfig.Config, tools.ToolCatalog) bridgeruntime.SelectorEngine,
) *sessionTurnPreparer {
	if runtimeFactory == nil {
		runtimeFactory = newAgentRuntimeFactory()
	}
	return &sessionTurnPreparer{
		runtimeFactory:  runtimeFactory,
		configStore:     configStore,
		sessionStore:    sessionStore,
		runRegistry:     runRegistry,
		selectorFactory: selectorFactory,
	}
}

func newTurnPreparationInput(userInput llm.Message, traceID string) turnPreparationInput {
	message := strings.TrimSpace(userInput.Text)
	return turnPreparationInput{
		startedAt:      time.Now().UTC(),
		rawUserMessage: message,
		userMessage:    message,
		traceID:        strings.TrimSpace(traceID),
	}
}

func isResumeLikeInput(userInput llm.Message) bool {
	return strings.TrimSpace(userInput.Text) == "" && !hasAgentInputImages(userInput)
}

func (p *sessionTurnPreparer) prepare(ctx context.Context, userInput llm.Message, sessionID string, traceID string) (*sessionTurnState, error) {
	return p.prepareWithRuntimeOverrides(ctx, userInput, sessionID, traceID, nil)
}

func (p *sessionTurnPreparer) prepareWithRuntimeOverrides(
	ctx context.Context,
	userInput llm.Message,
	sessionID string,
	traceID string,
	runtimeOverrides *TaskRuntimeOverrides,
) (*sessionTurnState, error) {
	if p == nil {
		p = newSessionTurnPreparer(nil, nil, nil, nil, nil)
	}
	input := newTurnPreparationInput(userInput, traceID)
	deps, historyBuilder, persistence, err := p.buildPrepareDependencies(runtimeOverrides)
	if err != nil {
		return nil, err
	}
	state, err := p.prepareSessionTurnState(ctx, deps, historyBuilder, persistence, sessionID, input)
	if err != nil {
		deps.Close()
		return nil, err
	}
	return state, nil
}

func (p *sessionTurnPreparer) prepareSessionTurnState(
	ctx context.Context,
	deps agentRuntimeDependencies,
	historyBuilder *SessionHistoryBuilder,
	persistence *SessionTurnCommitter,
	sessionID string,
	input turnPreparationInput,
) (*sessionTurnState, error) {
	sess, created, err := historyBuilder.LoadOrCreateSession(sessionID)
	if err != nil {
		return nil, &sessionTurnSetupError{sessionID: strings.TrimSpace(sessionID), err: err}
	}
	if created {
		titleTask := p.prepareCreatedSessionTitleTask(sess, deps, input)
		if err := p.persistCreatedSession(sess); err != nil {
			return nil, &sessionTurnSetupError{sessionID: strings.TrimSpace(sess.ID), err: err}
		}
		startSessionTitleTask(titleTask)
	}
	if isResumeLikeInput(llm.Message{Role: llm.RoleUser, Text: input.rawUserMessage}) && !hasAnsweredHumanResponse(sess) {
		return nil, &sessionTurnSetupError{
			sessionID:  strings.TrimSpace(sess.ID),
			statusCode: http.StatusBadRequest,
			err:        errors.New("resume requires pending human answers"),
		}
	}
	execCtx, cleanup, err := p.prepareExecutionContext(ctx, sess, deps.registry, input.traceID)
	if err != nil {
		return nil, &sessionTurnSetupError{sessionID: strings.TrimSpace(sess.ID), statusCode: http.StatusConflict, err: err}
	}
	history, preTurnMessages, catalog, _, err := p.prepareHistoryAndEnvironment(
		execCtx,
		deps,
		historyBuilder,
		sess,
		input.rawUserMessage,
		input.traceID,
	)
	if err != nil {
		cleanup()
		return nil, &sessionTurnSetupError{sessionID: strings.TrimSpace(sess.ID), err: err}
	}
	runAgent := p.buildTurnAgent(deps, catalog, sess, history)
	return &sessionTurnState{
		sessionStore:    p.sessionStore,
		deps:            deps,
		persistence:     persistence,
		sess:            sess,
		agent:           runAgent,
		execCtx:         execCtx,
		traceID:         input.traceID,
		userMessage:     input.userMessage,
		preTurnMessages: preTurnMessages,
		turnStartedAt:   input.startedAt,
		cleanup:         cleanup,
	}, nil
}

func (p *sessionTurnPreparer) prepareExecutionContext(
	ctx context.Context,
	sess *session.Session,
	registry *tools.Registry,
	traceID string,
) (context.Context, func(), error) {
	execCtx, cleanup, err := p.registerRun(ctx, sess.ID, traceID)
	if err != nil {
		return ctx, func() {}, err
	}
	execCtx = tools.WithSession(execCtx, sess)
	execCtx = tools.WithSessionCheckpoint(execCtx, p.sessionStore)
	if err := autoResumePendingHumanTools(execCtx, registry, sess, traceID); err != nil {
		cleanup()
		return ctx, func() {}, err
	}
	return execCtx, cleanup, nil
}

func (p *sessionTurnPreparer) buildTurnAgent(
	deps agentRuntimeDependencies,
	catalog tools.ToolCatalog,
	sess *session.Session,
	history *agent.History,
) *agent.Agent {
	runAgent := agent.NewAgentWithHistory(deps.client, catalog, history, deps.cfg.MaxTurns)
	runAgent.SetResponseOptions(llm.CloneResponseOptions(deps.cfg.ResponseOptions))
	runAgent.SetCompletionRetryPolicy(agent.NewCompletionRetryPolicy(
		deps.cfg.LLMCompletionRetryCount,
		time.Duration(deps.cfg.LLMCompletionRetryIntervalMS)*time.Millisecond,
	))
	p.attachCompletionPromptRefresh(runAgent, deps, sess, catalog)
	return runAgent
}

func (p *sessionTurnPreparer) prepareHistoryAndEnvironment(
	ctx context.Context,
	deps agentRuntimeDependencies,
	historyBuilder *SessionHistoryBuilder,
	sess *session.Session,
	rawUserMessage string,
	traceID string,
) (*agent.History, []llm.Message, tools.ToolCatalog, string, error) {
	preTurnMessages := llm.CloneMessages(sess.Messages)
	askHumanContinuation := hasAnsweredHumanResponse(sess)
	sess.AdvanceToolTurn(deps.cfg.ToolSearch.IdleTurns)
	pruneInvisibleSessionSkills(deps.cfg, sess)
	historyBuilder.SetTraceID(traceID)
	history := historyBuilder.BuildHistory(sess)

	catalog, systemPrompt, err := p.selectToolsForTurn(ctx, deps, sess, history, rawUserMessage, askHumanContinuation, traceID)
	if err != nil {
		return nil, nil, nil, "", err
	}
	finalPrompt, err := p.buildTurnSystemPrompt(
		deps,
		sess,
		catalog,
		systemPrompt,
	)
	if err != nil {
		return nil, nil, nil, "", err
	}
	if finalPrompt != "" {
		history.UpdateSystemPrompt(finalPrompt)
	}
	return history, preTurnMessages, catalog, systemPrompt, nil
}

func hasAnsweredHumanResponse(sess *session.Session) bool {
	return sess != nil && len(sess.HumanAnswers) > 0
}

func (p *sessionTurnPreparer) registerRun(ctx context.Context, sessionID string, traceID string) (context.Context, func(), error) {
	if p == nil || p.runRegistry == nil {
		return ctx, func() {}, nil
	}
	trimmedSessionID := strings.TrimSpace(sessionID)
	if trimmedSessionID == "" {
		return ctx, func() {}, nil
	}
	execCtx, cancel := context.WithCancel(ctx)
	if err := p.runRegistry.Register(trimmedSessionID, strings.TrimSpace(traceID), cancel); err != nil {
		cancel()
		return ctx, func() {}, err
	}
	return execCtx, func() {
		p.runRegistry.Unregister(trimmedSessionID)
		cancel()
	}, nil
}

func (p *sessionTurnPreparer) persistCreatedSession(sess *session.Session) error {
	if p == nil || p.sessionStore == nil || sess == nil {
		return nil
	}
	return p.sessionStore.Save(sess)
}
