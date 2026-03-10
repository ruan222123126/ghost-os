package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ghost-os/bridge/llm"
)

func TestMemoryManagerConfigFromAppConfigDefaultsToStablePublicPath(t *testing.T) {
	baseDir := t.TempDir()
	t.Setenv("GHOST_CONFIG_PATH", filepath.Join(baseDir, "config.toml"))
	cfg := Config{
		Memory: MemoryRuntimeConfig{
			WarmPath:          filepath.Join(baseDir, "warm.json"),
			ColdPath:          filepath.Join(baseDir, "cold"),
			LedgerPath:        filepath.Join(baseDir, "cold", "ledger"),
			LedgerReadEnabled: true,
			AutoRecallEnabled: true,
			AutoRecallLimit:   5,
			GraphEnabled:      true,
			GraphPath:         filepath.Join(baseDir, "graph"),
			DecisionEnabled:   true,
			DecisionPath:      filepath.Join(baseDir, "decision"),
		},
	}

	memoryCfg := memoryManagerConfigFromAppConfig(cfg, nil, nil)
	if memoryCfg.Ledger.DualWrite {
		t.Fatal("expected ledger dual-write to stay off in stable app config")
	}
	if memoryCfg.Ledger.ShadowCompare {
		t.Fatal("expected ledger shadow compare to stay off in stable app config")
	}
	if memoryCfg.Truth.Enabled || memoryCfg.Vector.Enabled || memoryCfg.Index.Enabled || memoryCfg.Recall.ShadowEnabled {
		t.Fatalf("expected experimental memory path to stay off by default, got %+v", memoryCfg)
	}
}

func TestMemoryManagerConfigFromAppConfigIgnoresDeprecatedExperimentalFields(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "config.toml")
	t.Setenv("GHOST_CONFIG_PATH", configPath)

	if err := os.WriteFile(configPath, []byte("memory_ledger_dual_write = true\nmemory_ledger_shadow_compare = true\n"), 0o600); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	loaded, err := loadConfigWithRuntime(runtimeConfig{
		Provider: llm.ProviderCustom,
		BaseURL:  "https://example.com/v1",
		Model:    "gpt-4o",
	})
	if err != nil {
		t.Fatalf("loadConfigWithRuntime: %v", err)
	}

	memoryCfg := memoryManagerConfigFromAppConfig(loaded, nil, nil)
	if memoryCfg.Ledger.DualWrite || memoryCfg.Ledger.ShadowCompare {
		t.Fatalf("expected deprecated experimental ledger toggles to stay ignored, got %+v", memoryCfg.Ledger)
	}
}

func TestMemoryQueryParamsIgnoreExperimentalFields(t *testing.T) {
	var params memoryQueryParams
	if err := json.Unmarshal([]byte(`{
		"semantic_query":"deploy plan",
		"include_graph":true,
		"graph_debug":true,
		"include_vector":true,
		"vector_debug":true,
		"truth_debug":true,
		"rerank_debug":true,
		"bucket_debug":true,
		"intent_debug":true
	}`), &params); err != nil {
		t.Fatalf("unmarshal params: %v", err)
	}

	query, err := buildMemoryQuery(params)
	if err != nil {
		t.Fatalf("buildMemoryQuery: %v", err)
	}
	if !query.IncludeGraph || !query.GraphDebug {
		t.Fatalf("expected stable graph params to survive, got %+v", query)
	}
	if query.IncludeVector || query.VectorDebug || query.IntentDebug || query.TruthDebug || query.RerankDebug || query.BucketDebug {
		t.Fatalf("expected experimental query params to be ignored by public contract, got %+v", query)
	}
	if !query.Debug {
		t.Fatal("expected stable graph debug to keep public debug response enabled")
	}
}

func TestConfigExampleOmitsExperimentalMemoryFields(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "docs", "config.example.toml"))
	if err != nil {
		t.Fatalf("read config example: %v", err)
	}
	text := string(raw)
	for _, forbidden := range []string{
		"memory_ledger_dual_write",
		"memory_ledger_shadow_compare",
		"memory_truth",
		"memory_vector",
		"memory_index",
		"shadow_recall",
	} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("expected config example to omit %q, got %q", forbidden, text)
		}
	}
}
