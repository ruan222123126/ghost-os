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
	webSearchTavilyAPIKey := "file-tavily-key"
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
	nativeBinaryPath := "/tmp/native-bin"
	nativeBinaryRoots := []string{"/tmp/native-root", "/opt/native"}
	nativeBinaryCandidates := []string{"native", "native.exe"}
	maxTurns := 42
	workerMaxFiles := 8
	toolSelectorEnabled := true
	toolSelectorMode := "rules"
	toolAllowlist := []string{"search_files", "read_file"}
	toolBlocklist := []string{"bash_exec"}
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
		WebSearchTavilyAPIKey:             &webSearchTavilyAPIKey,
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
		NativeBinaryPath:                  &nativeBinaryPath,
		NativeBinaryRoots:                 nativeBinaryRoots,
		NativeBinaryCandidates:            nativeBinaryCandidates,
		MaxTurns:                          &maxTurns,
		WorkerMaxFiles:                    &workerMaxFiles,
		ToolSelectorEnabled:               &toolSelectorEnabled,
		ToolSelectorMode:                  &toolSelectorMode,
		ToolAllowlist:                     toolAllowlist,
		ToolBlocklist:                     toolBlocklist,
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
	if cfg.Worker.Model != workerModel {
		t.Fatalf("unexpected worker model: got %q want %q", cfg.Worker.Model, workerModel)
	}
	if cfg.PromptsPath != promptsPath {
		t.Fatalf("unexpected prompts path: got %q want %q", cfg.PromptsPath, promptsPath)
	}
	if cfg.SessionsPath != sessionsPath {
		t.Fatalf("unexpected sessions path: got %q want %q", cfg.SessionsPath, sessionsPath)
	}
	if cfg.RSS.FeedsPath != rssFeedsPath {
		t.Fatalf("unexpected rss feeds path: got %q want %q", cfg.RSS.FeedsPath, rssFeedsPath)
	}
	if cfg.RSS.InboxPath != rssInboxPath {
		t.Fatalf("unexpected rss inbox path: got %q want %q", cfg.RSS.InboxPath, rssInboxPath)
	}
	if cfg.RSS.PollEnabled {
		t.Fatalf("expected rss poll to be disabled")
	}
	if cfg.RSS.PollInterval != 20*time.Minute {
		t.Fatalf("unexpected rss poll interval: got %s want %s", cfg.RSS.PollInterval, 20*time.Minute)
	}
	if cfg.RSS.PollMaxItemsPerFeed != rssPollMaxItemsPerFeed {
		t.Fatalf("unexpected rss poll max items: got %d want %d", cfg.RSS.PollMaxItemsPerFeed, rssPollMaxItemsPerFeed)
	}
	if cfg.RSS.AIBatchSize != rssAIBatchSize {
		t.Fatalf("unexpected rss ai batch size: got %d want %d", cfg.RSS.AIBatchSize, rssAIBatchSize)
	}
	if cfg.WebSearchTavilyAPIKey != webSearchTavilyAPIKey {
		t.Fatalf("unexpected tavily api key: got %q want %q", cfg.WebSearchTavilyAPIKey, webSearchTavilyAPIKey)
	}
	if cfg.NativeBinaryPath != nativeBinaryPath {
		t.Fatalf("unexpected native binary path: got %q want %q", cfg.NativeBinaryPath, nativeBinaryPath)
	}
	if len(cfg.NativeBinaryRoots) != len(nativeBinaryRoots) || cfg.NativeBinaryRoots[0] != nativeBinaryRoots[0] || cfg.NativeBinaryRoots[1] != nativeBinaryRoots[1] {
		t.Fatalf("unexpected native binary roots: got %v want %v", cfg.NativeBinaryRoots, nativeBinaryRoots)
	}
	if len(cfg.NativeBinaryCandidates) != len(nativeBinaryCandidates) || cfg.NativeBinaryCandidates[0] != nativeBinaryCandidates[0] || cfg.NativeBinaryCandidates[1] != nativeBinaryCandidates[1] {
		t.Fatalf("unexpected native binary candidates: got %v want %v", cfg.NativeBinaryCandidates, nativeBinaryCandidates)
	}
	if cfg.Memory.WarmPath != memoryWarmPath {
		t.Fatalf("unexpected warm path: got %q want %q", cfg.Memory.WarmPath, memoryWarmPath)
	}
	if cfg.Memory.ColdPath != memoryColdPath {
		t.Fatalf("unexpected cold path: got %q want %q", cfg.Memory.ColdPath, memoryColdPath)
	}
	if cfg.Memory.GraphPath != memoryGraphPath {
		t.Fatalf("unexpected graph path: got %q want %q", cfg.Memory.GraphPath, memoryGraphPath)
	}
	if cfg.Memory.WarmTTL.Hours() != 48 {
		t.Fatalf("unexpected warm ttl: got %s want 48h", cfg.Memory.WarmTTL)
	}
	if !cfg.Memory.TemporalDecayEnabled {
		t.Fatalf("expected temporal decay to be enabled")
	}
	if cfg.Memory.TemporalDecayHalfLife.Hours() != 96 {
		t.Fatalf("unexpected temporal half life: got %s want 96h", cfg.Memory.TemporalDecayHalfLife)
	}
	if !cfg.Memory.AnchorEnabled {
		t.Fatalf("expected anchor extraction to be enabled")
	}
	if cfg.Memory.AnchorMinWeight != memoryAnchorMinWeight {
		t.Fatalf("unexpected anchor min weight: got %v want %v", cfg.Memory.AnchorMinWeight, memoryAnchorMinWeight)
	}
	if cfg.Memory.EvolutionInterval.Hours() != 2 {
		t.Fatalf("unexpected evolution interval: got %s want 2h", cfg.Memory.EvolutionInterval)
	}
	if !cfg.Memory.EvolutionUseWorker {
		t.Fatalf("expected evolution worker usage to be enabled")
	}
	if cfg.Memory.EvolutionBatchSize != memoryEvolutionBatchSize {
		t.Fatalf("unexpected evolution batch size: got %d want %d", cfg.Memory.EvolutionBatchSize, memoryEvolutionBatchSize)
	}
	if !cfg.Memory.GraphEnabled {
		t.Fatalf("expected graph memory to be enabled")
	}
	if !cfg.Memory.GraphExtractOnArchive || cfg.Memory.GraphExtractOnEvolve {
		t.Fatalf("unexpected graph extraction flags: archive=%v evolve=%v", cfg.Memory.GraphExtractOnArchive, cfg.Memory.GraphExtractOnEvolve)
	}
	if cfg.Memory.GraphMaxHops != memoryGraphMaxHops || cfg.Memory.GraphMaxHits != memoryGraphMaxHits {
		t.Fatalf("unexpected graph hops/hits: hops=%d hits=%d", cfg.Memory.GraphMaxHops, cfg.Memory.GraphMaxHits)
	}
	if cfg.Memory.GraphMinConfidence != memoryGraphMinConfidence {
		t.Fatalf("unexpected graph min confidence: got %v want %v", cfg.Memory.GraphMinConfidence, memoryGraphMinConfidence)
	}
	if cfg.Memory.GraphNamespace != memoryGraphNamespace {
		t.Fatalf("unexpected graph namespace: got %q want %q", cfg.Memory.GraphNamespace, memoryGraphNamespace)
	}
	if !cfg.Memory.GraphDebugEnabled {
		t.Fatalf("expected graph debug to be enabled")
	}
	if !cfg.Memory.DecisionEnabled {
		t.Fatalf("expected decision memory to be enabled")
	}
	if cfg.Memory.DecisionCaptureOnTurn {
		t.Fatalf("expected decision capture on turn to be disabled")
	}
	if cfg.Memory.DecisionPath != memoryDecisionPath {
		t.Fatalf("unexpected decision path: got %q want %q", cfg.Memory.DecisionPath, memoryDecisionPath)
	}
	if cfg.Memory.DecisionMaxHits != memoryDecisionMaxHits {
		t.Fatalf("unexpected decision max hits: got %d want %d", cfg.Memory.DecisionMaxHits, memoryDecisionMaxHits)
	}
	if cfg.Memory.DecisionMinConfidence != memoryDecisionMinConfidence || cfg.Memory.DecisionMinReuseScore != memoryDecisionMinReuseScore {
		t.Fatalf("unexpected decision thresholds: confidence=%v reuse=%v", cfg.Memory.DecisionMinConfidence, cfg.Memory.DecisionMinReuseScore)
	}
	if !cfg.Memory.DecisionRecipeEnabled {
		t.Fatalf("expected decision recipe to be enabled")
	}
	if cfg.Memory.DecisionRecipeInterval.Hours() != 6 {
		t.Fatalf("unexpected decision recipe interval: got %s want 6h", cfg.Memory.DecisionRecipeInterval)
	}
	if cfg.Memory.DecisionRecipeMinSupport != memoryDecisionRecipeMinSupport {
		t.Fatalf("unexpected decision recipe min support: got %d want %d", cfg.Memory.DecisionRecipeMinSupport, memoryDecisionRecipeMinSupport)
	}
	if !cfg.Memory.DecisionDebugEnabled {
		t.Fatalf("expected decision debug to be enabled")
	}
	if cfg.Memory.DecisionSelectorHintEnabled {
		t.Fatalf("expected decision selector hint to be disabled")
	}
	if cfg.MaxTurns != maxTurns {
		t.Fatalf("unexpected max turns: got %d want %d", cfg.MaxTurns, maxTurns)
	}
	if cfg.Worker.MaxFiles != workerMaxFiles {
		t.Fatalf("unexpected worker max files: got %d want %d", cfg.Worker.MaxFiles, workerMaxFiles)
	}
	if !cfg.ToolSelector.Enabled {
		t.Fatalf("expected tool selector to be enabled")
	}
	if cfg.ToolSelector.Mode != toolSelectorMode {
		t.Fatalf("unexpected tool selector mode: got %q want %q", cfg.ToolSelector.Mode, toolSelectorMode)
	}
	if len(cfg.ToolSelector.Allowlist) != 2 || cfg.ToolSelector.Allowlist[0] != "read_file" || cfg.ToolSelector.Allowlist[1] != "search_files" {
		t.Fatalf("unexpected tool allowlist: %v", cfg.ToolSelector.Allowlist)
	}
	if len(cfg.ToolSelector.Blocklist) != 1 || cfg.ToolSelector.Blocklist[0] != "bash_exec" {
		t.Fatalf("unexpected tool blocklist: %v", cfg.ToolSelector.Blocklist)
	}
}
