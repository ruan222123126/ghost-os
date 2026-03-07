package app

import (
	"os"
	"path/filepath"
	"testing"
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
		ModelProvider:    &modelProvider,
		Model:            &model,
		ChatPath:         &chatPath,
		NativePersistent: &nativePersistent,
		ModelProviders: []providerConfig{{
			Name:    "crs",
			BaseURL: "https://lldai.online/openai",
			APIKey:  optionalStringPointer("file-key"),
			Models:  []string{"gpt-5.4", "gpt-4"},
		}},
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
	memoryWarmTTL := "48h"
	memoryTemporalHalfLife := "96h"
	memoryTemporalDecayEnabled := true
	memoryAnchorEnabled := true
	memoryAnchorMinWeight := 0.75
	memoryEvolutionInterval := "2h"
	memoryEvolutionUseWorker := true
	memoryEvolutionBatchSize := 12
	maxTurns := 42
	workerMaxFiles := 8
	toolSelectorEnabled := true
	toolSelectorMode := "rules"
	bindAddr := "0.0.0.0:9090"
	apiToken := "secret-token"
	if err := writeBridgeFileConfig(configPathFromEnv(), bridgeFileConfig{
		ModelProvider: &modelProvider,
		Model:         &model,
		ModelProviders: []providerConfig{{
			Name:    "openai",
			BaseURL: defaultBaseURL,
			APIKey:  optionalStringPointer("file-key"),
		}},
		WorkerModel:                 &workerModel,
		PromptsPath:                 &promptsPath,
		SessionsPath:                &sessionsPath,
		MemoryWarmPath:              &memoryWarmPath,
		MemoryColdPath:              &memoryColdPath,
		MemoryWarmTTL:               &memoryWarmTTL,
		MemoryTemporalDecayEnabled:  &memoryTemporalDecayEnabled,
		MemoryTemporalDecayHalfLife: &memoryTemporalHalfLife,
		MemoryAnchorEnabled:         &memoryAnchorEnabled,
		MemoryAnchorMinWeight:       &memoryAnchorMinWeight,
		MemoryEvolutionInterval:     &memoryEvolutionInterval,
		MemoryEvolutionUseWorker:    &memoryEvolutionUseWorker,
		MemoryEvolutionBatchSize:    &memoryEvolutionBatchSize,
		MaxTurns:                    &maxTurns,
		WorkerMaxFiles:              &workerMaxFiles,
		ToolSelectorEnabled:         &toolSelectorEnabled,
		ToolSelectorMode:            &toolSelectorMode,
		BindAddr:                    &bindAddr,
		APIToken:                    &apiToken,
		CORSOrigins:                 []string{"http://localhost:5173"},
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

func TestLoadBridgeFileConfigMigratesLegacyYAMLToToml(t *testing.T) {
	legacyPath := filepath.Join(t.TempDir(), "config.yaml")
	t.Setenv("GHOST_CONFIG_PATH", legacyPath)

	provider := "custom"
	apiKey := "legacy-key"
	baseURL := "http://localhost:11434/v1"
	model := "qwen-coder"
	if err := writeLegacyBridgeYAMLConfig(configPathFromEnv(), bridgeFileConfig{
		Provider: &provider,
		APIKey:   &apiKey,
		BaseURL:  &baseURL,
		Model:    &model,
	}); err != nil {
		t.Fatalf("write legacy config file: %v", err)
	}

	cfg, configPath, err := loadBridgeFileConfig()
	if err != nil {
		t.Fatalf("loadBridgeFileConfig: %v", err)
	}
	if filepath.Ext(configPath) != ".toml" {
		t.Fatalf("unexpected config path: got %q want .toml suffix", configPath)
	}
	if cfg.ModelProvider == nil || *cfg.ModelProvider != provider {
		t.Fatalf("unexpected model provider: %#v", cfg.ModelProvider)
	}
	if len(cfg.ModelProviders) != 1 {
		t.Fatalf("unexpected provider count: got %d want 1", len(cfg.ModelProviders))
	}
	if cfg.ModelProviders[0].BaseURL != baseURL {
		t.Fatalf("unexpected migrated base url: got %q want %q", cfg.ModelProviders[0].BaseURL, baseURL)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(legacyPath), "config.toml")); err != nil {
		t.Fatalf("expected migrated toml config: %v", err)
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
		BaseURL: "https://lldai.online/openai",
		APIKey:  optionalStringPointer("sk-xxx"),
		Models:  []string{"gpt-5.4", "gpt-4"},
	}); err != nil {
		t.Fatalf("AddProvider: %v", err)
	}
	if err := store.AddProvider(providerConfig{
		Name:    "openai",
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
	if fileCfg.ModelProvider == nil || *fileCfg.ModelProvider != "openai" {
		t.Fatalf("unexpected active provider: %#v", fileCfg.ModelProvider)
	}
	if len(fileCfg.ModelProviders) != 1 {
		t.Fatalf("unexpected provider count: got %d want 1", len(fileCfg.ModelProviders))
	}
	if fileCfg.ModelProviders[0].APIKey == nil || *fileCfg.ModelProviders[0].APIKey != "sk-yyy" {
		t.Fatalf("expected update to preserve api key, got %#v", fileCfg.ModelProviders[0].APIKey)
	}
	if runtime := store.RuntimeConfig(); runtime.ProviderName != "openai" {
		t.Fatalf("unexpected runtime provider: got %q want %q", runtime.ProviderName, "openai")
	}
}
