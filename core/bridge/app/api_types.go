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
	IncludeVector     bool                   `json:"include_vector,omitempty"`
	GraphHops         int                    `json:"graph_hops,omitempty"`
	GraphPredicates   []string               `json:"graph_predicates,omitempty"`
	GraphDebug        bool                   `json:"graph_debug,omitempty"`
	VectorDebug       bool                   `json:"vector_debug,omitempty"`
	IntentDebug       bool                   `json:"intent_debug,omitempty"`
	TruthDebug        bool                   `json:"truth_debug,omitempty"`
	RerankDebug       bool                   `json:"rerank_debug,omitempty"`
	BucketDebug       bool                   `json:"bucket_debug,omitempty"`
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

type memoryDecisionQueryParams struct {
	Namespace         string                 `json:"namespace,omitempty"`
	TimeRange         *memoryTimeRangeParams `json:"time_range,omitempty"`
	Limit             int                    `json:"limit,omitempty"`
	SemanticQuery     string                 `json:"semantic_query,omitempty"`
	Keywords          []string               `json:"keywords,omitempty"`
	DecisionTypes     []string               `json:"decision_types,omitempty"`
	DecisionReuseOnly bool                   `json:"decision_reuse_only,omitempty"`
	EnvironmentStrict bool                   `json:"environment_strict,omitempty"`
	MinReuseScore     float64                `json:"min_reuse_score,omitempty"`
}

type memoryDecisionStatsParams struct {
	Namespace string `json:"namespace,omitempty"`
}

type memoryDecisionRebuildParams struct {
	Namespace      string                 `json:"namespace,omitempty"`
	TimeRange      *memoryTimeRangeParams `json:"time_range,omitempty"`
	MaxSessions    int                    `json:"max_sessions,omitempty"`
	DryRun         bool                   `json:"dry_run,omitempty"`
	IncludeRecipes *bool                  `json:"include_recipes,omitempty"`
	RebuildMemos   *bool                  `json:"rebuild_memos,omitempty"`
	ResetNamespace bool                   `json:"reset_namespace,omitempty"`
}

type memoryHygieneRunParams struct {
	Scope          string   `json:"scope,omitempty"`
	Limit          int      `json:"limit,omitempty"`
	DryRun         bool     `json:"dry_run,omitempty"`
	MinConfidence  float64  `json:"min_confidence,omitempty"`
	MaxVotesPerRun int      `json:"max_votes_per_run,omitempty"`
	ReasonCodes    []string `json:"reason_codes,omitempty"`
}

type memoryHygieneRunPayload struct {
	Scanned               int    `json:"scanned"`
	Scored                int    `json:"scored"`
	SuppressedCandidates  int    `json:"suppressed_candidates"`
	QuarantinedCandidates int    `json:"quarantined_candidates"`
	DryRun                bool   `json:"dry_run"`
	TraceID               string `json:"trace_id"`
}
