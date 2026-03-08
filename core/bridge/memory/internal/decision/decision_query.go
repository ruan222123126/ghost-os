package decision

import (
	"math"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func (d *DecisionService) Retrieve(query MemoryQuery, scope SessionScope) ([]MemoryEntry, []DecisionHit, error) {
	if d == nil || !d.Enabled() || !query.IncludeDecision {
		return nil, nil, nil
	}
	queryText := buildDecisionQueryText(query)
	if queryText == "" {
		return nil, nil, nil
	}
	if query.EnvironmentStrict && decisionEnvironmentFromQuery(query, scope) == nil {
		return nil, nil, nil
	}

	now := time.Now().UTC()
	hits := make([]DecisionHit, 0, d.maxHits*2)
	for _, memoID := range d.candidateMemoIDs(query, scope) {
		memo, ok := d.store.Memo(memoID)
		if !ok || memo.Namespace != d.resolveDecisionNamespace(query, scope) {
			continue
		}
		score, whyMatched, caution, ok := d.scoreDecisionMemo(memo, query, scope, now)
		if !ok {
			continue
		}
		hit := decisionHitFromMemo(memo, score, whyMatched, caution)
		if !decisionHitAllowed(query, hit) {
			continue
		}
		hits = append(hits, hit)
	}
	for _, recipeID := range d.candidateRecipeIDs(query, scope) {
		recipe, ok := d.store.Recipe(recipeID)
		if !ok || recipe.Namespace != d.resolveDecisionNamespace(query, scope) {
			continue
		}
		score, whyMatched, caution, reuseScore, ok := d.scoreDecisionRecipe(recipe, query, scope, now)
		if !ok {
			continue
		}
		hit := d.decisionHitFromRecipe(recipe, score, reuseScore, whyMatched, caution)
		if !decisionHitAllowed(query, hit) {
			continue
		}
		hits = append(hits, hit)
	}

	if len(hits) == 0 {
		return nil, nil, nil
	}
	sort.SliceStable(hits, func(i, j int) bool {
		if hits[i].Score != hits[j].Score {
			return hits[i].Score > hits[j].Score
		}
		if decisionHitPriority(hits[i]) != decisionHitPriority(hits[j]) {
			return decisionHitPriority(hits[i]) > decisionHitPriority(hits[j])
		}
		if !hits[i].Timestamp.Equal(hits[j].Timestamp) {
			return hits[i].Timestamp.After(hits[j].Timestamp)
		}
		return decisionHitStableID(hits[i]) < decisionHitStableID(hits[j])
	})
	if len(hits) > d.maxHits {
		hits = hits[:d.maxHits]
	}

	entries := make([]MemoryEntry, 0, len(hits))
	for _, hit := range hits {
		entries = append(entries, memoryEntryFromDecisionHit(hit))
	}
	return entries, cloneDecisionHits(hits), nil
}

func (d *DecisionService) BuildSelectorHint(query MemoryQuery, scope SessionScope) (string, []DecisionHit, error) {
	_, hits, err := d.Retrieve(query, scope)
	if err != nil || len(hits) == 0 {
		return "", hits, err
	}
	selectedRecipe, report, advisory := d.SelectRecipeReuse(query, scope, hits, nil, nil, nil, time.Now().UTC())
	if selectedRecipe != nil && report != nil && advisory != nil {
		d.stageRecipeRun(scope, query.SemanticQuery, *selectedRecipe, *report, *advisory)
		return d.formatSelectorHintSelection(advisory, report, hits), cloneDecisionHits(hits), nil
	}
	if d.metrics != nil && report != nil && strings.TrimSpace(report.FallbackReason) != "" {
		d.metrics.recipeFallbacks.Add(1)
	}
	return d.formatSelectorHint(hits), cloneDecisionHits(hits), nil
}

func (d *DecisionService) resolveDecisionNamespace(query MemoryQuery, scope SessionScope) string {
	if query.Metadata != nil {
		if raw, ok := query.Metadata["namespace"].(string); ok && strings.TrimSpace(raw) != "" {
			return normalizeDecisionNamespace(raw)
		}
	}
	if env := decisionEnvironmentFromQuery(query, scope); env != nil && strings.TrimSpace(env.GraphNamespace) != "" {
		return normalizeDecisionNamespace(env.GraphNamespace)
	}
	return defaultDecisionNamespace
}

func buildDecisionQueryText(query MemoryQuery) string {
	parts := make([]string, 0, len(query.Keywords)+1)
	if semantic := strings.TrimSpace(query.SemanticQuery); semantic != "" {
		parts = append(parts, semantic)
	}
	parts = append(parts, query.Keywords...)
	return strings.TrimSpace(strings.Join(parts, " "))
}

func (d *DecisionService) candidateMemoIDs(query MemoryQuery, scope SessionScope) []string {
	if d == nil || d.store == nil {
		return nil
	}
	namespace := d.resolveDecisionNamespace(query, scope)
	queryText := buildDecisionQueryText(query)
	terms := decisionCandidateTerms(query, scope)
	graphRefs := decisionGraphRefsFromQuery(query)
	toolNames := decisionEnvironmentToolNames(decisionEnvironmentFromQuery(query, scope))
	limit := max(d.maxHits*3, d.maxHits+4)

	seen := make(map[string]struct{}, limit)
	ids := make([]string, 0, limit)
	appendID := func(id string) {
		trimmed := strings.TrimSpace(id)
		if trimmed == "" {
			return
		}
		if _, ok := seen[trimmed]; ok {
			return
		}
		memo, ok := d.store.memoByID[trimmed]
		if !ok || memo.Namespace != namespace {
			return
		}
		seen[trimmed] = struct{}{}
		ids = append(ids, trimmed)
	}

	d.store.mu.RLock()
	for intentKey, memoIDs := range d.store.memoIDsByIntentKey {
		if !decisionKeyMatchesQuery(intentKey, queryText, terms) {
			continue
		}
		for _, id := range memoIDs {
			appendID(id)
		}
	}
	for anchorKey, memoIDs := range d.store.memosByAnchorKey {
		if !decisionLookupMatchesTerms(anchorKey, terms) {
			continue
		}
		for _, id := range memoIDs {
			appendID(id)
		}
	}
	for _, toolName := range toolNames {
		for _, id := range d.store.memosByToolName[strings.TrimSpace(toolName)] {
			appendID(id)
		}
	}
	for _, graphRef := range graphRefs {
		for _, id := range d.store.memosByGraphNode[strings.TrimSpace(graphRef)] {
			appendID(id)
		}
	}
	if len(ids) < limit {
		for _, memo := range d.store.memos {
			if memo.Namespace != namespace {
				continue
			}
			appendID(memo.ID)
		}
	}
	d.store.mu.RUnlock()
	return ids
}

func (d *DecisionService) candidateRecipeIDs(query MemoryQuery, scope SessionScope) []string {
	if d == nil || d.store == nil {
		return nil
	}
	namespace := d.resolveDecisionNamespace(query, scope)
	queryText := buildDecisionQueryText(query)
	terms := decisionCandidateTerms(query, scope)
	limit := max(d.maxHits*2, d.maxHits+2)

	seen := make(map[string]struct{}, limit)
	ids := make([]string, 0, limit)
	appendID := func(id string) {
		trimmed := strings.TrimSpace(id)
		if trimmed == "" {
			return
		}
		if _, ok := seen[trimmed]; ok {
			return
		}
		recipe, ok := d.store.recipeByID[trimmed]
		if !ok || recipe.Namespace != namespace {
			return
		}
		seen[trimmed] = struct{}{}
		ids = append(ids, trimmed)
	}

	d.store.mu.RLock()
	for intentKey, recipeIDs := range d.store.recipeIDsByIntentKey {
		if !decisionKeyMatchesQuery(intentKey, queryText, terms) {
			continue
		}
		for _, id := range recipeIDs {
			appendID(id)
		}
	}
	if len(ids) < limit {
		for _, recipe := range d.store.recipes {
			if recipe.Namespace != namespace {
				continue
			}
			appendID(recipe.ID)
		}
	}
	d.store.mu.RUnlock()
	return ids
}

func (d *DecisionService) scoreDecisionMemo(memo DecisionMemo, query MemoryQuery, scope SessionScope, now time.Time) (float64, string, string, bool) {
	memo = normalizeDecisionMemo(memo)
	queryText := buildDecisionQueryText(query)
	minConfidence := maxFloat(clamp01(query.MinConfidence), d.minConfidence)
	if memo.Confidence > 0 && memo.Confidence < minConfidence {
		return 0, "", "", false
	}
	currentEnv := decisionEnvironmentFromQuery(query, scope)
	if query.EnvironmentStrict && currentEnv == nil {
		return 0, "", "", false
	}
	envScore := environmentCompatibility(currentEnv, memo.Environment)
	if query.EnvironmentStrict && (decisionEnvironmentMismatch(currentEnv, memo.Environment) || envScore < 0.55) {
		return 0, "", "", false
	}

	intentScore := decisionTextOverlap(queryText,
		memo.IntentKey,
		memo.IntentSummary,
		memo.ProblemSummary,
		memo.ContextSummary,
		memo.StrategySummary,
		memo.OutcomeSummary,
	)
	anchorScore := decisionAnchorOverlap(query, scope, memo.AnchorKeys)
	graphScore := decisionGraphOverlap(query, memo.GraphNodeRefs)
	toolScore := decisionToolPatternOverlap(currentEnv, decisionToolNamesFromUses(memo.ToolsUsed))
	recency := scoreDecisionRecency(now, memo.LastUsedAt, memo.CreatedAt)
	outcomeWeight := scoreDecisionOutcome(memo.Outcome)
	reuseScore := clamp01(memo.ReuseScore)
	caution := buildDecisionMemoCaution(memo)
	recommended := memo.Outcome == DecisionOutcomeSuccess || (memo.Outcome == DecisionOutcomePartial && caution == "")
	minReuse := maxFloat(clamp01(query.MinReuseScore), d.minReuseScore)
	if query.DecisionReuseOnly && recommended && reuseScore < minReuse {
		return 0, "", caution, false
	}

	score := intentScore*0.35 + envScore*0.18 + reuseScore*0.12 + anchorScore*0.10 + graphScore*0.10 + toolScore*0.08 + outcomeWeight*0.05 + recency*0.02
	whyMatched := formatDecisionWhyMatched(intentScore, anchorScore, graphScore, toolScore, envScore)
	if score < 0.18 {
		return 0, whyMatched, caution, false
	}
	return clamp01(score), whyMatched, caution, true
}

func (d *DecisionService) scoreDecisionRecipe(recipe DecisionRecipe, query MemoryQuery, scope SessionScope, now time.Time) (float64, string, string, float64, bool) {
	recipe = normalizeDecisionRecipe(recipe)
	queryText := buildDecisionQueryText(query)
	minConfidence := maxFloat(clamp01(query.MinConfidence), d.minConfidence)
	if recipe.Confidence > 0 && recipe.Confidence < minConfidence {
		return 0, "", "", 0, false
	}
	currentEnv := decisionEnvironmentFromQuery(query, scope)
	if query.EnvironmentStrict && currentEnv == nil {
		return 0, "", "", 0, false
	}
	envScore, hasEnvEvidence := d.recipeEnvironmentCompatibility(currentEnv, recipe)
	if query.EnvironmentStrict && (!hasEnvEvidence || envScore < 0.55) {
		return 0, "", "", 0, false
	}

	supportScore := clamp01(float64(recipe.SupportCount) / float64(max(d.recipeMinSupport+1, 4)))
	reuseScore := clamp01(recipe.SuccessRate*0.55 + supportScore*0.45)
	if query.DecisionReuseOnly && reuseScore < maxFloat(clamp01(query.MinReuseScore), d.minReuseScore) {
		return 0, "", "", 0, false
	}
	intentScore := decisionTextOverlap(queryText,
		recipe.IntentKey,
		strings.Join(recipe.TriggerPhrases, " "),
		strings.Join(recipe.Preconditions, " "),
		recipe.StrategySummary,
		decisionRecipeActionText(recipe),
	)
	anchorScore := decisionAnchorOverlap(query, scope, recipe.AnchorKeys)
	graphScore := decisionGraphOverlap(query, recipe.GraphRefs)
	toolScore := decisionToolPatternOverlap(currentEnv, recipe.RecommendedTools)
	recency := scoreDecisionRecency(now, recipe.UpdatedAt, recipe.CreatedAt)
	strategyWeight := clamp01(recipe.SuccessRate*0.65 + supportScore*0.35)
	caution := firstNonEmptySlice(recipe.AvoidPatterns)
	whyMatched := formatDecisionWhyMatched(intentScore, anchorScore, graphScore, toolScore, envScore)
	if supportScore >= 0.6 {
		if whyMatched == "" {
			whyMatched = "backed by multiple successful runs"
		} else {
			whyMatched += " + backed by multiple successful runs"
		}
	}
	score := intentScore*0.35 + envScore*0.18 + reuseScore*0.12 + anchorScore*0.10 + graphScore*0.10 + toolScore*0.08 + strategyWeight*0.05 + recency*0.02
	if score < 0.18 {
		return 0, whyMatched, caution, reuseScore, false
	}
	return clamp01(score), whyMatched, caution, reuseScore, true
}

func environmentCompatibility(current *DecisionEnvFingerprint, memoEnv DecisionEnvFingerprint) float64 {
	if current == nil {
		return 0
	}
	currentEnv := normalizeDecisionEnvFingerprint(*current)
	otherEnv := normalizeDecisionEnvFingerprint(memoEnv)
	matchedWeight := 0.0
	totalWeight := 0.0
	addExact := func(weight float64, currentValue, otherValue string) {
		if strings.TrimSpace(currentValue) == "" || strings.TrimSpace(otherValue) == "" {
			return
		}
		totalWeight += weight
		if decisionLookup(currentValue) == decisionLookup(otherValue) {
			matchedWeight += weight
		}
	}
	addExact(0.34, cleanDecisionPath(currentEnv.WorkspaceRoot), cleanDecisionPath(otherEnv.WorkspaceRoot))
	addExact(0.18, firstNonEmpty(currentEnv.Platform, currentEnv.OS), firstNonEmpty(otherEnv.Platform, otherEnv.OS))
	addExact(0.18, currentEnv.GraphNamespace, otherEnv.GraphNamespace)
	addExact(0.05, currentEnv.Domain, otherEnv.Domain)
	addExact(0.03, currentEnv.TargetAppOrSite, otherEnv.TargetAppOrSite)

	toolOverlap := decisionToolPatternOverlap(&currentEnv, otherEnv.ToolNames)
	if len(currentEnv.ToolNames) > 0 && len(otherEnv.ToolNames) > 0 {
		totalWeight += 0.18
		matchedWeight += 0.18 * toolOverlap
	} else if currentEnv.ToolsetSignature != "" && otherEnv.ToolsetSignature != "" {
		totalWeight += 0.18
		if decisionLookup(currentEnv.ToolsetSignature) == decisionLookup(otherEnv.ToolsetSignature) {
			matchedWeight += 0.18
		}
	}

	pathOverlap := decisionStringSetOverlap(currentEnv.PathHints, otherEnv.PathHints)
	if len(currentEnv.PathHints) > 0 && len(otherEnv.PathHints) > 0 {
		totalWeight += 0.04
		matchedWeight += 0.04 * pathOverlap
	}
	if totalWeight == 0 {
		return 0
	}
	return clamp01(matchedWeight / totalWeight)
}

func decisionGraphOverlap(query MemoryQuery, refs []string) float64 {
	return decisionStringSetOverlap(decisionGraphRefsFromQuery(query), refs)
}

func decisionAnchorOverlap(query MemoryQuery, scope SessionScope, keys []string) float64 {
	return decisionStringSetOverlap(decisionCandidateTerms(query, scope), keys)
}

func decisionToolPatternOverlap(current *DecisionEnvFingerprint, toolNames []string) float64 {
	if current == nil {
		return 0
	}
	return decisionStringSetOverlap(current.ToolNames, toolNames)
}

func decisionEnvironmentMismatch(current *DecisionEnvFingerprint, memoEnv DecisionEnvFingerprint) bool {
	if current == nil {
		return false
	}
	currentEnv := normalizeDecisionEnvFingerprint(*current)
	otherEnv := normalizeDecisionEnvFingerprint(memoEnv)
	if currentEnv.WorkspaceRoot != "" && otherEnv.WorkspaceRoot != "" && cleanDecisionPath(currentEnv.WorkspaceRoot) != cleanDecisionPath(otherEnv.WorkspaceRoot) {
		return true
	}
	if currentEnv.GraphNamespace != "" && otherEnv.GraphNamespace != "" && currentEnv.GraphNamespace != otherEnv.GraphNamespace {
		return true
	}
	if firstNonEmpty(currentEnv.Platform, currentEnv.OS) != "" && firstNonEmpty(otherEnv.Platform, otherEnv.OS) != "" && firstNonEmpty(currentEnv.Platform, currentEnv.OS) != firstNonEmpty(otherEnv.Platform, otherEnv.OS) {
		return true
	}
	return len(currentEnv.ToolNames) > 0 && len(otherEnv.ToolNames) > 0 && decisionStringSetOverlap(currentEnv.ToolNames, otherEnv.ToolNames) == 0
}

func decisionEnvironmentFromQuery(query MemoryQuery, scope SessionScope) *DecisionEnvFingerprint {
	if query.Environment != nil {
		env := cloneDecisionEnvFingerprint(*query.Environment)
		return &env
	}
	if scope.Environment != nil {
		env := cloneDecisionEnvFingerprint(*scope.Environment)
		return &env
	}
	return nil
}

func decisionHitFromMemo(memo DecisionMemo, score float64, whyMatched string, caution string) DecisionHit {
	hitType := DecisionHitTypeMemo
	if memo.Outcome == DecisionOutcomeFailure || memo.Outcome == DecisionOutcomeAwaitingHuman || memo.Outcome == DecisionOutcomeCancelled || (memo.Outcome == DecisionOutcomePartial && caution != "") {
		hitType = DecisionHitTypeWarning
	}
	hit := DecisionHit{
		Type:       hitType,
		Namespace:  memo.Namespace,
		IntentKey:  memo.IntentKey,
		MemoID:     memo.ID,
		SessionID:  memo.SessionID,
		Summary:    decisionMemoSummary(memo),
		Reason:     whyMatched,
		WhyMatched: whyMatched,
		Caution:    caution,
		Outcome:    memo.Outcome,
		Score:      score,
		Confidence: memo.Confidence,
		ReuseScore: memo.ReuseScore,
		Timestamp:  latestDecisionTime(memo.LastUsedAt, memo.CreatedAt),
		GraphRefs:  append([]string(nil), memo.GraphNodeRefs...),
		AnchorKeys: append([]string(nil), memo.AnchorKeys...),
	}
	return normalizeDecisionHit(hit)
}

func (d *DecisionService) decisionHitFromRecipe(recipe DecisionRecipe, score float64, reuseScore float64, whyMatched string, caution string) DecisionHit {
	outcome := DecisionOutcomeSuccess
	if recipe.SuccessRate > 0 && recipe.SuccessRate < 0.7 {
		outcome = DecisionOutcomePartial
	}
	hit := DecisionHit{
		Type:       DecisionHitTypeRecipe,
		Namespace:  recipe.Namespace,
		IntentKey:  recipe.IntentKey,
		RecipeID:   recipe.ID,
		Summary:    decisionRecipeSummary(recipe),
		Reason:     whyMatched,
		WhyMatched: whyMatched,
		Caution:    caution,
		Outcome:    outcome,
		Score:      score,
		Confidence: recipe.Confidence,
		ReuseScore: reuseScore,
		Timestamp:  latestDecisionTime(recipe.UpdatedAt, recipe.CreatedAt),
		GraphRefs:  append([]string(nil), recipe.GraphRefs...),
		AnchorKeys: append([]string(nil), recipe.AnchorKeys...),
	}
	if sessionID := d.recipeSourceSessionID(recipe); sessionID != "" {
		hit.SessionID = sessionID
	}
	return normalizeDecisionHit(hit)
}

func decisionHitAllowed(query MemoryQuery, hit DecisionHit) bool {
	allowed, restricted := normalizedDecisionTypeSet(query.DecisionTypes)
	if !restricted {
		return true
	}
	_, ok := allowed[normalizeDecisionHitType(hit.Type)]
	return ok
}

func normalizedDecisionTypeSet(hitTypes []string) (map[string]struct{}, bool) {
	if len(hitTypes) == 0 {
		return nil, false
	}
	out := make(map[string]struct{}, len(hitTypes))
	for _, hitType := range hitTypes {
		normalized := normalizeDecisionHitType(hitType)
		if normalized == "" {
			continue
		}
		out[normalized] = struct{}{}
	}
	return out, true
}

func decisionHitPriority(hit DecisionHit) int {
	switch normalizeDecisionHitType(hit.Type) {
	case DecisionHitTypeRecipe:
		return 3
	case DecisionHitTypeMemo:
		return 2
	case DecisionHitTypeWarning:
		return 1
	default:
		return 0
	}
}

func decisionHitStableID(hit DecisionHit) string {
	return firstNonEmpty(hit.MemoID, hit.RecipeID, hit.Summary)
}

func decisionTextOverlap(queryText string, corpus ...string) float64 {
	queryLookup := decisionLookup(queryText)
	if queryLookup == "" {
		return 0
	}
	corpusLookup := decisionLookup(strings.Join(corpus, " "))
	if corpusLookup == "" {
		return 0
	}
	phraseHit := 0.0
	if strings.Contains(corpusLookup, queryLookup) {
		phraseHit = 1
	}
	terms := strings.Fields(queryLookup)
	matched := 0
	for _, term := range terms {
		if strings.Contains(corpusLookup, term) {
			matched++
		}
	}
	termScore := 0.0
	if len(terms) > 0 {
		termScore = float64(matched) / float64(len(terms))
	}
	return clamp01(phraseHit*0.45 + termScore*0.55)
}

func decisionCandidateTerms(query MemoryQuery, scope SessionScope) []string {
	terms := append([]string(nil), queryTerms(query)...)
	if env := decisionEnvironmentFromQuery(query, scope); env != nil {
		terms = append(terms, env.PathHints...)
		terms = append(terms, env.TargetAppOrSite)
	}
	return uniqueNonEmptyDecisionTerms(terms)
}

func decisionGraphRefsFromQuery(query MemoryQuery) []string {
	if query.Metadata == nil {
		return nil
	}
	return decisionStringSlice(query.Metadata["graph_node_refs"])
}

func decisionEnvironmentToolNames(env *DecisionEnvFingerprint) []string {
	if env == nil {
		return nil
	}
	return append([]string(nil), env.ToolNames...)
}

func decisionKeyMatchesQuery(key string, queryText string, terms []string) bool {
	keyLookup := decisionLookup(key)
	if keyLookup == "" {
		return false
	}
	queryLookup := decisionLookup(queryText)
	if queryLookup != "" && (strings.Contains(queryLookup, keyLookup) || strings.Contains(keyLookup, queryLookup)) {
		return true
	}
	return decisionLookupMatchesTerms(key, terms)
}

func decisionLookupMatchesTerms(value string, terms []string) bool {
	lookup := decisionLookup(value)
	if lookup == "" {
		return false
	}
	for _, term := range terms {
		termLookup := decisionLookup(term)
		if termLookup == "" {
			continue
		}
		if strings.Contains(lookup, termLookup) || strings.Contains(termLookup, lookup) {
			return true
		}
	}
	return false
}

func decisionLookup(value string) string {
	replacer := strings.NewReplacer(".", " ", "_", " ", "-", " ", "/", " ", ":", " ", ",", " ")
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(replacer.Replace(value)))), " ")
}

func uniqueNonEmptyDecisionTerms(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		lookup := decisionLookup(value)
		if lookup == "" {
			continue
		}
		if _, ok := seen[lookup]; ok {
			continue
		}
		seen[lookup] = struct{}{}
		out = append(out, value)
	}
	return out
}

func decisionStringSetOverlap(left []string, right []string) float64 {
	leftSet := decisionLookupSet(left)
	rightSet := decisionLookupSet(right)
	if len(leftSet) == 0 || len(rightSet) == 0 {
		return 0
	}
	intersections := 0
	for key := range leftSet {
		if _, ok := rightSet[key]; ok {
			intersections++
		}
	}
	if intersections == 0 {
		return 0
	}
	denominator := len(leftSet)
	if len(rightSet) > denominator {
		denominator = len(rightSet)
	}
	if denominator == 0 {
		return 0
	}
	return clamp01(float64(intersections) / float64(denominator))
}

func decisionLookupSet(values []string) map[string]struct{} {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]struct{}, len(values))
	for _, value := range values {
		lookup := decisionLookup(value)
		if lookup == "" {
			continue
		}
		out[lookup] = struct{}{}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func decisionToolNamesFromUses(uses []DecisionToolUse) []string {
	if len(uses) == 0 {
		return nil
	}
	names := make([]string, 0, len(uses))
	for _, use := range uses {
		if name := strings.TrimSpace(use.Name); name != "" {
			names = append(names, name)
		}
	}
	return names
}

func buildDecisionMemoCaution(memo DecisionMemo) string {
	if memo.Outcome == DecisionOutcomeAwaitingHuman {
		return firstNonEmptySlice(memo.NeedsHumanFor)
	}
	return firstNonEmpty(
		firstNonEmptySlice(memo.AvoidPatterns),
		firstNonEmptySlice(memo.FailureReasons),
		firstNonEmptySlice(memo.ValidationChecks),
	)
}

func decisionMemoSummary(memo DecisionMemo) string {
	return summarizeDecisionText(firstNonEmpty(
		memo.StrategySummary,
		memo.OutcomeSummary,
		memo.IntentSummary,
		memo.ContextSummary,
		memo.ProblemSummary,
		firstNonEmptySlice(memo.ValidationChecks),
	), decisionRecallLineMaxLen)
}

func decisionRecipeSummary(recipe DecisionRecipe) string {
	summary := firstNonEmpty(
		recipe.StrategySummary,
		firstNonEmptySlice(recipe.Preconditions),
		firstNonEmptySlice(recipe.TriggerPhrases),
	)
	if action := firstNonEmpty(decisionRecipeFirstAction(recipe)); action != "" {
		summary = firstNonEmpty(summary, "use the proven sequence") + "; start with " + action
	}
	return summarizeDecisionText(summary, decisionRecallLineMaxLen)
}

func decisionRecipeActionText(recipe DecisionRecipe) string {
	if len(recipe.OrderedActions) == 0 {
		return ""
	}
	parts := make([]string, 0, len(recipe.OrderedActions)*3)
	for _, step := range recipe.OrderedActions {
		parts = append(parts, step.Title, step.Instruction, step.Validation, step.ToolName)
	}
	return strings.Join(parts, " ")
}

func decisionRecipeFirstAction(recipe DecisionRecipe) string {
	for _, step := range recipe.OrderedActions {
		if action := firstNonEmpty(step.Instruction, step.Title); action != "" {
			return summarizeDecisionText(action, 96)
		}
	}
	return ""
}

func (d *DecisionService) recipeEnvironmentCompatibility(current *DecisionEnvFingerprint, recipe DecisionRecipe) (float64, bool) {
	if d == nil || d.store == nil || current == nil || len(recipe.SourceMemoIDs) == 0 {
		return 0, false
	}
	best := 0.0
	found := false
	for _, memoID := range recipe.SourceMemoIDs {
		memo, ok := d.store.Memo(memoID)
		if !ok {
			continue
		}
		found = true
		score := environmentCompatibility(current, memo.Environment)
		if score > best {
			best = score
		}
	}
	return best, found
}

func scoreDecisionOutcome(outcome string) float64 {
	switch normalizeDecisionOutcome(outcome) {
	case DecisionOutcomeSuccess:
		return 1
	case DecisionOutcomePartial:
		return 0.7
	case DecisionOutcomeAwaitingHuman:
		return 0.58
	case DecisionOutcomeFailure:
		return 0.42
	case DecisionOutcomeCancelled:
		return 0.32
	default:
		return 0.5
	}
}

func scoreDecisionRecency(now time.Time, values ...time.Time) float64 {
	when := latestDecisionTime(values...)
	if when.IsZero() {
		return 0.35
	}
	age := now.Sub(when.UTC())
	if age < 0 {
		age = 0
	}
	halfLife := 45 * 24 * time.Hour
	return clamp01(math.Exp2(-age.Hours()/halfLife.Hours()) + 0.08)
}

func latestDecisionTime(values ...time.Time) time.Time {
	var latest time.Time
	for _, value := range values {
		if value.IsZero() {
			continue
		}
		value = value.UTC()
		if latest.IsZero() || value.After(latest) {
			latest = value
		}
	}
	return latest
}

func formatDecisionWhyMatched(intentScore float64, anchorScore float64, graphScore float64, toolScore float64, envScore float64) string {
	parts := make([]string, 0, 4)
	if intentScore >= 0.35 {
		parts = append(parts, "matched intent")
	}
	if envScore >= 0.75 {
		parts = append(parts, "same workspace")
	} else if envScore >= 0.45 {
		parts = append(parts, "compatible environment")
	}
	if toolScore >= 0.34 {
		parts = append(parts, "shared tools")
	}
	if anchorScore >= 0.25 {
		parts = append(parts, "shared anchors")
	}
	if graphScore >= 0.25 {
		parts = append(parts, "shared graph refs")
	}
	return strings.Join(parts, " + ")
}

func firstNonEmptySlice(values []string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func decisionStringSlice(value any) []string {
	switch typed := value.(type) {
	case []string:
		return uniqueStrings(typed)
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			text, ok := item.(string)
			if !ok {
				continue
			}
			out = append(out, text)
		}
		return uniqueStrings(out)
	default:
		return nil
	}
}

func cleanDecisionPath(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return ""
	}
	return filepath.Clean(trimmed)
}

func (d *DecisionService) recipeSourceSessionID(recipe DecisionRecipe) string {
	if d == nil || d.store == nil {
		return ""
	}
	for _, memoID := range recipe.SourceMemoIDs {
		memo, ok := d.store.Memo(memoID)
		if !ok {
			continue
		}
		if memo.SessionID != "" {
			return memo.SessionID
		}
	}
	return ""
}

func cloneDecisionHits(hits []DecisionHit) []DecisionHit {
	if len(hits) == 0 {
		return nil
	}
	out := make([]DecisionHit, len(hits))
	for i := range hits {
		out[i] = cloneDecisionHit(hits[i])
	}
	return out
}
