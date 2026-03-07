package app

import (
	"testing"
	"time"

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

func TestLoadConfigWithRuntime_LoadsMemoryDecisionSettings(t *testing.T) {
	t.Setenv("GHOST_CONFIG_PATH", t.TempDir()+"/config.toml")
	t.Setenv("GHOST_MEMORY_DECISION_ENABLED", "true")
	t.Setenv("GHOST_MEMORY_DECISION_CAPTURE_ON_TURN", "false")
	t.Setenv("GHOST_MEMORY_DECISION_PATH", "/tmp/decision")
	t.Setenv("GHOST_MEMORY_DECISION_MAX_HITS", "6")
	t.Setenv("GHOST_MEMORY_DECISION_MIN_CONFIDENCE", "0.81")
	t.Setenv("GHOST_MEMORY_DECISION_MIN_REUSE_SCORE", "0.77")
	t.Setenv("GHOST_MEMORY_DECISION_RECIPE_ENABLED", "false")
	t.Setenv("GHOST_MEMORY_DECISION_RECIPE_INTERVAL", "12h")
	t.Setenv("GHOST_MEMORY_DECISION_RECIPE_MIN_SUPPORT", "5")
	t.Setenv("GHOST_MEMORY_DECISION_DEBUG_ENABLED", "true")
	t.Setenv("GHOST_MEMORY_DECISION_SELECTOR_HINT_ENABLED", "false")

	cfg, err := loadConfigWithRuntime(runtimeConfig{
		Provider: llm.ProviderCustom,
		BaseURL:  "https://example.com/v1",
		Model:    "gpt-4o",
	})
	if err != nil {
		t.Fatalf("loadConfigWithRuntime returned error: %v", err)
	}

	if !cfg.MemoryDecisionEnabled {
		t.Fatalf("expected decision memory to be enabled")
	}
	if cfg.MemoryDecisionCaptureOnTurn {
		t.Fatalf("expected decision capture on turn to be disabled")
	}
	if cfg.MemoryDecisionPath != "/tmp/decision" {
		t.Fatalf("unexpected decision path: got %q want %q", cfg.MemoryDecisionPath, "/tmp/decision")
	}
	if cfg.MemoryDecisionMaxHits != 6 {
		t.Fatalf("unexpected decision max hits: got %d want %d", cfg.MemoryDecisionMaxHits, 6)
	}
	if cfg.MemoryDecisionMinConfidence != 0.81 {
		t.Fatalf("unexpected decision min confidence: got %v want %v", cfg.MemoryDecisionMinConfidence, 0.81)
	}
	if cfg.MemoryDecisionMinReuseScore != 0.77 {
		t.Fatalf("unexpected decision min reuse score: got %v want %v", cfg.MemoryDecisionMinReuseScore, 0.77)
	}
	if cfg.MemoryDecisionRecipeEnabled {
		t.Fatalf("expected decision recipe toggle to be false")
	}
	if cfg.MemoryDecisionRecipeInterval != 12*time.Hour {
		t.Fatalf("unexpected decision recipe interval: got %s want %s", cfg.MemoryDecisionRecipeInterval, 12*time.Hour)
	}
	if cfg.MemoryDecisionRecipeMinSupport != 5 {
		t.Fatalf("unexpected decision recipe min support: got %d want %d", cfg.MemoryDecisionRecipeMinSupport, 5)
	}
	if !cfg.MemoryDecisionDebugEnabled {
		t.Fatalf("expected decision debug to be enabled")
	}
	if cfg.MemoryDecisionSelectorHintEnabled {
		t.Fatalf("expected selector hint toggle to be disabled")
	}
}

func TestLoadConfigWithRuntime_LoadsMemoryEnhancementSettings(t *testing.T) {
	t.Setenv("GHOST_CONFIG_PATH", t.TempDir()+"/config.toml")
	t.Setenv("GHOST_MEMORY_TEMPORAL_DECAY_ENABLED", "true")
	t.Setenv("GHOST_MEMORY_TEMPORAL_DECAY_HALF_LIFE", "120h")
	t.Setenv("GHOST_MEMORY_ANCHOR_ENABLED", "true")
	t.Setenv("GHOST_MEMORY_ANCHOR_MIN_WEIGHT", "0.82")
	t.Setenv("GHOST_MEMORY_EVOLUTION_USE_WORKER", "false")
	t.Setenv("GHOST_MEMORY_EVOLUTION_BATCH_SIZE", "9")
	t.Setenv("GHOST_MEMORY_GRAPH_ENABLED", "true")
	t.Setenv("GHOST_MEMORY_GRAPH_PATH", "/tmp/graph")
	t.Setenv("GHOST_MEMORY_GRAPH_EXTRACT_ON_ARCHIVE", "true")
	t.Setenv("GHOST_MEMORY_GRAPH_EXTRACT_ON_EVOLVE", "false")
	t.Setenv("GHOST_MEMORY_GRAPH_MAX_HOPS", "2")
	t.Setenv("GHOST_MEMORY_GRAPH_MAX_HITS", "7")
	t.Setenv("GHOST_MEMORY_GRAPH_MIN_CONFIDENCE", "0.8")
	t.Setenv("GHOST_MEMORY_GRAPH_NAMESPACE", "workspace:test")
	t.Setenv("GHOST_MEMORY_GRAPH_DEBUG_ENABLED", "true")

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
	if !cfg.MemoryGraphEnabled {
		t.Fatal("expected MemoryGraphEnabled to be true")
	}
	if cfg.MemoryGraphPath != "/tmp/graph" {
		t.Fatalf("unexpected MemoryGraphPath: %q", cfg.MemoryGraphPath)
	}
	if !cfg.MemoryGraphExtractOnArchive || cfg.MemoryGraphExtractOnEvolve {
		t.Fatalf("unexpected graph extraction flags: archive=%v evolve=%v", cfg.MemoryGraphExtractOnArchive, cfg.MemoryGraphExtractOnEvolve)
	}
	if cfg.MemoryGraphMaxHops != 2 || cfg.MemoryGraphMaxHits != 7 {
		t.Fatalf("unexpected graph hop/hit limits: hops=%d hits=%d", cfg.MemoryGraphMaxHops, cfg.MemoryGraphMaxHits)
	}
	if cfg.MemoryGraphMinConfidence != 0.8 {
		t.Fatalf("unexpected MemoryGraphMinConfidence: %v", cfg.MemoryGraphMinConfidence)
	}
	if cfg.MemoryGraphNamespace != "workspace:test" {
		t.Fatalf("unexpected MemoryGraphNamespace: %q", cfg.MemoryGraphNamespace)
	}
	if !cfg.MemoryGraphDebugEnabled {
		t.Fatal("expected MemoryGraphDebugEnabled to be true")
	}
}
