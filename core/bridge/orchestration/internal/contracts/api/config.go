package api

type ConfigResponse struct {
	Provider                          string `json:"provider"`
	ProviderType                      string `json:"provider_type"`
	BaseURL                           string `json:"base_url"`
	Model                             string `json:"model"`
	ChatPath                          string `json:"chat_path"`
	ProjectRoot                       string `json:"project_root"`
	MaxTurns                          int    `json:"max_turns"`
	TaskExecutionTimeoutMs            int    `json:"task_execution_timeout_ms"`
	RelayDefaultStopPolicy            string `json:"relay_default_stop_policy"`
	RelayDefaultMaxRounds             int    `json:"relay_default_max_rounds"`
	RelayDefaultExecutionTimeoutMs    int    `json:"relay_default_execution_timeout_ms"`
	LlmCompletionRetryCount           int    `json:"llm_completion_retry_count"`
	LlmCompletionRetryIntervalMs      int    `json:"llm_completion_retry_interval_ms"`
	APIKeySet                         bool   `json:"api_key_set"`
	ModelSelectionEnabled             bool   `json:"model_selection_enabled"`
	SessionHumanLogFullEnabled        bool   `json:"session_human_log_full_enabled"`
	SessionSystemPromptVisibleEnabled bool   `json:"session_system_prompt_visible_enabled"`
	AssistantMarkdownEnabled          bool   `json:"assistant_markdown_enabled"`
	ToolCallCompactOutputEnabled      bool   `json:"tool_call_compact_output_enabled"`
	MemoryModeEnabled                 bool   `json:"memory_mode_enabled"`
	MicrocompactEnabled               bool   `json:"microcompact_enabled"`
	SessionTitleMode                  string `json:"session_title_mode"`
	WebSearchTavilyURL                string `json:"web_search_tavily_url"`
	WebSearchExaURL                   string `json:"web_search_exa_url"`
	WebSearchTavilyAPIKeySet          bool   `json:"web_search_tavily_api_key_set"`
	WebSearchExaAPIKeySet             bool   `json:"web_search_exa_api_key_set"`
}

type ConfigUpdateRequest struct {
	Provider                          *string `json:"provider,omitempty"`
	APIKey                            *string `json:"api_key,omitempty"`
	BaseURL                           *string `json:"base_url,omitempty"`
	Model                             *string `json:"model,omitempty"`
	ChatPath                          *string `json:"chat_path,omitempty"`
	ProjectRoot                       *string `json:"project_root,omitempty"`
	MaxTurns                          *int    `json:"max_turns,omitempty"`
	TaskExecutionTimeoutMs            *int    `json:"task_execution_timeout_ms,omitempty"`
	RelayDefaultStopPolicy            *string `json:"relay_default_stop_policy,omitempty"`
	RelayDefaultMaxRounds             *int    `json:"relay_default_max_rounds,omitempty"`
	RelayDefaultExecutionTimeoutMs    *int    `json:"relay_default_execution_timeout_ms,omitempty"`
	LlmCompletionRetryCount           *int    `json:"llm_completion_retry_count,omitempty"`
	LlmCompletionRetryIntervalMs      *int    `json:"llm_completion_retry_interval_ms,omitempty"`
	SessionHumanLogFullEnabled        *bool   `json:"session_human_log_full_enabled,omitempty"`
	SessionSystemPromptVisibleEnabled *bool   `json:"session_system_prompt_visible_enabled,omitempty"`
	AssistantMarkdownEnabled          *bool   `json:"assistant_markdown_enabled,omitempty"`
	ToolCallCompactOutputEnabled      *bool   `json:"tool_call_compact_output_enabled,omitempty"`
	MemoryModeEnabled                 *bool   `json:"memory_mode_enabled,omitempty"`
	MicrocompactEnabled               *bool   `json:"microcompact_enabled,omitempty"`
	SessionTitleMode                  *string `json:"session_title_mode,omitempty"`
	WebSearchTavilyURL                *string `json:"web_search_tavily_url,omitempty"`
	WebSearchExaURL                   *string `json:"web_search_exa_url,omitempty"`
	WebSearchTavilyAPIKey             *string `json:"web_search_tavily_api_key,omitempty"`
	WebSearchExaAPIKey                *string `json:"web_search_exa_api_key,omitempty"`
	TraceID                           string  `json:"trace_id,omitempty"`
}

type ProviderConfigResponse struct {
	Name                       string         `json:"name"`
	Type                       string         `json:"type"`
	BaseURL                    string         `json:"base_url"`
	Models                     []string       `json:"models,omitempty"`
	ContextWindowTokens        int            `json:"context_window_tokens,omitempty"`
	ResponseReserveTokens      int            `json:"response_reserve_tokens,omitempty"`
	ModelContextWindowTokens   map[string]int `json:"model_context_window_tokens,omitempty"`
	ModelResponseReserveTokens map[string]int `json:"model_response_reserve_tokens,omitempty"`
	APIKeySet                  bool           `json:"api_key_set"`
}

type ProviderConfigInput struct {
	Name                       string         `json:"name"`
	Type                       string         `json:"type"`
	BaseURL                    *string        `json:"base_url,omitempty"`
	APIKey                     *string        `json:"api_key,omitempty"`
	Models                     []string       `json:"models,omitempty"`
	ContextWindowTokens        int            `json:"context_window_tokens,omitempty"`
	ResponseReserveTokens      int            `json:"response_reserve_tokens,omitempty"`
	ModelContextWindowTokens   map[string]int `json:"model_context_window_tokens,omitempty"`
	ModelResponseReserveTokens map[string]int `json:"model_response_reserve_tokens,omitempty"`
	TraceID                    string         `json:"trace_id,omitempty"`
}

type ProviderListResponse struct {
	Providers      []ProviderConfigResponse `json:"providers"`
	ActiveProvider string                   `json:"active_provider"`
}

type SetActiveProviderRequest struct {
	Name    string `json:"name"`
	TraceID string `json:"trace_id,omitempty"`
}
