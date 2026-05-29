package orchestration

import (
	"ghost-os/bridge/agent"
	bridgeconfig "ghost-os/bridge/config"
	bridgeruntime "ghost-os/bridge/runtime"
	"ghost-os/bridge/tools"
)

type agentRuntimeDependencies struct {
	cfg                  bridgeconfig.Config
	client               agent.Completer
	registry             *tools.Registry
	systemPrompt         string
	systemPromptOverride bool
	systemPromptFiles    *bridgeconfig.SystemPromptFiles
	cleanup              func()
}

func (d agentRuntimeDependencies) Close() {
	if d.cleanup != nil {
		d.cleanup()
	}
}

type AgentRuntimeFactory interface {
	Build(store bridgeconfig.Store) (agentRuntimeDependencies, error)
}

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

// runtimeFactoryAdapter 将 runtime 包导出的工厂转换为 orchestration 内部依赖结构，
// 以便保持编排层测试替身和字段级装配不变。
type runtimeFactoryAdapter struct {
	inner bridgeruntime.AgentRuntimeFactory
}

func (f runtimeFactoryAdapter) Build(store bridgeconfig.Store) (agentRuntimeDependencies, error) {
	var innerStore *bridgeruntime.ConfigStore
	if store != nil {
		innerStore = bridgeruntime.WrapConfigStore(store)
	}
	deps, err := f.inner.Build(innerStore)
	if err != nil {
		return agentRuntimeDependencies{}, err
	}
	return agentRuntimeDependencies{
		cfg:          deps.Config(),
		client:       deps.Client(),
		registry:     deps.Registry(),
		systemPrompt: deps.SystemPrompt(),
		cleanup:      deps.Close,
	}, nil
}

func newAgentRuntimeFactory() AgentRuntimeFactory {
	return runtimeFactoryAdapter{inner: bridgeruntime.NewAgentRuntimeFactory()}
}
