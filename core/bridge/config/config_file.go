package config

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
	Name                       string
	Type                       llm.Provider
	BaseURL                    string
	APIKey                     *string
	Models                     []string
	ContextWindowTokens        int
	ResponseReserveTokens      int
	ModelContextWindowTokens   map[string]int
	ModelResponseReserveTokens map[string]int
}

type providerFileConfig struct {
	Type                       llm.Provider   `toml:"type,omitempty"`
	BaseURL                    string         `toml:"base_url,omitempty"`
	APIKey                     *string        `toml:"api_key,omitempty"`
	Models                     []string       `toml:"models,omitempty"`
	ContextWindowTokens        int            `toml:"context_window_tokens,omitempty"`
	ResponseReserveTokens      int            `toml:"response_reserve_tokens,omitempty"`
	ModelContextWindowTokens   map[string]int `toml:"model_context_window_tokens,omitempty"`
	ModelResponseReserveTokens map[string]int `toml:"model_response_reserve_tokens,omitempty"`
}

type legacyProviderConfig struct {
	Name    string       `toml:"name,omitempty"`
	Type    llm.Provider `toml:"type,omitempty"`
	BaseURL string       `toml:"base_url,omitempty"`
	APIKey  *string      `toml:"api_key,omitempty"`
	Models  []string     `toml:"models,omitempty"`
}

type bridgeFileConfig struct {
	ActiveProvider                        *string                       `toml:"active_provider,omitempty"`
	Providers                             map[string]providerFileConfig `toml:"providers,omitempty"`
	ModelProvider                         *string                       `toml:"model_provider,omitempty"`
	ModelProviders                        []legacyProviderConfig        `toml:"model_providers,omitempty"`
	Provider                              *string                       `toml:"provider,omitempty"`
	APIKey                                *string                       `toml:"api_key,omitempty"`
	BaseURL                               *string                       `toml:"base_url,omitempty"`
	Model                                 *string                       `toml:"model,omitempty"`
	ChatPath                              *string                       `toml:"chat_path,omitempty"`
	ProjectRoot                           *string                       `toml:"project_root,omitempty"`
	NativePersistent                      *bool                         `toml:"native_persistent,omitempty"`
	NativeBinaryPath                      *string                       `toml:"native_binary_path,omitempty"`
	NativeBinaryRoots                     []string                      `toml:"native_binary_roots,omitempty"`
	NativeBinaryCandidates                []string                      `toml:"native_binary_candidates,omitempty"`
	NativeAllowedReadPaths                []string                      `toml:"native_allowed_read_paths,omitempty"`
	NativeAllowedWritePaths               []string                      `toml:"native_allowed_write_paths,omitempty"`
	WorkerModel                           *string                       `toml:"worker_model,omitempty"`
	PromptsPath                           *string                       `toml:"prompts_path,omitempty"`
	PromptsDir                            *string                       `toml:"prompts_dir,omitempty"`
	PromptsCoreFiles                      []string                      `toml:"prompts_core_files,omitempty"`
	SessionsPath                          *string                       `toml:"sessions_path,omitempty"`
	RSSFeedsPath                          *string                       `toml:"rss_feeds_path,omitempty"`
	RSSInboxPath                          *string                       `toml:"rss_inbox_path,omitempty"`
	RSSBriefingsPath                      *string                       `toml:"rss_briefings_path,omitempty"`
	RSSReportsPath                        *string                       `toml:"rss_reports_path,omitempty"`
	RSSPollEnabled                        *bool                         `toml:"rss_poll_enabled,omitempty"`
	RSSPollInterval                       *string                       `toml:"rss_poll_interval,omitempty"`
	RSSPollMaxItemsPerFeed                *int                          `toml:"rss_poll_max_items_per_feed,omitempty"`
	RSSAIBatchSize                        *int                          `toml:"rss_ai_batch_size,omitempty"`
	RSSBriefingEnabled                    *bool                         `toml:"rss_briefing_enabled,omitempty"`
	RSSBriefingInterval                   *string                       `toml:"rss_briefing_interval,omitempty"`
	WebSearchTavilyAPIKey                 *string                       `toml:"web_search_tavily_api_key,omitempty"`
	ProviderHeaders                       map[string]string             `toml:"provider_headers,omitempty"`
	AnthropicVersion                      *string                       `toml:"anthropic_version,omitempty"`
	AnthropicMaxTokens                    *int                          `toml:"anthropic_max_tokens,omitempty"`
	ProMaxIterations                      *int                          `toml:"pro_max_iterations,omitempty"`
	MaxTurns                              *int                          `toml:"max_turns,omitempty"`
	WorkerMaxConcurrency                  *int                          `toml:"worker_max_concurrency,omitempty"`
	WorkerMaxFiles                        *int                          `toml:"worker_max_files,omitempty"`
	WorkerMaxFileChunks                   *int                          `toml:"worker_max_file_chunks,omitempty"`
	ToolSelectorEnabled                   *bool                         `toml:"tool_selector_enabled,omitempty"`
	ToolSelectorMode                      *string                       `toml:"tool_selector_mode,omitempty"`
	ToolSelectorModel                     *string                       `toml:"tool_selector_model,omitempty"`
	ToolSelectorTimeoutMS                 *int                          `toml:"tool_selector_timeout_ms,omitempty"`
	ToolSelectorConfidence                *float64                      `toml:"tool_selector_confidence,omitempty"`
	ToolSelectorShadow                    *bool                         `toml:"tool_selector_shadow,omitempty"`
	ToolSelectorRecentMsgs                *int                          `toml:"tool_selector_recent_messages,omitempty"`
	ToolAllowlistOnly                     *bool                         `toml:"tool_allowlist_only,omitempty"`
	ToolAllowlist                         []string                      `toml:"tool_allowlist,omitempty"`
	ToolBlocklist                         []string                      `toml:"tool_blocklist,omitempty"`
	MemoryAugmentationEnabled             *bool                         `toml:"memory_augmentation_enabled,omitempty"`
	MemoryAugmentationLearningEnabled     *bool                         `toml:"memory_augmentation_learning_enabled,omitempty"`
	MemoryAugmentationRecallEnabled       *bool                         `toml:"memory_augmentation_recall_enabled,omitempty"`
	MemoryAugmentationMaxRecallItems      *int                          `toml:"memory_augmentation_max_recall_items,omitempty"`
	MemoryAugmentationMinConfidence       *float64                      `toml:"memory_augmentation_min_confidence,omitempty"`
	MemoryAugmentationSessionScopeEnabled *bool                         `toml:"memory_augmentation_session_scope_enabled,omitempty"`
	MemoryAugmentationUserScopeEnabled    *bool                         `toml:"memory_augmentation_user_scope_enabled,omitempty"`
	MemoryAugmentationLLMModel            *string                       `toml:"memory_augmentation_llm_model,omitempty"`
	MemoryAugmentationUserScopeID         *string                       `toml:"memory_augmentation_user_scope_id,omitempty"`
	BindAddr                              *string                       `toml:"bind_addr,omitempty"`
	APIToken                              *string                       `toml:"api_token,omitempty"`
	CORSOrigins                           []string                      `toml:"cors_origins,omitempty"`
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

func parseStringCSV(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}

	values := make([]string, 0)
	for _, item := range strings.Split(raw, ",") {
		if value := strings.TrimSpace(item); value != "" {
			values = append(values, value)
		}
	}
	if len(values) == 0 {
		return nil
	}
	return values
}

func normalizeConfiguredPathList(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		normalized = append(normalized, trimmed)
	}
	if len(normalized) == 0 {
		return nil
	}
	return normalized
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
