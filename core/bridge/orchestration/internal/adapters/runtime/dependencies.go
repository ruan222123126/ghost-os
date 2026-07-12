package runtime

import (
	"ghost-os/bridge/agent"
	bridgeconfig "ghost-os/bridge/config"
	apprelay "ghost-os/bridge/orchestration/internal/app/agentturn/relay"
	ownerapp "ghost-os/bridge/orchestration/internal/app/orchestrations/owner"
	appworkflows "ghost-os/bridge/orchestration/internal/app/workflows"
	"ghost-os/bridge/session"
	"ghost-os/bridge/tools"
)

type Dependencies interface {
	Config() bridgeconfig.Config
	Client() agent.Completer
	Registry() *tools.Registry
	SystemPrompt() string
	SystemPromptOverride() bool
	SystemPromptFiles() *bridgeconfig.SystemPromptFiles
	RuntimeSelection() *session.RuntimeSelection
	Close()
}

func ToRelayDependencies(deps Dependencies) apprelay.RuntimeDependencies {
	if deps == nil {
		return apprelay.RuntimeDependencies{}
	}
	return apprelay.RuntimeDependencies{
		Config:               deps.Config(),
		Client:               deps.Client(),
		Registry:             deps.Registry(),
		SystemPrompt:         deps.SystemPrompt(),
		SystemPromptOverride: deps.SystemPromptOverride(),
		Cleanup:              deps.Close,
	}
}

func ToOwnerDecisionDependencies(deps Dependencies) ownerapp.DecisionRuntimeDependencies {
	if deps == nil {
		return ownerapp.DecisionRuntimeDependencies{}
	}
	return ownerapp.DecisionRuntimeDependencies{
		Config:               deps.Config(),
		Client:               deps.Client(),
		Registry:             deps.Registry(),
		SystemPrompt:         deps.SystemPrompt(),
		SystemPromptOverride: deps.SystemPromptOverride(),
		SystemPromptFiles:    deps.SystemPromptFiles(),
		Cleanup:              deps.Close,
	}
}

func ToWorkflowDependencies(deps Dependencies) appworkflows.RuntimeDependencies {
	if deps == nil {
		return appworkflows.RuntimeDependencies{}
	}
	return appworkflows.RuntimeDependencies{
		Client:   deps.Client(),
		Registry: deps.Registry(),
		Cleanup:  deps.Close,
	}
}
