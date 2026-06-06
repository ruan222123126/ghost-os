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

type bridgeFileConfig struct {
	ActiveProvider                 *string                       `toml:"active_provider,omitempty"`
	Providers                      map[string]providerFileConfig `toml:"providers,omitempty"`
	Model                          *string                       `toml:"model,omitempty"`
	ModelSelectionEnabled          *bool                         `toml:"model_selection_enabled,omitempty"`
	ChatPath                       *string                       `toml:"chat_path,omitempty"`
	ResponsePromptCacheKey         *string                       `toml:"response_prompt_cache_key,omitempty"`
	ResponsePromptCacheRetention   *string                       `toml:"response_prompt_cache_retention,omitempty"`
	ResponseSafetyIdentifier       *string                       `toml:"response_safety_identifier,omitempty"`
	ResponseStore                  *bool                         `toml:"response_store,omitempty"`
	ResponseMetadata               map[string]string             `toml:"response_metadata,omitempty"`
	CodexStatelessRetryEnabled     *bool                         `toml:"codex_stateless_retry_enabled,omitempty"`
	ProjectRoot                    *string                       `toml:"project_root,omitempty"`
	ScriptExecSandboxMemoryMB      *int                          `toml:"script_exec_sandbox_memory_mb,omitempty"`
	NativePersistent               *bool                         `toml:"native_persistent,omitempty"`
	NativeBinaryPath               *string                       `toml:"native_binary_path,omitempty"`
	NativeBinaryRoots              []string                      `toml:"native_binary_roots,omitempty"`
	NativeBinaryCandidates         []string                      `toml:"native_binary_candidates,omitempty"`
	CodexCLIPath                   *string                       `toml:"codex_cli_path,omitempty"`
	NodeBinPath                    *string                       `toml:"node_bin_path,omitempty"`
	NativeAllowedReadPaths         []string                      `toml:"native_allowed_read_paths,omitempty"`
	NativeAllowedWritePaths        []string                      `toml:"native_allowed_write_paths,omitempty"`
	WorkerModel                    *string                       `toml:"worker_model,omitempty"`
	PromptsPath                    *string                       `toml:"prompts_path,omitempty"`
	PromptsDir                     *string                       `toml:"prompts_dir,omitempty"`
	PromptsCoreFiles               []string                      `toml:"prompts_core_files,omitempty"`
	PromptsRuntimeConstraintFiles  []string                      `toml:"prompts_runtime_constraint_files,omitempty"`
	PromptsResponseRuleFiles       []string                      `toml:"prompts_response_rule_files,omitempty"`
	TasksPath                      *string                       `toml:"tasks_path,omitempty"`
	TaskExecutionTimeoutMS         *int                          `toml:"task_execution_timeout_ms,omitempty"`
	WorkflowToolAllowlist          []string                      `toml:"workflow_tool_allowlist,omitempty"`
	SessionsPath                   *string                       `toml:"sessions_path,omitempty"`
	SessionHumanLogFullEnabled     *bool                         `toml:"session_human_log_full_enabled,omitempty"`
	SessionSystemPromptVisible     *bool                         `toml:"session_system_prompt_visible_enabled,omitempty"`
	AssistantMarkdownEnabled       *bool                         `toml:"assistant_markdown_enabled,omitempty"`
	ToolCallCompactOutputEnabled   *bool                         `toml:"tool_call_compact_output_enabled,omitempty"`
	MemoryModeEnabled              *bool                         `toml:"memory_mode_enabled,omitempty"`
	MicrocompactEnabled            *bool                         `toml:"microcompact_enabled,omitempty"`
	SessionTitleMode               *string                       `toml:"session_title_mode,omitempty"`
	WebSearchTavilyURL             *string                       `toml:"web_search_tavily_url,omitempty"`
	WebSearchExaURL                *string                       `toml:"web_search_exa_url,omitempty"`
	WebSearchTavilyAPIKey          *string                       `toml:"web_search_tavily_api_key,omitempty"`
	WebSearchExaAPIKey             *string                       `toml:"web_search_exa_api_key,omitempty"`
	LLMCompletionRetryCount        *int                          `toml:"llm_completion_retry_count,omitempty"`
	LLMCompletionRetryIntervalMS   *int                          `toml:"llm_completion_retry_interval_ms,omitempty"`
	ProviderHeaders                map[string]string             `toml:"provider_headers,omitempty"`
	AnthropicVersion               *string                       `toml:"anthropic_version,omitempty"`
	AnthropicMaxTokens             *int                          `toml:"anthropic_max_tokens,omitempty"`
	RelayDefaultStopPolicy         *string                       `toml:"relay_default_stop_policy,omitempty"`
	RelayDefaultMaxRounds          *int                          `toml:"relay_default_max_rounds,omitempty"`
	RelayDefaultExecutionTimeoutMS *int                          `toml:"relay_default_execution_timeout_ms,omitempty"`
	MaxTurns                       *int                          `toml:"max_turns,omitempty"`
	WorkerMaxConcurrency           *int                          `toml:"worker_max_concurrency,omitempty"`
	WorkerMaxFiles                 *int                          `toml:"worker_max_files,omitempty"`
	WorkerMaxFileChunks            *int                          `toml:"worker_max_file_chunks,omitempty"`
	ToolSelectorEnabled            *bool                         `toml:"tool_selector_enabled,omitempty"`
	ToolSelectorMode               *string                       `toml:"tool_selector_mode,omitempty"`
	ToolSelectorModel              *string                       `toml:"tool_selector_model,omitempty"`
	ToolSelectorTimeoutMS          *int                          `toml:"tool_selector_timeout_ms,omitempty"`
	ToolSelectorConfidence         *float64                      `toml:"tool_selector_confidence,omitempty"`
	ToolSelectorShadow             *bool                         `toml:"tool_selector_shadow,omitempty"`
	ToolSelectorRecentMsgs         *int                          `toml:"tool_selector_recent_messages,omitempty"`
	ToolSearchEnabled              *bool                         `toml:"tool_search_enabled,omitempty"`
	ToolSearchIdleTurns            *int                          `toml:"tool_search_idle_turns,omitempty"`
	ToolAllowlistOnly              *bool                         `toml:"tool_allowlist_only,omitempty"`
	ToolAllowlist                  []string                      `toml:"tool_allowlist,omitempty"`
	ToolBlocklist                  []string                      `toml:"tool_blocklist,omitempty"`
	SkillBlocklist                 []string                      `toml:"skill_blocklist,omitempty"`
	ToolPromptOverrides            map[string]string             `toml:"tool_prompt_overrides,omitempty"`
	BindAddr                       *string                       `toml:"bind_addr,omitempty"`
	APIToken                       *string                       `toml:"api_token,omitempty"`
	CORSOrigins                    []string                      `toml:"cors_origins,omitempty"`
}
