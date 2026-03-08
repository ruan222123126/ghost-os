package decision

import (
	"crypto/sha1"
	"encoding/hex"
	"sort"
	"strings"
	"time"
)

type recipeSelectionInput struct {
	Query        MemoryQuery
	Scope        SessionScope
	Hits         []DecisionHit
	Entries      []MemoryEntry
	Plan         *QueryIntentPlan
	RerankReport *HybridRerankReport
	Now          time.Time
}

type recipeSelectionCandidate struct {
	Recipe              DecisionRecipe
	Hit                 DecisionHit
	SelectionScore      float64
	SelectionConfidence float64
	Reasons             []string
	AdvisoryOnly        bool
	Lineage             DecisionLineage
}

func (d *DecisionService) SelectRecipeReuse(query MemoryQuery, scope SessionScope, hits []DecisionHit, entries []MemoryEntry, plan *QueryIntentPlan, rerankReport *HybridRerankReport, now time.Time) (*DecisionRecipe, *RecipeSelectionReport, *RecipeAdvisory) {
	input := recipeSelectionInput{Query: query, Scope: scope, Hits: hits, Entries: entries, Plan: plan, RerankReport: rerankReport, Now: now}
	if d == nil || !d.recipeReuseEnabledForQuery() {
		return nil, &RecipeSelectionReport{FallbackReason: "recipe reuse disabled"}, nil
	}
	currentTime := input.Now.UTC()
	if currentTime.IsZero() {
		currentTime = time.Now().UTC()
	}
	recipeHits := make([]DecisionHit, 0, len(input.Hits))
	for _, rawHit := range input.Hits {
		hit := normalizeDecisionHit(rawHit)
		if hit.Type != DecisionHitTypeRecipe || hit.RecipeID == "" {
			continue
		}
		recipeHits = append(recipeHits, hit)
	}
	if len(recipeHits) == 0 {
		reason := "no recipe candidates survived recall"
		return nil, &RecipeSelectionReport{FallbackReason: reason}, nil
	}

	candidates := make([]recipeSelectionCandidate, 0, len(recipeHits))
	for _, hit := range recipeHits {
		recipe, ok := d.store.Recipe(hit.RecipeID)
		if !ok {
			continue
		}
		candidate := d.scoreRecipeSelectionCandidate(recipe, hit, input, currentTime)
		if candidate.SelectionScore <= 0 {
			continue
		}
		candidates = append(candidates, candidate)
	}
	if len(candidates) == 0 {
		reason := "recipe candidates fell below selection threshold"
		return nil, &RecipeSelectionReport{FallbackReason: reason}, nil
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].SelectionScore != candidates[j].SelectionScore {
			return candidates[i].SelectionScore > candidates[j].SelectionScore
		}
		if candidates[i].SelectionConfidence != candidates[j].SelectionConfidence {
			return candidates[i].SelectionConfidence > candidates[j].SelectionConfidence
		}
		if !candidates[i].Recipe.UpdatedAt.Equal(candidates[j].Recipe.UpdatedAt) {
			return candidates[i].Recipe.UpdatedAt.After(candidates[j].Recipe.UpdatedAt)
		}
		return candidates[i].Recipe.ID < candidates[j].Recipe.ID
	})
	best := candidates[0]
	selected := cloneDecisionRecipe(best.Recipe)
	grayHit, applyByDefault := d.recipeRolloutDecision(input.Scope.SessionID, input.Query.SemanticQuery, best.SelectionConfidence)
	advisory := d.buildRecipeAdvisory(selected, input.Hits, best.SelectionConfidence, applyByDefault && !best.AdvisoryOnly)
	if advisory == nil {
		return nil, &RecipeSelectionReport{FallbackReason: "selected recipe could not build advisory"}, nil
	}
	if best.AdvisoryOnly {
		advisory.ApplyByDefault = false
		advisory.LowConfidenceOnly = true
	}
	report := &RecipeSelectionReport{
		SelectedRecipeID: selected.ID,
		SelectionScore:   best.SelectionConfidence,
		WhySelected:      append([]string(nil), best.Reasons...),
		GrayHit:          grayHit,
		ApplyByDefault:   advisory.ApplyByDefault,
		Lineage:          cloneDecisionLineage(best.Lineage),
	}
	return &selected, report, advisory
}

func (d *DecisionService) scoreRecipeSelectionCandidate(recipe DecisionRecipe, hit DecisionHit, input recipeSelectionInput, now time.Time) recipeSelectionCandidate {
	recipe = normalizeDecisionRecipe(recipe)
	hit = normalizeDecisionHit(hit)
	contextLineage := d.selectionContextLineage(input.Hits)
	matchedClaimIDs := intersectStrings(contextLineage.SourceClaimIDs, recipe.SourceClaimIDs)
	matchedEvidenceIDs := intersectStrings(contextLineage.SourceEvidenceIDs, recipe.SourceEvidenceIDs)
	conflictedClaimIDs := intersectStrings(contextLineage.SourceClaimIDs, recipe.ContradictedClaimIDs)
	missingRequiredClaimIDs := diffStrings(recipe.SourceClaimIDs, matchedClaimIDs)
	currentEnv := decisionEnvironmentFromQuery(input.Query, input.Scope)
	envScore, hasEnvEvidence := d.recipeEnvironmentCompatibility(currentEnv, recipe)
	hitScore := maxFloat(hit.Score, hit.ReuseScore)
	supportScore := clamp01(float64(recipe.SupportCount) / float64(max(d.recipeMinSupport+2, 5)))
	successScore := clamp01(recipe.SuccessRate)
	freshnessScore := scoreDecisionRecency(now, recipe.UpdatedAt, recipe.CreatedAt, recipe.LastAppliedAt, recipe.LastSelectedAt)
	coverageScore := recipeSelectionCoverage(input.Query, input.Plan, recipe)
	hybridBoost := recipeSelectionHybridBoost(hit, input.RerankReport)
	if !hasEnvEvidence && currentEnv == nil {
		envScore = 0.2
	}
	score := hitScore*0.34 + envScore*0.18 + successScore*0.18 + supportScore*0.10 + freshnessScore*0.08 + coverageScore*0.07 + hybridBoost*0.05
	score -= recipeStatusPenalty(recipe.Status)
	if len(conflictedClaimIDs) > 0 {
		score -= clamp01(float64(len(conflictedClaimIDs))/float64(max(len(recipe.ContradictedClaimIDs), 1))) * 0.22
	}
	selectionConfidence := clamp01(recipe.Confidence*0.55 + score*0.45)
	advisoryOnly := selectionConfidence < d.recipeSelectionThreshold() || recipe.SuccessRate < d.recipeSuccessThreshold()
	if len(conflictedClaimIDs) > 0 && len(matchedClaimIDs) == 0 {
		advisoryOnly = true
	}
	if selectionConfidence < 0.35 {
		return recipeSelectionCandidate{}
	}
	reasons := make([]string, 0, 5)
	if hitScore >= 0.45 {
		reasons = append(reasons, "matched intent recall")
	}
	if envScore >= 0.75 {
		reasons = append(reasons, "environment matched strongly")
	} else if envScore >= 0.45 {
		reasons = append(reasons, "environment stayed compatible")
	}
	if successScore >= d.recipeSuccessThreshold() {
		reasons = append(reasons, "history shows strong success rate")
	}
	if supportScore >= 0.6 {
		reasons = append(reasons, "backed by multiple supporting runs")
	}
	if coverageScore >= 0.35 {
		reasons = append(reasons, "risk and validation coverage present")
	}
	if len(matchedClaimIDs) > 0 {
		reasons = append(reasons, "matched supporting claims")
	}
	if len(conflictedClaimIDs) > 0 {
		reasons = append(reasons, "conflicting claims kept selection conservative")
	}
	if advisoryOnly {
		reasons = append(reasons, "kept advisory-only because confidence is still warming up")
	}
	return recipeSelectionCandidate{
		Recipe:              recipe,
		Hit:                 hit,
		SelectionScore:      clamp01(score),
		SelectionConfidence: selectionConfidence,
		Reasons:             uniqueStrings(reasons),
		AdvisoryOnly:        advisoryOnly,
		Lineage: normalizeDecisionLineage(DecisionLineage{
			SourceClaimIDs:             append([]string(nil), recipe.SourceClaimIDs...),
			SourceEvidenceIDs:          append([]string(nil), recipe.SourceEvidenceIDs...),
			ContradictedClaimIDs:       append([]string(nil), recipe.ContradictedClaimIDs...),
			SelectionClaimIDs:          matchedClaimIDs,
			SelectionEvidenceIDs:       matchedEvidenceIDs,
			MatchedClaimIDs:            matchedClaimIDs,
			MatchedEvidenceIDs:         matchedEvidenceIDs,
			MissingRequiredClaimIDs:    missingRequiredClaimIDs,
			ConflictedClaimIDs:         conflictedClaimIDs,
			LineageSummary:             decisionLineageSummary("selected", matchedClaimIDs, matchedEvidenceIDs, recipe.SourceMemoIDs),
			LineageVersion:             decisionLineageVersion,
			DistillerVersion:           firstNonEmpty(recipe.DistillerVersion, decisionDistillerVersion),
			FallbackDueToClaimConflict: len(conflictedClaimIDs) > 0 && advisoryOnly,
		}),
	}
}

func (d *DecisionService) selectionContextLineage(hits []DecisionHit) DecisionLineage {
	if d == nil || d.store == nil || len(hits) == 0 {
		return DecisionLineage{}
	}
	context := DecisionLineage{}
	for _, rawHit := range hits {
		hit := normalizeDecisionHit(rawHit)
		if hit.Type != DecisionHitTypeMemo || strings.TrimSpace(hit.MemoID) == "" {
			continue
		}
		memo, ok := d.store.Memo(hit.MemoID)
		if !ok {
			continue
		}
		context = mergeDecisionLineage(context, decisionLineageFromMemo(memo))
	}
	return context
}

func recipeSelectionCoverage(query MemoryQuery, plan *QueryIntentPlan, recipe DecisionRecipe) float64 {
	terms := make([]string, 0, len(recipe.ValidationChecklist)+len(recipe.Preconditions))
	terms = append(terms, recipe.ValidationChecklist...)
	terms = append(terms, recipe.Preconditions...)
	queryTerms := append([]string(nil), query.Keywords...)
	if plan != nil {
		queryTerms = append(queryTerms, plan.Constraints...)
		queryTerms = append(queryTerms, plan.Risks...)
		queryTerms = append(queryTerms, plan.Environment...)
	}
	if semantic := strings.TrimSpace(query.SemanticQuery); semantic != "" {
		queryTerms = append(queryTerms, semantic)
	}
	return decisionStringSetOverlap(queryTerms, terms)
}

func recipeSelectionHybridBoost(hit DecisionHit, report *HybridRerankReport) float64 {
	if report == nil || len(report.Candidates) == 0 {
		return 0
	}
	entryID := decisionEntryID(hit)
	for index, item := range report.Candidates {
		if strings.TrimSpace(item.EntryID) != entryID {
			continue
		}
		return clamp01(item.HybridScore * (1 - float64(index)/float64(max(len(report.Candidates), 1))))
	}
	return 0
}

func recipeStatusPenalty(status string) float64 {
	switch normalizeRecipeStatus(status) {
	case RecipeStatusConflicted:
		return 0.22
	case RecipeStatusDeprecated:
		return 0.30
	default:
		return 0
	}
}

func (d *DecisionService) buildRecipeAdvisory(recipe DecisionRecipe, hits []DecisionHit, confidence float64, applyByDefault bool) *RecipeAdvisory {
	recipe = normalizeDecisionRecipe(recipe)
	startWith := firstNonEmpty(decisionRecipeFirstAction(recipe), recipe.StrategySummary)
	if startWith == "" {
		return nil
	}
	askHuman := recipeAskHumanHints(recipe, hits)
	return &RecipeAdvisory{
		RecipeID:          recipe.ID,
		Source:            "selected_recipe",
		StartWith:         summarizeDecisionText(startWith, 140),
		Avoid:             summarizeDecisionTexts(recipe.AvoidPatterns, 120),
		Validate:          summarizeDecisionTexts(recipe.ValidationChecklist, 120),
		AskHumanIf:        askHuman,
		RecommendedTools:  append([]string(nil), recipe.RecommendedTools...),
		Confidence:        clamp01(confidence),
		ApplyByDefault:    applyByDefault,
		LowConfidenceOnly: false,
	}
}

func recipeAskHumanHints(recipe DecisionRecipe, hits []DecisionHit) []string {
	out := make([]string, 0, 3)
	appendIfHuman := func(value string) {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return
		}
		lower := strings.ToLower(trimmed)
		if strings.Contains(lower, "human") || strings.Contains(lower, "approval") || strings.Contains(lower, "confirm") || strings.Contains(lower, "credential") || strings.Contains(lower, "permission") || strings.Contains(lower, "secret") {
			out = append(out, summarizeDecisionText(trimmed, 120))
		}
	}
	for _, item := range recipe.Preconditions {
		appendIfHuman(item)
	}
	for _, item := range recipe.AvoidPatterns {
		appendIfHuman(item)
	}
	for _, hit := range hits {
		appendIfHuman(hit.Caution)
	}
	return uniqueStrings(out)
}

func (d *DecisionService) stageRecipeRun(scope SessionScope, queryText string, recipe DecisionRecipe, report RecipeSelectionReport, advisory RecipeAdvisory) {
	if d == nil || !d.recipeExecutionTrackingEnabled || strings.TrimSpace(scope.SessionID) == "" || strings.TrimSpace(queryText) == "" {
		return
	}
	selectedAt := time.Now().UTC()
	run := RecipeRun{
		ID:         "pending:" + buildPendingRecipeRunKey(scope.SessionID, queryText),
		Namespace:  recipe.Namespace,
		RecipeID:   recipe.ID,
		SessionID:  strings.TrimSpace(scope.SessionID),
		SelectedAt: selectedAt,
		IntentKey:  recipe.IntentKey,
		Selection:  cloneRecipeSelectionReport(report),
		Advisory:   cloneRecipeAdvisory(advisory),
	}
	if scope.Environment != nil {
		run.EnvironmentFingerprint = cloneDecisionEnvFingerprint(*scope.Environment)
	}
	key := buildPendingRecipeRunKey(scope.SessionID, queryText)
	d.pendingMu.Lock()
	d.pendingRecipeRuns[key] = run
	d.pendingMu.Unlock()
	if report.GrayHit && d.metrics != nil {
		d.metrics.recipeDefaultGrayHits.Add(1)
	}
	d.recordRecipeSelection(recipe.ID, selectedAt)
}

func buildPendingRecipeRunKey(sessionID string, queryText string) string {
	sum := sha1.Sum([]byte(strings.TrimSpace(sessionID) + "|" + normalizeRecallText(queryText)))
	return hex.EncodeToString(sum[:8])
}
