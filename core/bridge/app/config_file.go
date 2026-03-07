package app

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	defaultConfigPath       = "~/.ghost-os/config.toml"
	defaultLegacyConfigPath = "~/.ghost-os/config.yaml"
)

type providerConfig struct {
	Name    string   `yaml:"name,omitempty" toml:"name,omitempty"`
	BaseURL string   `yaml:"base_url,omitempty" toml:"base_url,omitempty"`
	APIKey  *string  `yaml:"api_key,omitempty" toml:"api_key,omitempty"`
	Models  []string `yaml:"models,omitempty" toml:"models,omitempty"`
}

type bridgeFileConfig struct {
	ModelProvider                  *string           `yaml:"model_provider,omitempty" toml:"model_provider,omitempty"`
	ModelProviders                 []providerConfig  `yaml:"model_providers,omitempty" toml:"model_providers,omitempty"`
	Provider                       *string           `yaml:"provider,omitempty" toml:"provider,omitempty"`
	APIKey                         *string           `yaml:"api_key,omitempty" toml:"api_key,omitempty"`
	BaseURL                        *string           `yaml:"base_url,omitempty" toml:"base_url,omitempty"`
	Model                          *string           `yaml:"model,omitempty" toml:"model,omitempty"`
	ChatPath                       *string           `yaml:"chat_path,omitempty" toml:"chat_path,omitempty"`
	NativePersistent               *bool             `yaml:"native_persistent,omitempty" toml:"native_persistent,omitempty"`
	WorkerModel                    *string           `yaml:"worker_model,omitempty" toml:"worker_model,omitempty"`
	PromptsPath                    *string           `yaml:"prompts_path,omitempty" toml:"prompts_path,omitempty"`
	SessionsPath                   *string           `yaml:"sessions_path,omitempty" toml:"sessions_path,omitempty"`
	MemoryWarmPath                 *string           `yaml:"memory_warm_path,omitempty" toml:"memory_warm_path,omitempty"`
	MemoryColdPath                 *string           `yaml:"memory_cold_path,omitempty" toml:"memory_cold_path,omitempty"`
	MemoryAutoRecallEnabled        *bool             `yaml:"memory_auto_recall_enabled,omitempty" toml:"memory_auto_recall_enabled,omitempty"`
	MemoryAutoRecallLimit          *int              `yaml:"memory_auto_recall_limit,omitempty" toml:"memory_auto_recall_limit,omitempty"`
	MemoryWarmTTL                  *string           `yaml:"memory_warm_ttl,omitempty" toml:"memory_warm_ttl,omitempty"`
	MemoryTemporalDecayEnabled     *bool             `yaml:"memory_temporal_decay_enabled,omitempty" toml:"memory_temporal_decay_enabled,omitempty"`
	MemoryTemporalDecayHalfLife    *string           `yaml:"memory_temporal_decay_half_life,omitempty" toml:"memory_temporal_decay_half_life,omitempty"`
	MemoryAnchorEnabled            *bool             `yaml:"memory_anchor_enabled,omitempty" toml:"memory_anchor_enabled,omitempty"`
	MemoryAnchorMinWeight          *float64          `yaml:"memory_anchor_min_weight,omitempty" toml:"memory_anchor_min_weight,omitempty"`
	MemoryEvolutionInterval        *string           `yaml:"memory_evolution_interval,omitempty" toml:"memory_evolution_interval,omitempty"`
	MemoryEvolutionEnabled         *bool             `yaml:"memory_evolution_enabled,omitempty" toml:"memory_evolution_enabled,omitempty"`
	MemoryEvolutionUseWorker       *bool             `yaml:"memory_evolution_use_worker,omitempty" toml:"memory_evolution_use_worker,omitempty"`
	MemoryEvolutionBatchSize       *int              `yaml:"memory_evolution_batch_size,omitempty" toml:"memory_evolution_batch_size,omitempty"`
	MemoryGraphEnabled             *bool             `yaml:"memory_graph_enabled,omitempty" toml:"memory_graph_enabled,omitempty"`
	MemoryGraphPath                *string           `yaml:"memory_graph_path,omitempty" toml:"memory_graph_path,omitempty"`
	MemoryGraphExtractOnArchive    *bool             `yaml:"memory_graph_extract_on_archive,omitempty" toml:"memory_graph_extract_on_archive,omitempty"`
	MemoryGraphExtractOnEvolve     *bool             `yaml:"memory_graph_extract_on_evolve,omitempty" toml:"memory_graph_extract_on_evolve,omitempty"`
	MemoryGraphMaxHops             *int              `yaml:"memory_graph_max_hops,omitempty" toml:"memory_graph_max_hops,omitempty"`
	MemoryGraphMaxHits             *int              `yaml:"memory_graph_max_hits,omitempty" toml:"memory_graph_max_hits,omitempty"`
	MemoryGraphMinConfidence       *float64          `yaml:"memory_graph_min_confidence,omitempty" toml:"memory_graph_min_confidence,omitempty"`
	MemoryGraphNamespace           *string           `yaml:"memory_graph_namespace,omitempty" toml:"memory_graph_namespace,omitempty"`
	MemoryGraphDebugEnabled        *bool             `yaml:"memory_graph_debug_enabled,omitempty" toml:"memory_graph_debug_enabled,omitempty"`
	MemoryDecisionEnabled          *bool             `yaml:"memory_decision_enabled,omitempty" toml:"memory_decision_enabled,omitempty"`
	MemoryDecisionPath             *string           `yaml:"memory_decision_path,omitempty" toml:"memory_decision_path,omitempty"`
	MemoryDecisionMaxHits          *int              `yaml:"memory_decision_max_hits,omitempty" toml:"memory_decision_max_hits,omitempty"`
	MemoryDecisionMinConfidence    *float64          `yaml:"memory_decision_min_confidence,omitempty" toml:"memory_decision_min_confidence,omitempty"`
	MemoryDecisionMinReuseScore    *float64          `yaml:"memory_decision_min_reuse_score,omitempty" toml:"memory_decision_min_reuse_score,omitempty"`
	MemoryDecisionRecipeEnabled    *bool             `yaml:"memory_decision_recipe_enabled,omitempty" toml:"memory_decision_recipe_enabled,omitempty"`
	MemoryDecisionRecipeInterval   *string           `yaml:"memory_decision_recipe_interval,omitempty" toml:"memory_decision_recipe_interval,omitempty"`
	MemoryDecisionRecipeMinSupport *int              `yaml:"memory_decision_recipe_min_support,omitempty" toml:"memory_decision_recipe_min_support,omitempty"`
	MemoryDecisionDebugEnabled     *bool             `yaml:"memory_decision_debug_enabled,omitempty" toml:"memory_decision_debug_enabled,omitempty"`
	ProviderHeaders                map[string]string `yaml:"provider_headers,omitempty" toml:"provider_headers,omitempty"`
	AnthropicVersion               *string           `yaml:"anthropic_version,omitempty" toml:"anthropic_version,omitempty"`
	AnthropicMaxTokens             *int              `yaml:"anthropic_max_tokens,omitempty" toml:"anthropic_max_tokens,omitempty"`
	MaxTurns                       *int              `yaml:"max_turns,omitempty" toml:"max_turns,omitempty"`
	WorkerMaxConcurrency           *int              `yaml:"worker_max_concurrency,omitempty" toml:"worker_max_concurrency,omitempty"`
	WorkerMaxFiles                 *int              `yaml:"worker_max_files,omitempty" toml:"worker_max_files,omitempty"`
	WorkerMaxFileChunks            *int              `yaml:"worker_max_file_chunks,omitempty" toml:"worker_max_file_chunks,omitempty"`
	ToolSelectorEnabled            *bool             `yaml:"tool_selector_enabled,omitempty" toml:"tool_selector_enabled,omitempty"`
	ToolSelectorMode               *string           `yaml:"tool_selector_mode,omitempty" toml:"tool_selector_mode,omitempty"`
	ToolSelectorModel              *string           `yaml:"tool_selector_model,omitempty" toml:"tool_selector_model,omitempty"`
	ToolSelectorTimeoutMS          *int              `yaml:"tool_selector_timeout_ms,omitempty" toml:"tool_selector_timeout_ms,omitempty"`
	ToolSelectorConfidence         *float64          `yaml:"tool_selector_confidence,omitempty" toml:"tool_selector_confidence,omitempty"`
	ToolSelectorShadow             *bool             `yaml:"tool_selector_shadow,omitempty" toml:"tool_selector_shadow,omitempty"`
	ToolSelectorRecentMsgs         *int              `yaml:"tool_selector_recent_messages,omitempty" toml:"tool_selector_recent_messages,omitempty"`
	BindAddr                       *string           `yaml:"bind_addr,omitempty" toml:"bind_addr,omitempty"`
	APIToken                       *string           `yaml:"api_token,omitempty" toml:"api_token,omitempty"`
	CORSOrigins                    []string          `yaml:"cors_origins,omitempty" toml:"cors_origins,omitempty"`
}

func configPathFromEnv() string {
	return getenvDefault("GHOST_CONFIG_PATH", defaultConfigPath)
}

func loadLegacyBridgeYAMLConfig(path string) (bridgeFileConfig, error) {
	resolvedPath, err := resolveUserPath(path)
	if err != nil {
		return bridgeFileConfig{}, fmt.Errorf("resolve config path: %w", err)
	}

	raw, err := os.ReadFile(resolvedPath)
	if err != nil {
		return bridgeFileConfig{}, err
	}
	if strings.TrimSpace(string(raw)) == "" {
		return bridgeFileConfig{}, nil
	}

	var cfg bridgeFileConfig
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return bridgeFileConfig{}, fmt.Errorf("parse config file %s: %w", resolvedPath, err)
	}
	return cfg, nil
}

func writeLegacyBridgeYAMLConfig(path string, cfg bridgeFileConfig) error {
	resolvedPath, err := resolveUserPath(path)
	if err != nil {
		return fmt.Errorf("resolve config path: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(resolvedPath), 0o755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	raw, err := yaml.Marshal(&cfg)
	if err != nil {
		return fmt.Errorf("encode config file: %w", err)
	}
	if err := os.WriteFile(resolvedPath, raw, 0o600); err != nil {
		return fmt.Errorf("write config file %s: %w", resolvedPath, err)
	}
	return nil
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
