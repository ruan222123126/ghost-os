package app

const defaultMaxRequestBodyBytes int64 = 1 << 20

type agentParams struct {
	Message   string `json:"message"`
	SessionID string `json:"session_id,omitempty"`
}

type agentStopParams struct {
	SessionID string `json:"session_id,omitempty"`
	TraceID   string `json:"trace_id,omitempty"`
}

type sessionIDParams struct {
	ID string `json:"id"`
}

type sessionDeleteResponse struct {
	ID      string `json:"id"`
	Deleted bool   `json:"deleted"`
}

type providerCreateRequest = providerConfigInput

type providerUpdateRequest = providerConfigInput

type memoryTimeRangeParams struct {
	Start string `json:"start,omitempty"`
	End   string `json:"end,omitempty"`
}

type memoryQueryParams struct {
	SessionID         string                 `json:"session_id,omitempty"`
	TimeRange         *memoryTimeRangeParams `json:"time_range,omitempty"`
	Limit             int                    `json:"limit,omitempty"`
	Keywords          []string               `json:"keywords,omitempty"`
	Metadata          map[string]any         `json:"metadata,omitempty"`
	IncludeMarkdown   bool                   `json:"include_markdown,omitempty"`
	IncludeGraph      *bool                  `json:"include_graph,omitempty"`
	IncludeDecision   *bool                  `json:"include_decision,omitempty"`
	SemanticQuery     string                 `json:"semantic_query,omitempty"`
	GraphHops         int                    `json:"graph_hops,omitempty"`
	GraphPredicates   []string               `json:"graph_predicates,omitempty"`
	GraphDebug        bool                   `json:"graph_debug,omitempty"`
	DecisionDebug     bool                   `json:"decision_debug,omitempty"`
	DecisionReuseOnly bool                   `json:"decision_reuse_only,omitempty"`
	DecisionTypes     []string               `json:"decision_types,omitempty"`
	EnvironmentStrict bool                   `json:"environment_strict,omitempty"`
	MinReuseScore     float64                `json:"min_reuse_score,omitempty"`
}

type memoryArchiveParams struct {
	SessionID string `json:"session_id"`
}

type memoryArchiveResponse struct {
	SessionID string `json:"session_id"`
	Archived  bool   `json:"archived"`
}
