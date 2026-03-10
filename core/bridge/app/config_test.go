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
	t.Setenv("GHOST_TOOL_ALLOWLIST", "search_files, read_file")
	t.Setenv("GHOST_TOOL_BLOCKLIST", "bash_exec, ask_human")

	cfg, err := loadConfigWithRuntime(runtimeConfig{
		Provider: llm.ProviderCustom,
		BaseURL:  "https://example.com/v1",
		Model:    "gpt-4o",
	})
	if err != nil {
		t.Fatalf("loadConfigWithRuntime returned error: %v", err)
	}

	if !cfg.ToolSelector.Enabled {
		t.Fatal("expected ToolSelectorEnabled to be true")
	}
	if cfg.ToolSelector.Mode != "llm" {
		t.Fatalf("unexpected ToolSelectorMode: %q", cfg.ToolSelector.Mode)
	}
	if cfg.ToolSelector.Model != "gpt-4o-mini" {
		t.Fatalf("unexpected ToolSelectorModel: %q", cfg.ToolSelector.Model)
	}
	if cfg.ToolSelector.TimeoutMS != 900 {
		t.Fatalf("unexpected ToolSelectorTimeoutMS: %d", cfg.ToolSelector.TimeoutMS)
	}
	if cfg.ToolSelector.Confidence != 0.88 {
		t.Fatalf("unexpected ToolSelectorConfidence: %v", cfg.ToolSelector.Confidence)
	}
	if !cfg.ToolSelector.Shadow {
		t.Fatal("expected ToolSelectorShadow to be true")
	}
	if cfg.ToolSelector.RecentMsgs != 4 {
		t.Fatalf("unexpected ToolSelectorRecentMsgs: %d", cfg.ToolSelector.RecentMsgs)
	}
	if got, want := cfg.ToolSelector.Allowlist, []string{"read_file", "search_files"}; len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("unexpected ToolAllowlist: got %v want %v", got, want)
	}
	if got, want := cfg.ToolSelector.Blocklist, []string{"bash_exec"}; len(got) != len(want) || got[0] != want[0] {
		t.Fatalf("unexpected ToolBlocklist: got %v want %v", got, want)
	}
}

func TestInferProviderTypeDetectsCodexModel(t *testing.T) {
	if got := inferProviderType("", "", "codex-mini-latest"); got != llm.ProviderCodex {
		t.Fatalf("unexpected provider type: got %q want %q", got, llm.ProviderCodex)
	}
}

func TestValidateRuntimeForExecutionRequiresAPIKeyForCodex(t *testing.T) {
	err := validateRuntimeForExecution(runtimeConfig{
		Provider: llm.ProviderCodex,
		BaseURL:  "https://api.openai.com/v1",
		Model:    "codex-mini-latest",
	})
	if err == nil {
		t.Fatal("expected codex api key validation error")
	}
}

func TestLoadConfigWithRuntime_RejectsToolListOverlap(t *testing.T) {
	t.Setenv("GHOST_CONFIG_PATH", t.TempDir()+"/config.toml")
	t.Setenv("GHOST_TOOL_ALLOWLIST", "read_file")
	t.Setenv("GHOST_TOOL_BLOCKLIST", "read_file")

	_, err := loadConfigWithRuntime(runtimeConfig{
		Provider: llm.ProviderCustom,
		BaseURL:  "https://example.com/v1",
		Model:    "gpt-4o",
	})
	if err == nil {
		t.Fatal("expected overlap error")
	}
}

func TestLoadConfigWithRuntime_RejectsUnknownToolInList(t *testing.T) {
	t.Setenv("GHOST_CONFIG_PATH", t.TempDir()+"/config.toml")
	t.Setenv("GHOST_TOOL_ALLOWLIST", "ghost_tool")

	_, err := loadConfigWithRuntime(runtimeConfig{
		Provider: llm.ProviderCustom,
		BaseURL:  "https://example.com/v1",
		Model:    "gpt-4o",
	})
	if err == nil {
		t.Fatal("expected unknown tool error")
	}
}

func TestLoadConfigWithRuntime_LoadsWebSearchTavilyAPIKeyFromEnv(t *testing.T) {
	t.Setenv("GHOST_CONFIG_PATH", t.TempDir()+"/config.toml")
	t.Setenv("GHOST_WEB_SEARCH_TAVILY_API_KEY", "env-tavily-key")

	cfg, err := loadConfigWithRuntime(runtimeConfig{
		Provider: llm.ProviderCustom,
		BaseURL:  "https://example.com/v1",
		Model:    "gpt-4o",
	})
	if err != nil {
		t.Fatalf("loadConfigWithRuntime returned error: %v", err)
	}
	if cfg.WebSearchTavilyAPIKey != "env-tavily-key" {
		t.Fatalf("unexpected tavily api key: got %q want %q", cfg.WebSearchTavilyAPIKey, "env-tavily-key")
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

	if !cfg.Memory.DecisionEnabled {
		t.Fatalf("expected decision memory to be enabled")
	}
	if cfg.Memory.DecisionCaptureOnTurn {
		t.Fatalf("expected decision capture on turn to be disabled")
	}
	if cfg.Memory.DecisionPath != "/tmp/decision" {
		t.Fatalf("unexpected decision path: got %q want %q", cfg.Memory.DecisionPath, "/tmp/decision")
	}
	if cfg.Memory.DecisionMaxHits != 6 {
		t.Fatalf("unexpected decision max hits: got %d want %d", cfg.Memory.DecisionMaxHits, 6)
	}
	if cfg.Memory.DecisionMinConfidence != 0.81 {
		t.Fatalf("unexpected decision min confidence: got %v want %v", cfg.Memory.DecisionMinConfidence, 0.81)
	}
	if cfg.Memory.DecisionMinReuseScore != 0.77 {
		t.Fatalf("unexpected decision min reuse score: got %v want %v", cfg.Memory.DecisionMinReuseScore, 0.77)
	}
	if cfg.Memory.DecisionRecipeEnabled {
		t.Fatalf("expected decision recipe toggle to be false")
	}
	if cfg.Memory.DecisionRecipeInterval != 12*time.Hour {
		t.Fatalf("unexpected decision recipe interval: got %s want %s", cfg.Memory.DecisionRecipeInterval, 12*time.Hour)
	}
	if cfg.Memory.DecisionRecipeMinSupport != 5 {
		t.Fatalf("unexpected decision recipe min support: got %d want %d", cfg.Memory.DecisionRecipeMinSupport, 5)
	}
	if !cfg.Memory.DecisionDebugEnabled {
		t.Fatalf("expected decision debug to be enabled")
	}
	if cfg.Memory.DecisionSelectorHintEnabled {
		t.Fatalf("expected selector hint toggle to be disabled")
	}
	if cfg.RSS.FeedsPath != defaultRSSFeedsPath {
		t.Fatalf("unexpected rss feeds path: got %q want %q", cfg.RSS.FeedsPath, defaultRSSFeedsPath)
	}
}

func TestLoadConfigWithRuntime_LoadsRSSFeedsPath(t *testing.T) {
	t.Setenv("GHOST_CONFIG_PATH", t.TempDir()+"/config.toml")
	t.Setenv("GHOST_RSS_FEEDS_PATH", "/tmp/rss-feeds.json")
	t.Setenv("GHOST_RSS_INBOX_PATH", "/tmp/rss-inbox.json")
	t.Setenv("GHOST_RSS_POLL_ENABLED", "false")
	t.Setenv("GHOST_RSS_POLL_INTERVAL", "10m")
	t.Setenv("GHOST_RSS_POLL_MAX_ITEMS_PER_FEED", "15")
	t.Setenv("GHOST_RSS_AI_BATCH_SIZE", "7")

	cfg, err := loadConfigWithRuntime(runtimeConfig{
		Provider: llm.ProviderCustom,
		BaseURL:  "https://example.com/v1",
		Model:    "gpt-4o",
	})
	if err != nil {
		t.Fatalf("loadConfigWithRuntime returned error: %v", err)
	}
	if cfg.RSS.FeedsPath != "/tmp/rss-feeds.json" {
		t.Fatalf("unexpected rss feeds path: got %q want %q", cfg.RSS.FeedsPath, "/tmp/rss-feeds.json")
	}
	if cfg.RSS.InboxPath != "/tmp/rss-inbox.json" {
		t.Fatalf("unexpected rss inbox path: got %q want %q", cfg.RSS.InboxPath, "/tmp/rss-inbox.json")
	}
	if cfg.RSS.PollEnabled {
		t.Fatal("expected rss poll to be disabled")
	}
	if cfg.RSS.PollInterval != 10*time.Minute {
		t.Fatalf("unexpected rss poll interval: got %s want %s", cfg.RSS.PollInterval, 10*time.Minute)
	}
	if cfg.RSS.PollMaxItemsPerFeed != 15 {
		t.Fatalf("unexpected rss max items: got %d want %d", cfg.RSS.PollMaxItemsPerFeed, 15)
	}
	if cfg.RSS.AIBatchSize != 7 {
		t.Fatalf("unexpected rss ai batch size: got %d want %d", cfg.RSS.AIBatchSize, 7)
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
	if !cfg.Memory.TemporalDecayEnabled {
		t.Fatal("expected MemoryTemporalDecayEnabled to be true")
	}
	if cfg.Memory.TemporalDecayHalfLife.Hours() != 120 {
		t.Fatalf("unexpected MemoryTemporalDecayHalfLife: %s", cfg.Memory.TemporalDecayHalfLife)
	}
	if !cfg.Memory.AnchorEnabled {
		t.Fatal("expected MemoryAnchorEnabled to be true")
	}
	if cfg.Memory.AnchorMinWeight != 0.82 {
		t.Fatalf("unexpected MemoryAnchorMinWeight: %v", cfg.Memory.AnchorMinWeight)
	}
	if cfg.Memory.EvolutionUseWorker {
		t.Fatal("expected MemoryEvolutionUseWorker to be false")
	}
	if cfg.Memory.EvolutionBatchSize != 9 {
		t.Fatalf("unexpected MemoryEvolutionBatchSize: %d", cfg.Memory.EvolutionBatchSize)
	}
	if !cfg.Memory.GraphEnabled {
		t.Fatal("expected MemoryGraphEnabled to be true")
	}
	if cfg.Memory.GraphPath != "/tmp/graph" {
		t.Fatalf("unexpected MemoryGraphPath: %q", cfg.Memory.GraphPath)
	}
	if !cfg.Memory.GraphExtractOnArchive || cfg.Memory.GraphExtractOnEvolve {
		t.Fatalf("unexpected graph extraction flags: archive=%v evolve=%v", cfg.Memory.GraphExtractOnArchive, cfg.Memory.GraphExtractOnEvolve)
	}
	if cfg.Memory.GraphMaxHops != 2 || cfg.Memory.GraphMaxHits != 7 {
		t.Fatalf("unexpected graph hop/hit limits: hops=%d hits=%d", cfg.Memory.GraphMaxHops, cfg.Memory.GraphMaxHits)
	}
	if cfg.Memory.GraphMinConfidence != 0.8 {
		t.Fatalf("unexpected MemoryGraphMinConfidence: %v", cfg.Memory.GraphMinConfidence)
	}
	if cfg.Memory.GraphNamespace != "workspace:test" {
		t.Fatalf("unexpected MemoryGraphNamespace: %q", cfg.Memory.GraphNamespace)
	}
	if !cfg.Memory.GraphDebugEnabled {
		t.Fatal("expected MemoryGraphDebugEnabled to be true")
	}
}
