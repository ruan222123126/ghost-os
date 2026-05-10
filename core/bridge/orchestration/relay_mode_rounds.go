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
	if req.relay.StopPolicy != taskRelayStopPolicyMaxRounds || round <= req.relay.MaxRounds {
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
	_, runErr := turnAgent.RunWithTraceID(ctx, userPrompt, roundTraceID)
	if runErr == nil {
		return relayModeResult{}, false, fmt.Errorf("relay round %d ended without relay handoff tool", round)
	}
	var handoffErr *agent.ErrIterationHandoff
	if !errors.As(runErr, &handoffErr) {
		return relayModeResult{}, false, runErr
	}
	return r.recordRound(req, round, roundTraceID, handoffErr)
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
