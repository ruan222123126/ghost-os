package orchestration

import (
	"errors"
	"fmt"
	"strings"
)

type agentResponseMeta struct {
	Mode             string
	IterationCount   int
	StoppedBy        string
	FinalChangeLog   string
	IterationSummary []agentIterationSummaryItem
}

// newAgentResponsePayload 构造并校验 AGENT_SEND 成功响应，确保会话结束契约稳定。
func newAgentResponsePayload(message string, sessionID string, sessionEnd *assistantSessionEndSignalPayload, meta agentResponseMeta) (agentResponse, error) {
	trimmedMessage := strings.TrimSpace(message)
	trimmedSessionID := strings.TrimSpace(sessionID)
	ended := sessionEnd != nil
	payload := agentResponse{
		Message:        trimmedMessage,
		SessionID:      trimmedSessionID,
		SessionEnded:   ended,
		SessionEnd:     sessionEnd,
		Mode:           strings.TrimSpace(meta.Mode),
		IterationCount: meta.IterationCount,
		StoppedBy:      strings.TrimSpace(meta.StoppedBy),
		FinalChangeLog: strings.TrimSpace(meta.FinalChangeLog),
	}
	if len(meta.IterationSummary) > 0 {
		payload.IterationSummary = cloneIterationSummary(meta.IterationSummary)
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
	case proModePro, proModeProx:
		if payload.IterationCount <= 0 {
			return errors.New("agent response iteration_count must be > 0 for pro/prox mode")
		}
		if strings.TrimSpace(payload.StoppedBy) == "" {
			return errors.New("agent response stopped_by is required for pro/prox mode")
		}
		return nil
	case agentModePlan:
		return nil
	default:
		return errors.New("agent response mode is invalid")
	}
}

func cloneIterationSummary(records []agentIterationSummaryItem) []agentIterationSummaryItem {
	if len(records) == 0 {
		return nil
	}
	out := make([]agentIterationSummaryItem, len(records))
	copy(out, records)
	return out
}
