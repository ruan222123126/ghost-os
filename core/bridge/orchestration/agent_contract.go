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
	IterationSummary []map[string]any
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
	if payload.Mode != "" {
		switch payload.Mode {
		case proModePro, proModeProx:
		default:
			return errors.New("agent response mode is invalid")
		}
		if payload.IterationCount <= 0 {
			return errors.New("agent response iteration_count must be > 0 when mode is set")
		}
		if strings.TrimSpace(payload.StoppedBy) == "" {
			return errors.New("agent response stopped_by is required when mode is set")
		}
	}
	return nil
}

func cloneIterationSummary(records []map[string]any) []map[string]any {
	if len(records) == 0 {
		return nil
	}
	out := make([]map[string]any, 0, len(records))
	for _, record := range records {
		cloned := make(map[string]any, len(record))
		for key, value := range record {
			cloned[key] = value
		}
		out = append(out, cloned)
	}
	return out
}
