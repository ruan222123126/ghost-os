package orchestration

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"ghost-os/bridge/agent"
	"ghost-os/bridge/llm"
	bridgeruntime "ghost-os/bridge/runtime"
	"ghost-os/bridge/session"
	"ghost-os/bridge/streaming"
	bridgeTasks "ghost-os/bridge/tasks"
	"ghost-os/bridge/tools"
)

const relayRoundRepairTraceSuffix = "-repair"

func (r relayModeRunner) runRounds(ctx context.Context, req relayRunDeps) (relayModeResult, error) {
	baseCatalog := tools.NewPromptOverrideCatalog(req.deps.registry, req.deps.cfg.ToolSelector.PromptOverrides)
	catalog := newRelayModeCatalog(
		bridgeruntime.NewToolSelectionPolicy(req.deps.cfg).ResidentCatalog(baseCatalog),
		req.relay.StopPolicy == taskRelayStopPolicyAIDecides,
	)
	systemPrompt, err := buildRelaySystemPrompt(req.deps, catalog, req.relay)
	if err != nil {
		return relayModeResult{}, err
	}
	for round := 1; ; round++ {
		if limitResult, done, err := r.checkRelayLimit(ctx, req, round); done {
			return limitResult, err
		}
		result, done, err := r.runRound(ctx, req, round, systemPrompt, catalog)
		if done || err != nil {
			return result, err
		}
	}
}

func buildRelaySystemPrompt(deps agentRuntimeDependencies, catalog tools.ToolCatalog, relay TaskRelayConfig) (string, error) {
	if deps.systemPromptOverride {
		return buildRelayModeSystemPrompt(deps.systemPrompt, relay), nil
	}
	basePrompt, err := bridgeruntime.BuildSystemPromptForCatalog(deps.cfg, catalog)
	if err != nil {
		return "", err
	}
	return buildRelayModeSystemPrompt(basePrompt, relay), nil
}

func (r relayModeRunner) checkRelayLimit(
	ctx context.Context,
	req relayRunDeps,
	round int,
) (relayModeResult, bool, error) {
	if err := ctx.Err(); err != nil {
		return relayModeResult{}, true, err
	}
	if req.relay.MaxRounds <= 0 {
		return relayModeResult{}, true, fmt.Errorf("relay max_rounds must be > 0")
	}
	if round <= req.relay.MaxRounds {
		return relayModeResult{}, false, nil
	}
	records := cloneRelayRecords(req.session.RelayRuntime)
	return relayModeResult{
		Message:   buildRelayMaxRoundsMessage(req.task.Message, records, req.relay.MaxRounds),
		StoppedBy: relayModeStopMaxRounds,
		Records:   records,
	}, true, nil
}

func (r relayModeRunner) runRound(
	ctx context.Context,
	req relayRunDeps,
	round int,
	systemPrompt string,
	catalog tools.ToolCatalog,
) (relayModeResult, bool, error) {
	handle, err := r.startRoundCard(ctx, req, round)
	if err != nil {
		return relayModeResult{}, false, err
	}
	turnAgent := newRelayTurnAgent(req.deps, catalog, systemPrompt)
	turnAgent.SetStreamLifecyclePayloadBuilder(newSessionStreamLifecyclePayloadBuilderForSessionID(req.session.ID))
	roundTraceID := fmt.Sprintf("%s-relay-%d", strings.TrimSpace(req.traceID), round)
	userPrompt := buildRelayModeUserPrompt(req.task.Message, cloneRelayRecords(req.session.RelayRuntime), round, req.relay)
	runSink := newSessionDraftCheckpointSink(newTaskRunCardStreamSink(handle), r.sessionStore, req.session)
	handoffErr, handoffTraceID, err := runRelayRoundWithRepair(
		ctx,
		req.deps,
		systemPrompt,
		turnAgent,
		req.session.ID,
		userPrompt,
		round,
		roundTraceID,
		req.relay,
		runSink,
	)
	if err != nil {
		if handle != nil {
			_ = handle.Finish(ctx, taskRunCardFinishInput{
				status:          relayRoundStatusFromError(ctx, err),
				errorText:       err.Error(),
				sourceSessionID: req.session.ID,
				finishedAt:      time.Now().UTC(),
			})
		}
		return relayModeResult{}, false, err
	}
	record := relayRecordFromHandoff(round, handoffTraceID, handoffErr)
	result, done, recordErr := r.recordRound(req, record, handoffErr)
	summary := relayRoundSummary(record)
	status := taskRunStatusSuccess
	errorText := ""
	if recordErr != nil {
		status = relayRoundStatusFromError(ctx, recordErr)
		errorText = recordErr.Error()
	}
	if handle != nil {
		if finishErr := handle.Finish(ctx, taskRunCardFinishInput{
			status:          status,
			preview:         summary,
			errorText:       errorText,
			finalText:       summary,
			sourceSessionID: req.session.ID,
			finishedAt:      time.Now().UTC(),
		}); finishErr != nil {
			return relayModeResult{}, false, finishErr
		}
	}
	return result, done, recordErr
}

func newRelayTurnAgent(deps agentRuntimeDependencies, catalog tools.ToolCatalog, systemPrompt string) *agent.Agent {
	turnAgent := agent.NewAgentWithHistory(deps.client, catalog, agent.NewHistory(systemPrompt), deps.cfg.MaxTurns)
	turnAgent.SetResponseOptions(llm.CloneResponseOptions(deps.cfg.ResponseOptions))
	turnAgent.SetCompletionRetryPolicy(agent.NewCompletionRetryPolicy(
		deps.cfg.LLMCompletionRetryCount,
		time.Duration(deps.cfg.LLMCompletionRetryIntervalMS)*time.Millisecond,
	))
	return turnAgent
}

func runRelayRoundWithRepair(
	ctx context.Context,
	deps agentRuntimeDependencies,
	systemPrompt string,
	turnAgent *agent.Agent,
	sessionID string,
	userPrompt string,
	round int,
	roundTraceID string,
	relay TaskRelayConfig,
	sink streaming.Sink,
) (*agent.ErrIterationHandoff, string, error) {
	output, runErr := turnAgent.RunMessageStreamWithTraceID(ctx, llm.Message{
		Role: llm.RoleUser,
		Text: userPrompt,
	}, roundTraceID, sink)
	handoffErr, err := relayHandoffFromRunErr(runErr)
	if err != nil {
		return nil, "", err
	}
	if handoffErr != nil {
		return handoffErr, roundTraceID, nil
	}
	repairTraceID := roundTraceID + relayRoundRepairTraceSuffix
	repairPrompt := buildRelayModeRepairPrompt(output, relay)
	repairAgent := newRelayTurnAgent(deps, newRelayModeCatalog(nil, relay.StopPolicy == taskRelayStopPolicyAIDecides), systemPrompt)
	repairAgent.SetStreamLifecyclePayloadBuilder(newSessionStreamLifecyclePayloadBuilderForSessionID(sessionID))
	repairAgent.SetToolChoice("required")
	_, repairErr := repairAgent.RunMessageStreamWithTraceID(ctx, llm.Message{
		Role: llm.RoleUser,
		Text: repairPrompt,
	}, repairTraceID, sink)
	handoffErr, err = relayHandoffFromRunErr(repairErr)
	if err != nil {
		return nil, "", err
	}
	if handoffErr == nil {
		return nil, "", fmt.Errorf("relay round %d ended without relay handoff tool", round)
	}
	return handoffErr, repairTraceID, nil
}

func relayHandoffFromRunErr(runErr error) (*agent.ErrIterationHandoff, error) {
	if runErr == nil {
		return nil, nil
	}
	var handoffErr *agent.ErrIterationHandoff
	if errors.As(runErr, &handoffErr) {
		return handoffErr, nil
	}
	return nil, runErr
}

func (r relayModeRunner) recordRound(
	req relayRunDeps,
	record session.RelayRecord,
	handoffErr *agent.ErrIterationHandoff,
) (relayModeResult, bool, error) {
	req.session.AppendRelayRecord(record)
	if err := r.sessionStore.Save(req.session); err != nil {
		return relayModeResult{}, false, err
	}
	if !handoffErr.Completed {
		return relayModeResult{}, false, nil
	}
	return relayModeResult{
		Message:        handoffErr.FinalMessage,
		StoppedBy:      relayModeStopCompleted,
		FinalChangeLog: handoffErr.FinalChangeLog,
		Records:        cloneRelayRecords(req.session.RelayRuntime),
	}, true, nil
}

func (r relayModeRunner) startRoundCard(
	ctx context.Context,
	req relayRunDeps,
	round int,
) (*taskRunCardHandle, error) {
	recorder := taskRunCardRecorderFromContext(ctx)
	if recorder == nil {
		return nil, nil
	}
	return recorder.StartCard(ctx, taskRunCardStartInput{
		kind:            bridgeTasks.RunCardKindRelayRound,
		title:           taskRunCardTaskTitle(req.task),
		nodeID:          strings.TrimSpace(req.task.ID),
		nodeType:        taskKindAgentMessage,
		round:           round,
		sourceSessionID: strings.TrimSpace(req.session.ID),
		startedAt:       time.Now().UTC(),
	})
}

func relayRoundStatusFromError(ctx context.Context, err error) string {
	switch {
	case errors.Is(ctx.Err(), context.Canceled), errors.Is(err, context.Canceled):
		return taskRunStatusCancelled
	default:
		return taskRunStatusError
	}
}

func relayRoundSummary(record session.RelayRecord) string {
	lines := make([]string, 0, 5)
	if record.Did != "" {
		lines = append(lines, "did: "+record.Did)
	}
	if record.Remaining != "" {
		lines = append(lines, "remaining: "+record.Remaining)
	}
	if record.NextStep != "" {
		lines = append(lines, "next_step: "+record.NextStep)
	}
	if len(record.FailedAttempts) > 0 {
		lines = append(lines, "failed_attempts: "+strings.Join(record.FailedAttempts, " | "))
	}
	if record.FinalChangeLog != "" {
		lines = append(lines, "final_change_log: "+record.FinalChangeLog)
	}
	if len(lines) == 0 {
		return fmt.Sprintf("relay round %d recorded", record.Round)
	}
	return strings.Join(lines, "\n")
}

func relayRecordFromHandoff(round int, traceID string, handoffErr *agent.ErrIterationHandoff) session.RelayRecord {
	return session.RelayRecord{
		Round:          round,
		Did:            handoffErr.Did,
		Remaining:      handoffErr.Remaining,
		FailedAttempts: append([]string(nil), handoffErr.FailedAttempts...),
		NextStep:       handoffErr.NextStep,
		Completed:      handoffErr.Completed,
		TraceID:        traceID,
		RecordedAt:     time.Now().UTC(),
		FinalChangeLog: handoffErr.FinalChangeLog,
	}
}
