package config

// Snapshot is the public runtime config layer exposed outside config.
type Snapshot struct {
	Provider                   string                          `json:"provider"`
	ProviderType               string                          `json:"provider_type"`
	BaseURL                    string                          `json:"base_url"`
	Model                      string                          `json:"model"`
	ChatPath                   string                          `json:"chat_path"`
	APIKeySet                  bool                            `json:"api_key_set"`
	ModelSelectionEnabled      bool                            `json:"model_selection_enabled"`
	GraphQLDefaultSource       string                          `json:"graphql_default_source"`
	GraphQLToolRuntimeEnabled  bool                            `json:"graphql_tool_runtime_enabled"`
	GraphQLTextSanitizeEnabled bool                            `json:"graphql_text_sanitize_enabled"`
	GraphQLSources             []GraphQLSourceSnapshot         `json:"graphql_sources"`
	GraphQLMutationPolicies    []GraphQLMutationPolicySnapshot `json:"graphql_mutation_policies"`
	SessionHumanLogFullEnabled bool                            `json:"session_human_log_full_enabled"`
	WebRooterEnabled           bool                            `json:"web_rooter_enabled"`
	WebRooterBaseURL           string                          `json:"web_rooter_base_url"`
	WebRooterTimeoutMS         int                             `json:"web_rooter_timeout_ms"`
	WebRooterAPITokenSet       bool                            `json:"web_rooter_api_token_set"`
	WebSearchTavilyURL         string                          `json:"web_search_tavily_url"`
	WebSearchExaURL            string                          `json:"web_search_exa_url"`
	WebSearchTavilyAPIKeySet   bool                            `json:"web_search_tavily_api_key_set"`
	WebSearchExaAPIKeySet      bool                            `json:"web_search_exa_api_key_set"`
}

type UpdateRequest struct {
	Provider                   *string                      `json:"provider,omitempty"`
	APIKey                     *string                      `json:"api_key,omitempty"`
	BaseURL                    *string                      `json:"base_url,omitempty"`
	Model                      *string                      `json:"model,omitempty"`
	ChatPath                   *string                      `json:"chat_path,omitempty"`
	GraphQLDefaultSource       *string                      `json:"graphql_default_source,omitempty"`
	GraphQLToolRuntimeEnabled  *bool                        `json:"graphql_tool_runtime_enabled,omitempty"`
	GraphQLTextSanitizeEnabled *bool                        `json:"graphql_text_sanitize_enabled,omitempty"`
	GraphQLSources             []GraphQLSourceInput         `json:"graphql_sources,omitempty"`
	GraphQLSourceUpsert        *GraphQLSourceInput          `json:"graphql_source_upsert,omitempty"`
	GraphQLMutationPolicies    []GraphQLMutationPolicyInput `json:"graphql_mutation_policies,omitempty"`
	SessionHumanLogFullEnabled *bool                        `json:"session_human_log_full_enabled,omitempty"`
	WebRooterEnabled           *bool                        `json:"web_rooter_enabled,omitempty"`
	WebRooterBaseURL           *string                      `json:"web_rooter_base_url,omitempty"`
	WebRooterAPIToken          *string                      `json:"web_rooter_api_token,omitempty"`
	WebRooterTimeoutMS         *int                         `json:"web_rooter_timeout_ms,omitempty"`
	WebSearchTavilyURL         *string                      `json:"web_search_tavily_url,omitempty"`
	WebSearchExaURL            *string                      `json:"web_search_exa_url,omitempty"`
	WebSearchTavilyAPIKey      *string                      `json:"web_search_tavily_api_key,omitempty"`
	WebSearchExaAPIKey         *string                      `json:"web_search_exa_api_key,omitempty"`
	TraceID                    string                       `json:"trace_id,omitempty"`
}

type configUpdateRequest = UpdateRequest
