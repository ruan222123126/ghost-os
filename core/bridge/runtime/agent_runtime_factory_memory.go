package runtime

import (
	"os"
	"strings"

	"ghost-os/bridge/llm"
	"ghost-os/bridge/memoryaug"
	"ghost-os/bridge/memorystore"
	"ghost-os/bridge/tools"
)

type memoryRuntimeResources struct {
	planner memoryaug.IntentPlanner
	recall  memoryaug.RecallService
	learn   memoryaug.LearningService
	cleanup func()
}

func (d memoryRuntimeResources) Close() {
	if d.cleanup != nil {
		d.cleanup()
	}
}

func setupMemoryAugmentation(cfg Config, registry *tools.Registry) (memoryRuntimeResources, error) {
	settings := memorySettingsFromConfig(cfg)
	if !settings.Enabled {
		return memoryRuntimeResources{}, nil
	}
	return setupEnabledMemoryAugmentation(cfg, registry, settings)
}

func setupEnabledMemoryAugmentation(
	cfg Config,
	registry *tools.Registry,
	settings memoryaug.Settings,
) (memoryRuntimeResources, error) {
	store, err := memorystore.NewStoreWithOptions(memorystore.StoreOptions{
		Path:               strings.TrimSpace(os.Getenv("GHOST_MEMORY_PATH")),
		DefaultUserScopeID: settings.UserScopeID,
	})
	if err != nil {
		return memoryRuntimeResources{}, err
	}
	memoryClient := llm.NewClientWithOptions(providerClientOptions(cfg, strings.TrimSpace(settings.LLMModel)))
	planner, recall, learn := newMemoryRuntimeServices(settings, store, memoryClient)
	registerMemoryRuntimeTools(registry, store, planner, recall, settings.UserScopeID)
	return memoryRuntimeResources{
		planner: planner,
		recall:  recall,
		learn:   learn,
		cleanup: func() {
			_ = store.Close()
		},
	}, nil
}

func newMemoryRuntimeServices(
	settings memoryaug.Settings,
	store *memorystore.Store,
	memoryClient *llm.Client,
) (memoryaug.IntentPlanner, memoryaug.RecallService, memoryaug.LearningService) {
	planner := memoryaug.NewIntentPlanner(settings, store, memoryClient)
	recall := memoryaug.NewRecallService(settings, store)
	learn := memoryaug.NewLearningService(
		settings,
		store,
		memoryaug.NewLLMExtractor(memoryClient),
		memoryaug.NewEventLLMExtractor(memoryClient),
	)
	return planner, recall, learn
}

func registerMemoryRuntimeTools(
	registry *tools.Registry,
	store *memorystore.Store,
	planner memoryaug.IntentPlanner,
	recall memoryaug.RecallService,
	userScopeID string,
) {
	registry.Register(tools.NewMemoryManageTool(store))
	registry.Register(tools.NewMemoryLearnedListTool(store))
	registry.Register(tools.NewMemoryRecallDebugTool(planner, recall, userScopeID))
}

func memorySettingsFromConfig(cfg Config) memoryaug.Settings {
	return memoryaug.Settings{
		Enabled:             cfg.MemoryAugmentation.Enabled,
		LearningEnabled:     cfg.MemoryAugmentation.LearningEnabled,
		RecallEnabled:       cfg.MemoryAugmentation.RecallEnabled,
		MaxRecallItems:      cfg.MemoryAugmentation.MaxRecallItems,
		MinConfidence:       cfg.MemoryAugmentation.MinConfidence,
		SessionScopeEnabled: cfg.MemoryAugmentation.SessionScopeEnabled,
		UserScopeEnabled:    cfg.MemoryAugmentation.UserScopeEnabled,
		LLMModel:            cfg.MemoryAugmentation.LLMModel,
		UserScopeID:         cfg.MemoryAugmentation.UserScopeID,
	}
}
