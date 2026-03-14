package config

import "ghost-os/bridge/llm"

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
	WebSearchExaAPIKey                    *string                       `toml:"web_search_exa_api_key,omitempty"`
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
	ToolSearchEnabled                     *bool                         `toml:"tool_search_enabled,omitempty"`
	ToolSearchIdleTurns                   *int                          `toml:"tool_search_idle_turns,omitempty"`
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
