package orchestration

import (
	"strings"

	bridgeconfig "ghost-os/bridge/config"
)

type requestRuntimeOptions struct {
	ProjectRoot string
}

type requestRuntimeAwareRunner interface {
	withRequestRuntimeOptions(*requestRuntimeOptions) SessionTurnRunner
}

func normalizeRequestRuntimeOptions(rawProjectRoot string) (*requestRuntimeOptions, error) {
	trimmed := strings.TrimSpace(rawProjectRoot)
	if trimmed == "" {
		return nil, nil
	}

	projectRoot, err := bridgeconfig.NormalizeProjectRoot(trimmed)
	if err != nil {
		return nil, err
	}
	return &requestRuntimeOptions{ProjectRoot: projectRoot}, nil
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
	if input == nil {
		return nil
	}
	return &requestRuntimeOptions{ProjectRoot: input.ProjectRoot}
}
