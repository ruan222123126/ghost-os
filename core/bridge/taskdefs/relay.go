package taskdefs

import "strings"

const (
	AgentModeSingle = "single"
	AgentModeRelay  = "relay"

	RelayStopPolicyAIDecides = "ai_decides"
	RelayStopPolicyMaxRounds = "max_rounds"
)

type TaskRelayConfig struct {
	StopPolicy         string `json:"stop_policy"`
	MaxRounds          int    `json:"max_rounds,omitempty"`
	ExecutionTimeoutMS *int   `json:"execution_timeout_ms,omitempty"`
}

func NormalizeAgentMode(mode string) string {
	switch strings.TrimSpace(mode) {
	case "", AgentModeSingle:
		return AgentModeSingle
	case AgentModeRelay:
		return AgentModeRelay
	default:
		return strings.TrimSpace(mode)
	}
}

func CloneTaskRelayConfig(input *TaskRelayConfig) *TaskRelayConfig {
	if input == nil {
		return nil
	}
	return &TaskRelayConfig{
		StopPolicy:         strings.TrimSpace(input.StopPolicy),
		MaxRounds:          input.MaxRounds,
		ExecutionTimeoutMS: cloneOptionalIntPointer(input.ExecutionTimeoutMS),
	}
}
