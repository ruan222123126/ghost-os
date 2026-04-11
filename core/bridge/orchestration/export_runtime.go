package orchestration

import (
	"ghost-os/bridge/agent"
	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/tools"
)

type RuntimeDependencies = agentRuntimeDependencies
type RuntimeCompleter = agent.Completer
type RuntimeToolRegistry = tools.Registry

func NewRuntimeDependencies(
	cfg bridgeconfig.Config,
	client RuntimeCompleter,
	registry *RuntimeToolRegistry,
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
