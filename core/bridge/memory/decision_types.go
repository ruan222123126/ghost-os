package memory

import (
	"strings"
	"time"
)

const (
	DecisionOutcomeSuccess       = "success"
	DecisionOutcomePartial       = "partial"
	DecisionOutcomeFailure       = "failure"
	DecisionOutcomeAwaitingHuman = "awaiting_human"
	DecisionOutcomeCancelled     = "cancelled"

	DecisionHitTypeMemo    = "memo"
	DecisionHitTypeRecipe  = "recipe"
	DecisionHitTypeWarning = "warning"

	defaultDecisionNamespace        = "default"
	defaultDecisionMaxHits          = 4
	defaultDecisionMinConfidence    = 0.75
	defaultDecisionMinReuseScore    = 0.70
	defaultDecisionRecipeInterval   = 6 * time.Hour
	defaultDecisionRecipeMinSupport = 3
	defaultDecisionMemosPathName    = "memos.json"
	defaultDecisionRecipesPathName  = "recipes.json"
	defaultDecisionClustersPathName = "clusters.json"
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

// DecisionEnvFingerprint 保存可复用决策的轻量环境指纹。
type DecisionEnvFingerprint struct {
	OS               string   `json:"os,omitempty"`
	Shell            string   `json:"shell,omitempty"`
	WorkspaceRoot    string   `json:"workspace_root,omitempty"`
	Provider         string   `json:"provider,omitempty"`
	Model            string   `json:"model,omitempty"`
	GraphNamespace   string   `json:"graph_namespace,omitempty"`
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
	TriggerPhrases      []string     `json:"trigger_phrases,omitempty"`
	Preconditions       []string     `json:"preconditions,omitempty"`
	StrategySummary     string       `json:"strategy_summary,omitempty"`
	RecommendedTools    []string     `json:"recommended_tools,omitempty"`
	OrderedActions      []RecipeStep `json:"ordered_actions,omitempty"`
	ValidationChecklist []string     `json:"validation_checklist,omitempty"`
	AvoidPatterns       []string     `json:"avoid_patterns,omitempty"`
	SupportCount        int          `json:"support_count,omitempty"`
	SuccessRate         float64      `json:"success_rate,omitempty"`
	Confidence          float64      `json:"confidence,omitempty"`
	SourceMemoIDs       []string     `json:"source_memo_ids,omitempty"`
	GraphRefs           []string     `json:"graph_refs,omitempty"`
	AnchorKeys          []string     `json:"anchor_keys,omitempty"`
	CreatedAt           time.Time    `json:"created_at,omitempty"`
	UpdatedAt           time.Time    `json:"updated_at,omitempty"`
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
	Type       string   `json:"type"`
	Namespace  string   `json:"namespace,omitempty"`
	IntentKey  string   `json:"intent_key,omitempty"`
	MemoID     string   `json:"memo_id,omitempty"`
	RecipeID   string   `json:"recipe_id,omitempty"`
	Summary    string   `json:"summary,omitempty"`
	Reason     string   `json:"reason,omitempty"`
	Outcome    string   `json:"outcome,omitempty"`
	Score      float64  `json:"score,omitempty"`
	Confidence float64  `json:"confidence,omitempty"`
	ReuseScore float64  `json:"reuse_score,omitempty"`
	GraphRefs  []string `json:"graph_refs,omitempty"`
	AnchorKeys []string `json:"anchor_keys,omitempty"`
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

func normalizeDecisionNamespace(namespace string) string {
	trimmed := strings.TrimSpace(namespace)
	if trimmed == "" {
		return defaultDecisionNamespace
	}
	return trimmed
}

func normalizeDecisionOutcome(outcome string) string {
	switch strings.ToLower(strings.TrimSpace(outcome)) {
	case DecisionOutcomeSuccess:
		return DecisionOutcomeSuccess
	case DecisionOutcomePartial:
		return DecisionOutcomePartial
	case DecisionOutcomeFailure:
		return DecisionOutcomeFailure
	case DecisionOutcomeAwaitingHuman:
		return DecisionOutcomeAwaitingHuman
	case DecisionOutcomeCancelled:
		return DecisionOutcomeCancelled
	default:
		return ""
	}
}

func normalizeDecisionHitType(hitType string) string {
	switch strings.ToLower(strings.TrimSpace(hitType)) {
	case DecisionHitTypeMemo:
		return DecisionHitTypeMemo
	case DecisionHitTypeRecipe:
		return DecisionHitTypeRecipe
	case DecisionHitTypeWarning:
		return DecisionHitTypeWarning
	default:
		return ""
	}
}

func normalizeDecisionStep(step DecisionStep) DecisionStep {
	out := step
	out.Title = strings.TrimSpace(out.Title)
	out.Summary = strings.TrimSpace(out.Summary)
	out.Outcome = normalizeDecisionOutcome(out.Outcome)
	return out
}

func normalizeDecisionSteps(steps []DecisionStep) []DecisionStep {
	if len(steps) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(steps))
	out := make([]DecisionStep, 0, len(steps))
	for _, step := range steps {
		normalized := normalizeDecisionStep(step)
		if normalized.Title == "" && normalized.Summary == "" && normalized.Outcome == "" {
			continue
		}
		fingerprint := strings.Join([]string{normalized.Title, normalized.Summary, normalized.Outcome}, "|")
		if _, ok := seen[fingerprint]; ok {
			continue
		}
		seen[fingerprint] = struct{}{}
		out = append(out, normalized)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func normalizeDecisionToolUse(tool DecisionToolUse) DecisionToolUse {
	out := tool
	out.Name = strings.TrimSpace(out.Name)
	out.Purpose = strings.TrimSpace(out.Purpose)
	out.InputSummary = strings.TrimSpace(out.InputSummary)
	out.OutputSummary = strings.TrimSpace(out.OutputSummary)
	return out
}

func normalizeDecisionToolUses(tools []DecisionToolUse) []DecisionToolUse {
	if len(tools) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(tools))
	out := make([]DecisionToolUse, 0, len(tools))
	for _, tool := range tools {
		normalized := normalizeDecisionToolUse(tool)
		if normalized.Name == "" && normalized.Purpose == "" && normalized.InputSummary == "" && normalized.OutputSummary == "" {
			continue
		}
		fingerprint := strings.Join([]string{normalized.Name, normalized.Purpose, normalized.InputSummary, normalized.OutputSummary}, "|")
		if _, ok := seen[fingerprint]; ok {
			continue
		}
		seen[fingerprint] = struct{}{}
		out = append(out, normalized)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func normalizeDecisionQuestion(question DecisionQuestion) DecisionQuestion {
	out := question
	out.Question = strings.TrimSpace(out.Question)
	out.Answer = strings.TrimSpace(out.Answer)
	if !out.AskedAt.IsZero() {
		out.AskedAt = out.AskedAt.UTC()
	}
	if !out.AnsweredAt.IsZero() {
		out.AnsweredAt = out.AnsweredAt.UTC()
	}
	return out
}

func normalizeDecisionQuestions(questions []DecisionQuestion) []DecisionQuestion {
	if len(questions) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(questions))
	out := make([]DecisionQuestion, 0, len(questions))
	for _, question := range questions {
		normalized := normalizeDecisionQuestion(question)
		if normalized.Question == "" && normalized.Answer == "" && normalized.AskedAt.IsZero() && normalized.AnsweredAt.IsZero() {
			continue
		}
		fingerprint := strings.Join([]string{
			normalized.Question,
			normalized.Answer,
			normalized.AskedAt.UTC().Format(time.RFC3339Nano),
			normalized.AnsweredAt.UTC().Format(time.RFC3339Nano),
		}, "|")
		if _, ok := seen[fingerprint]; ok {
			continue
		}
		seen[fingerprint] = struct{}{}
		out = append(out, normalized)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func normalizeDecisionEnvFingerprint(env DecisionEnvFingerprint) DecisionEnvFingerprint {
	out := env
	out.OS = strings.TrimSpace(out.OS)
	out.Shell = strings.TrimSpace(out.Shell)
	out.WorkspaceRoot = strings.TrimSpace(out.WorkspaceRoot)
	out.Provider = strings.TrimSpace(out.Provider)
	out.Model = strings.TrimSpace(out.Model)
	out.GraphNamespace = strings.TrimSpace(out.GraphNamespace)
	out.ToolNames = uniqueStrings(out.ToolNames)
	return out
}

func normalizeDecisionMemo(memo DecisionMemo) DecisionMemo {
	out := memo
	out.ID = strings.TrimSpace(out.ID)
	out.Namespace = normalizeDecisionNamespace(out.Namespace)
	out.SessionID = strings.TrimSpace(out.SessionID)
	out.TraceID = strings.TrimSpace(out.TraceID)
	out.TurnID = strings.TrimSpace(out.TurnID)
	out.IntentKey = strings.TrimSpace(out.IntentKey)
	out.IntentSummary = strings.TrimSpace(out.IntentSummary)
	out.ProblemSummary = strings.TrimSpace(out.ProblemSummary)
	out.ContextSummary = strings.TrimSpace(out.ContextSummary)
	out.Constraints = uniqueStrings(out.Constraints)
	out.Assumptions = uniqueStrings(out.Assumptions)
	out.StrategySummary = strings.TrimSpace(out.StrategySummary)
	out.KeySteps = normalizeDecisionSteps(out.KeySteps)
	out.ToolsUsed = normalizeDecisionToolUses(out.ToolsUsed)
	out.QuestionsAsked = normalizeDecisionQuestions(out.QuestionsAsked)
	out.NeedsHumanFor = uniqueStrings(out.NeedsHumanFor)
	out.Outcome = normalizeDecisionOutcome(out.Outcome)
	out.OutcomeSummary = strings.TrimSpace(out.OutcomeSummary)
	out.FailureReasons = uniqueStrings(out.FailureReasons)
	out.AvoidPatterns = uniqueStrings(out.AvoidPatterns)
	out.ValidationChecks = uniqueStrings(out.ValidationChecks)
	out.GraphNodeRefs = uniqueStrings(out.GraphNodeRefs)
	out.GraphEdgeRefs = uniqueStrings(out.GraphEdgeRefs)
	out.AnchorKeys = uniqueStrings(out.AnchorKeys)
	out.Confidence = clamp01(out.Confidence)
	out.ReuseScore = clamp01(out.ReuseScore)
	if out.CreatedAt.IsZero() {
		out.CreatedAt = time.Now().UTC()
	} else {
		out.CreatedAt = out.CreatedAt.UTC()
	}
	if out.LastUsedAt.IsZero() {
		out.LastUsedAt = out.CreatedAt.UTC()
	} else {
		out.LastUsedAt = out.LastUsedAt.UTC()
	}
	if out.AccessCount < 0 {
		out.AccessCount = 0
	}
	out.Environment = normalizeDecisionEnvFingerprint(out.Environment)
	return out
}

func normalizeRecipeStep(step RecipeStep) RecipeStep {
	out := step
	out.Title = strings.TrimSpace(out.Title)
	out.Instruction = strings.TrimSpace(out.Instruction)
	out.ToolName = strings.TrimSpace(out.ToolName)
	out.Validation = strings.TrimSpace(out.Validation)
	return out
}

func normalizeRecipeSteps(steps []RecipeStep) []RecipeStep {
	if len(steps) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(steps))
	out := make([]RecipeStep, 0, len(steps))
	for _, step := range steps {
		normalized := normalizeRecipeStep(step)
		if normalized.Title == "" && normalized.Instruction == "" && normalized.ToolName == "" && normalized.Validation == "" {
			continue
		}
		fingerprint := strings.Join([]string{normalized.Title, normalized.Instruction, normalized.ToolName, normalized.Validation}, "|")
		if _, ok := seen[fingerprint]; ok {
			continue
		}
		seen[fingerprint] = struct{}{}
		out = append(out, normalized)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func normalizeDecisionRecipe(recipe DecisionRecipe) DecisionRecipe {
	out := recipe
	out.ID = strings.TrimSpace(out.ID)
	out.Namespace = normalizeDecisionNamespace(out.Namespace)
	out.IntentKey = strings.TrimSpace(out.IntentKey)
	out.TriggerPhrases = uniqueStrings(out.TriggerPhrases)
	out.Preconditions = uniqueStrings(out.Preconditions)
	out.StrategySummary = strings.TrimSpace(out.StrategySummary)
	out.RecommendedTools = uniqueStrings(out.RecommendedTools)
	out.OrderedActions = normalizeRecipeSteps(out.OrderedActions)
	out.ValidationChecklist = uniqueStrings(out.ValidationChecklist)
	out.AvoidPatterns = uniqueStrings(out.AvoidPatterns)
	if out.SupportCount < 0 {
		out.SupportCount = 0
	}
	out.SuccessRate = clamp01(out.SuccessRate)
	out.Confidence = clamp01(out.Confidence)
	out.SourceMemoIDs = uniqueStrings(out.SourceMemoIDs)
	out.GraphRefs = uniqueStrings(out.GraphRefs)
	out.AnchorKeys = uniqueStrings(out.AnchorKeys)
	if out.CreatedAt.IsZero() {
		out.CreatedAt = time.Now().UTC()
	} else {
		out.CreatedAt = out.CreatedAt.UTC()
	}
	if out.UpdatedAt.IsZero() {
		out.UpdatedAt = out.CreatedAt.UTC()
	} else {
		out.UpdatedAt = out.UpdatedAt.UTC()
	}
	return out
}

func normalizeDecisionCluster(cluster DecisionCluster) DecisionCluster {
	out := cluster
	out.ID = strings.TrimSpace(out.ID)
	out.Namespace = normalizeDecisionNamespace(out.Namespace)
	out.IntentKey = strings.TrimSpace(out.IntentKey)
	out.EnvironmentKey = strings.TrimSpace(out.EnvironmentKey)
	out.MemoIDs = uniqueStrings(out.MemoIDs)
	if out.SuccessCount < 0 {
		out.SuccessCount = 0
	}
	if out.FailureCount < 0 {
		out.FailureCount = 0
	}
	out.RecipeID = strings.TrimSpace(out.RecipeID)
	if out.UpdatedAt.IsZero() {
		out.UpdatedAt = time.Now().UTC()
	} else {
		out.UpdatedAt = out.UpdatedAt.UTC()
	}
	return out
}

func normalizeDecisionHit(hit DecisionHit) DecisionHit {
	out := hit
	out.Type = normalizeDecisionHitType(out.Type)
	out.Namespace = normalizeDecisionNamespace(out.Namespace)
	out.IntentKey = strings.TrimSpace(out.IntentKey)
	out.MemoID = strings.TrimSpace(out.MemoID)
	out.RecipeID = strings.TrimSpace(out.RecipeID)
	out.Summary = strings.TrimSpace(out.Summary)
	out.Reason = strings.TrimSpace(out.Reason)
	out.Outcome = normalizeDecisionOutcome(out.Outcome)
	out.Score = clamp01(out.Score)
	out.Confidence = clamp01(out.Confidence)
	out.ReuseScore = clamp01(out.ReuseScore)
	out.GraphRefs = uniqueStrings(out.GraphRefs)
	out.AnchorKeys = uniqueStrings(out.AnchorKeys)
	return out
}

func cloneDecisionMemo(memo DecisionMemo) DecisionMemo {
	out := memo
	out.Constraints = append([]string(nil), memo.Constraints...)
	out.Assumptions = append([]string(nil), memo.Assumptions...)
	out.KeySteps = cloneDecisionSteps(memo.KeySteps)
	out.ToolsUsed = cloneDecisionToolUses(memo.ToolsUsed)
	out.QuestionsAsked = cloneDecisionQuestions(memo.QuestionsAsked)
	out.NeedsHumanFor = append([]string(nil), memo.NeedsHumanFor...)
	out.FailureReasons = append([]string(nil), memo.FailureReasons...)
	out.AvoidPatterns = append([]string(nil), memo.AvoidPatterns...)
	out.ValidationChecks = append([]string(nil), memo.ValidationChecks...)
	out.GraphNodeRefs = append([]string(nil), memo.GraphNodeRefs...)
	out.GraphEdgeRefs = append([]string(nil), memo.GraphEdgeRefs...)
	out.AnchorKeys = append([]string(nil), memo.AnchorKeys...)
	out.Environment = cloneDecisionEnvFingerprint(memo.Environment)
	return out
}

func cloneDecisionMemos(memos []DecisionMemo) []DecisionMemo {
	if len(memos) == 0 {
		return nil
	}
	out := make([]DecisionMemo, len(memos))
	for i := range memos {
		out[i] = cloneDecisionMemo(memos[i])
	}
	return out
}

func cloneDecisionSteps(steps []DecisionStep) []DecisionStep {
	if len(steps) == 0 {
		return nil
	}
	out := make([]DecisionStep, len(steps))
	copy(out, steps)
	return out
}

func cloneDecisionToolUses(tools []DecisionToolUse) []DecisionToolUse {
	if len(tools) == 0 {
		return nil
	}
	out := make([]DecisionToolUse, len(tools))
	copy(out, tools)
	return out
}

func cloneDecisionQuestions(questions []DecisionQuestion) []DecisionQuestion {
	if len(questions) == 0 {
		return nil
	}
	out := make([]DecisionQuestion, len(questions))
	copy(out, questions)
	return out
}

func cloneDecisionEnvFingerprint(env DecisionEnvFingerprint) DecisionEnvFingerprint {
	out := env
	out.ToolNames = append([]string(nil), env.ToolNames...)
	return out
}

func cloneDecisionRecipe(recipe DecisionRecipe) DecisionRecipe {
	out := recipe
	out.TriggerPhrases = append([]string(nil), recipe.TriggerPhrases...)
	out.Preconditions = append([]string(nil), recipe.Preconditions...)
	out.RecommendedTools = append([]string(nil), recipe.RecommendedTools...)
	out.OrderedActions = cloneRecipeSteps(recipe.OrderedActions)
	out.ValidationChecklist = append([]string(nil), recipe.ValidationChecklist...)
	out.AvoidPatterns = append([]string(nil), recipe.AvoidPatterns...)
	out.SourceMemoIDs = append([]string(nil), recipe.SourceMemoIDs...)
	out.GraphRefs = append([]string(nil), recipe.GraphRefs...)
	out.AnchorKeys = append([]string(nil), recipe.AnchorKeys...)
	return out
}

func cloneDecisionRecipes(recipes []DecisionRecipe) []DecisionRecipe {
	if len(recipes) == 0 {
		return nil
	}
	out := make([]DecisionRecipe, len(recipes))
	for i := range recipes {
		out[i] = cloneDecisionRecipe(recipes[i])
	}
	return out
}

func cloneRecipeSteps(steps []RecipeStep) []RecipeStep {
	if len(steps) == 0 {
		return nil
	}
	out := make([]RecipeStep, len(steps))
	copy(out, steps)
	return out
}

func cloneDecisionCluster(cluster DecisionCluster) DecisionCluster {
	out := cluster
	out.MemoIDs = append([]string(nil), cluster.MemoIDs...)
	return out
}

func cloneDecisionClusters(clusters []DecisionCluster) []DecisionCluster {
	if len(clusters) == 0 {
		return nil
	}
	out := make([]DecisionCluster, len(clusters))
	for i := range clusters {
		out[i] = cloneDecisionCluster(clusters[i])
	}
	return out
}

func cloneDecisionHit(hit DecisionHit) DecisionHit {
	out := hit
	out.GraphRefs = append([]string(nil), hit.GraphRefs...)
	out.AnchorKeys = append([]string(nil), hit.AnchorKeys...)
	return out
}

func normalizeDecisionMemos(memos []DecisionMemo) []DecisionMemo {
	if len(memos) == 0 {
		return nil
	}
	out := make([]DecisionMemo, 0, len(memos))
	indexByID := make(map[string]int, len(memos))
	for _, memo := range memos {
		normalized := normalizeDecisionMemo(memo)
		if normalized.ID == "" {
			continue
		}
		if idx, ok := indexByID[normalized.ID]; ok {
			out[idx] = normalized
			continue
		}
		indexByID[normalized.ID] = len(out)
		out = append(out, normalized)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func normalizeDecisionRecipes(recipes []DecisionRecipe) []DecisionRecipe {
	if len(recipes) == 0 {
		return nil
	}
	out := make([]DecisionRecipe, 0, len(recipes))
	indexByID := make(map[string]int, len(recipes))
	for _, recipe := range recipes {
		normalized := normalizeDecisionRecipe(recipe)
		if normalized.ID == "" {
			continue
		}
		if idx, ok := indexByID[normalized.ID]; ok {
			out[idx] = normalized
			continue
		}
		indexByID[normalized.ID] = len(out)
		out = append(out, normalized)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func normalizeDecisionClusters(clusters []DecisionCluster) []DecisionCluster {
	if len(clusters) == 0 {
		return nil
	}
	out := make([]DecisionCluster, 0, len(clusters))
	indexByID := make(map[string]int, len(clusters))
	for _, cluster := range clusters {
		normalized := normalizeDecisionCluster(cluster)
		if normalized.ID == "" {
			continue
		}
		if idx, ok := indexByID[normalized.ID]; ok {
			out[idx] = normalized
			continue
		}
		indexByID[normalized.ID] = len(out)
		out = append(out, normalized)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
