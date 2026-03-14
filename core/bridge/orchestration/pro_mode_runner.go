package orchestration

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"ghost-os/bridge/agent"
	"ghost-os/bridge/llm"
	"ghost-os/bridge/session"
	"ghost-os/bridge/tools"
)

type proModeRunner struct {
	runtimeFactory AgentRuntimeFactory
	configStore    *ConfigStore
	sessionStore   *session.Store
	runRegistry    *RunRegistry
}

func newProModeRunner(service *bridgeService) proModeRunner {
	if service == nil {
		return proModeRunner{}
	}
	factory := service.runtimeFactory
	if factory == nil {
		factory = newAgentRuntimeFactoryWithTaskManager(service.taskToolManager())
	}
	return proModeRunner{
		runtimeFactory: factory,
		configStore:    service.configStore,
		sessionStore:   service.sessionStore,
		runRegistry:    service.runRegistry,
	}
}

func (r proModeRunner) Execute(ctx context.Context, prepared preparedAgentTurnRequest, traceID string) (agentResponse, error) {
	if r.runtimeFactory == nil {
		return agentResponse{}, errors.New("agent runtime factory is not configured")
	}
	deps, err := r.runtimeFactory.Build(r.configStore)
	if err != nil {
		return agentResponse{}, err
	}
	defer deps.Close()

	request, _, err := parseProModeRequest(prepared.message, deps.cfg.ProMaxIterations)
	if err != nil {
		return agentResponse{}, err
	}

	sess, err := loadOrCreateIterationSession(r.sessionStore, prepared.sessionID, deps.systemPrompt)
	if err != nil {
		return agentResponse{}, err
	}
	execCtx, cleanup, err := r.register(ctx, sess.ID, traceID)
	if err != nil {
		return agentResponse{}, err
	}
	defer cleanup()

	return r.run(tools.WithSession(execCtx, sess), deps, sess, request, prepared.message, traceID)
}

func loadOrCreateIterationSession(store *session.Store, sessionID string, systemPrompt string) (*session.Session, error) {
	if store == nil {
		return nil, errors.New("session store is not configured")
	}
	if strings.TrimSpace(sessionID) == "" {
		return session.NewSession(systemPrompt), nil
	}
	return store.Load(strings.TrimSpace(sessionID))
}

func (r proModeRunner) register(ctx context.Context, sessionID string, traceID string) (context.Context, func(), error) {
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

func (r proModeRunner) run(
	ctx context.Context,
	deps agentRuntimeDependencies,
	sess *session.Session,
	request proModeRequest,
	originalMessage string,
	traceID string,
) (agentResponse, error) {
	sess.AddMessage(llm.Message{Role: llm.RoleUser, Text: strings.TrimSpace(originalMessage)})
	sess.StartIterationRuntime(request.Mode, request.OriginalTask, request.MaxIterations, request.Unlimited)
	if err := r.sessionStore.Save(sess); err != nil {
		return agentResponse{}, err
	}

	result, runErr := r.runIterations(ctx, deps, sess, request, traceID)
	if runErr != nil {
		if err := r.finishErroredRun(sess, runErr); err != nil {
			return agentResponse{}, err
		}
		return agentResponse{}, runErr
	}

	sess.AddMessage(llm.Message{Role: llm.RoleAssistant, Text: result.Message})
	sess.FinishIterationRuntime(proModeResultStatus(result.StoppedBy), result.StoppedBy, result.Message, result.FinalChangeLog)
	if err := r.sessionStore.Save(sess); err != nil {
		return agentResponse{}, err
	}

	return newAgentResponsePayload(result.Message, sess.ID, nil, agentResponseMeta{
		Mode:             request.Mode,
		IterationCount:   len(result.Records),
		StoppedBy:        result.StoppedBy,
		FinalChangeLog:   result.FinalChangeLog,
		IterationSummary: buildAgentIterationRecords(result.Records),
	})
}

func (r proModeRunner) finishErroredRun(sess *session.Session, runErr error) error {
	status := proModeStatusError
	stoppedBy := proModeStopError
	if errors.Is(runErr, context.Canceled) {
		status = proModeStatusCancelled
		stoppedBy = proModeStopCancelled
	}
	sess.FinishIterationRuntime(status, stoppedBy, "", "")
	return r.sessionStore.Save(sess)
}

func (r proModeRunner) runIterations(
	ctx context.Context,
	deps agentRuntimeDependencies,
	sess *session.Session,
	request proModeRequest,
	traceID string,
) (proModeResult, error) {
	catalog := newProModeCatalog(newToolSelectionPolicy(deps.cfg.ToolSelector).scopeCatalog(deps.registry), request.Mode == proModePro)
	systemPrompt, err := buildProModeSystemPrompt(deps.cfg, catalog, request)
	if err != nil {
		return proModeResult{}, err
	}

	for iteration := 1; ; iteration++ {
		if limitResult, done, err := r.checkLimit(ctx, sess, request, iteration); done {
			return limitResult, err
		}
		result, done, err := r.runIteration(ctx, deps, sess, request, traceID, iteration, systemPrompt, catalog)
		if done || err != nil {
			return result, err
		}
	}
}

func (r proModeRunner) checkLimit(
	ctx context.Context,
	sess *session.Session,
	request proModeRequest,
	iteration int,
) (proModeResult, bool, error) {
	if err := ctx.Err(); err != nil {
		return proModeResult{}, true, err
	}
	if request.Unlimited || request.MaxIterations <= 0 || iteration <= request.MaxIterations {
		return proModeResult{}, false, nil
	}
	records := cloneIterationRecords(sess.IterationRuntime)
	return proModeResult{
		Message:   buildProModeMaxLimitMessage(request, records),
		StoppedBy: proModeStopMaxLimit,
		Records:   records,
	}, true, nil
}

func (r proModeRunner) runIteration(
	ctx context.Context,
	deps agentRuntimeDependencies,
	sess *session.Session,
	request proModeRequest,
	traceID string,
	iteration int,
	systemPrompt string,
	catalog tools.ToolCatalog,
) (proModeResult, bool, error) {
	history := agent.NewHistory(systemPrompt)
	turnAgent := agent.NewAgentWithHistory(deps.client, catalog, history, deps.cfg.MaxTurns)
	iterationTraceID := fmt.Sprintf("%s-pro-%d", strings.TrimSpace(traceID), iteration)
	userPrompt := buildProModeUserPrompt(request, cloneIterationRecords(sess.IterationRuntime), iteration)
	_, runErr := turnAgent.RunWithTraceID(ctx, userPrompt, iterationTraceID)
	if runErr == nil {
		return proModeResult{}, false, fmt.Errorf("pro iteration %d ended without pro_update_record or pro_complete", iteration)
	}

	var handoffErr *agent.ErrIterationHandoff
	if !errors.As(runErr, &handoffErr) {
		return proModeResult{}, false, runErr
	}

	record := session.IterationRecord{
		Iteration:      iteration,
		Did:            handoffErr.Did,
		Remaining:      handoffErr.Remaining,
		Completed:      handoffErr.Completed,
		TraceID:        iterationTraceID,
		RecordedAt:     time.Now().UTC(),
		FinalChangeLog: handoffErr.FinalChangeLog,
	}
	sess.AppendIterationRecord(record)
	if err := r.sessionStore.Save(sess); err != nil {
		return proModeResult{}, false, err
	}
	if !handoffErr.Completed {
		return proModeResult{}, false, nil
	}
	return proModeResult{
		Message:        handoffErr.FinalMessage,
		StoppedBy:      proModeStopCompleted,
		FinalChangeLog: handoffErr.FinalChangeLog,
		Records:        cloneIterationRecords(sess.IterationRuntime),
	}, true, nil
}
