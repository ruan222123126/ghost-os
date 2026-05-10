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
	runner, err := requestScopedAgentRunner(s.agentRunner, prepared.RequestRuntime)
	if err != nil {
		return "", "", err
	}
	if prepared.RuntimeOverrides != nil {
		if hasAgentInputImages(prepared.UserInput) {
			return "", "", errRuntimeOverrideWithImages
		}
		runner, ok := runner.(SessionTurnRunnerWithOverrides)
		if !ok {
			return "", "", errRuntimeOverrideRunnerRequired
		}
		return runner.RunTurnWithOverrides(
			ctx,
			prepared.Message,
			prepared.SessionID,
			traceID,
			prepared.RuntimeOverrides,
		)
	}
	if hasAgentInputImages(prepared.UserInput) {
		runner, ok := runner.(StructuredSessionTurnRunner)
		if !ok {
			return "", "", errStructuredAgentRunnerRequired
		}
		return runner.RunTurnInput(ctx, prepared.UserInput, prepared.SessionID, traceID)
	}
	return runner.RunTurn(ctx, prepared.Message, prepared.SessionID, traceID)
}

func (s *bridgeService) runPreparedAgentTurnStream(
	ctx context.Context,
	prepared preparedAgentTurnRequest,
	traceID string,
	sink streaming.Sink,
) (string, string, error) {
	runner, err := requestScopedAgentRunner(s.agentRunner, prepared.RequestRuntime)
	if err != nil {
		return "", "", err
	}
	if hasAgentInputImages(prepared.UserInput) {
		runner, ok := runner.(StructuredSessionTurnRunner)
		if !ok {
			return "", "", errStructuredAgentRunnerRequired
		}
		return runner.RunTurnStreamInput(ctx, prepared.UserInput, prepared.SessionID, traceID, sink)
	}
	return runner.RunTurnStream(ctx, prepared.Message, prepared.SessionID, traceID, sink)
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
