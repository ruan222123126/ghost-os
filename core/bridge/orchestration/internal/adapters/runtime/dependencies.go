package runtime

import (
	"ghost-os/bridge/agent"
	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/orchestration/internal/adapters/toolruntime"
	agentturnplan "ghost-os/bridge/orchestration/internal/app/agentturn/plan"
	apprelay "ghost-os/bridge/orchestration/internal/app/agentturn/relay"
	ownerapp "ghost-os/bridge/orchestration/internal/app/orchestrations/owner"
	appworkflows "ghost-os/bridge/orchestration/internal/app/workflows"
	"ghost-os/bridge/tools"
)

type Dependencies interface {
	Config() bridgeconfig.Config
	Client() agent.Completer
	Registry() *tools.Registry
	SystemPrompt() string
	SystemPromptOverride() bool
	SystemPromptFiles() *bridgeconfig.SystemPromptFiles
	Close()
}

func ToPlanDependencies(deps Dependencies) agentturnplan.RuntimeDependencies {
	if deps == nil {
		return agentturnplan.RuntimeDependencies{}
	}
	return agentturnplan.RuntimeDependencies{
		Config:       deps.Config(),
		Client:       deps.Client(),
		SystemPrompt: deps.SystemPrompt(),
		Cleanup:      deps.Close,
	}
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

func ToToolRuntimeDependencies(deps Dependencies) toolruntime.RuntimeDependencies {
	if deps == nil {
		return nil
	}
	return toolRuntimeDependencies{deps: deps}
}

type toolRuntimeDependencies struct {
	deps Dependencies
}

func (d toolRuntimeDependencies) Registry() *tools.Registry {
	if d.deps == nil {
		return nil
	}
	return d.deps.Registry()
}

func (d toolRuntimeDependencies) Close() {
	if d.deps != nil {
		d.deps.Close()
	}
}
