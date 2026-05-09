package config

// Snapshot is the public runtime config layer exposed outside config.
type Snapshot struct {
	Provider                     string `json:"provider"`
	ProviderType                 string `json:"provider_type"`
	BaseURL                      string `json:"base_url"`
	Model                        string `json:"model"`
	ChatPath                     string `json:"chat_path"`
	ProjectRoot                  string `json:"project_root"`
	MaxTurns                     int    `json:"max_turns"`
	TaskExecutionTimeoutMS       int    `json:"task_execution_timeout_ms"`
	LLMCompletionRetryCount      int    `json:"llm_completion_retry_count"`
	LLMCompletionRetryIntervalMS int    `json:"llm_completion_retry_interval_ms"`
	APIKeySet                    bool   `json:"api_key_set"`
	ModelSelectionEnabled        bool   `json:"model_selection_enabled"`
	SessionHumanLogFullEnabled   bool   `json:"session_human_log_full_enabled"`
	SessionSystemPromptVisible   bool   `json:"session_system_prompt_visible_enabled"`
	AssistantMarkdownEnabled     bool   `json:"assistant_markdown_enabled"`
	ToolCallCompactOutputEnabled bool   `json:"tool_call_compact_output_enabled"`
	MemoryModeEnabled            bool   `json:"memory_mode_enabled"`
	MicrocompactEnabled          bool   `json:"microcompact_enabled"`
	WebSearchTavilyURL           string `json:"web_search_tavily_url"`
	WebSearchExaURL              string `json:"web_search_exa_url"`
	WebSearchTavilyAPIKeySet     bool   `json:"web_search_tavily_api_key_set"`
	WebSearchExaAPIKeySet        bool   `json:"web_search_exa_api_key_set"`
}

type UpdateRequest struct {
	Provider                     *string `json:"provider,omitempty"`
	APIKey                       *string `json:"api_key,omitempty"`
	BaseURL                      *string `json:"base_url,omitempty"`
	Model                        *string `json:"model,omitempty"`
	ChatPath                     *string `json:"chat_path,omitempty"`
	ProjectRoot                  *string `json:"project_root,omitempty"`
	MaxTurns                     *int    `json:"max_turns,omitempty"`
	TaskExecutionTimeoutMS       *int    `json:"task_execution_timeout_ms,omitempty"`
	LLMCompletionRetryCount      *int    `json:"llm_completion_retry_count,omitempty"`
	LLMCompletionRetryIntervalMS *int    `json:"llm_completion_retry_interval_ms,omitempty"`
	SessionHumanLogFullEnabled   *bool   `json:"session_human_log_full_enabled,omitempty"`
	SessionSystemPromptVisible   *bool   `json:"session_system_prompt_visible_enabled,omitempty"`
	AssistantMarkdownEnabled     *bool   `json:"assistant_markdown_enabled,omitempty"`
	ToolCallCompactOutputEnabled *bool   `json:"tool_call_compact_output_enabled,omitempty"`
	MemoryModeEnabled            *bool   `json:"memory_mode_enabled,omitempty"`
	MicrocompactEnabled          *bool   `json:"microcompact_enabled,omitempty"`
	WebSearchTavilyURL           *string `json:"web_search_tavily_url,omitempty"`
	WebSearchExaURL              *string `json:"web_search_exa_url,omitempty"`
	WebSearchTavilyAPIKey        *string `json:"web_search_tavily_api_key,omitempty"`
	WebSearchExaAPIKey           *string `json:"web_search_exa_api_key,omitempty"`
	TraceID                      string  `json:"trace_id,omitempty"`
}

type configUpdateRequest = UpdateRequest
