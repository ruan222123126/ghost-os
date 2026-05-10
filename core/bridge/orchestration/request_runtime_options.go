package orchestration

import (
	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/orchestration/internal/app/agentturn"
)

type requestRuntimeOptions = agentturn.RequestRuntimeOptions

type requestRuntimeAwareRunner interface {
	withRequestRuntimeOptions(*requestRuntimeOptions) SessionTurnRunner
}

func normalizeRequestRuntimeOptions(rawProjectRoot string) (*requestRuntimeOptions, error) {
	return agentturn.NormalizeRequestRuntimeOptions(rawProjectRoot)
}

func applyRequestRuntimeOptionsToStore(
	store bridgeconfig.Store,
	options *requestRuntimeOptions,
) bridgeconfig.Store {
	if options == nil {
		return store
	}
	return bridgeconfig.WithProjectRootOverride(store, options.ProjectRoot)
}

func cloneRequestRuntimeOptions(input *requestRuntimeOptions) *requestRuntimeOptions {
	return agentturn.CloneRequestRuntimeOptions(input)
}
