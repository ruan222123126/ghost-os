package app

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"ghost-os/bridge/llm"
)

const defaultConfigPath = "~/.ghost-os/config.toml"

type providerConfig struct {
	Name    string
	Type    llm.Provider
	BaseURL string
	APIKey  *string
	Models  []string
}

type providerFileConfig struct {
	Type    llm.Provider `toml:"type,omitempty"`
	BaseURL string       `toml:"base_url,omitempty"`
	APIKey  *string      `toml:"api_key,omitempty"`
	Models  []string     `toml:"models,omitempty"`
}

type legacyProviderConfig struct {
	Name    string       `toml:"name,omitempty"`
	Type    llm.Provider `toml:"type,omitempty"`
	BaseURL string       `toml:"base_url,omitempty"`
	APIKey  *string      `toml:"api_key,omitempty"`
	Models  []string     `toml:"models,omitempty"`
}

type bridgeFileConfig struct {
	ActiveProvider                               *string                       `toml:"active_provider,omitempty"`
	Providers                                    map[string]providerFileConfig `toml:"providers,omitempty"`
	ModelProvider                                *string                       `toml:"model_provider,omitempty"`
	ModelProviders                               []legacyProviderConfig        `toml:"model_providers,omitempty"`
	Provider                                     *string                       `toml:"provider,omitempty"`
	APIKey                                       *string                       `toml:"api_key,omitempty"`
	BaseURL                                      *string                       `toml:"base_url,omitempty"`
	Model                                        *string                       `toml:"model,omitempty"`
	ChatPath                                     *string                       `toml:"chat_path,omitempty"`
	NativePersistent                             *bool                         `toml:"native_persistent,omitempty"`
	WorkerModel                                  *string                       `toml:"worker_model,omitempty"`
	PromptsPath                                  *string                       `toml:"prompts_path,omitempty"`
	SessionsPath                                 *string                       `toml:"sessions_path,omitempty"`
	MemoryWarmPath                               *string                       `toml:"memory_warm_path,omitempty"`
	MemoryColdPath                               *string                       `toml:"memory_cold_path,omitempty"`
	MemoryLedgerPath                             *string                       `toml:"memory_ledger_path,omitempty"`
	MemoryLedgerDualWrite                        *bool                         `toml:"memory_ledger_dual_write,omitempty"`
	MemoryLedgerReadEnabled                      *bool                         `toml:"memory_ledger_read_enabled,omitempty"`
	MemoryLedgerShadowCompare                    *bool                         `toml:"memory_ledger_shadow_compare,omitempty"`
	MemoryAutoRecallEnabled                      *bool                         `toml:"memory_auto_recall_enabled,omitempty"`
	MemoryAutoRecallLimit                        *int                          `toml:"memory_auto_recall_limit,omitempty"`
	MemoryWarmTTL                                *string                       `toml:"memory_warm_ttl,omitempty"`
	MemoryTemporalDecayEnabled                   *bool                         `toml:"memory_temporal_decay_enabled,omitempty"`
	MemoryTemporalDecayHalfLife                  *string                       `toml:"memory_temporal_decay_half_life,omitempty"`
	MemoryAnchorEnabled                          *bool                         `toml:"memory_anchor_enabled,omitempty"`
	MemoryAnchorMinWeight                        *float64                      `toml:"memory_anchor_min_weight,omitempty"`
	MemoryEvolutionInterval                      *string                       `toml:"memory_evolution_interval,omitempty"`
	MemoryEvolutionEnabled                       *bool                         `toml:"memory_evolution_enabled,omitempty"`
	MemoryEvolutionUseWorker                     *bool                         `toml:"memory_evolution_use_worker,omitempty"`
	MemoryEvolutionBatchSize                     *int                          `toml:"memory_evolution_batch_size,omitempty"`
	MemoryGraphEnabled                           *bool                         `toml:"memory_graph_enabled,omitempty"`
	MemoryGraphPath                              *string                       `toml:"memory_graph_path,omitempty"`
	MemoryGraphExtractOnArchive                  *bool                         `toml:"memory_graph_extract_on_archive,omitempty"`
	MemoryGraphExtractOnEvolve                   *bool                         `toml:"memory_graph_extract_on_evolve,omitempty"`
	MemoryGraphMaxHops                           *int                          `toml:"memory_graph_max_hops,omitempty"`
	MemoryGraphMaxHits                           *int                          `toml:"memory_graph_max_hits,omitempty"`
	MemoryGraphMinConfidence                     *float64                      `toml:"memory_graph_min_confidence,omitempty"`
	MemoryGraphNamespace                         *string                       `toml:"memory_graph_namespace,omitempty"`
	MemoryGraphDebugEnabled                      *bool                         `toml:"memory_graph_debug_enabled,omitempty"`
	MemoryDecisionEnabled                        *bool                         `toml:"memory_decision_enabled,omitempty"`
	MemoryDecisionCaptureOnTurn                  *bool                         `toml:"memory_decision_capture_on_turn,omitempty"`
	MemoryDecisionPath                           *string                       `toml:"memory_decision_path,omitempty"`
	MemoryDecisionMaxHits                        *int                          `toml:"memory_decision_max_hits,omitempty"`
	MemoryDecisionMinConfidence                  *float64                      `toml:"memory_decision_min_confidence,omitempty"`
	MemoryDecisionMinReuseScore                  *float64                      `toml:"memory_decision_min_reuse_score,omitempty"`
	MemoryDecisionRecipeEnabled                  *bool                         `toml:"memory_decision_recipe_enabled,omitempty"`
	MemoryDecisionRecipeInterval                 *string                       `toml:"memory_decision_recipe_interval,omitempty"`
	MemoryDecisionRecipeMinSupport               *int                          `toml:"memory_decision_recipe_min_support,omitempty"`
	MemoryDecisionDebugEnabled                   *bool                         `toml:"memory_decision_debug_enabled,omitempty"`
	MemoryDecisionSelectorHintEnabled            *bool                         `toml:"memory_decision_selector_hint_enabled,omitempty"`
	MemoryDecisionRecipeReuseEnabled             *bool                         `toml:"memory_decision_recipe_reuse_enabled,omitempty"`
	MemoryDecisionRecipeExecutionTrackingEnabled *bool                         `toml:"memory_decision_recipe_execution_tracking_enabled,omitempty"`
	MemoryDecisionRecipeBackfillEnabled          *bool                         `toml:"memory_decision_recipe_backfill_enabled,omitempty"`
	MemoryDecisionRecipeDefaultEnabled           *bool                         `toml:"memory_decision_recipe_default_enabled,omitempty"`
	MemoryDecisionRecipeDefaultGrayPercent       *int                          `toml:"memory_decision_recipe_default_gray_percent,omitempty"`
	MemoryDecisionRecipeMinSelectionConfidence   *float64                      `toml:"memory_decision_recipe_min_selection_confidence,omitempty"`
	MemoryDecisionRecipeMinSuccessRate           *float64                      `toml:"memory_decision_recipe_min_success_rate,omitempty"`
	MemoryDecisionRecipeBackfillBatchSize        *int                          `toml:"memory_decision_recipe_backfill_batch_size,omitempty"`
	MemoryDecisionRecipeBackfillInterval         *string                       `toml:"memory_decision_recipe_backfill_interval,omitempty"`
	ProviderHeaders                              map[string]string             `toml:"provider_headers,omitempty"`
	AnthropicVersion                             *string                       `toml:"anthropic_version,omitempty"`
	AnthropicMaxTokens                           *int                          `toml:"anthropic_max_tokens,omitempty"`
	MaxTurns                                     *int                          `toml:"max_turns,omitempty"`
	WorkerMaxConcurrency                         *int                          `toml:"worker_max_concurrency,omitempty"`
	WorkerMaxFiles                               *int                          `toml:"worker_max_files,omitempty"`
	WorkerMaxFileChunks                          *int                          `toml:"worker_max_file_chunks,omitempty"`
	ToolSelectorEnabled                          *bool                         `toml:"tool_selector_enabled,omitempty"`
	ToolSelectorMode                             *string                       `toml:"tool_selector_mode,omitempty"`
	ToolSelectorModel                            *string                       `toml:"tool_selector_model,omitempty"`
	ToolSelectorTimeoutMS                        *int                          `toml:"tool_selector_timeout_ms,omitempty"`
	ToolSelectorConfidence                       *float64                      `toml:"tool_selector_confidence,omitempty"`
	ToolSelectorShadow                           *bool                         `toml:"tool_selector_shadow,omitempty"`
	ToolSelectorRecentMsgs                       *int                          `toml:"tool_selector_recent_messages,omitempty"`
	BindAddr                                     *string                       `toml:"bind_addr,omitempty"`
	APIToken                                     *string                       `toml:"api_token,omitempty"`
	CORSOrigins                                  []string                      `toml:"cors_origins,omitempty"`
}

func configPathFromEnv() string {
	return getenvDefault("GHOST_CONFIG_PATH", defaultConfigPath)
}

func resolveUserPath(pathValue string) (string, error) {
	trimmed := strings.TrimSpace(pathValue)
	if trimmed == "" {
		return "", errors.New("path is empty")
	}

	if trimmed == "~" || strings.HasPrefix(trimmed, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve user home directory: %w", err)
		}
		if trimmed == "~" {
			return homeDir, nil
		}
		return filepath.Join(homeDir, strings.TrimPrefix(trimmed, "~/")), nil
	}

	return filepath.Clean(trimmed), nil
}

func valueOrEnv(raw *string, envName, fallback string) string {
	if raw != nil {
		return strings.TrimSpace(*raw)
	}
	return getenvDefault(envName, fallback)
}

func boolOrEnv(raw *bool, envName string, fallback bool) bool {
	if raw != nil {
		return *raw
	}
	return parseBoolEnv(envName, fallback)
}

func intOrEnv(raw *int, envName string, fallback int) int {
	if raw != nil {
		if *raw > 0 {
			return *raw
		}
		return fallback
	}
	return parsePositiveIntEnv(envName, fallback)
}

func floatOrEnv(raw *float64, envName string, fallback float64) float64 {
	if raw != nil {
		if *raw >= 0 && *raw <= 1 {
			return *raw
		}
		return fallback
	}
	return parseFloatEnv(envName, fallback)
}

func durationOrEnv(raw *string, envName string, fallback time.Duration) time.Duration {
	if raw != nil {
		value, err := time.ParseDuration(strings.TrimSpace(*raw))
		if err != nil || value <= 0 {
			return fallback
		}
		return value
	}
	return parseDurationEnv(envName, fallback)
}

func headersOrEnv(raw map[string]string) (map[string]string, error) {
	if len(raw) > 0 {
		return normalizeProviderHeaders(raw)
	}
	return parseProviderHeaders(os.Getenv("GHOST_PROVIDER_HEADERS"))
}

func normalizeProviderHeaders(raw map[string]string) (map[string]string, error) {
	if len(raw) == 0 {
		return nil, nil
	}

	out := make(map[string]string, len(raw))
	for key, value := range raw {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" {
			return nil, fmt.Errorf("invalid provider_headers: header key cannot be empty")
		}
		out[trimmedKey] = strings.TrimSpace(value)
	}
	return out, nil
}

func corsOriginsOrEnv(raw []string) []string {
	if raw != nil {
		return normalizeOrigins(raw)
	}
	return parseOriginsCSV(getenvDefault("GHOST_CORS_ORIGINS", ""))
}

func normalizeOrigins(origins []string) []string {
	trimmed := make([]string, 0, len(origins))
	for _, origin := range origins {
		value := strings.TrimSpace(origin)
		if value == "" {
			continue
		}
		trimmed = append(trimmed, value)
	}
	return trimmed
}

func parseOriginsCSV(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	return normalizeOrigins(strings.Split(raw, ","))
}

func cloneStringPointer(raw *string) *string {
	if raw == nil {
		return nil
	}
	value := strings.TrimSpace(*raw)
	return &value
}

func cloneOptionalStringPointer(raw *string) *string {
	if raw == nil {
		return nil
	}
	value := strings.TrimSpace(*raw)
	if value == "" {
		return nil
	}
	return &value
}

func optionalStringPointer(raw string) *string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return nil
	}
	return &value
}

func stringPointer(raw string) *string {
	value := strings.TrimSpace(raw)
	return &value
}

func stringValue(raw *string) string {
	if raw == nil {
		return ""
	}
	return strings.TrimSpace(*raw)
}
