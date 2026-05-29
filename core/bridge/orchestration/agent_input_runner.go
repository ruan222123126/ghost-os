package orchestration

import (
	"context"
	"errors"

	"ghost-os/bridge/llm"
	"ghost-os/bridge/orchestration/internal/app/agentturn"
	"ghost-os/bridge/streaming"
)

var errStructuredAgentRunnerRequired = errors.New("configured agent runner does not support image input")
var errRuntimeOverrideRunnerRequired = errors.New("configured agent runner does not support runtime_overrides")
var errRuntimeOverrideWithImages = errors.New("runtime_overrides do not support image input")
var errRequestRuntimeRunnerRequired = errors.New("configured agent runner does not support request runtime options")

func buildAgentUserInput(message string, images []sessionImageContent) (llm.Message, string, error) {
	return agentturn.BuildUserInput(message, images)
}

func normalizeAgentImages(images []sessionImageContent) ([]llm.ContentPart, error) {
	if len(images) == 0 {
		return nil, nil
	}
	message, _, err := agentturn.BuildUserInput("", images)
	return message.Content, err
}

func normalizeAgentImage(image sessionImageContent, index int) (llm.ContentPart, error) {
	parts, err := normalizeAgentImages([]sessionImageContent{image})
	if err != nil {
		return llm.ContentPart{}, err
	}
	if len(parts) == 0 {
		return llm.ContentPart{}, errAgentMessageRequired
	}
	return parts[0], nil
}

func hasAgentInputImages(message llm.Message) bool {
	return agentturn.HasInputImages(message)
}

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
