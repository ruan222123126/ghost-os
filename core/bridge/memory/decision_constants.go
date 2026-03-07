package memory

import "time"

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
