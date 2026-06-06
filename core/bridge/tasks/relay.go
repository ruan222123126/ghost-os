package tasks

import "ghost-os/bridge/taskdefs"

const (
	AgentModeSingle = taskdefs.AgentModeSingle
	AgentModeRelay  = taskdefs.AgentModeRelay

	RelayStopPolicyAIDecides = taskdefs.RelayStopPolicyAIDecides
	RelayStopPolicyMaxRounds = taskdefs.RelayStopPolicyMaxRounds
)

type TaskRelayConfig = taskdefs.TaskRelayConfig

func NormalizeAgentMode(mode string) string {
	return taskdefs.NormalizeAgentMode(mode)
}

func CloneTaskRelayConfig(input *TaskRelayConfig) *TaskRelayConfig {
	return taskdefs.CloneTaskRelayConfig(input)
}

func IsRelayAgentTask(task ScheduledTask) bool {
	return NormalizeKind(task.TaskKind) == KindAgentMessage &&
		NormalizeAgentMode(task.AgentMode) == AgentModeRelay
}
