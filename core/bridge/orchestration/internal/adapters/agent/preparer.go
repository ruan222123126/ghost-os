package agent

import (
	"context"

	bridgeagent "ghost-os/bridge/agent"
	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/llm"
	runtimeadapter "ghost-os/bridge/orchestration/internal/adapters/runtime"
	"ghost-os/bridge/orchestration/internal/app/agentturn"
	"ghost-os/bridge/orchestration/internal/app/agentturn/sessionprep"
	"ghost-os/bridge/orchestration/internal/app/agentturn/turnstate"
	appsessions "ghost-os/bridge/orchestration/internal/app/sessions"
	bridgeruntime "ghost-os/bridge/runtime"
	"ghost-os/bridge/session"
	bridgeTasks "ghost-os/bridge/tasks"
	"ghost-os/bridge/tools"
)

type RuntimeDependencies = runtimeadapter.Dependencies

type RuntimeFactory interface {
	Build(store bridgeconfig.Store) (RuntimeDependencies, error)
}

type SessionTurnPreparerConfig struct {
	RuntimeFactory  RuntimeFactory
	ConfigStore     bridgeconfig.Store
	SessionStore    *session.Store
	RunRegistry     agentturn.SessionRunRegistry
	SelectorFactory func(bridgeconfig.Config, tools.ToolCatalog) bridgeruntime.SelectorEngine
	Logger          agentturn.Logger
}

type SessionTurnPreparer struct {
	config SessionTurnPreparerConfig
}

func NewSessionTurnPreparer(config SessionTurnPreparerConfig) SessionTurnPreparer {
	return SessionTurnPreparer{config: config}
}

func (p SessionTurnPreparer) PrepareWithRuntimeOverrides(
	ctx context.Context,
	userInput llm.Message,
	sessionID string,
	traceID string,
	runtimeOverrides *bridgeTasks.TaskRuntimeOverrides,
) (*turnstate.State, error) {
	input := agentturn.NewTurnPreparationInput(userInput, traceID)
	deps, historyBuilder, persistence, err := p.BuildPrepareDependencies(runtimeOverrides)
	if err != nil {
		return nil, err
	}
	state, err := p.sessionPrep().PrepareState(ctx, sessionprep.Command{
		Deps:           sessionPrepDependencies(deps),
		HistoryBuilder: historyBuilder,
		Persistence:    persistence,
		SessionID:      sessionID,
		Input:          input,
	})
	if err != nil {
		deps.Close()
		return nil, err
	}
	return state, nil
}

func (p SessionTurnPreparer) BuildPrepareDependencies(
	runtimeOverrides *bridgeTasks.TaskRuntimeOverrides,
) (RuntimeDependencies, *appsessions.HistoryBuilder, turnstate.Persistence, error) {
	prepared, err := sessionprep.BuildDependencies(sessionprep.DependencyCommand{
		RuntimeFactory:   sessionPrepRuntimeFactory{inner: p.config.RuntimeFactory},
		ConfigStore:      p.config.ConfigStore,
		SessionStore:     p.config.SessionStore,
		RuntimeOverrides: runtimeOverrides,
	})
	if err != nil {
		return nil, nil, nil, err
	}
	return runtimeDependencies{deps: prepared.Deps}, prepared.HistoryBuilder, prepared.Persistence, nil
}

func (p SessionTurnPreparer) PrepareExecutionContext(
	ctx context.Context,
	sess *session.Session,
	registry *tools.Registry,
	traceID string,
) (context.Context, func(), error) {
	return p.sessionPrep().PrepareExecutionContext(ctx, sess, registry, traceID)
}

func (p SessionTurnPreparer) BuildTurnAgent(
	deps RuntimeDependencies,
	catalog tools.ToolCatalog,
	sess *session.Session,
	history *bridgeagent.History,
) (*bridgeagent.Agent, error) {
	return p.sessionPrep().BuildTurnAgent(sessionPrepDependencies(deps), catalog, sess, history)
}

func (p SessionTurnPreparer) sessionPrep() sessionprep.Preparer {
	preparer := sessionprep.Preparer{
		SessionStore:    p.config.SessionStore,
		SelectorFactory: p.config.SelectorFactory,
		AgentBuilder:    TurnAgentBuilder{},
		Logger:          p.config.Logger,
	}
	if p.config.RunRegistry != nil {
		preparer.RunRegistry = p.config.RunRegistry
	}
	return preparer
}

type sessionPrepRuntimeFactory struct {
	inner RuntimeFactory
}

func (f sessionPrepRuntimeFactory) Build(store bridgeconfig.Store) (sessionprep.RuntimeDependencies, error) {
	if f.inner == nil {
		return sessionprep.RuntimeDependencies{}, sessionprep.ErrRuntimeFactoryRequired
	}
	deps, err := f.inner.Build(store)
	if err != nil {
		return sessionprep.RuntimeDependencies{}, err
	}
	return sessionPrepDependencies(deps), nil
}

func sessionPrepDependencies(deps RuntimeDependencies) sessionprep.RuntimeDependencies {
	if deps == nil {
		return sessionprep.RuntimeDependencies{}
	}
	return sessionprep.RuntimeDependencies{
		Config:               deps.Config(),
		Client:               deps.Client(),
		Registry:             deps.Registry(),
		SystemPrompt:         deps.SystemPrompt(),
		SystemPromptOverride: deps.SystemPromptOverride(),
		SystemPromptFiles:    deps.SystemPromptFiles(),
		Cleanup:              deps.Close,
	}
}

type runtimeDependencies struct {
	deps sessionprep.RuntimeDependencies
}

func (d runtimeDependencies) Config() bridgeconfig.Config {
	return d.deps.Config
}

func (d runtimeDependencies) Client() bridgeagent.Completer {
	return d.deps.Client
}

func (d runtimeDependencies) Registry() *tools.Registry {
	return d.deps.Registry
}

func (d runtimeDependencies) SystemPrompt() string {
	return d.deps.SystemPrompt
}

func (d runtimeDependencies) SystemPromptOverride() bool {
	return d.deps.SystemPromptOverride
}

func (d runtimeDependencies) SystemPromptFiles() *bridgeconfig.SystemPromptFiles {
	return d.deps.SystemPromptFiles
}

func (d runtimeDependencies) Close() {
	d.deps.Close()
}
