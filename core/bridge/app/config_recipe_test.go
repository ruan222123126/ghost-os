package app

import (
	"testing"
	"time"

	"ghost-os/bridge/llm"
)

func TestLoadConfigWithRuntime_LoadsDecisionRecipeReuseSettingsFromEnv(t *testing.T) {
	t.Setenv("GHOST_CONFIG_PATH", t.TempDir()+"/config.toml")
	t.Setenv("GHOST_MEMORY_DECISION_RECIPE_REUSE_ENABLED", "false")
	t.Setenv("GHOST_MEMORY_DECISION_RECIPE_EXECUTION_TRACKING_ENABLED", "true")
	t.Setenv("GHOST_MEMORY_DECISION_RECIPE_BACKFILL_ENABLED", "false")
	t.Setenv("GHOST_MEMORY_DECISION_RECIPE_DEFAULT_ENABLED", "true")
	t.Setenv("GHOST_MEMORY_DECISION_RECIPE_DEFAULT_GRAY_PERCENT", "25")
	t.Setenv("GHOST_MEMORY_DECISION_RECIPE_MIN_SELECTION_CONFIDENCE", "0.68")
	t.Setenv("GHOST_MEMORY_DECISION_RECIPE_MIN_SUCCESS_RATE", "0.64")
	t.Setenv("GHOST_MEMORY_DECISION_RECIPE_BACKFILL_BATCH_SIZE", "17")
	t.Setenv("GHOST_MEMORY_DECISION_RECIPE_BACKFILL_INTERVAL", "45m")

	cfg, err := loadConfigWithRuntime(runtimeConfig{
		Provider: llm.ProviderCustom,
		BaseURL:  "https://example.com/v1",
		Model:    "gpt-4o",
	})
	if err != nil {
		t.Fatalf("loadConfigWithRuntime returned error: %v", err)
	}

	if !cfg.MemoryDecisionRecipeReuseEnabledSet || cfg.MemoryDecisionRecipeReuseEnabled {
		t.Fatalf("unexpected recipe reuse toggle: enabled=%v set=%v", cfg.MemoryDecisionRecipeReuseEnabled, cfg.MemoryDecisionRecipeReuseEnabledSet)
	}
	if !cfg.MemoryDecisionRecipeExecutionTrackingEnabledSet || !cfg.MemoryDecisionRecipeExecutionTrackingEnabled {
		t.Fatalf("unexpected recipe execution tracking toggle: enabled=%v set=%v", cfg.MemoryDecisionRecipeExecutionTrackingEnabled, cfg.MemoryDecisionRecipeExecutionTrackingEnabledSet)
	}
	if !cfg.MemoryDecisionRecipeBackfillEnabledSet || cfg.MemoryDecisionRecipeBackfillEnabled {
		t.Fatalf("unexpected recipe backfill toggle: enabled=%v set=%v", cfg.MemoryDecisionRecipeBackfillEnabled, cfg.MemoryDecisionRecipeBackfillEnabledSet)
	}
	if !cfg.MemoryDecisionRecipeDefaultEnabledSet || !cfg.MemoryDecisionRecipeDefaultEnabled {
		t.Fatalf("unexpected recipe default toggle: enabled=%v set=%v", cfg.MemoryDecisionRecipeDefaultEnabled, cfg.MemoryDecisionRecipeDefaultEnabledSet)
	}
	if cfg.MemoryDecisionRecipeDefaultGrayPercent != 25 {
		t.Fatalf("unexpected recipe gray percent: got %d want %d", cfg.MemoryDecisionRecipeDefaultGrayPercent, 25)
	}
	if cfg.MemoryDecisionRecipeMinSelectionConfidence != 0.68 {
		t.Fatalf("unexpected recipe min selection confidence: got %v want %v", cfg.MemoryDecisionRecipeMinSelectionConfidence, 0.68)
	}
	if cfg.MemoryDecisionRecipeMinSuccessRate != 0.64 {
		t.Fatalf("unexpected recipe min success rate: got %v want %v", cfg.MemoryDecisionRecipeMinSuccessRate, 0.64)
	}
	if cfg.MemoryDecisionRecipeBackfillBatchSize != 17 {
		t.Fatalf("unexpected recipe backfill batch size: got %d want %d", cfg.MemoryDecisionRecipeBackfillBatchSize, 17)
	}
	if cfg.MemoryDecisionRecipeBackfillInterval != 45*time.Minute {
		t.Fatalf("unexpected recipe backfill interval: got %s want %s", cfg.MemoryDecisionRecipeBackfillInterval, 45*time.Minute)
	}

	memoryCfg := memoryManagerConfigFromAppConfig(cfg, nil, nil)
	if !memoryCfg.RecipeReuseEnabledSet || memoryCfg.RecipeReuseEnabled {
		t.Fatalf("unexpected memory recipe reuse config: %+v", memoryCfg)
	}
	if !memoryCfg.RecipeExecutionTrackingEnabledSet || !memoryCfg.RecipeExecutionTrackingEnabled {
		t.Fatalf("unexpected memory recipe execution tracking config: %+v", memoryCfg)
	}
	if !memoryCfg.RecipeBackfillEnabledSet || memoryCfg.RecipeBackfillEnabled {
		t.Fatalf("unexpected memory recipe backfill config: %+v", memoryCfg)
	}
	if !memoryCfg.RecipeDefaultEnabledSet || !memoryCfg.RecipeDefaultEnabled {
		t.Fatalf("unexpected memory recipe default config: %+v", memoryCfg)
	}
	if memoryCfg.RecipeDefaultGrayPercent != 25 || memoryCfg.RecipeMinSelectionConfidence != 0.68 || memoryCfg.RecipeMinSuccessRate != 0.64 {
		t.Fatalf("unexpected memory recipe thresholds: %+v", memoryCfg)
	}
	if memoryCfg.RecipeBackfillBatchSize != 17 || memoryCfg.RecipeBackfillInterval != 45*time.Minute {
		t.Fatalf("unexpected memory recipe backfill settings: %+v", memoryCfg)
	}
}

func TestLoadConfigWithRuntime_LoadsDecisionRecipeReuseSettingsFromFile(t *testing.T) {
	configPath := t.TempDir() + "/config.toml"
	t.Setenv("GHOST_CONFIG_PATH", configPath)

	recipeReuseEnabled := true
	recipeExecutionTrackingEnabled := false
	recipeBackfillEnabled := true
	recipeDefaultEnabled := false
	recipeDefaultGrayPercent := 40
	recipeMinSelectionConfidence := 0.74
	recipeMinSuccessRate := 0.69
	recipeBackfillBatchSize := 23
	recipeBackfillInterval := "90m"
	if err := writeBridgeFileConfig(configPathFromEnv(), bridgeFileConfig{
		MemoryDecisionRecipeReuseEnabled:             &recipeReuseEnabled,
		MemoryDecisionRecipeExecutionTrackingEnabled: &recipeExecutionTrackingEnabled,
		MemoryDecisionRecipeBackfillEnabled:          &recipeBackfillEnabled,
		MemoryDecisionRecipeDefaultEnabled:           &recipeDefaultEnabled,
		MemoryDecisionRecipeDefaultGrayPercent:       &recipeDefaultGrayPercent,
		MemoryDecisionRecipeMinSelectionConfidence:   &recipeMinSelectionConfidence,
		MemoryDecisionRecipeMinSuccessRate:           &recipeMinSuccessRate,
		MemoryDecisionRecipeBackfillBatchSize:        &recipeBackfillBatchSize,
		MemoryDecisionRecipeBackfillInterval:         &recipeBackfillInterval,
	}); err != nil {
		t.Fatalf("writeBridgeFileConfig: %v", err)
	}

	cfg, err := loadConfigWithRuntime(runtimeConfig{
		Provider: llm.ProviderCustom,
		BaseURL:  "https://example.com/v1",
		Model:    "gpt-4o",
	})
	if err != nil {
		t.Fatalf("loadConfigWithRuntime returned error: %v", err)
	}

	if !cfg.MemoryDecisionRecipeReuseEnabledSet || !cfg.MemoryDecisionRecipeReuseEnabled {
		t.Fatalf("unexpected recipe reuse toggle from file: enabled=%v set=%v", cfg.MemoryDecisionRecipeReuseEnabled, cfg.MemoryDecisionRecipeReuseEnabledSet)
	}
	if !cfg.MemoryDecisionRecipeExecutionTrackingEnabledSet || cfg.MemoryDecisionRecipeExecutionTrackingEnabled {
		t.Fatalf("unexpected execution tracking toggle from file: enabled=%v set=%v", cfg.MemoryDecisionRecipeExecutionTrackingEnabled, cfg.MemoryDecisionRecipeExecutionTrackingEnabledSet)
	}
	if !cfg.MemoryDecisionRecipeBackfillEnabledSet || !cfg.MemoryDecisionRecipeBackfillEnabled {
		t.Fatalf("unexpected backfill toggle from file: enabled=%v set=%v", cfg.MemoryDecisionRecipeBackfillEnabled, cfg.MemoryDecisionRecipeBackfillEnabledSet)
	}
	if !cfg.MemoryDecisionRecipeDefaultEnabledSet || cfg.MemoryDecisionRecipeDefaultEnabled {
		t.Fatalf("unexpected default toggle from file: enabled=%v set=%v", cfg.MemoryDecisionRecipeDefaultEnabled, cfg.MemoryDecisionRecipeDefaultEnabledSet)
	}
	if cfg.MemoryDecisionRecipeDefaultGrayPercent != 40 {
		t.Fatalf("unexpected gray percent from file: got %d want %d", cfg.MemoryDecisionRecipeDefaultGrayPercent, 40)
	}
	if cfg.MemoryDecisionRecipeMinSelectionConfidence != 0.74 || cfg.MemoryDecisionRecipeMinSuccessRate != 0.69 {
		t.Fatalf("unexpected thresholds from file: selection=%v success=%v", cfg.MemoryDecisionRecipeMinSelectionConfidence, cfg.MemoryDecisionRecipeMinSuccessRate)
	}
	if cfg.MemoryDecisionRecipeBackfillBatchSize != 23 {
		t.Fatalf("unexpected batch size from file: got %d want %d", cfg.MemoryDecisionRecipeBackfillBatchSize, 23)
	}
	if cfg.MemoryDecisionRecipeBackfillInterval != 90*time.Minute {
		t.Fatalf("unexpected backfill interval from file: got %s want %s", cfg.MemoryDecisionRecipeBackfillInterval, 90*time.Minute)
	}
}
