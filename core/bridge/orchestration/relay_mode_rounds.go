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
	turnAgent := newRelayTurnAgent(req.deps, catalog, systemPrompt)
	roundTraceID := fmt.Sprintf("%s-relay-%d", strings.TrimSpace(req.traceID), round)
	userPrompt := buildRelayModeUserPrompt(req.task.Message, cloneRelayRecords(req.session.RelayRuntime), round, req.relay)
	handoffErr, handoffTraceID, err := runRelayRoundWithRepair(ctx, req.deps, systemPrompt, turnAgent, userPrompt, round, roundTraceID, req.relay)
	if err != nil {
		return relayModeResult{}, false, err
	}
	return r.recordRound(req, round, handoffTraceID, handoffErr)
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
	userPrompt string,
	round int,
	roundTraceID string,
	relay TaskRelayConfig,
) (*agent.ErrIterationHandoff, string, error) {
	output, runErr := turnAgent.RunWithTraceID(ctx, userPrompt, roundTraceID)
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
	repairAgent.SetToolChoice("required")
	_, repairErr := repairAgent.RunWithTraceID(ctx, repairPrompt, repairTraceID)
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
	round int,
	roundTraceID string,
	handoffErr *agent.ErrIterationHandoff,
) (relayModeResult, bool, error) {
	req.session.AppendRelayRecord(relayRecordFromHandoff(round, roundTraceID, handoffErr))
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
