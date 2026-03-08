package decision

import (
	"crypto/sha1"
	"encoding/hex"
	"strconv"
	"strings"
	"time"
)

func decisionArchiveWindow(archives []ColdArchive, opts DecisionRebuildOptions) ([]ColdArchive, string) {
	if len(archives) == 0 {
		return nil, ""
	}
	start := parseDecisionRebuildCursor(opts.CursorCheckpoint)
	if start < 0 {
		start = 0
	}
	if start >= len(archives) {
		return nil, strconv.Itoa(len(archives))
	}
	end := len(archives)
	if opts.MaxSessions > 0 && start+opts.MaxSessions < end {
		end = start + opts.MaxSessions
	}
	batchSize := opts.BatchSize
	if batchSize <= 0 {
		batchSize = end - start
	}
	if start+batchSize < end {
		end = start + batchSize
	}
	return archives[start:end], strconv.Itoa(end)
}

func parseDecisionRebuildCursor(cursor string) int {
	value, err := strconv.Atoi(strings.TrimSpace(cursor))
	if err != nil || value < 0 {
		return 0
	}
	return value
}

func (d *DecisionService) rebuildHistoricalRecipeRuns(namespace string, memos []DecisionMemo, dryRun bool) (created int, updated int, err error) {
	if d == nil || d.store == nil || !d.recipeBackfillEnabled {
		return 0, 0, nil
	}
	recipes := d.store.ListRecipes(namespace)
	if len(recipes) == 0 || len(memos) == 0 {
		return 0, 0, nil
	}
	existing := make(map[string]struct{}, len(d.store.ListRecipeRuns(namespace)))
	for _, run := range d.store.ListRecipeRuns(namespace) {
		existing[run.ID] = struct{}{}
	}
	for _, memo := range memos {
		recipe, ok := d.matchRecipeForMemo(memo, recipes)
		if !ok {
			continue
		}
		run := approximateRecipeRunFromMemo(recipe, memo)
		if _, seen := existing[run.ID]; seen {
			updated++
		} else {
			created++
		}
		if dryRun {
			continue
		}
		if _, err := d.store.UpsertRecipeRun(run); err != nil {
			return created, updated, err
		}
	}
	if !dryRun {
		err = d.rebuildRecipeStatsFromRuns(namespace)
	}
	return created, updated, err
}

func (d *DecisionService) matchRecipeForMemo(memo DecisionMemo, recipes []DecisionRecipe) (DecisionRecipe, bool) {
	memo = normalizeDecisionMemo(memo)
	bestScore := 0.0
	var best DecisionRecipe
	for _, recipe := range recipes {
		recipe = normalizeDecisionRecipe(recipe)
		score := 0.0
		if recipe.IntentKey != "" && recipe.IntentKey == memo.IntentKey {
			score += 0.45
		} else if decisionLookupMatchesTerms(recipe.IntentKey, []string{memo.IntentKey, memo.IntentSummary, memo.ProblemSummary}) {
			score += 0.25
		}
		if containsRecipeBackfillString(recipe.SourceMemoIDs, memo.ID) {
			score += 0.30
		}
		memoEnvKey := decisionClusterEnvironmentKey(memo.Environment, memo.GraphNodeRefs, memo.AnchorKeys)
		if recipe.EnvironmentKey != "" && memoEnvKey != "" {
			if recipe.EnvironmentKey == memoEnvKey {
				score += 0.15
			} else if strings.Contains(recipe.EnvironmentKey, memoEnvKey) || strings.Contains(memoEnvKey, recipe.EnvironmentKey) {
				score += 0.08
			}
		}
		score += clamp01(recipe.SuccessRate) * 0.10
		if score > bestScore {
			bestScore = score
			best = recipe
		}
	}
	if bestScore < 0.35 {
		return DecisionRecipe{}, false
	}
	return best, true
}

func approximateRecipeRunFromMemo(recipe DecisionRecipe, memo DecisionMemo) RecipeRun {
	selectedAt := latestDecisionTime(memo.LastUsedAt, memo.CreatedAt)
	if selectedAt.IsZero() {
		selectedAt = time.Now().UTC()
	}
	actualTools := decisionToolNamesFromUses(memo.ToolsUsed)
	steps := buildRecipeRunSteps(recipe, actualTools, nil, memo.Outcome)
	feedback := RecipeFeedback{
		Outcome:            memo.Outcome,
		FailureReasons:     append([]string(nil), memo.FailureReasons...),
		HumanBlocked:       memo.HumanBlocked,
		ValidationFailures: append([]string(nil), memo.ValidationChecks...),
		ReuseHelpful:       normalizeDecisionOutcome(memo.Outcome) == DecisionOutcomeSuccess,
	}
	selectionScore := clamp01(recipe.SuccessRate*0.6 + recipe.Confidence*0.4)
	return normalizeRecipeRun(RecipeRun{
		DecisionLineage: DecisionLineage{
			SourceClaimIDs:       append([]string(nil), recipe.SourceClaimIDs...),
			SourceEvidenceIDs:    append([]string(nil), recipe.SourceEvidenceIDs...),
			SelectionClaimIDs:    append([]string(nil), recipe.SourceClaimIDs...),
			SelectionEvidenceIDs: append([]string(nil), recipe.SourceEvidenceIDs...),
			ExecutionEvidenceIDs: append([]string(nil), memo.SourceEvidenceIDs...),
			EmittedClaimIDs:      append([]string(nil), memo.DerivedClaimIDs...),
			LineageSummary:       decisionLineageSummary("backfill", append(append([]string(nil), recipe.SourceClaimIDs...), memo.DerivedClaimIDs...), append(append([]string(nil), recipe.SourceEvidenceIDs...), memo.SourceEvidenceIDs...), []string{memo.ID}),
			LineageVersion:       decisionLineageVersion,
			LineagePartial:       true,
		},
		ID:                     buildApproximateRecipeRunID(recipe, memo),
		Namespace:              recipe.Namespace,
		RecipeID:               recipe.ID,
		SessionID:              memo.SessionID,
		TraceID:                memo.TraceID,
		TurnID:                 memo.TurnID,
		SelectedAt:             selectedAt,
		CompletedAt:            selectedAt,
		IntentKey:              memo.IntentKey,
		EnvironmentFingerprint: cloneDecisionEnvFingerprint(memo.Environment),
		Selection: RecipeSelectionReport{
			SelectedRecipeID: recipe.ID,
			SelectionScore:   selectionScore,
			WhySelected:      []string{"backfilled from historical memo cluster"},
			Lineage: DecisionLineage{
				SelectionClaimIDs:    append([]string(nil), recipe.SourceClaimIDs...),
				SelectionEvidenceIDs: append([]string(nil), recipe.SourceEvidenceIDs...),
				LineageVersion:       decisionLineageVersion,
				LineagePartial:       true,
			},
		},
		Advisory: RecipeAdvisory{
			RecipeID:          recipe.ID,
			Source:            "backfill",
			StartWith:         firstNonEmpty(decisionRecipeFirstAction(recipe), recipe.StrategySummary),
			Avoid:             append([]string(nil), recipe.AvoidPatterns...),
			Validate:          append([]string(nil), recipe.ValidationChecklist...),
			RecommendedTools:  append([]string(nil), recipe.RecommendedTools...),
			Confidence:        selectionScore,
			ApplyByDefault:    false,
			LowConfidenceOnly: true,
		},
		Steps:       steps,
		Feedback:    normalizeRecipeFeedback(feedback),
		ActualTools: actualTools,
	})
}

func buildApproximateRecipeRunID(recipe DecisionRecipe, memo DecisionMemo) string {
	seed := strings.Join([]string{recipe.Namespace, recipe.ID, memo.SessionID, memo.TurnID, memo.ID}, "|")
	sum := sha1.Sum([]byte(seed))
	return "recipe-run-" + hex.EncodeToString(sum[:8])
}

func containsRecipeBackfillString(values []string, target string) bool {
	target = strings.TrimSpace(target)
	if target == "" {
		return false
	}
	for _, value := range values {
		if strings.TrimSpace(value) == target {
			return true
		}
	}
	return false
}
