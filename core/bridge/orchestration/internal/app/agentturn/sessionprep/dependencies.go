package sessionprep

import (
	"errors"
	"fmt"
	"strings"

	bridgeconfig "ghost-os/bridge/config"
	"ghost-os/bridge/internal/runtimeutil"
	"ghost-os/bridge/orchestration/internal/app/agentturn/turnstate"
	appsessions "ghost-os/bridge/orchestration/internal/app/sessions"
	apptasks "ghost-os/bridge/orchestration/internal/app/tasks"
	bridgeruntime "ghost-os/bridge/runtime"
	"ghost-os/bridge/session"
	bridgeTasks "ghost-os/bridge/tasks"
)

var ErrRuntimeFactoryRequired = errors.New("session turn runtime factory is not configured")

type RuntimeFactory interface {
	Build(store bridgeconfig.Store) (RuntimeDependencies, error)
}

type DependencyCommand struct {
	RuntimeFactory   RuntimeFactory
	ConfigStore      bridgeconfig.Store
	SessionStore     *session.Store
	RuntimeOverrides *bridgeTasks.TaskRuntimeOverrides
}

type PreparedDependencies struct {
	Deps           RuntimeDependencies
	HistoryBuilder *appsessions.HistoryBuilder
	Persistence    turnstate.Persistence
}

func BuildDependencies(cmd DependencyCommand) (PreparedDependencies, error) {
	if cmd.RuntimeFactory == nil {
		return PreparedDependencies{}, ErrRuntimeFactoryRequired
	}
	normalized, err := apptasks.NormalizeTaskRuntimeOverrides(cmd.RuntimeOverrides)
	if err != nil {
		return PreparedDependencies{}, err
	}
	runtimeStore := apptasks.NewRuntimeOverrideStore(cmd.ConfigStore, normalized)
	deps, err := cmd.RuntimeFactory.Build(runtimeStore)
	if err != nil {
		return PreparedDependencies{}, err
	}
	depsWithPreset, err := applyTaskRuntimePresetOverride(deps, cmd.ConfigStore, normalized)
	if err != nil {
		deps.Close()
		return PreparedDependencies{}, err
	}
	deps = depsWithPreset
	deps = applyTaskRuntimePromptOverride(deps, normalized)
	deps.RuntimeSelection = runtimeutil.BuildGhostRuntimeSelection(
		runtimeStore,
		deps.Config,
		session.RuntimeSelectionModeDefault,
	)
	return PreparedDependencies{
		Deps: deps,
		HistoryBuilder: appsessions.NewHistoryBuilderFromConfig(
			deps.Config,
			deps.SystemPrompt,
			cmd.SessionStore,
			"",
		),
		Persistence: turnstate.NewCommitter(cmd.SessionStore),
	}, nil
}

func applyTaskRuntimePresetOverride(
	deps RuntimeDependencies,
	configStore bridgeconfig.Store,
	runtimeOverrides *bridgeTasks.TaskRuntimeOverrides,
) (RuntimeDependencies, error) {
	if runtimeOverrides == nil || strings.TrimSpace(runtimeOverrides.PresetID) == "" {
		return deps, nil
	}
	if configStore == nil {
		return RuntimeDependencies{}, errors.New("preset_id requires config store")
	}
	files, err := configStore.SystemPrompts()
	if err != nil {
		return RuntimeDependencies{}, err
	}
	presets, err := configStore.Presets()
	if err != nil {
		return RuntimeDependencies{}, err
	}
	preset, ok := bridgeconfig.FindPresetByID(presets, runtimeOverrides.PresetID)
	if !ok {
		return RuntimeDependencies{}, fmt.Errorf("preset_id %q is not configured", strings.TrimSpace(runtimeOverrides.PresetID))
	}
	presetFiles, err := bridgeconfig.ApplyPresetToSystemPromptFiles(files, preset)
	if err != nil {
		return RuntimeDependencies{}, err
	}
	prompt, err := bridgeruntime.BuildSystemPromptForSessionWithFiles(
		deps.Config,
		bridgeruntime.NewToolSelectionPolicy(deps.Config).ResidentCatalog(deps.Registry),
		nil,
		deps.Config.ToolSearch.IdleTurns,
		presetFiles,
	)
	if err != nil {
		return RuntimeDependencies{}, err
	}
	deps.SystemPrompt = strings.TrimSpace(prompt)
	deps.SystemPromptFiles = &presetFiles
	return deps, nil
}

func applyTaskRuntimePromptOverride(
	deps RuntimeDependencies,
	runtimeOverrides *bridgeTasks.TaskRuntimeOverrides,
) RuntimeDependencies {
	if runtimeOverrides == nil || strings.TrimSpace(runtimeOverrides.SystemPrompt) == "" {
		return deps
	}
	deps.SystemPrompt = strings.TrimSpace(runtimeOverrides.SystemPrompt)
	deps.SystemPromptOverride = true
	return deps
}
