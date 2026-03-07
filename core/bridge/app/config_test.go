package app

import (
	"testing"

	"ghost-os/bridge/llm"
)

func TestLoadConfigWithRuntime_LoadsToolSelectorSettings(t *testing.T) {
	t.Setenv("GHOST_CONFIG_PATH", t.TempDir()+"/config.toml")
	t.Setenv("GHOST_TOOL_SELECTOR_ENABLED", "true")
	t.Setenv("GHOST_TOOL_SELECTOR_MODE", "llm")
	t.Setenv("GHOST_TOOL_SELECTOR_MODEL", "gpt-4o-mini")
	t.Setenv("GHOST_TOOL_SELECTOR_TIMEOUT_MS", "900")
	t.Setenv("GHOST_TOOL_SELECTOR_CONFIDENCE", "0.88")
	t.Setenv("GHOST_TOOL_SELECTOR_SHADOW", "true")
	t.Setenv("GHOST_TOOL_SELECTOR_RECENT_MESSAGES", "4")

	cfg, err := loadConfigWithRuntime(runtimeConfig{
		Provider: llm.ProviderCustom,
		BaseURL:  "https://example.com/v1",
		Model:    "gpt-4o",
	})
	if err != nil {
		t.Fatalf("loadConfigWithRuntime returned error: %v", err)
	}

	if !cfg.ToolSelectorEnabled {
		t.Fatal("expected ToolSelectorEnabled to be true")
	}
	if cfg.ToolSelectorMode != "llm" {
		t.Fatalf("unexpected ToolSelectorMode: %q", cfg.ToolSelectorMode)
	}
	if cfg.ToolSelectorModel != "gpt-4o-mini" {
		t.Fatalf("unexpected ToolSelectorModel: %q", cfg.ToolSelectorModel)
	}
	if cfg.ToolSelectorTimeoutMS != 900 {
		t.Fatalf("unexpected ToolSelectorTimeoutMS: %d", cfg.ToolSelectorTimeoutMS)
	}
	if cfg.ToolSelectorConfidence != 0.88 {
		t.Fatalf("unexpected ToolSelectorConfidence: %v", cfg.ToolSelectorConfidence)
	}
	if !cfg.ToolSelectorShadow {
		t.Fatal("expected ToolSelectorShadow to be true")
	}
	if cfg.ToolSelectorRecentMsgs != 4 {
		t.Fatalf("unexpected ToolSelectorRecentMsgs: %d", cfg.ToolSelectorRecentMsgs)
	}
}

func TestParseFloatEnv_FallsBackOnInvalidValues(t *testing.T) {
	for _, raw := range []string{"", "bad", "-1", "2"} {
		t.Run(raw, func(t *testing.T) {
			t.Setenv("GHOST_TOOL_SELECTOR_CONFIDENCE", raw)
			if got := parseFloatEnv("GHOST_TOOL_SELECTOR_CONFIDENCE", 0.75); got != 0.75 {
				t.Fatalf("expected fallback 0.75, got %v", got)
			}
		})
	}
}

func TestLoadConfigWithRuntime_LoadsMemoryEnhancementSettings(t *testing.T) {
	t.Setenv("GHOST_MEMORY_TEMPORAL_DECAY_ENABLED", "true")
	t.Setenv("GHOST_MEMORY_TEMPORAL_DECAY_HALF_LIFE", "120h")
	t.Setenv("GHOST_MEMORY_ANCHOR_ENABLED", "true")
	t.Setenv("GHOST_MEMORY_ANCHOR_MIN_WEIGHT", "0.82")
	t.Setenv("GHOST_MEMORY_EVOLUTION_USE_WORKER", "false")
	t.Setenv("GHOST_MEMORY_EVOLUTION_BATCH_SIZE", "9")

	cfg, err := loadConfigWithRuntime(runtimeConfig{
		Provider: llm.ProviderCustom,
		BaseURL:  "https://example.com/v1",
		Model:    "gpt-4o",
	})
	if err != nil {
		t.Fatalf("loadConfigWithRuntime returned error: %v", err)
	}
	if !cfg.MemoryTemporalDecayEnabled {
		t.Fatal("expected MemoryTemporalDecayEnabled to be true")
	}
	if cfg.MemoryTemporalDecayHalfLife.Hours() != 120 {
		t.Fatalf("unexpected MemoryTemporalDecayHalfLife: %s", cfg.MemoryTemporalDecayHalfLife)
	}
	if !cfg.MemoryAnchorEnabled {
		t.Fatal("expected MemoryAnchorEnabled to be true")
	}
	if cfg.MemoryAnchorMinWeight != 0.82 {
		t.Fatalf("unexpected MemoryAnchorMinWeight: %v", cfg.MemoryAnchorMinWeight)
	}
	if cfg.MemoryEvolutionUseWorker {
		t.Fatal("expected MemoryEvolutionUseWorker to be false")
	}
	if cfg.MemoryEvolutionBatchSize != 9 {
		t.Fatalf("unexpected MemoryEvolutionBatchSize: %d", cfg.MemoryEvolutionBatchSize)
	}
}
