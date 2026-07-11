package relay

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"ghost-os/bridge/agent"
	"ghost-os/bridge/llm"
	apptasks "ghost-os/bridge/orchestration/internal/app/tasks"
	"ghost-os/bridge/orchestration/internal/domain/runtimeopts"
	internaltrace "ghost-os/bridge/orchestration/internal/trace"
	bridgeruntime "ghost-os/bridge/runtime"
	"ghost-os/bridge/session"
	"ghost-os/bridge/streaming"
	"ghost-os/bridge/taskdefs"
	"ghost-os/bridge/tools"
)

const (
	relayRoundRepairTraceSuffix        = "-repair"
	relayRoundProtocolErrorTraceSuffix = "-protocol-error"
)

type relayRoundRunOutput struct {
	handoff   *agent.ErrIterationHandoff
	traceID   string
	finalText string
}

func (r Runner) runRounds(ctx context.Context, req RunRequest) (Result, error) {
	baseCatalog := tools.NewPromptOverrideCatalog(req.Deps.Registry, req.Deps.Config.ToolSelector.PromptOverrides)
	catalog := newRelayModeCatalog(
		bridgeruntime.NewToolSelectionPolicy(req.Deps.Config).ResidentCatalog(baseCatalog),
		req.Relay.StopPolicy == taskdefs.RelayStopPolicyAIDecides,
	)
	systemPrompt, err := buildRelaySystemPrompt(req.Deps, catalog, req.Relay)
	if err != nil {
		return Result{}, err
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

func buildRelaySystemPrompt(deps RuntimeDependencies, catalog tools.ToolCatalog, relay taskdefs.TaskRelayConfig) (string, error) {
	if deps.SystemPromptOverride {
		return buildRelayModeSystemPrompt(deps.SystemPrompt, relay), nil
	}
	basePrompt, err := bridgeruntime.BuildSystemPromptForCatalog(deps.Config, catalog)
	if err != nil {
		return "", err
	}
	return buildRelayModeSystemPrompt(basePrompt, relay), nil
}

func (r Runner) checkRelayLimit(
	ctx context.Context,
	req RunRequest,
	round int,
) (Result, bool, error) {
	if err := ctx.Err(); err != nil {
		return Result{}, true, err
	}
	if req.Relay.MaxRounds <= 0 {
		return Result{}, true, fmt.Errorf("relay max_rounds must be > 0")
	}
	if round <= req.Relay.MaxRounds {
		return Result{}, false, nil
	}
	records := cloneRelayRecords(req.Session.RelayRuntime)
	return Result{
		Message:   buildRelayMaxRoundsMessage(req.Task.Message, records, req.Relay.MaxRounds),
		StoppedBy: StopMaxRounds,
		Records:   records,
	}, true, nil
}

func (r Runner) runRound(
	ctx context.Context,
	req RunRequest,
	round int,
	systemPrompt string,
	catalog tools.ToolCatalog,
) (Result, bool, error) {
	handle, err := r.startRoundCard(ctx, req, round)
	if err != nil {
		return Result{}, false, err
	}
	turnAgent, err := newRelayTurnAgent(req.Deps, catalog, systemPrompt)
	if err != nil {
		return Result{}, false, err
	}
	turnAgent.SetStreamLifecyclePayloadBuilder(sessionLifecyclePayloadBuilder(req.Session.ID))
	roundTraceID := fmt.Sprintf("%s-relay-%d", strings.TrimSpace(req.TraceID), round)
	userPrompt := buildRelayModeUserPrompt(req.Task.Message, cloneRelayRecords(req.Session.RelayRuntime), round, req.Relay)
	runSink := internaltrace.NewSessionDraftCheckpointSink(newRuncardStreamSink(handle), r.SessionStore, req.Session)
	roundOutput, err := runRelayRoundWithRepair(
		ctx,
		req.Deps,
		systemPrompt,
		turnAgent,
		req.Session.ID,
		userPrompt,
		round,
		roundTraceID,
		req.Relay,
		runSink,
	)
	if err != nil {
		finishRelayRoundCardError(ctx, handle, req.Session.ID, roundOutput, err)
		return Result{}, false, err
	}
	record := relayRecordFromHandoff(round, roundOutput.traceID, roundOutput.handoff)
	result, done, recordErr := r.recordRound(req, record, roundOutput.handoff)
	summary := relayRoundSummary(record)
	status := taskdefs.RunStatusSuccess
	errorText := ""
	if recordErr != nil {
		status = relayRoundStatusFromError(ctx, recordErr)
		errorText = recordErr.Error()
	}
	if finishErr := finishRelayRoundCard(ctx, handle, req.Session.ID, status, summary, errorText); finishErr != nil {
		return Result{}, false, finishErr
	}
	return result, done, recordErr
}

func newRelayTurnAgent(deps RuntimeDependencies, catalog tools.ToolCatalog, systemPrompt string) (*agent.Agent, error) {
	turnAgent := agent.NewAgentWithHistory(deps.Client, catalog, agent.NewHistory(systemPrompt), deps.Config.MaxTurns)
	turnAgent.SetResponseOptions(llm.CloneResponseOptions(deps.Config.ResponseOptions))
	retry, err := runtimeopts.NormalizeRetryPolicy(
		deps.Config.LLMCompletionRetryCount,
		deps.Config.LLMCompletionRetryIntervalMS,
	)
	if err != nil {
		return nil, err
	}
	turnAgent.SetCompletionRetryPolicy(agent.NewCompletionRetryPolicy(
		retry.Count,
		time.Duration(retry.IntervalMS)*time.Millisecond,
	))
	return turnAgent, nil
}

func runRelayRoundWithRepair(
	ctx context.Context,
	deps RuntimeDependencies,
	systemPrompt string,
	turnAgent *agent.Agent,
	sessionID string,
	userPrompt string,
	round int,
	roundTraceID string,
	relay taskdefs.TaskRelayConfig,
	sink streaming.Sink,
) (relayRoundRunOutput, error) {
	output, runErr := turnAgent.RunMessageStreamWithTraceID(ctx, llm.Message{
		Role: llm.RoleUser,
		Text: userPrompt,
	}, roundTraceID, sink)
	handoffErr, err := relayHandoffFromRunErr(runErr)
	if err != nil {
		return relayRoundRunOutput{finalText: output}, err
	}
	if handoffErr != nil {
		return relayRoundRunOutput{handoff: handoffErr, traceID: roundTraceID}, nil
	}
	repairTraceID := roundTraceID + relayRoundRepairTraceSuffix
	repairPrompt := buildRelayModeRepairPrompt(output, relay)
	repairAgent, err := newRelayTurnAgent(deps, newRelayModeCatalog(nil, relay.StopPolicy == taskdefs.RelayStopPolicyAIDecides), systemPrompt)
	if err != nil {
		return relayRoundRunOutput{finalText: output}, err
	}
	repairAgent.SetStreamLifecyclePayloadBuilder(sessionLifecyclePayloadBuilder(sessionID))
	repairAgent.SetToolChoice("required")
	repairOutput, repairErr := repairAgent.RunMessageStreamWithTraceID(ctx, llm.Message{
		Role: llm.RoleUser,
		Text: repairPrompt,
	}, repairTraceID, sink)
	handoffErr, err = relayHandoffFromRunErr(repairErr)
	if err != nil {
		return relayRoundRunOutput{finalText: relayVisibleOutput(output, repairOutput)}, err
	}
	if handoffErr == nil {
		return runRelayRoundProtocolErrorCorrection(
			ctx,
			repairAgent,
			round,
			roundTraceID+relayRoundProtocolErrorTraceSuffix,
			relay,
			[]string{output, repairOutput},
			sink,
		)
	}
	return relayRoundRunOutput{handoff: handoffErr, traceID: repairTraceID}, nil
}

func relayMissingHandoffError(round int) error {
	return fmt.Errorf("relay round %d ended without relay handoff tool", round)
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

func (r Runner) recordRound(
	req RunRequest,
	record session.RelayRecord,
	handoffErr *agent.ErrIterationHandoff,
) (Result, bool, error) {
	req.Session.AppendRelayRecord(record)
	if err := r.SessionStore.Save(req.Session); err != nil {
		return Result{}, false, err
	}
	if !handoffErr.Completed {
		return Result{}, false, nil
	}
	return Result{
		Message:        handoffErr.FinalMessage,
		StoppedBy:      StopCompleted,
		FinalChangeLog: handoffErr.FinalChangeLog,
		Records:        cloneRelayRecords(req.Session.RelayRuntime),
	}, true, nil
}

func (r Runner) startRoundCard(
	ctx context.Context,
	req RunRequest,
	round int,
) (*runcardHandle, error) {
	recorder := runcardRecorderFromContext(ctx)
	if recorder == nil {
		return nil, nil
	}
	return recorder.StartCard(ctx, runcardStartInput{
		kind:            taskdefs.RunCardKindRelayRound,
		title:           apptasks.AgentRunCardTaskTitle(req.Task),
		nodeID:          strings.TrimSpace(req.Task.ID),
		nodeType:        taskdefs.KindAgentMessage,
		round:           round,
		sourceSessionID: strings.TrimSpace(req.Session.ID),
		startedAt:       time.Now().UTC(),
	})
}

func relayRoundStatusFromError(ctx context.Context, err error) string {
	switch {
	case errors.Is(ctx.Err(), context.Canceled), errors.Is(err, context.Canceled):
		return taskdefs.RunStatusCancelled
	default:
		return taskdefs.RunStatusError
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
