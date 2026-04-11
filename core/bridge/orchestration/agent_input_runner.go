package orchestration

import (
	"context"
	"errors"

	"ghost-os/bridge/streaming"
)

var errStructuredAgentRunnerRequired = errors.New("configured agent runner does not support image input")
var errRuntimeOverrideRunnerRequired = errors.New("configured agent runner does not support runtime_overrides")
var errRuntimeOverrideWithImages = errors.New("runtime_overrides do not support image input")

func (s *bridgeService) runPreparedAgentTurn(
	ctx context.Context,
	prepared preparedAgentTurnRequest,
	traceID string,
) (string, string, error) {
	if prepared.runtimeOverrides != nil {
		if hasAgentInputImages(prepared.userInput) {
			return "", "", errRuntimeOverrideWithImages
		}
		runner, ok := s.agentRunner.(SessionTurnRunnerWithOverrides)
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
		runner, ok := s.agentRunner.(StructuredSessionTurnRunner)
		if !ok {
			return "", "", errStructuredAgentRunnerRequired
		}
		return runner.RunTurnInput(ctx, prepared.userInput, prepared.sessionID, traceID)
	}
	return s.agentRunner.RunTurn(ctx, prepared.message, prepared.sessionID, traceID)
}

func (s *bridgeService) runPreparedAgentTurnStream(
	ctx context.Context,
	prepared preparedAgentTurnRequest,
	traceID string,
	sink streaming.Sink,
) (string, string, error) {
	if hasAgentInputImages(prepared.userInput) {
		runner, ok := s.agentRunner.(StructuredSessionTurnRunner)
		if !ok {
			return "", "", errStructuredAgentRunnerRequired
		}
		return runner.RunTurnStreamInput(ctx, prepared.userInput, prepared.sessionID, traceID, sink)
	}
	return s.agentRunner.RunTurnStream(ctx, prepared.message, prepared.sessionID, traceID, sink)
}
