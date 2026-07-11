package sessionprep

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"ghost-os/bridge/agent"
	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/llm"
	"ghost-os/bridge/orchestration/internal/app/agentturn"
	"ghost-os/bridge/orchestration/internal/app/agentturn/toolselect"
	"ghost-os/bridge/orchestration/internal/app/agentturn/turnstate"
	appsessions "ghost-os/bridge/orchestration/internal/app/sessions"
	bridgeruntime "ghost-os/bridge/runtime"
	"ghost-os/bridge/session"
	"ghost-os/bridge/tools"
)

type RuntimeDependencies struct {
	Config               bridgeconfig.Config
	Client               agent.Completer
	Registry             *tools.Registry
	SystemPrompt         string
	SystemPromptOverride bool
	SystemPromptFiles    *bridgeconfig.SystemPromptFiles
	Cleanup              func()
}

func (d RuntimeDependencies) Close() {
	if d.Cleanup != nil {
		d.Cleanup()
	}
}

type Preparer struct {
	SessionStore    *session.Store
	RunRegistry     agentturn.SessionRunRegistry
	SelectorFactory func(bridgeconfig.Config, tools.ToolCatalog) bridgeruntime.SelectorEngine
	AgentBuilder    TurnAgentBuilder
	Logger          agentturn.Logger
}

type TurnAgentBuilder interface {
	BuildTurnAgent(req TurnAgentRequest) (*agent.Agent, error)
}

type TurnAgentRequest struct {
	Config               bridgeconfig.Config
	Client               agent.Completer
	Catalog              tools.ToolCatalog
	Session              *session.Session
	History              *agent.History
	SystemPrompt         string
	SystemPromptOverride bool
	SystemPromptFiles    *bridgeconfig.SystemPromptFiles
}

type Command struct {
	Deps           RuntimeDependencies
	HistoryBuilder *appsessions.HistoryBuilder
	Persistence    turnstate.Persistence
	SessionID      string
	Input          agentturn.TurnPreparationInput
}

func (p Preparer) PrepareState(ctx context.Context, cmd Command) (*turnstate.State, error) {
	sess, created, err := cmd.HistoryBuilder.LoadOrCreateSession(cmd.SessionID)
	if err != nil {
		return nil, &agentturn.SessionSetupError{SessionID: strings.TrimSpace(cmd.SessionID), Err: err}
	}
	if created {
		if err := p.persistCreatedSession(sess, cmd.Deps, cmd.Input); err != nil {
			return nil, err
		}
	}
	if isInvalidResume(cmd.Input, sess) {
		return nil, &agentturn.SessionSetupError{
			SessionID:  strings.TrimSpace(sess.ID),
			StatusCode: http.StatusBadRequest,
			Err:        errors.New("resume requires pending human answers"),
		}
	}
	execCtx, cleanup, err := p.prepareExecutionContext(ctx, sess, cmd.Deps.Registry, cmd.Input.TraceID)
	if err != nil {
		return nil, &agentturn.SessionSetupError{SessionID: strings.TrimSpace(sess.ID), StatusCode: http.StatusConflict, Err: err}
	}
	history, preTurnMessages, catalog, err := p.prepareHistoryAndEnvironment(
		execCtx,
		cmd.Deps,
		cmd.HistoryBuilder,
		sess,
		cmd.Input.RawUserMessage,
		cmd.Input.TraceID,
	)
	if err != nil {
		cleanup()
		return nil, &agentturn.SessionSetupError{SessionID: strings.TrimSpace(sess.ID), Err: err}
	}
	runAgent, err := p.buildTurnAgent(cmd.Deps, catalog, sess, history)
	if err != nil {
		cleanup()
		return nil, &agentturn.SessionSetupError{SessionID: strings.TrimSpace(sess.ID), Err: err}
	}
	return &turnstate.State{
		SessionStore:    p.SessionStore,
		RuntimeCleanup:  cmd.Deps.Close,
		Persistence:     cmd.Persistence,
		Session:         sess,
		Agent:           runAgent,
		ExecCtx:         execCtx,
		TraceID:         cmd.Input.TraceID,
		UserMessage:     cmd.Input.UserMessage,
		PreTurnMessages: preTurnMessages,
		TurnStartedAt:   cmd.Input.StartedAt,
		Cleanup:         cleanup,
	}, nil
}

func (p Preparer) persistCreatedSession(
	sess *session.Session,
	deps RuntimeDependencies,
	input agentturn.TurnPreparationInput,
) error {
	titleTask := p.prepareCreatedSessionTitleTask(sess, deps, input)
	if err := agentturn.PersistCreatedSession(p.SessionStore, sess); err != nil {
		return &agentturn.SessionSetupError{SessionID: strings.TrimSpace(sess.ID), Err: err}
	}
	agentturn.StartSessionTitleTask(titleTask)
	return nil
}

func (p Preparer) prepareCreatedSessionTitleTask(
	sess *session.Session,
	deps RuntimeDependencies,
	input agentturn.TurnPreparationInput,
) *agentturn.SessionTitleTask {
	if sess == nil {
		return nil
	}
	prepared := agentturn.PrepareCreatedSessionTitle(agentturn.SessionTitlePreparationInput{
		Mode:        deps.Config.SessionTitleMode,
		Store:       p.SessionStore,
		Completer:   deps.Client,
		Logger:      p.Logger,
		SessionID:   sess.ID,
		UserMessage: input.UserMessage,
		TraceID:     input.TraceID,
	})
	if prepared.HasImmediateTitle {
		sess.Title = prepared.ImmediateTitle
	}
	return prepared.Task
}

func isInvalidResume(input agentturn.TurnPreparationInput, sess *session.Session) bool {
	userInput := llm.Message{Role: llm.RoleUser, Text: input.RawUserMessage}
	return agentturn.IsResumeLikeInput(userInput) && !agentturn.HasAnsweredHumanResponse(sess)
}

func (p Preparer) prepareExecutionContext(
	ctx context.Context,
	sess *session.Session,
	registry *tools.Registry,
	traceID string,
) (context.Context, func(), error) {
	return p.PrepareExecutionContext(ctx, sess, registry, traceID)
}

func (p Preparer) PrepareExecutionContext(
	ctx context.Context,
	sess *session.Session,
	registry *tools.Registry,
	traceID string,
) (context.Context, func(), error) {
	execCtx, cleanup, err := agentturn.RegisterSessionRun(ctx, p.RunRegistry, sess.ID, traceID)
	if err != nil {
		return ctx, func() {}, err
	}
	execCtx = tools.WithSession(execCtx, sess)
	execCtx = tools.WithSessionCheckpoint(execCtx, p.SessionStore)
	if err := agentturn.AutoResumePendingHumanTools(execCtx, registry, sess, traceID); err != nil {
		cleanup()
		return ctx, func() {}, err
	}
	return execCtx, cleanup, nil
}

func (p Preparer) buildTurnAgent(
	deps RuntimeDependencies,
	catalog tools.ToolCatalog,
	sess *session.Session,
	history *agent.History,
) (*agent.Agent, error) {
	return p.BuildTurnAgent(deps, catalog, sess, history)
}

func (p Preparer) BuildTurnAgent(
	deps RuntimeDependencies,
	catalog tools.ToolCatalog,
	sess *session.Session,
	history *agent.History,
) (*agent.Agent, error) {
	if p.AgentBuilder == nil {
		return nil, errors.New("session turn agent builder is not configured")
	}
	return p.AgentBuilder.BuildTurnAgent(TurnAgentRequest{
		Config:               deps.Config,
		Client:               deps.Client,
		Catalog:              catalog,
		Session:              sess,
		History:              history,
		SystemPrompt:         deps.SystemPrompt,
		SystemPromptOverride: deps.SystemPromptOverride,
		SystemPromptFiles:    deps.SystemPromptFiles,
	})
}

func (p Preparer) prepareHistoryAndEnvironment(
	ctx context.Context,
	deps RuntimeDependencies,
	historyBuilder *appsessions.HistoryBuilder,
	sess *session.Session,
	rawUserMessage string,
	traceID string,
) (*agent.History, []llm.Message, tools.ToolCatalog, error) {
	preTurnMessages := llm.CloneMessages(sess.Messages)
	askHumanContinuation := agentturn.HasAnsweredHumanResponse(sess)
	sess.AdvanceToolTurn(deps.Config.ToolSearch.IdleTurns)
	agentturn.PruneInvisibleSessionSkills(deps.Config, sess)
	historyBuilder.SetTraceID(traceID)
	history := historyBuilder.BuildHistory(sess)

	catalog, systemPrompt, err := toolselect.SelectForTurn(ctx, toolselect.Request{
		Config:               deps.Config,
		Registry:             deps.Registry,
		Session:              sess,
		History:              history,
		UserMessage:          rawUserMessage,
		AskHumanContinuation: askHumanContinuation,
		TraceID:              traceID,
		SelectorFactory:      p.SelectorFactory,
	})
	if err != nil {
		return nil, nil, nil, err
	}
	finalPrompt, err := agentturn.BuildCompletionSystemPrompt(agentturn.CompletionPromptRequest{
		Config:               deps.Config,
		Catalog:              catalog,
		Session:              sess,
		SystemPrompt:         systemPrompt,
		FallbackSystemPrompt: deps.SystemPrompt,
		SystemPromptOverride: deps.SystemPromptOverride,
		SystemPromptFiles:    deps.SystemPromptFiles,
	})
	if err != nil {
		return nil, nil, nil, err
	}
	if finalPrompt != "" {
		history.UpdateSystemPrompt(finalPrompt)
	}
	return history, preTurnMessages, catalog, nil
}
