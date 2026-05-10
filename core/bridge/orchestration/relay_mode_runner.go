package orchestration

import (
	"context"
	"errors"
	"strings"

	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/llm"
	"ghost-os/bridge/session"
	"ghost-os/bridge/tools"
)

const (
	relayModeStatusCompleted  = "completed"
	relayModeStatusIncomplete = "incomplete"
	relayModeStatusCancelled  = "cancelled"
	relayModeStatusError      = "error"
	relayModeStopCompleted    = "relay_complete"
	relayModeStopMaxRounds    = "max_rounds"
	relayModeStopCancelled    = "cancelled"
	relayModeStopError        = "error"
)

type relayModeRunner struct {
	runtimeFactory AgentRuntimeFactory
	configStore    bridgeconfig.Store
	sessionStore   *session.Store
	runRegistry    *RunRegistry
}

type relayModeResult struct {
	Message        string
	StoppedBy      string
	FinalChangeLog string
	Records        []session.RelayRecord
	SessionID      string
}

func newRelayModeRunner(service *bridgeService) relayModeRunner {
	if service == nil {
		return relayModeRunner{}
	}
	factory := service.runtimeFactory
	if factory == nil {
		factory = newAgentRuntimeFactory()
	}
	return relayModeRunner{
		runtimeFactory: factory,
		configStore:    service.configStore,
		sessionStore:   service.sessionStore,
		runRegistry:    service.runRegistry,
	}
}

func (r relayModeRunner) ExecuteTask(
	ctx context.Context,
	task ScheduledTask,
	traceID string,
) (relayModeResult, error) {
	if task.Relay == nil {
		return relayModeResult{}, errors.New("relay config is required")
	}
	deps, err := r.buildDependencies(task.RuntimeOverrides)
	if err != nil {
		return relayModeResult{}, err
	}
	defer deps.Close()
	relay := *task.Relay
	sess, err := loadOrCreateRelaySession(r.sessionStore, task.SessionID, deps.systemPrompt)
	if err != nil {
		return relayModeResult{}, err
	}
	execCtx, cleanup, err := r.register(ctx, sess.ID, traceID)
	if err != nil {
		return relayModeResult{}, err
	}
	defer cleanup()
	return r.run(tools.WithSessionCheckpoint(tools.WithSession(execCtx, sess), r.sessionStore), relayRunDeps{
		deps:    deps,
		relay:   relay,
		task:    task,
		session: sess,
		traceID: traceID,
	})
}

type relayRunDeps struct {
	deps    agentRuntimeDependencies
	relay   TaskRelayConfig
	task    ScheduledTask
	session *session.Session
	traceID string
}

func (r relayModeRunner) buildDependencies(runtimeOverrides *TaskRuntimeOverrides) (agentRuntimeDependencies, error) {
	preparer := newSessionTurnPreparer(r.runtimeFactory, r.configStore, r.sessionStore, r.runRegistry, nil)
	deps, _, _, err := preparer.buildPrepareDependencies(runtimeOverrides)
	return deps, err
}

func loadOrCreateRelaySession(store *session.Store, sessionID string, systemPrompt string) (*session.Session, error) {
	if store == nil {
		return nil, errors.New("session store is not configured")
	}
	if strings.TrimSpace(sessionID) == "" {
		return session.NewSession(systemPrompt), nil
	}
	return store.Load(strings.TrimSpace(sessionID))
}

func (r relayModeRunner) register(ctx context.Context, sessionID string, traceID string) (context.Context, func(), error) {
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

func (r relayModeRunner) run(ctx context.Context, req relayRunDeps) (relayModeResult, error) {
	req.session.AddMessage(llm.Message{Role: llm.RoleUser, Text: strings.TrimSpace(req.task.Message)})
	req.session.StartRelayRuntime(req.task.Message, req.relay.StopPolicy, req.relay.MaxRounds, req.relay.ExecutionTimeoutMS)
	if err := r.sessionStore.Save(req.session); err != nil {
		return relayModeResult{}, err
	}
	result, runErr := r.runRounds(ctx, req)
	if runErr != nil {
		if err := r.finishErroredRun(req.session, runErr); err != nil {
			return relayModeResult{}, err
		}
		return relayModeResult{}, runErr
	}
	req.session.AddMessage(llm.Message{Role: llm.RoleAssistant, Text: result.Message})
	req.session.FinishRelayRuntime(relayResultStatus(result.StoppedBy), result.StoppedBy, result.Message, result.FinalChangeLog)
	if err := r.sessionStore.Save(req.session); err != nil {
		return relayModeResult{}, err
	}
	result.SessionID = req.session.ID
	return result, nil
}
