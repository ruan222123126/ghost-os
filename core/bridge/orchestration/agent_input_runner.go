package orchestration

import (
	"context"
	"errors"

	"ghost-os/bridge/streaming"
)

var errStructuredAgentRunnerRequired = errors.New("configured agent runner does not support image input")
var errRuntimeOverrideRunnerRequired = errors.New("configured agent runner does not support runtime_overrides")
var errRuntimeOverrideWithImages = errors.New("runtime_overrides do not support image input")
var errRequestRuntimeRunnerRequired = errors.New("configured agent runner does not support request runtime options")

func (s *bridgeService) runPreparedAgentTurn(
	ctx context.Context,
	prepared preparedAgentTurnRequest,
	traceID string,
) (string, string, error) {
	runner, err := requestScopedAgentRunner(s.agentRunner, prepared.requestRuntime)
	if err != nil {
		return "", "", err
	}
	if prepared.runtimeOverrides != nil {
		if hasAgentInputImages(prepared.userInput) {
			return "", "", errRuntimeOverrideWithImages
		}
		runner, ok := runner.(SessionTurnRunnerWithOverrides)
		if !ok {
			return "", "", errRuntimeOverrideRunnerRequired
		}
		return runner.RunTurnWithOverrides(
			ctx,
			prepared.message,
			prepared.sessionID,
			traceID,
			prepared.runtimeOverrides,
		)
	}
	if hasAgentInputImages(prepared.userInput) {
		runner, ok := runner.(StructuredSessionTurnRunner)
		if !ok {
			return "", "", errStructuredAgentRunnerRequired
		}
		return runner.RunTurnInput(ctx, prepared.userInput, prepared.sessionID, traceID)
	}
	return runner.RunTurn(ctx, prepared.message, prepared.sessionID, traceID)
}

func (s *bridgeService) runPreparedAgentTurnStream(
	ctx context.Context,
	prepared preparedAgentTurnRequest,
	traceID string,
	sink streaming.Sink,
) (string, string, error) {
	runner, err := requestScopedAgentRunner(s.agentRunner, prepared.requestRuntime)
	if err != nil {
		return "", "", err
	}
	if hasAgentInputImages(prepared.userInput) {
		runner, ok := runner.(StructuredSessionTurnRunner)
		if !ok {
			return "", "", errStructuredAgentRunnerRequired
		}
		return runner.RunTurnStreamInput(ctx, prepared.userInput, prepared.sessionID, traceID, sink)
	}
	return runner.RunTurnStream(ctx, prepared.message, prepared.sessionID, traceID, sink)
}

func requestScopedAgentRunner(
	runner SessionTurnRunner,
	options *requestRuntimeOptions,
) (SessionTurnRunner, error) {
	if options == nil {
		return runner, nil
	}
	aware, ok := runner.(requestRuntimeAwareRunner)
	if !ok {
		return nil, errRequestRuntimeRunnerRequired
	}
	return aware.withRequestRuntimeOptions(options), nil
}
