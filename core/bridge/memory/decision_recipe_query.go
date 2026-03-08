package memory

import (
	"strings"
	"time"
)

func (s *QueryService) decorateRecipeSelection(result MemoryQueryResult, query MemoryQuery, scope SessionScope) MemoryQueryResult {
	if s == nil || s.decision == nil || len(result.DecisionHits) == 0 {
		return result
	}
	selectedRecipe, report, advisory := s.decision.SelectRecipeReuse(query, scope, result.DecisionHits, result.Entries, result.IntentPlan, result.RerankReport, time.Now().UTC())
	if selectedRecipe != nil {
		cloned := cloneDecisionRecipe(*selectedRecipe)
		result.SelectedRecipe = &cloned
	}
	if advisory != nil {
		cloned := cloneRecipeAdvisory(*advisory)
		result.RecipeAdvisory = &cloned
	}
	if report != nil {
		cloned := cloneRecipeSelectionReport(*report)
		result.RecipeSelection = &cloned
	}
	return result
}

func recipeContextLines(advisory RecipeAdvisory) []string {
	advisory = normalizeRecipeAdvisory(advisory)
	if !advisory.ApplyByDefault {
		return nil
	}
	lines := make([]string, 0, 2)
	if advisory.StartWith != "" {
		lines = append(lines, summarizeLine("Selected recipe: start with "+selectorHintClause(advisory.StartWith), 170))
	}
	parts := make([]string, 0, 2)
	if len(advisory.Avoid) > 0 {
		parts = append(parts, "avoid "+selectorHintClause(advisory.Avoid[0]))
	}
	if len(advisory.Validate) > 0 {
		parts = append(parts, "validate "+selectorHintClause(advisory.Validate[0]))
	}
	if len(parts) > 0 {
		lines = append(lines, summarizeLine(strings.Join(parts, "; "), 170))
	}
	return lines
}
