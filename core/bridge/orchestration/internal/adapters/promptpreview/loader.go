package promptpreview

import (
	"errors"
	"fmt"

	bridgeconfig "ghost-os/bridge/config"
	appprompts "ghost-os/bridge/orchestration/internal/app/prompts"
	bridgeruntime "ghost-os/bridge/runtime"
	"ghost-os/bridge/tools"
)

type RuntimeDependencies interface {
	Config() bridgeconfig.Config
	Registry() *tools.Registry
	Close()
}

type RuntimeFactory interface {
	Build(store bridgeconfig.Store) (RuntimeDependencies, error)
}

type RuntimeFactoryFunc func(bridgeconfig.Store) (RuntimeDependencies, error)

func (f RuntimeFactoryFunc) Build(store bridgeconfig.Store) (RuntimeDependencies, error) {
	return f(store)
}

type Loader struct {
	Store          bridgeconfig.Store
	RuntimeFactory RuntimeFactory
}

func (l Loader) Load(cfg bridgeconfig.Config) (appprompts.Preview, error) {
	catalog, cleanup, err := l.loadCatalog(cfg)
	if err != nil {
		return appprompts.Preview{}, err
	}
	defer cleanup()

	rendered, err := bridgeruntime.BuildSystemPromptForCatalog(cfg, catalog)
	if err != nil {
		return appprompts.Preview{}, fmt.Errorf("build system prompt preview: %w", err)
	}
	return appprompts.Preview{
		RenderedPrompt:  rendered,
		ToolDefinitions: appprompts.ToolDefinitionsFrom(catalog.ToolDefs()),
	}, nil
}

func (l Loader) loadCatalog(cfg bridgeconfig.Config) (tools.ToolCatalog, func(), error) {
	if l.RuntimeFactory == nil {
		return nil, func() {}, errors.New("runtime factory unavailable for system prompt preview")
	}

	deps, err := l.RuntimeFactory.Build(l.Store)
	if err != nil {
		return nil, func() {}, fmt.Errorf("build runtime dependencies for system prompt preview: %w", err)
	}
	if deps == nil {
		return nil, func() {}, errors.New("runtime dependencies unavailable for system prompt preview")
	}
	if deps.Registry() == nil {
		deps.Close()
		return nil, func() {}, errors.New("tool registry unavailable for system prompt preview")
	}
	baseCatalog := tools.NewPromptOverrideCatalog(deps.Registry(), deps.Config().ToolSelector.PromptOverrides)
	return bridgeruntime.NewToolSelectionPolicy(cfg).ResidentCatalog(baseCatalog), deps.Close, nil
}
