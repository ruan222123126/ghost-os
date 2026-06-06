package orchestration

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"ghost-os/bridge/llm"
	"ghost-os/bridge/orchestration/internal/app/agentturn"
	"ghost-os/bridge/streaming"
)

type agentResponseMeta struct {
	Mode string
}

// newAgentResponsePayload 构造并校验 AGENT_SEND 成功响应，确保会话结束契约稳定。
func newAgentResponsePayload(message string, sessionID string, sessionEnd *assistantSessionEndSignalPayload, meta agentResponseMeta) (agentResponse, error) {
	trimmedMessage := strings.TrimSpace(message)
	trimmedSessionID := strings.TrimSpace(sessionID)
	ended := sessionEnd != nil
	payload := agentResponse{
		Message:      trimmedMessage,
		SessionID:    trimmedSessionID,
		SessionEnded: ended,
		SessionEnd:   sessionEnd,
		Mode:         strings.TrimSpace(meta.Mode),
	}
	if err := validateAgentResponsePayload(payload); err != nil {
		return agentResponse{}, err
	}
	return payload, nil
}

// validateAgentResponsePayload 在跨进程返回前执行最小契约校验。
func validateAgentResponsePayload(payload agentResponse) error {
	if err := validateAgentResponseRequiredFields(payload); err != nil {
		return err
	}
	if err := validateAgentResponseSessionEnd(payload); err != nil {
		return err
	}
	if payload.Mode == "" {
		return nil
	}
	if err := validateAgentResponseMode(payload); err != nil {
		return err
	}
	return nil
}

func validateAgentResponseRequiredFields(payload agentResponse) error {
	if strings.TrimSpace(payload.Message) == "" {
		return errors.New("agent response message is empty")
	}
	if strings.TrimSpace(payload.SessionID) == "" {
		return errors.New("agent response session_id is empty")
	}
	return nil
}

func validateAgentResponseSessionEnd(payload agentResponse) error {
	if payload.SessionEnded && payload.SessionEnd == nil {
		return errors.New("agent response session_end is required when session_ended=true")
	}
	if !payload.SessionEnded && payload.SessionEnd != nil {
		return errors.New("agent response session_end must be empty when session_ended=false")
	}
	if payload.SessionEnd == nil {
		return nil
	}
	if strings.TrimSpace(payload.SessionEnd.Signal) != busAssistantSessionEndSignal {
		return fmt.Errorf("agent response session_end.signal must be %q", busAssistantSessionEndSignal)
	}
	if strings.TrimSpace(payload.SessionEnd.Message) == "" {
		return errors.New("agent response session_end.message is empty")
	}
	if strings.TrimSpace(payload.SessionEnd.Message) != strings.TrimSpace(payload.Message) {
		return errors.New("agent response session_end.message must equal message")
	}
	return nil
}

func validateAgentResponseMode(payload agentResponse) error {
	switch payload.Mode {
	case agentModePlan:
		return nil
	default:
		return errors.New("agent response mode is invalid")
	}
}

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
