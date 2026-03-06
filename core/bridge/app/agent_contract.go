package app

import (
	"errors"
	"fmt"
	"strings"
)

// newAgentResponsePayload 构造并校验 AGENT_SEND 成功响应，确保会话结束契约稳定。
func newAgentResponsePayload(message string, sessionID string, sessionEnd *assistantSessionEndSignalPayload) (agentResponse, error) {
	trimmedMessage := strings.TrimSpace(message)
	trimmedSessionID := strings.TrimSpace(sessionID)
	ended := sessionEnd != nil
	payload := agentResponse{
		Message:      trimmedMessage,
		SessionID:    trimmedSessionID,
		SessionEnded: ended,
		SessionEnd:   sessionEnd,
	}
	if err := validateAgentResponsePayload(payload); err != nil {
		return agentResponse{}, err
	}
	return payload, nil
}

// validateAgentResponsePayload 在跨进程返回前执行最小契约校验。
func validateAgentResponsePayload(payload agentResponse) error {
	if strings.TrimSpace(payload.Message) == "" {
		return errors.New("agent response message is empty")
	}
	if strings.TrimSpace(payload.SessionID) == "" {
		return errors.New("agent response session_id is empty")
	}
	if payload.SessionEnded && payload.SessionEnd == nil {
		return errors.New("agent response session_end is required when session_ended=true")
	}
	if !payload.SessionEnded && payload.SessionEnd != nil {
		return errors.New("agent response session_end must be empty when session_ended=false")
	}
	if payload.SessionEnd != nil {
		if strings.TrimSpace(payload.SessionEnd.Signal) != busAssistantSessionEndSignal {
			return fmt.Errorf("agent response session_end.signal must be %q", busAssistantSessionEndSignal)
		}
		if strings.TrimSpace(payload.SessionEnd.Message) == "" {
			return errors.New("agent response session_end.message is empty")
		}
		if strings.TrimSpace(payload.SessionEnd.Message) != strings.TrimSpace(payload.Message) {
			return errors.New("agent response session_end.message must equal message")
		}
	}
	return nil
}
