package memory

import (
	"strings"
	"time"
)

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
	out.Platform = strings.TrimSpace(out.Platform)
	out.Shell = strings.TrimSpace(out.Shell)
	out.WorkspaceRoot = strings.TrimSpace(out.WorkspaceRoot)
	out.Provider = strings.TrimSpace(out.Provider)
	out.Model = strings.TrimSpace(out.Model)
	out.GraphNamespace = strings.TrimSpace(out.GraphNamespace)
	out.Domain = strings.TrimSpace(out.Domain)
	out.ToolsetSignature = strings.TrimSpace(out.ToolsetSignature)
	out.PathHints = uniqueStrings(out.PathHints)
	out.TargetAppOrSite = strings.TrimSpace(out.TargetAppOrSite)
	out.ToolNames = uniqueStrings(out.ToolNames)
	return out
}

func normalizeDecisionAnsweredQuestion(question DecisionAnsweredQuestion) DecisionAnsweredQuestion {
	out := question
	out.QuestionID = strings.TrimSpace(out.QuestionID)
	out.Prompt = strings.TrimSpace(out.Prompt)
	out.ToolCallID = strings.TrimSpace(out.ToolCallID)
	out.TraceID = strings.TrimSpace(out.TraceID)
	out.Answer = strings.TrimSpace(out.Answer)
	if !out.AskedAt.IsZero() {
		out.AskedAt = out.AskedAt.UTC()
	}
	if !out.AnsweredAt.IsZero() {
		out.AnsweredAt = out.AnsweredAt.UTC()
	}
	return out
}

func normalizeDecisionAnsweredQuestions(questions []DecisionAnsweredQuestion) []DecisionAnsweredQuestion {
	if len(questions) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(questions))
	out := make([]DecisionAnsweredQuestion, 0, len(questions))
	for _, question := range questions {
		normalized := normalizeDecisionAnsweredQuestion(question)
		if normalized.QuestionID == "" && normalized.Prompt == "" && normalized.ToolCallID == "" && normalized.Answer == "" {
			continue
		}
		fingerprint := strings.Join([]string{
			normalized.QuestionID,
			normalized.Prompt,
			normalized.ToolCallID,
			normalized.TraceID,
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
	out.SessionID = strings.TrimSpace(out.SessionID)
	out.Summary = strings.TrimSpace(out.Summary)
	out.WhyMatched = strings.TrimSpace(firstNonEmpty(out.WhyMatched, out.Reason))
	out.Reason = strings.TrimSpace(firstNonEmpty(out.Reason, out.WhyMatched))
	out.Caution = strings.TrimSpace(out.Caution)
	out.Outcome = normalizeDecisionOutcome(out.Outcome)
	out.Score = clamp01(out.Score)
	out.Confidence = clamp01(out.Confidence)
	out.ReuseScore = clamp01(out.ReuseScore)
	if !out.Timestamp.IsZero() {
		out.Timestamp = out.Timestamp.UTC()
	}
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
	out.PathHints = append([]string(nil), env.PathHints...)
	out.ToolNames = append([]string(nil), env.ToolNames...)
	return out
}

func cloneDecisionAnsweredQuestions(questions []DecisionAnsweredQuestion) []DecisionAnsweredQuestion {
	if len(questions) == 0 {
		return nil
	}
	out := make([]DecisionAnsweredQuestion, len(questions))
	copy(out, questions)
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
