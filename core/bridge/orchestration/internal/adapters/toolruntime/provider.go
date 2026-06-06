package toolruntime

import (
	"fmt"

	bridgeconfig "ghost-os/bridge/config"
	apptools "ghost-os/bridge/orchestration/internal/app/tools"
	"ghost-os/bridge/orchestration/internal/contracts/bus"
	bridgetools "ghost-os/bridge/tools"
)

type RuntimeDependencies interface {
	Registry() *bridgetools.Registry
	Close()
}

type RuntimeFactory interface {
	Build(store bridgeconfig.Store) (RuntimeDependencies, error)
}

type RuntimeFactoryFunc func(bridgeconfig.Store) (RuntimeDependencies, error)

func (f RuntimeFactoryFunc) Build(store bridgeconfig.Store) (RuntimeDependencies, error) {
	return f(store)
}

type Provider struct {
	Store          bridgeconfig.Store
	RuntimeFactory RuntimeFactory
}

func (p Provider) ListToolInputSchemas() (map[string]map[string]any, error) {
	if p.RuntimeFactory == nil {
		return nil, nil
	}
	deps, err := p.RuntimeFactory.Build(p.Store)
	if err != nil {
		return nil, err
	}
	if deps == nil {
		return nil, nil
	}
	defer deps.Close()
	if deps.Registry() == nil {
		return map[string]map[string]any{}, nil
	}
	return apptools.CollectSchemas(deps.Registry().ToolDefs())
}

func (p Provider) Tool(name string) (bridgetools.Tool, func(), error) {
	if p.RuntimeFactory == nil {
		return nil, func() {}, unavailable("runtime factory is not configured")
	}
	deps, err := p.RuntimeFactory.Build(p.Store)
	if err != nil {
		return nil, func() {}, unavailable("build runtime dependencies: %w", err)
	}
	if deps == nil {
		return nil, func() {}, unavailable("runtime factory is not configured")
	}
	if deps.Registry() == nil {
		deps.Close()
		return nil, func() {}, unavailable("tool registry is not configured")
	}
	tool := deps.Registry().Get(name)
	if tool == nil {
		deps.Close()
		return nil, func() {}, unavailable("tool %q is not available", name)
	}
	return tool, deps.Close, nil
}

func unavailable(format string, args ...any) error {
	return bus.WrapError(bus.ServiceErrorUnavailable, fmt.Errorf(format, args...))
}
