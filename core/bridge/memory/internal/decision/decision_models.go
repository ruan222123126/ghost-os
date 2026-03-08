package decision

import (
	"time"

	"ghost-os/bridge/llm"
)

// DecisionStep 表示一次决策里的关键动作摘要。
type DecisionStep struct {
	Title   string `json:"title,omitempty"`
	Summary string `json:"summary,omitempty"`
	Outcome string `json:"outcome,omitempty"`
}

// DecisionToolUse 记录一次关键工具使用概览。
type DecisionToolUse struct {
	Name          string `json:"name"`
	Purpose       string `json:"purpose,omitempty"`
	InputSummary  string `json:"input_summary,omitempty"`
	OutputSummary string `json:"output_summary,omitempty"`
}

// DecisionQuestion 记录一次人工问答节点。
type DecisionQuestion struct {
	Question   string    `json:"question"`
	Answer     string    `json:"answer,omitempty"`
	AskedAt    time.Time `json:"asked_at,omitempty"`
	AnsweredAt time.Time `json:"answered_at,omitempty"`
}

// DecisionAnsweredQuestion 解耦 session 层的人类问答结果，供 decision capture 使用。
type DecisionAnsweredQuestion struct {
	QuestionID string    `json:"question_id,omitempty"`
	Prompt     string    `json:"prompt,omitempty"`
	ToolCallID string    `json:"tool_call_id,omitempty"`
	TraceID    string    `json:"trace_id,omitempty"`
	Answer     string    `json:"answer,omitempty"`
	AskedAt    time.Time `json:"asked_at,omitempty"`
	AnsweredAt time.Time `json:"answered_at,omitempty"`
}

// DecisionCaptureInput 描述一轮完成后用于提炼 decision memo 的输入。
type DecisionCaptureInput struct {
	Namespace         string                     `json:"namespace,omitempty"`
	SessionID         string                     `json:"session_id,omitempty"`
	TraceID           string                     `json:"trace_id,omitempty"`
	TurnID            string                     `json:"turn_id,omitempty"`
	UserMessage       string                     `json:"user_message,omitempty"`
	RecentHistory     []llm.Message              `json:"-"`
	NewMessages       []llm.Message              `json:"-"`
	Outcome           string                     `json:"outcome,omitempty"`
	AnsweredQuestions []DecisionAnsweredQuestion `json:"answered_questions,omitempty"`
	SessionEnded      bool                       `json:"session_ended,omitempty"`
	TurnStartedAt     time.Time                  `json:"turn_started_at,omitempty"`
	TurnFinishedAt    time.Time                  `json:"turn_finished_at,omitempty"`
	Environment       DecisionEnvFingerprint     `json:"environment,omitempty"`
	RecipeRun         *RecipeRun                 `json:"recipe_run,omitempty"`
	RecipeFeedback    *RecipeFeedback            `json:"recipe_feedback,omitempty"`
}

// DecisionEnvFingerprint 保存可复用决策的轻量环境指纹。
type DecisionEnvFingerprint struct {
	OS               string   `json:"os,omitempty"`
	Platform         string   `json:"platform,omitempty"`
	Shell            string   `json:"shell,omitempty"`
	WorkspaceRoot    string   `json:"workspace_root,omitempty"`
	Provider         string   `json:"provider,omitempty"`
	Model            string   `json:"model,omitempty"`
	GraphNamespace   string   `json:"graph_namespace,omitempty"`
	Domain           string   `json:"domain,omitempty"`
	ToolsetSignature string   `json:"toolset_signature,omitempty"`
	PathHints        []string `json:"path_hints,omitempty"`
	TargetAppOrSite  string   `json:"target_app_or_site,omitempty"`
	NativePersistent bool     `json:"native_persistent,omitempty"`
	ToolNames        []string `json:"tool_names,omitempty"`
}

// DecisionMemo 是单次有效决策的结构化快照。
type DecisionMemo struct {
	ID               string                 `json:"id"`
	Namespace        string                 `json:"namespace,omitempty"`
	SessionID        string                 `json:"session_id,omitempty"`
	TraceID          string                 `json:"trace_id,omitempty"`
	TurnID           string                 `json:"turn_id,omitempty"`
	IntentKey        string                 `json:"intent_key,omitempty"`
	IntentSummary    string                 `json:"intent_summary,omitempty"`
	ProblemSummary   string                 `json:"problem_summary,omitempty"`
	ContextSummary   string                 `json:"context_summary,omitempty"`
	Constraints      []string               `json:"constraints,omitempty"`
	Assumptions      []string               `json:"assumptions,omitempty"`
	StrategySummary  string                 `json:"strategy_summary,omitempty"`
	KeySteps         []DecisionStep         `json:"key_steps,omitempty"`
	ToolsUsed        []DecisionToolUse      `json:"tools_used,omitempty"`
	QuestionsAsked   []DecisionQuestion     `json:"questions_asked,omitempty"`
	HumanBlocked     bool                   `json:"human_blocked,omitempty"`
	NeedsHumanFor    []string               `json:"needs_human_for,omitempty"`
	Outcome          string                 `json:"outcome,omitempty"`
	OutcomeSummary   string                 `json:"outcome_summary,omitempty"`
	FailureReasons   []string               `json:"failure_reasons,omitempty"`
	AvoidPatterns    []string               `json:"avoid_patterns,omitempty"`
	ValidationChecks []string               `json:"validation_checks,omitempty"`
	GraphNodeRefs    []string               `json:"graph_node_refs,omitempty"`
	GraphEdgeRefs    []string               `json:"graph_edge_refs,omitempty"`
	AnchorKeys       []string               `json:"anchor_keys,omitempty"`
	Confidence       float64                `json:"confidence,omitempty"`
	ReuseScore       float64                `json:"reuse_score,omitempty"`
	CreatedAt        time.Time              `json:"created_at,omitempty"`
	LastUsedAt       time.Time              `json:"last_used_at,omitempty"`
	AccessCount      int                    `json:"access_count,omitempty"`
	Environment      DecisionEnvFingerprint `json:"environment,omitempty"`
}

// RecipeStep 表示 recipe 中的顺序化建议动作。
type RecipeStep struct {
	Title       string `json:"title,omitempty"`
	Instruction string `json:"instruction,omitempty"`
	ToolName    string `json:"tool_name,omitempty"`
	Validation  string `json:"validation,omitempty"`
}

// DecisionRecipe 是从多个 memo 聚合出的可复用流程模板。
type DecisionRecipe struct {
	ID                  string       `json:"id"`
	Namespace           string       `json:"namespace,omitempty"`
	IntentKey           string       `json:"intent_key,omitempty"`
	EnvironmentKey      string       `json:"environment_key,omitempty"`
	TriggerPhrases      []string     `json:"trigger_phrases,omitempty"`
	Preconditions       []string     `json:"preconditions,omitempty"`
	StrategySummary     string       `json:"strategy_summary,omitempty"`
	RecommendedTools    []string     `json:"recommended_tools,omitempty"`
	OrderedActions      []RecipeStep `json:"ordered_actions,omitempty"`
	ValidationChecklist []string     `json:"validation_checklist,omitempty"`
	AvoidPatterns       []string     `json:"avoid_patterns,omitempty"`
	Status              string       `json:"status,omitempty"`
	SupportCount        int          `json:"support_count,omitempty"`
	SuccessRate         float64      `json:"success_rate,omitempty"`
	Confidence          float64      `json:"confidence,omitempty"`
	SelectedCount       int          `json:"selected_count,omitempty"`
	AppliedCount        int          `json:"applied_count,omitempty"`
	SuccessCount        int          `json:"success_count,omitempty"`
	PartialCount        int          `json:"partial_count,omitempty"`
	FailureCount        int          `json:"failure_count,omitempty"`
	HumanBlockedCount   int          `json:"human_blocked_count,omitempty"`
	DeviationCount      int          `json:"deviation_count,omitempty"`
	SourceMemoIDs       []string     `json:"source_memo_ids,omitempty"`
	GraphRefs           []string     `json:"graph_refs,omitempty"`
	AnchorKeys          []string     `json:"anchor_keys,omitempty"`
	LastSelectedAt      time.Time    `json:"last_selected_at,omitempty"`
	LastAppliedAt       time.Time    `json:"last_applied_at,omitempty"`
	LastOutcome         string       `json:"last_outcome,omitempty"`
	LastOutcomeAt       time.Time    `json:"last_outcome_at,omitempty"`
	CreatedAt           time.Time    `json:"created_at,omitempty"`
	UpdatedAt           time.Time    `json:"updated_at,omitempty"`
}

// RecipeAdvisory 描述选中 recipe 后的执行 scaffold。
type RecipeAdvisory struct {
	RecipeID          string   `json:"recipe_id,omitempty"`
	Source            string   `json:"source,omitempty"`
	StartWith         string   `json:"start_with,omitempty"`
	Avoid             []string `json:"avoid,omitempty"`
	Validate          []string `json:"validate,omitempty"`
	AskHumanIf        []string `json:"ask_human_if,omitempty"`
	RecommendedTools  []string `json:"recommended_tools,omitempty"`
	Confidence        float64  `json:"confidence,omitempty"`
	ApplyByDefault    bool     `json:"apply_by_default,omitempty"`
	LowConfidenceOnly bool     `json:"low_confidence_only,omitempty"`
}

// RecipeRunStep 跟踪预期步骤与实际执行之间的偏差。
type RecipeRunStep struct {
	ExpectedStep     RecipeStep `json:"expected_step,omitempty"`
	ActualTool       string     `json:"actual_tool,omitempty"`
	Matched          bool       `json:"matched,omitempty"`
	Skipped          bool       `json:"skipped,omitempty"`
	DeviationReason  string     `json:"deviation_reason,omitempty"`
	ValidationResult string     `json:"validation_result,omitempty"`
}

// RecipeFeedback 描述一次 recipe 复用后的结果判定。
type RecipeFeedback struct {
	Outcome            string   `json:"outcome,omitempty"`
	FailureReasons     []string `json:"failure_reasons,omitempty"`
	HumanBlocked       bool     `json:"human_blocked,omitempty"`
	ValidationFailures []string `json:"validation_failures,omitempty"`
	ReuseHelpful       bool     `json:"reuse_helpful,omitempty"`
}

// RecipeSelectionReport 解释为何选中或为何回退。
type RecipeSelectionReport struct {
	SelectedRecipeID string   `json:"selected_recipe_id,omitempty"`
	SelectionScore   float64  `json:"selection_score,omitempty"`
	WhySelected      []string `json:"why_selected,omitempty"`
	FallbackReason   string   `json:"fallback_reason,omitempty"`
	GrayHit          bool     `json:"gray_hit,omitempty"`
	ApplyByDefault   bool     `json:"apply_by_default,omitempty"`
}

// RecipeRun 描述某次回合对 recipe 的实际复用记录。
type RecipeRun struct {
	ID                     string                 `json:"run_id"`
	Namespace              string                 `json:"namespace,omitempty"`
	RecipeID               string                 `json:"recipe_id,omitempty"`
	SessionID              string                 `json:"session_id,omitempty"`
	TraceID                string                 `json:"trace_id,omitempty"`
	TurnID                 string                 `json:"turn_id,omitempty"`
	SelectedAt             time.Time              `json:"selected_at,omitempty"`
	CompletedAt            time.Time              `json:"completed_at,omitempty"`
	IntentKey              string                 `json:"intent_key,omitempty"`
	EnvironmentFingerprint DecisionEnvFingerprint `json:"environment_fingerprint,omitempty"`
	Selection              RecipeSelectionReport  `json:"selection,omitempty"`
	Advisory               RecipeAdvisory         `json:"advisory,omitempty"`
	Steps                  []RecipeRunStep        `json:"steps,omitempty"`
	Feedback               RecipeFeedback         `json:"feedback,omitempty"`
	ActualTools            []string               `json:"actual_tools,omitempty"`
}

// DecisionCluster 仅作为后续 distill 的持久化壳。
type DecisionCluster struct {
	ID             string    `json:"id"`
	Namespace      string    `json:"namespace,omitempty"`
	IntentKey      string    `json:"intent_key,omitempty"`
	EnvironmentKey string    `json:"environment_key,omitempty"`
	MemoIDs        []string  `json:"memo_ids,omitempty"`
	SuccessCount   int       `json:"success_count,omitempty"`
	FailureCount   int       `json:"failure_count,omitempty"`
	RecipeID       string    `json:"recipe_id,omitempty"`
	UpdatedAt      time.Time `json:"updated_at,omitempty"`
}

// DecisionHit 描述未来 query/retrieve 的统一命中视图。
type DecisionHit struct {
	Type       string    `json:"type"`
	Namespace  string    `json:"namespace,omitempty"`
	IntentKey  string    `json:"intent_key,omitempty"`
	MemoID     string    `json:"memo_id,omitempty"`
	RecipeID   string    `json:"recipe_id,omitempty"`
	SessionID  string    `json:"session_id,omitempty"`
	Summary    string    `json:"summary,omitempty"`
	Reason     string    `json:"reason,omitempty"`
	WhyMatched string    `json:"why_matched,omitempty"`
	Caution    string    `json:"caution,omitempty"`
	Outcome    string    `json:"outcome,omitempty"`
	Score      float64   `json:"score,omitempty"`
	Confidence float64   `json:"confidence,omitempty"`
	ReuseScore float64   `json:"reuse_score,omitempty"`
	Timestamp  time.Time `json:"timestamp,omitempty"`
	GraphRefs  []string  `json:"graph_refs,omitempty"`
	AnchorKeys []string  `json:"anchor_keys,omitempty"`
}

// DecisionStats 描述当前 namespace 下的 sidecar 快照。
type DecisionStats struct {
	Namespace          string `json:"namespace,omitempty"`
	MemoCount          int    `json:"memo_count"`
	RecipeCount        int    `json:"recipe_count"`
	ClusterCount       int    `json:"cluster_count"`
	SuccessCount       int    `json:"success_count"`
	PartialCount       int    `json:"partial_count"`
	FailureCount       int    `json:"failure_count"`
	AwaitingHumanCount int    `json:"awaiting_human_count"`
	CancelledCount     int    `json:"cancelled_count"`
	HumanBlockedCount  int    `json:"human_blocked_count"`
}

// DecisionDistillStats 描述一次 recipe 蒸馏批次的输出摘要。
type DecisionDistillStats struct {
	Namespace       string `json:"namespace,omitempty"`
	MemosScanned    int    `json:"memos_scanned"`
	ClustersBuilt   int    `json:"clusters_built"`
	RecipesCreated  int    `json:"recipes_created"`
	RecipesUpdated  int    `json:"recipes_updated"`
	WarningsFolded  int    `json:"warnings_folded"`
	SkippedClusters int    `json:"skipped_clusters"`
}

// DecisionRebuildOptions 控制 decision memo/recipe 的离线重建流程。
type DecisionRebuildOptions struct {
	Namespace        string     `json:"namespace,omitempty"`
	TimeRange        *TimeRange `json:"time_range,omitempty"`
	MaxSessions      int        `json:"max_sessions,omitempty"`
	BatchSize        int        `json:"batch_size,omitempty"`
	CursorCheckpoint string     `json:"cursor_checkpoint,omitempty"`
	DryRun           bool       `json:"dry_run,omitempty"`
	IncludeRecipes   bool       `json:"include_recipes,omitempty"`
	RebuildMemos     bool       `json:"rebuild_memos,omitempty"`
	ResetNamespace   bool       `json:"reset_namespace,omitempty"`
}

// DecisionRebuildStats 描述一次离线 backfill/rebuild 的处理计数。
type DecisionRebuildStats struct {
	Namespace         string `json:"namespace,omitempty"`
	SessionsScanned   int    `json:"sessions_scanned"`
	MarkdownScanned   int    `json:"markdown_scanned"`
	MemosCaptured     int    `json:"memos_captured"`
	RecipesCreated    int    `json:"recipes_created"`
	RecipesUpdated    int    `json:"recipes_updated"`
	RecipeRunsCreated int    `json:"recipe_runs_created"`
	RecipeRunsUpdated int    `json:"recipe_runs_updated"`
	ClustersUpdated   int    `json:"clusters_updated"`
	SkippedEntries    int    `json:"skipped_entries"`
	CursorCheckpoint  string `json:"cursor_checkpoint,omitempty"`
}
