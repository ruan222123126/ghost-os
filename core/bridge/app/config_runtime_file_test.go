package app

import (
	"path/filepath"
	"testing"
	"time"

	"ghost-os/bridge/llm"
)

func TestRuntimeConfigFromEnvPrefersMultiProviderConfigFile(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.toml")
	t.Setenv("GHOST_CONFIG_PATH", configPath)
	t.Setenv("GHOST_PROVIDER", "openai")
	t.Setenv("GHOST_API_KEY", "env-key")
	t.Setenv("GHOST_BASE_URL", "https://env.example/v1")
	t.Setenv("GHOST_MODEL", "env-model")
	t.Setenv("GHOST_CHAT_PATH", "/env/chat")

	modelProvider := "crs"
	model := "gpt-5.4"
	chatPath := "/v1/chat/completions"
	nativePersistent := true
	if err := writeBridgeFileConfig(configPathFromEnv(), bridgeFileConfig{
		ActiveProvider:   &modelProvider,
		Model:            &model,
		ChatPath:         &chatPath,
		NativePersistent: &nativePersistent,
		Providers: map[string]providerFileConfig{
			"crs": {
				Type:    llm.ProviderCustom,
				BaseURL: "https://lldai.online/openai",
				APIKey:  optionalStringPointer("file-key"),
				Models:  []string{"gpt-5.4", "gpt-4"},
			},
		},
	}); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	runtime, err := runtimeConfigFromEnv()
	if err != nil {
		t.Fatalf("runtimeConfigFromEnv: %v", err)
	}
	if runtime.ProviderName != "crs" {
		t.Fatalf("unexpected provider name: got %q want %q", runtime.ProviderName, "crs")
	}
	if runtime.APIKey != "file-key" {
		t.Fatalf("unexpected api key: got %q want %q", runtime.APIKey, "file-key")
	}
	if runtime.BaseURL != "https://lldai.online/openai" {
		t.Fatalf("unexpected base url: got %q", runtime.BaseURL)
	}
	if runtime.Model != model {
		t.Fatalf("unexpected model: got %q want %q", runtime.Model, model)
	}
	if runtime.ChatPath != chatPath {
		t.Fatalf("unexpected chat path: got %q want %q", runtime.ChatPath, chatPath)
	}
	if !runtime.NativePersistent {
		t.Fatalf("expected native persistent to be true")
	}
}

func TestLoadConfigReadsStaticFieldsFromTomlConfig(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.toml")
	t.Setenv("GHOST_CONFIG_PATH", configPath)

	modelProvider := "openai"
	model := "gpt-4o-mini"
	workerModel := "gpt-4o-mini"
	promptsPath := "/tmp/prompts.yaml"
	sessionsPath := "/tmp/sessions"
	rssFeedsPath := "/tmp/rss/feeds.json"
	rssInboxPath := "/tmp/rss/inbox.json"
	rssPollEnabled := false
	rssPollInterval := "20m"
	rssPollMaxItemsPerFeed := 12
	rssAIBatchSize := 6
	memoryWarmPath := "/tmp/warm.json"
	memoryColdPath := "/tmp/cold"
	memoryGraphPath := "/tmp/graph"
	memoryWarmTTL := "48h"
	memoryTemporalHalfLife := "96h"
	memoryTemporalDecayEnabled := true
	memoryAnchorEnabled := true
	memoryAnchorMinWeight := 0.75
	memoryEvolutionInterval := "2h"
	memoryEvolutionUseWorker := true
	memoryEvolutionBatchSize := 12
	memoryGraphEnabled := true
	memoryGraphExtractOnArchive := true
	memoryGraphExtractOnEvolve := false
	memoryGraphMaxHops := 2
	memoryGraphMaxHits := 7
	memoryGraphMinConfidence := 0.8
	memoryGraphNamespace := "workspace:file"
	memoryGraphDebugEnabled := true
	memoryDecisionEnabled := true
	memoryDecisionCaptureOnTurn := false
	memoryDecisionPath := "/tmp/decision"
	memoryDecisionMaxHits := 4
	memoryDecisionMinConfidence := 0.79
	memoryDecisionMinReuseScore := 0.74
	memoryDecisionRecipeEnabled := true
	memoryDecisionRecipeInterval := "6h"
	memoryDecisionRecipeMinSupport := 5
	memoryDecisionDebugEnabled := true
	memoryDecisionSelectorHintEnabled := false
	maxTurns := 42
	workerMaxFiles := 8
	toolSelectorEnabled := true
	toolSelectorMode := "rules"
	bindAddr := "0.0.0.0:9090"
	apiToken := "secret-token"
	if err := writeBridgeFileConfig(configPathFromEnv(), bridgeFileConfig{
		ActiveProvider: &modelProvider,
		Model:          &model,
		Providers: map[string]providerFileConfig{
			"openai": {
				Type:    llm.ProviderOpenAI,
				BaseURL: defaultBaseURL,
				APIKey:  optionalStringPointer("file-key"),
			},
		},
		WorkerModel:                       &workerModel,
		PromptsPath:                       &promptsPath,
		SessionsPath:                      &sessionsPath,
		RSSFeedsPath:                      &rssFeedsPath,
		RSSInboxPath:                      &rssInboxPath,
		RSSPollEnabled:                    &rssPollEnabled,
		RSSPollInterval:                   &rssPollInterval,
		RSSPollMaxItemsPerFeed:            &rssPollMaxItemsPerFeed,
		RSSAIBatchSize:                    &rssAIBatchSize,
		MemoryWarmPath:                    &memoryWarmPath,
		MemoryColdPath:                    &memoryColdPath,
		MemoryGraphPath:                   &memoryGraphPath,
		MemoryWarmTTL:                     &memoryWarmTTL,
		MemoryTemporalDecayEnabled:        &memoryTemporalDecayEnabled,
		MemoryTemporalDecayHalfLife:       &memoryTemporalHalfLife,
		MemoryAnchorEnabled:               &memoryAnchorEnabled,
		MemoryAnchorMinWeight:             &memoryAnchorMinWeight,
		MemoryEvolutionInterval:           &memoryEvolutionInterval,
		MemoryEvolutionUseWorker:          &memoryEvolutionUseWorker,
		MemoryEvolutionBatchSize:          &memoryEvolutionBatchSize,
		MemoryGraphEnabled:                &memoryGraphEnabled,
		MemoryGraphExtractOnArchive:       &memoryGraphExtractOnArchive,
		MemoryGraphExtractOnEvolve:        &memoryGraphExtractOnEvolve,
		MemoryGraphMaxHops:                &memoryGraphMaxHops,
		MemoryGraphMaxHits:                &memoryGraphMaxHits,
		MemoryGraphMinConfidence:          &memoryGraphMinConfidence,
		MemoryGraphNamespace:              &memoryGraphNamespace,
		MemoryGraphDebugEnabled:           &memoryGraphDebugEnabled,
		MemoryDecisionEnabled:             &memoryDecisionEnabled,
		MemoryDecisionCaptureOnTurn:       &memoryDecisionCaptureOnTurn,
		MemoryDecisionPath:                &memoryDecisionPath,
		MemoryDecisionMaxHits:             &memoryDecisionMaxHits,
		MemoryDecisionMinConfidence:       &memoryDecisionMinConfidence,
		MemoryDecisionMinReuseScore:       &memoryDecisionMinReuseScore,
		MemoryDecisionRecipeEnabled:       &memoryDecisionRecipeEnabled,
		MemoryDecisionRecipeInterval:      &memoryDecisionRecipeInterval,
		MemoryDecisionRecipeMinSupport:    &memoryDecisionRecipeMinSupport,
		MemoryDecisionDebugEnabled:        &memoryDecisionDebugEnabled,
		MemoryDecisionSelectorHintEnabled: &memoryDecisionSelectorHintEnabled,
		MaxTurns:                          &maxTurns,
		WorkerMaxFiles:                    &workerMaxFiles,
		ToolSelectorEnabled:               &toolSelectorEnabled,
		ToolSelectorMode:                  &toolSelectorMode,
		BindAddr:                          &bindAddr,
		APIToken:                          &apiToken,
		CORSOrigins:                       []string{"http://localhost:5173"},
	}); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.WorkerModel != workerModel {
		t.Fatalf("unexpected worker model: got %q want %q", cfg.WorkerModel, workerModel)
	}
	if cfg.PromptsPath != promptsPath {
		t.Fatalf("unexpected prompts path: got %q want %q", cfg.PromptsPath, promptsPath)
	}
	if cfg.SessionsPath != sessionsPath {
		t.Fatalf("unexpected sessions path: got %q want %q", cfg.SessionsPath, sessionsPath)
	}
	if cfg.RSSFeedsPath != rssFeedsPath {
		t.Fatalf("unexpected rss feeds path: got %q want %q", cfg.RSSFeedsPath, rssFeedsPath)
	}
	if cfg.RSSInboxPath != rssInboxPath {
		t.Fatalf("unexpected rss inbox path: got %q want %q", cfg.RSSInboxPath, rssInboxPath)
	}
	if cfg.RSSPollEnabled {
		t.Fatalf("expected rss poll to be disabled")
	}
	if cfg.RSSPollInterval != 20*time.Minute {
		t.Fatalf("unexpected rss poll interval: got %s want %s", cfg.RSSPollInterval, 20*time.Minute)
	}
	if cfg.RSSPollMaxItemsPerFeed != rssPollMaxItemsPerFeed {
		t.Fatalf("unexpected rss poll max items: got %d want %d", cfg.RSSPollMaxItemsPerFeed, rssPollMaxItemsPerFeed)
	}
	if cfg.RSSAIBatchSize != rssAIBatchSize {
		t.Fatalf("unexpected rss ai batch size: got %d want %d", cfg.RSSAIBatchSize, rssAIBatchSize)
	}
	if cfg.MemoryWarmPath != memoryWarmPath {
		t.Fatalf("unexpected warm path: got %q want %q", cfg.MemoryWarmPath, memoryWarmPath)
	}
	if cfg.MemoryColdPath != memoryColdPath {
		t.Fatalf("unexpected cold path: got %q want %q", cfg.MemoryColdPath, memoryColdPath)
	}
	if cfg.MemoryGraphPath != memoryGraphPath {
		t.Fatalf("unexpected graph path: got %q want %q", cfg.MemoryGraphPath, memoryGraphPath)
	}
	if cfg.MemoryWarmTTL.Hours() != 48 {
		t.Fatalf("unexpected warm ttl: got %s want 48h", cfg.MemoryWarmTTL)
	}
	if !cfg.MemoryTemporalDecayEnabled {
		t.Fatalf("expected temporal decay to be enabled")
	}
	if cfg.MemoryTemporalDecayHalfLife.Hours() != 96 {
		t.Fatalf("unexpected temporal half life: got %s want 96h", cfg.MemoryTemporalDecayHalfLife)
	}
	if !cfg.MemoryAnchorEnabled {
		t.Fatalf("expected anchor extraction to be enabled")
	}
	if cfg.MemoryAnchorMinWeight != memoryAnchorMinWeight {
		t.Fatalf("unexpected anchor min weight: got %v want %v", cfg.MemoryAnchorMinWeight, memoryAnchorMinWeight)
	}
	if cfg.MemoryEvolutionInterval.Hours() != 2 {
		t.Fatalf("unexpected evolution interval: got %s want 2h", cfg.MemoryEvolutionInterval)
	}
	if !cfg.MemoryEvolutionUseWorker {
		t.Fatalf("expected evolution worker usage to be enabled")
	}
	if cfg.MemoryEvolutionBatchSize != memoryEvolutionBatchSize {
		t.Fatalf("unexpected evolution batch size: got %d want %d", cfg.MemoryEvolutionBatchSize, memoryEvolutionBatchSize)
	}
	if !cfg.MemoryGraphEnabled {
		t.Fatalf("expected graph memory to be enabled")
	}
	if !cfg.MemoryGraphExtractOnArchive || cfg.MemoryGraphExtractOnEvolve {
		t.Fatalf("unexpected graph extraction flags: archive=%v evolve=%v", cfg.MemoryGraphExtractOnArchive, cfg.MemoryGraphExtractOnEvolve)
	}
	if cfg.MemoryGraphMaxHops != memoryGraphMaxHops || cfg.MemoryGraphMaxHits != memoryGraphMaxHits {
		t.Fatalf("unexpected graph hops/hits: hops=%d hits=%d", cfg.MemoryGraphMaxHops, cfg.MemoryGraphMaxHits)
	}
	if cfg.MemoryGraphMinConfidence != memoryGraphMinConfidence {
		t.Fatalf("unexpected graph min confidence: got %v want %v", cfg.MemoryGraphMinConfidence, memoryGraphMinConfidence)
	}
	if cfg.MemoryGraphNamespace != memoryGraphNamespace {
		t.Fatalf("unexpected graph namespace: got %q want %q", cfg.MemoryGraphNamespace, memoryGraphNamespace)
	}
	if !cfg.MemoryGraphDebugEnabled {
		t.Fatalf("expected graph debug to be enabled")
	}
	if !cfg.MemoryDecisionEnabled {
		t.Fatalf("expected decision memory to be enabled")
	}
	if cfg.MemoryDecisionCaptureOnTurn {
		t.Fatalf("expected decision capture on turn to be disabled")
	}
	if cfg.MemoryDecisionPath != memoryDecisionPath {
		t.Fatalf("unexpected decision path: got %q want %q", cfg.MemoryDecisionPath, memoryDecisionPath)
	}
	if cfg.MemoryDecisionMaxHits != memoryDecisionMaxHits {
		t.Fatalf("unexpected decision max hits: got %d want %d", cfg.MemoryDecisionMaxHits, memoryDecisionMaxHits)
	}
	if cfg.MemoryDecisionMinConfidence != memoryDecisionMinConfidence || cfg.MemoryDecisionMinReuseScore != memoryDecisionMinReuseScore {
		t.Fatalf("unexpected decision thresholds: confidence=%v reuse=%v", cfg.MemoryDecisionMinConfidence, cfg.MemoryDecisionMinReuseScore)
	}
	if !cfg.MemoryDecisionRecipeEnabled {
		t.Fatalf("expected decision recipe to be enabled")
	}
	if cfg.MemoryDecisionRecipeInterval.Hours() != 6 {
		t.Fatalf("unexpected decision recipe interval: got %s want 6h", cfg.MemoryDecisionRecipeInterval)
	}
	if cfg.MemoryDecisionRecipeMinSupport != memoryDecisionRecipeMinSupport {
		t.Fatalf("unexpected decision recipe min support: got %d want %d", cfg.MemoryDecisionRecipeMinSupport, memoryDecisionRecipeMinSupport)
	}
	if !cfg.MemoryDecisionDebugEnabled {
		t.Fatalf("expected decision debug to be enabled")
	}
	if cfg.MemoryDecisionSelectorHintEnabled {
		t.Fatalf("expected decision selector hint to be disabled")
	}
	if cfg.MaxTurns != maxTurns {
		t.Fatalf("unexpected max turns: got %d want %d", cfg.MaxTurns, maxTurns)
	}
	if cfg.WorkerMaxFiles != workerMaxFiles {
		t.Fatalf("unexpected worker max files: got %d want %d", cfg.WorkerMaxFiles, workerMaxFiles)
	}
	if !cfg.ToolSelectorEnabled {
		t.Fatalf("expected tool selector to be enabled")
	}
	if cfg.ToolSelectorMode != toolSelectorMode {
		t.Fatalf("unexpected tool selector mode: got %q want %q", cfg.ToolSelectorMode, toolSelectorMode)
	}
}
