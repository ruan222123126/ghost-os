package decision

import "time"

const (
	DecisionOutcomeSuccess       = "success"
	DecisionOutcomePartial       = "partial"
	DecisionOutcomeFailure       = "failure"
	DecisionOutcomeAwaitingHuman = "awaiting_human"
	DecisionOutcomeCancelled     = "cancelled"

	RecipeStatusActive     = "active"
	RecipeStatusDeprecated = "deprecated"
	RecipeStatusConflicted = "conflicted"

	RecipeValidationPassed  = "passed"
	RecipeValidationFailed  = "failed"
	RecipeValidationUnknown = "unknown"

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
	defaultDecisionRecipeRunsPath   = "recipe_runs.json"
	decisionLineageVersion          = "lineage.v1"
	decisionDistillerVersion        = "distill.claim-evidence.v1"
)
