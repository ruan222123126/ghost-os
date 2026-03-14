package orchestration

import (
	"ghost-os/bridge/agent"
	"ghost-os/bridge/tools"
)

type RuntimeDependencies = agentRuntimeDependencies

func NewRuntimeDependencies(
	cfg Config,
	client agent.Completer,
	registry *tools.Registry,
	systemPrompt string,
	cleanup func(),
) RuntimeDependencies {
	return agentRuntimeDependencies{
		cfg:          cfg,
		client:       client,
		registry:     registry,
		systemPrompt: systemPrompt,
		cleanup:      cleanup,
	}
}
