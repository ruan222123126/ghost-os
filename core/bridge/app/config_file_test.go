package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

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
		WorkerModel:                    &workerModel,
		PromptsPath:                    &promptsPath,
		SessionsPath:                   &sessionsPath,
		MemoryWarmPath:                 &memoryWarmPath,
		MemoryColdPath:                 &memoryColdPath,
		MemoryGraphPath:                &memoryGraphPath,
		MemoryWarmTTL:                  &memoryWarmTTL,
		MemoryTemporalDecayEnabled:     &memoryTemporalDecayEnabled,
		MemoryTemporalDecayHalfLife:    &memoryTemporalHalfLife,
		MemoryAnchorEnabled:            &memoryAnchorEnabled,
		MemoryAnchorMinWeight:          &memoryAnchorMinWeight,
		MemoryEvolutionInterval:        &memoryEvolutionInterval,
		MemoryEvolutionUseWorker:       &memoryEvolutionUseWorker,
		MemoryEvolutionBatchSize:       &memoryEvolutionBatchSize,
		MemoryGraphEnabled:             &memoryGraphEnabled,
		MemoryGraphExtractOnArchive:    &memoryGraphExtractOnArchive,
		MemoryGraphExtractOnEvolve:     &memoryGraphExtractOnEvolve,
		MemoryGraphMaxHops:             &memoryGraphMaxHops,
		MemoryGraphMaxHits:             &memoryGraphMaxHits,
		MemoryGraphMinConfidence:       &memoryGraphMinConfidence,
		MemoryGraphNamespace:           &memoryGraphNamespace,
		MemoryGraphDebugEnabled:        &memoryGraphDebugEnabled,
		MemoryDecisionEnabled:          &memoryDecisionEnabled,
		MemoryDecisionCaptureOnTurn:    &memoryDecisionCaptureOnTurn,
		MemoryDecisionPath:             &memoryDecisionPath,
		MemoryDecisionMaxHits:          &memoryDecisionMaxHits,
		MemoryDecisionMinConfidence:    &memoryDecisionMinConfidence,
		MemoryDecisionMinReuseScore:    &memoryDecisionMinReuseScore,
		MemoryDecisionRecipeEnabled:    &memoryDecisionRecipeEnabled,
		MemoryDecisionRecipeInterval:   &memoryDecisionRecipeInterval,
		MemoryDecisionRecipeMinSupport: &memoryDecisionRecipeMinSupport,
		MemoryDecisionDebugEnabled:     &memoryDecisionDebugEnabled,
		MaxTurns:                       &maxTurns,
		WorkerMaxFiles:                 &workerMaxFiles,
		ToolSelectorEnabled:            &toolSelectorEnabled,
		ToolSelectorMode:               &toolSelectorMode,
		BindAddr:                       &bindAddr,
		APIToken:                       &apiToken,
		CORSOrigins:                    []string{"http://localhost:5173"},
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
	if got := resolveBindAddr(8080); got != bindAddr {
		t.Fatalf("unexpected bind addr: got %q want %q", got, bindAddr)
	}
	auth := newAPITokenAuthFromEnv()
	if auth.token != apiToken {
		t.Fatalf("unexpected api token: got %q want %q", auth.token, apiToken)
	}
	policy := newCORSPolicyFromEnv()
	if !policy.allows("http://localhost:5173") {
		t.Fatalf("expected origin from config file to be allowed")
	}
	if policy.allows("https://evil.example") {
		t.Fatalf("unexpected allow for unconfigured origin")
	}
}

func TestLoadBridgeFileConfigMigratesLegacyProviderLayoutToNamedTables(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.toml")
	t.Setenv("GHOST_CONFIG_PATH", configPath)

	legacyConfig := []byte(`model_provider = "custom"
model = "qwen-coder"
memory_decision_enabled = true
memory_decision_capture_on_turn = false
memory_decision_path = "~/.ghost-os/memory/decision"
memory_decision_max_hits = 5
memory_decision_min_confidence = 0.76
memory_decision_min_reuse_score = 0.71
memory_decision_recipe_enabled = false
memory_decision_recipe_interval = "8h"
memory_decision_recipe_min_support = 4
memory_decision_debug_enabled = true

[[model_providers]]
name = "custom"
base_url = "http://localhost:11434/v1"
api_key = "legacy-key"
`)
	if err := os.WriteFile(configPath, legacyConfig, 0o600); err != nil {
		t.Fatalf("write legacy config file: %v", err)
	}

	cfg, loadedPath, err := loadBridgeFileConfig()
	if err != nil {
		t.Fatalf("loadBridgeFileConfig: %v", err)
	}
	if loadedPath != configPath {
		t.Fatalf("unexpected config path: got %q want %q", loadedPath, configPath)
	}
	if cfg.ActiveProvider == nil || *cfg.ActiveProvider != "custom" {
		t.Fatalf("unexpected active provider: %#v", cfg.ActiveProvider)
	}
	providers := normalizeProviderConfigs(cfg.Providers, stringValue(cfg.Model))
	if len(providers) != 1 {
		t.Fatalf("unexpected provider count: got %d want 1", len(providers))
	}
	if providers[0].Type != llm.ProviderCustom {
		t.Fatalf("unexpected provider type: got %q want %q", providers[0].Type, llm.ProviderCustom)
	}
	if providers[0].BaseURL != "http://localhost:11434/v1" {
		t.Fatalf("unexpected migrated base url: got %q", providers[0].BaseURL)
	}
	if cfg.MemoryDecisionRecipeInterval == nil || *cfg.MemoryDecisionRecipeInterval != "8h" {
		t.Fatalf("unexpected migrated decision recipe interval: %#v", cfg.MemoryDecisionRecipeInterval)
	}
	if cfg.MemoryDecisionCaptureOnTurn == nil || *cfg.MemoryDecisionCaptureOnTurn {
		t.Fatalf("unexpected migrated decision capture toggle: %#v", cfg.MemoryDecisionCaptureOnTurn)
	}
	raw, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read migrated config file: %v", err)
	}
	contents := string(raw)
	if strings.Contains(contents, "[[model_providers]]") || strings.Contains(contents, "model_provider") {
		t.Fatalf("expected legacy provider layout to be removed, got %s", contents)
	}
	if !strings.Contains(contents, "[providers.custom]") {
		t.Fatalf("expected named provider table, got %s", contents)
	}
}

func TestConfigStoreProviderCRUDPersistsToml(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.toml")
	t.Setenv("GHOST_CONFIG_PATH", configPath)

	store, err := NewConfigStoreFromEnv()
	if err != nil {
		t.Fatalf("NewConfigStoreFromEnv: %v", err)
	}

	if err := store.AddProvider(providerConfig{
		Name:    "crs",
		Type:    llm.ProviderCustom,
		BaseURL: "https://lldai.online/openai",
		APIKey:  optionalStringPointer("sk-xxx"),
		Models:  []string{"gpt-5.4", "gpt-4"},
	}); err != nil {
		t.Fatalf("AddProvider: %v", err)
	}
	if err := store.AddProvider(providerConfig{
		Name:    "openai",
		Type:    llm.ProviderOpenAI,
		BaseURL: defaultBaseURL,
		APIKey:  optionalStringPointer("sk-yyy"),
	}); err != nil {
		t.Fatalf("AddProvider second provider: %v", err)
	}
	if err := store.SetActiveProvider("openai"); err != nil {
		t.Fatalf("SetActiveProvider: %v", err)
	}
	if err := store.UpdateProvider("openai", providerConfig{
		Name:    "openai",
		Type:    llm.ProviderOpenAI,
		BaseURL: "https://api.openai.com/v1",
		Models:  []string{"gpt-5.4"},
	}); err != nil {
		t.Fatalf("UpdateProvider: %v", err)
	}
	if err := store.DeleteProvider("crs"); err != nil {
		t.Fatalf("DeleteProvider: %v", err)
	}

	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		t.Fatalf("loadBridgeFileConfig: %v", err)
	}
	if fileCfg.ActiveProvider == nil || *fileCfg.ActiveProvider != "openai" {
		t.Fatalf("unexpected active provider: %#v", fileCfg.ActiveProvider)
	}
	providers := normalizeProviderConfigs(fileCfg.Providers, stringValue(fileCfg.Model))
	if len(providers) != 1 {
		t.Fatalf("unexpected provider count: got %d want 1", len(providers))
	}
	if providers[0].APIKey == nil || *providers[0].APIKey != "sk-yyy" {
		t.Fatalf("expected update to preserve api key, got %#v", providers[0].APIKey)
	}
	if providers[0].Type != llm.ProviderOpenAI {
		t.Fatalf("expected provider type to persist, got %q", providers[0].Type)
	}
	if runtime := store.RuntimeConfig(); runtime.ProviderName != "openai" {
		t.Fatalf("unexpected runtime provider: got %q want %q", runtime.ProviderName, "openai")
	}
}
