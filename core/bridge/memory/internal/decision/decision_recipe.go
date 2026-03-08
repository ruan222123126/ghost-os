package decision

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"log"
	"math"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	decisionRecipeMinSuccessRate = 0.66
	decisionRecipeMaxPhrases     = 6
	decisionRecipeMaxItems       = 6
)

// DecisionDistiller 周期性把多个成功 memo 蒸馏为 recipe。
type DecisionDistiller struct {
	service    *DecisionService
	interval   time.Duration
	minSupport int
	stop       chan struct{}
	startOnce  sync.Once
	stopOnce   sync.Once
	wg         sync.WaitGroup
}

func NewDecisionDistiller(service *DecisionService, interval time.Duration, minSupport int) *DecisionDistiller {
	if interval <= 0 {
		interval = defaultDecisionRecipeInterval
	}
	if minSupport <= 0 {
		minSupport = defaultDecisionRecipeMinSupport
	}
	return &DecisionDistiller{
		service:    service,
		interval:   interval,
		minSupport: minSupport,
		stop:       make(chan struct{}),
	}
}

func (d *DecisionDistiller) Start() {
	if d == nil || d.service == nil || d.service.store == nil || !d.service.recipeEnabled || d.interval <= 0 {
		return
	}
	if d.stop == nil {
		d.stop = make(chan struct{})
	}
	if d.service.debugEnabled {
		log.Printf("[MEMORY] decision recipe distiller started: interval=%s min_support=%d", d.interval, d.minSupport)
	}
	d.startOnce.Do(func() {
		d.wg.Add(1)
		go func() {
			defer d.wg.Done()
			ticker := time.NewTicker(d.interval)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					if _, err := d.DistillAll(""); err != nil && d.service.debugEnabled {
						log.Printf("[MEMORY] decision recipe distill failed: %v", err)
					}
				case <-d.stop:
					return
				}
			}
		}()
	})
}

func (d *DecisionDistiller) Stop() {
	if d == nil {
		return
	}
	d.stopOnce.Do(func() {
		if d.stop != nil {
			close(d.stop)
		}
	})
	d.wg.Wait()
}

func (d *DecisionDistiller) DistillAll(namespace string) (DecisionDistillStats, error) {
	stats := DecisionDistillStats{Namespace: distillStatsNamespace(namespace)}
	if d == nil || d.service == nil || d.service.store == nil || !d.service.recipeEnabled {
		return stats, nil
	}
	if ns := strings.TrimSpace(namespace); ns != "" {
		return d.distillNamespace(normalizeDecisionNamespace(ns), d.service.store.ListMemos(ns))
	}

	memos := d.service.store.ListMemos("")
	if len(memos) == 0 {
		return stats, nil
	}

	byNamespace := make(map[string][]DecisionMemo)
	for _, memo := range memos {
		ns := normalizeDecisionNamespace(memo.Namespace)
		byNamespace[ns] = append(byNamespace[ns], memo)
	}
	namespaces := make([]string, 0, len(byNamespace))
	for ns := range byNamespace {
		namespaces = append(namespaces, ns)
	}
	sort.Strings(namespaces)
	for _, ns := range namespaces {
		nsStats, err := d.distillNamespace(ns, byNamespace[ns])
		if err != nil {
			return stats, err
		}
		stats.MemosScanned += nsStats.MemosScanned
		stats.ClustersBuilt += nsStats.ClustersBuilt
		stats.RecipesCreated += nsStats.RecipesCreated
		stats.RecipesUpdated += nsStats.RecipesUpdated
		stats.WarningsFolded += nsStats.WarningsFolded
		stats.SkippedClusters += nsStats.SkippedClusters
	}
	if len(namespaces) == 1 {
		stats.Namespace = namespaces[0]
	}
	return stats, nil
}

func (d *DecisionDistiller) distillNamespace(namespace string, memos []DecisionMemo) (DecisionDistillStats, error) {
	recipes, clusters, stats, err := d.DistillClusters(namespace, memos)
	if err != nil {
		return stats, err
	}
	changed := false
	for _, cluster := range clusters {
		if _, err := d.service.store.UpsertCluster(cluster); err != nil {
			return stats, fmt.Errorf("upsert decision cluster: %w", err)
		}
		changed = true
	}
	for _, recipe := range recipes {
		created, err := d.service.store.UpsertRecipe(recipe)
		if err != nil {
			return stats, fmt.Errorf("upsert decision recipe: %w", err)
		}
		if created {
			stats.RecipesCreated++
		} else {
			stats.RecipesUpdated++
		}
		changed = true
	}
	if changed {
		if err := d.service.store.Persist(); err != nil {
			return stats, fmt.Errorf("persist decision recipes: %w", err)
		}
	}
	return stats, nil
}

func (d *DecisionDistiller) DistillClusters(namespace string, memos []DecisionMemo) ([]DecisionRecipe, []DecisionCluster, DecisionDistillStats, error) {
	ns := normalizeDecisionNamespace(namespace)
	stats := DecisionDistillStats{Namespace: ns}
	if d == nil || d.service == nil || d.service.store == nil || !d.service.recipeEnabled {
		return nil, nil, stats, nil
	}

	filtered := make([]DecisionMemo, 0, len(memos))
	for _, rawMemo := range memos {
		memo := normalizeDecisionMemo(rawMemo)
		if memo.Namespace != ns {
			continue
		}
		filtered = append(filtered, memo)
	}
	stats.MemosScanned = len(filtered)
	if len(filtered) == 0 {
		return nil, nil, stats, nil
	}

	clustersByKey := make(map[string][]DecisionMemo)
	for _, memo := range filtered {
		clusterKey := decisionDistillClusterKey(memo)
		clustersByKey[clusterKey] = append(clustersByKey[clusterKey], memo)
	}
	clusterKeys := make([]string, 0, len(clustersByKey))
	for key := range clustersByKey {
		clusterKeys = append(clusterKeys, key)
	}
	sort.Strings(clusterKeys)

	clusters := make([]DecisionCluster, 0, len(clusterKeys))
	recipes := make([]DecisionRecipe, 0, len(clusterKeys))
	for _, clusterKey := range clusterKeys {
		clusterMemos := cloneDecisionMemos(clustersByKey[clusterKey])
		cluster := decisionBuildCluster(ns, clusterKey, clusterMemos)
		stats.ClustersBuilt++

		recipe, ok, warningsFolded := d.buildRecipe(ns, cluster, clusterMemos)
		stats.WarningsFolded += warningsFolded
		if ok {
			cluster.RecipeID = recipe.ID
			recipes = append(recipes, recipe)
		} else {
			stats.SkippedClusters++
		}
		clusters = append(clusters, normalizeDecisionCluster(cluster))
	}
	return recipes, clusters, stats, nil
}

func (d *DecisionDistiller) buildRecipe(namespace string, cluster DecisionCluster, memos []DecisionMemo) (DecisionRecipe, bool, int) {
	successMemos := make([]DecisionMemo, 0, len(memos))
	warningMemos := make([]DecisionMemo, 0, len(memos))
	for _, memo := range memos {
		if memo.Outcome == DecisionOutcomeSuccess {
			successMemos = append(successMemos, memo)
			continue
		}
		warningMemos = append(warningMemos, memo)
	}
	warnings := decisionCollectWarningPatterns(warningMemos, decisionRecipeMaxItems)
	warningsFolded := len(warnings)
	if len(successMemos) < d.minSupport {
		return DecisionRecipe{}, false, warningsFolded
	}
	successRate := float64(len(successMemos)) / float64(len(memos))
	if successRate < decisionRecipeMinSuccessRate {
		return DecisionRecipe{}, false, warningsFolded
	}
	preconditions, envStability := decisionRecipePreconditions(successMemos)
	if envStability <= 0 {
		return DecisionRecipe{}, false, warningsFolded
	}

	intentKey := decisionClusterIntentKey(successMemos)
	triggerPhrases := decisionCollectTriggerPhrases(successMemos, decisionRecipeMaxPhrases)
	strategySummary := decisionRecipeStrategySummary(successMemos)
	recommendedTools := decisionStableToolPrefix(successMemos)
	orderedActions := decisionStableRecipeActions(successMemos)
	if len(recommendedTools) == 0 {
		recommendedTools = decisionTopToolNames(successMemos, decisionRecipeMaxItems)
	}
	if len(orderedActions) == 0 {
		orderedActions = decisionRecipeStepsFromTools(recommendedTools)
	}
	validationChecklist := decisionCollectValidationChecks(successMemos, decisionRecipeMaxItems)
	graphRefs := decisionCollectStableGraphRefs(successMemos, decisionRecipeMaxItems)
	anchorKeys := decisionCollectStableAnchorKeys(successMemos, decisionRecipeMaxItems)
	sourceMemoIDs := make([]string, 0, len(successMemos))
	successClaimObservations := make([]string, 0, len(successMemos)*4)
	successEvidenceObservations := make([]string, 0, len(successMemos)*4)
	failureClaimObservations := make([]string, 0, len(warningMemos)*2)
	createdAt := time.Time{}
	updatedAt := time.Time{}
	for _, memo := range successMemos {
		sourceMemoIDs = append(sourceMemoIDs, memo.ID)
		successClaimObservations = append(successClaimObservations, memo.SourceClaimIDs...)
		successClaimObservations = append(successClaimObservations, memo.DerivedClaimIDs...)
		successEvidenceObservations = append(successEvidenceObservations, memo.SourceEvidenceIDs...)
		if createdAt.IsZero() || memo.CreatedAt.Before(createdAt) {
			createdAt = memo.CreatedAt.UTC()
		}
		latest := latestDecisionTime(memo.LastUsedAt, memo.CreatedAt)
		if updatedAt.IsZero() || latest.After(updatedAt) {
			updatedAt = latest
		}
	}
	for _, memo := range warningMemos {
		failureClaimObservations = append(failureClaimObservations, memo.SourceClaimIDs...)
		failureClaimObservations = append(failureClaimObservations, memo.DerivedClaimIDs...)
	}
	if strategySummary == "" && len(orderedActions) > 0 {
		strategySummary = firstNonEmpty(orderedActions[0].Instruction, orderedActions[0].Title)
	}
	confidence := decisionRecipeConfidence(successRate, len(successMemos), envStability)
	sourceClaimIDs := decisionCountStableIDs(successClaimObservations, len(successMemos), 0.5)
	sourceEvidenceIDs := uniqueStrings(successEvidenceObservations)
	contradictedClaimIDs := diffStrings(decisionCountStableIDs(failureClaimObservations, len(warningMemos), 0.5), sourceClaimIDs)
	recipe := DecisionRecipe{
		DecisionLineage: DecisionLineage{
			SourceEvidenceIDs:    sourceEvidenceIDs,
			SourceClaimIDs:       sourceClaimIDs,
			ContradictedClaimIDs: contradictedClaimIDs,
			LineageSummary:       decisionLineageSummary("distilled", sourceClaimIDs, sourceEvidenceIDs, sourceMemoIDs),
			LineageVersion:       decisionLineageVersion,
			DistillerVersion:     decisionDistillerVersion,
		},
		ID:                  buildDecisionRecipeID(namespace, cluster.EnvironmentKey, intentKey, triggerPhrases),
		Namespace:           normalizeDecisionNamespace(namespace),
		IntentKey:           firstNonEmpty(intentKey, decisionLookup(strings.Join(triggerPhrases, " "))),
		EnvironmentKey:      strings.TrimSpace(cluster.EnvironmentKey),
		TriggerPhrases:      triggerPhrases,
		Preconditions:       preconditions,
		StrategySummary:     strategySummary,
		RecommendedTools:    recommendedTools,
		OrderedActions:      orderedActions,
		ValidationChecklist: validationChecklist,
		AvoidPatterns:       warnings,
		Status:              RecipeStatusActive,
		SupportCount:        len(successMemos),
		SuccessRate:         successRate,
		Confidence:          confidence,
		SourceMemoIDs:       uniqueStrings(sourceMemoIDs),
		GraphRefs:           graphRefs,
		AnchorKeys:          anchorKeys,
		CreatedAt:           createdAt,
		UpdatedAt:           updatedAt,
	}
	return normalizeDecisionRecipe(recipe), true, warningsFolded
}

func distillStatsNamespace(namespace string) string {
	if strings.TrimSpace(namespace) == "" {
		return "*"
	}
	return normalizeDecisionNamespace(namespace)
}

func decisionDistillClusterKey(memo DecisionMemo) string {
	intentPart := decisionMemoClusterIntent(memo)
	envPart := decisionClusterEnvironmentKey(memo.Environment, memo.GraphNodeRefs, memo.AnchorKeys)
	if envPart == "" {
		return intentPart
	}
	return intentPart + "|" + envPart
}

func decisionMemoClusterIntent(memo DecisionMemo) string {
	if key := strings.TrimSpace(memo.IntentKey); key != "" {
		return key
	}
	candidates := []string{
		memo.IntentSummary,
		memo.ProblemSummary,
		memo.ContextSummary,
		memo.OutcomeSummary,
		firstNonEmptySlice(memo.ValidationChecks),
	}
	for _, candidate := range candidates {
		lookup := decisionLookup(candidate)
		if lookup != "" {
			return lookup
		}
	}
	if names := decisionToolNamesFromUses(memo.ToolsUsed); len(names) > 0 {
		return "tools:" + decisionLookup(strings.Join(names, " "))
	}
	return "memo:" + strings.TrimSpace(memo.ID)
}

func decisionClusterEnvironmentKey(env DecisionEnvFingerprint, graphRefs []string, anchorKeys []string) string {
	env = normalizeDecisionEnvFingerprint(env)
	parts := make([]string, 0, 6)
	if value := decisionLookup(env.Domain); value != "" {
		parts = append(parts, "domain="+value)
	}
	if value := decisionLookup(cleanDecisionPath(env.WorkspaceRoot)); value != "" {
		parts = append(parts, "workspace="+value)
	}
	if value := decisionLookup(firstNonEmpty(env.GraphNamespace, env.TargetAppOrSite)); value != "" {
		parts = append(parts, "scope="+value)
	}
	if value := decisionLookup(firstNonEmpty(env.Platform, env.OS)); value != "" {
		parts = append(parts, "platform="+value)
	}
	if refs := uniqueStrings(graphRefs); len(refs) > 0 {
		sorted := append([]string(nil), refs...)
		sort.Strings(sorted)
		parts = append(parts, "graph="+decisionLookup(strings.Join(sorted, ",")))
	}
	if keys := uniqueStrings(anchorKeys); len(keys) > 0 {
		sorted := append([]string(nil), keys...)
		sort.Strings(sorted)
		parts = append(parts, "anchor="+decisionLookup(strings.Join(sorted, ",")))
	}
	return strings.Join(parts, "|")
}

func decisionBuildCluster(namespace string, clusterKey string, memos []DecisionMemo) DecisionCluster {
	memoIDs := make([]string, 0, len(memos))
	successCount := 0
	failureCount := 0
	updatedAt := time.Time{}
	for _, memo := range memos {
		memoIDs = append(memoIDs, memo.ID)
		if memo.Outcome == DecisionOutcomeSuccess {
			successCount++
		} else {
			failureCount++
		}
		latest := latestDecisionTime(memo.LastUsedAt, memo.CreatedAt)
		if updatedAt.IsZero() || latest.After(updatedAt) {
			updatedAt = latest
		}
	}
	return DecisionCluster{
		ID:             buildDecisionClusterID(namespace, clusterKey),
		Namespace:      normalizeDecisionNamespace(namespace),
		IntentKey:      decisionClusterIntentKey(memos),
		EnvironmentKey: strings.TrimSpace(clusterKey[strings.Index(clusterKey, "|")+1:]),
		MemoIDs:        uniqueStrings(memoIDs),
		SuccessCount:   successCount,
		FailureCount:   failureCount,
		UpdatedAt:      updatedAt,
	}
}

func decisionClusterIntentKey(memos []DecisionMemo) string {
	counts := make(map[string]int)
	for _, memo := range memos {
		if key := strings.TrimSpace(memo.IntentKey); key != "" {
			counts[key]++
		}
	}
	if best, ok := decisionTopCountKey(counts); ok {
		return best
	}
	phrases := decisionCollectTriggerPhrases(memos, 1)
	if len(phrases) == 0 {
		return ""
	}
	return decisionLookup(phrases[0])
}

func decisionCollectTriggerPhrases(memos []DecisionMemo, limit int) []string {
	candidates := make([]string, 0, len(memos)*2)
	for _, memo := range memos {
		if phrase := summarizeDecisionText(memo.IntentSummary, 96); phrase != "" {
			candidates = append(candidates, phrase)
		}
		if phrase := summarizeDecisionText(memo.ProblemSummary, 96); phrase != "" {
			candidates = append(candidates, phrase)
		}
	}
	return decisionTopStrings(candidates, limit, decisionMajorityCount(len(memos)))
}

func decisionRecipePreconditions(memos []DecisionMemo) ([]string, float64) {
	preconditions := make([]string, 0, decisionRecipeMaxItems)
	counts := make(map[string]int)
	threshold := decisionMajorityCount(len(memos))
	addStable := func(prefix string, values []string) {
		for _, value := range decisionTopStrings(values, 1, threshold) {
			preconditions = append(preconditions, prefix+value)
		}
	}

	constraints := make([]string, 0, len(memos)*2)
	domains := make([]string, 0, len(memos))
	workspaces := make([]string, 0, len(memos))
	platforms := make([]string, 0, len(memos))
	scopes := make([]string, 0, len(memos))
	targets := make([]string, 0, len(memos))
	for _, memo := range memos {
		constraints = append(constraints, memo.Constraints...)
		if value := cleanDecisionPath(memo.Environment.WorkspaceRoot); value != "" {
			workspaces = append(workspaces, value)
			counts["workspace"]++
		}
		if value := memo.Environment.Domain; value != "" {
			domains = append(domains, value)
			counts["domain"]++
		}
		if value := firstNonEmpty(memo.Environment.Platform, memo.Environment.OS); value != "" {
			platforms = append(platforms, value)
			counts["platform"]++
		}
		if value := memo.Environment.GraphNamespace; value != "" {
			scopes = append(scopes, value)
			counts["scope"]++
		}
		if value := memo.Environment.TargetAppOrSite; value != "" {
			targets = append(targets, value)
			counts["target"]++
		}
	}
	addStable("workspace=", workspaces)
	addStable("domain=", domains)
	addStable("platform=", platforms)
	addStable("graph_namespace=", scopes)
	addStable("target=", targets)
	preconditions = append(preconditions, decisionTopStrings(constraints, decisionRecipeMaxItems-len(preconditions), threshold)...)
	preconditions = uniqueStrings(preconditions)
	if len(preconditions) > decisionRecipeMaxItems {
		preconditions = preconditions[:decisionRecipeMaxItems]
	}
	stableSignals := 0
	for _, count := range counts {
		if count >= threshold {
			stableSignals++
		}
	}
	if stableSignals == 0 {
		return preconditions, 0
	}
	return preconditions, clamp01(float64(stableSignals) / 5)
}

func decisionRecipeStrategySummary(memos []DecisionMemo) string {
	candidates := make([]string, 0, len(memos)*2)
	for _, memo := range memos {
		if summary := summarizeDecisionText(memo.StrategySummary, decisionOutcomeSummaryMaxLen); summary != "" {
			candidates = append(candidates, summary)
			continue
		}
		if summary := summarizeDecisionText(memo.OutcomeSummary, decisionOutcomeSummaryMaxLen); summary != "" {
			candidates = append(candidates, summary)
		}
	}
	best := decisionTopStrings(candidates, 1, 1)
	if len(best) == 0 {
		return ""
	}
	return best[0]
}

func decisionStableToolPrefix(memos []DecisionMemo) []string {
	sequences := make([][]string, 0, len(memos))
	for _, memo := range memos {
		names := uniqueStrings(decisionToolNamesFromUses(memo.ToolsUsed))
		if len(names) == 0 {
			continue
		}
		sequences = append(sequences, names)
	}
	if len(sequences) == 0 {
		return nil
	}
	threshold := decisionMajorityCount(len(sequences))
	prefix := make([]string, 0, decisionRecipeMaxItems)
	for index := 0; index < decisionRecipeMaxItems; index++ {
		counts := make(map[string]int)
		available := 0
		for _, seq := range sequences {
			if index >= len(seq) {
				continue
			}
			available++
			counts[seq[index]]++
		}
		if available < threshold {
			break
		}
		name, count, ok := decisionTopCountEntry(counts)
		if !ok || count < threshold {
			break
		}
		prefix = append(prefix, name)
	}
	return uniqueStrings(prefix)
}

func decisionStableRecipeActions(memos []DecisionMemo) []RecipeStep {
	sequences := make([][]RecipeStep, 0, len(memos))
	for _, memo := range memos {
		steps := decisionRecipeStepCandidates(memo)
		if len(steps) == 0 {
			continue
		}
		sequences = append(sequences, steps)
	}
	if len(sequences) == 0 {
		return nil
	}
	threshold := decisionMajorityCount(len(sequences))
	ordered := make([]RecipeStep, 0, decisionRecipeMaxItems)
	for index := 0; index < decisionRecipeMaxItems; index++ {
		counts := make(map[string]int)
		stepsByKey := make(map[string]RecipeStep)
		available := 0
		for _, seq := range sequences {
			if index >= len(seq) {
				continue
			}
			step := normalizeRecipeStep(seq[index])
			fingerprint := decisionRecipeStepFingerprint(step)
			if fingerprint == "" {
				continue
			}
			available++
			counts[fingerprint]++
			stepsByKey[fingerprint] = step
		}
		if available < threshold {
			break
		}
		fingerprint, count, ok := decisionTopCountEntry(counts)
		if !ok || count < threshold {
			break
		}
		ordered = append(ordered, stepsByKey[fingerprint])
	}
	return normalizeRecipeSteps(ordered)
}

func decisionRecipeStepCandidates(memo DecisionMemo) []RecipeStep {
	steps := make([]RecipeStep, 0, max(len(memo.KeySteps), len(memo.ToolsUsed)))
	for index, step := range memo.KeySteps {
		recipeStep := RecipeStep{
			Title:       summarizeDecisionText(step.Title, 96),
			Instruction: summarizeDecisionText(firstNonEmpty(step.Summary, step.Title), 120),
		}
		if index < len(memo.ToolsUsed) {
			recipeStep.ToolName = strings.TrimSpace(memo.ToolsUsed[index].Name)
		}
		steps = append(steps, recipeStep)
	}
	if len(steps) > 0 {
		return normalizeRecipeSteps(steps)
	}
	for _, tool := range memo.ToolsUsed {
		steps = append(steps, RecipeStep{
			Instruction: summarizeDecisionText(firstNonEmpty(tool.Purpose, "Use "+tool.Name), 120),
			ToolName:    strings.TrimSpace(tool.Name),
		})
	}
	return normalizeRecipeSteps(steps)
}

func decisionRecipeStepsFromTools(toolNames []string) []RecipeStep {
	if len(toolNames) == 0 {
		return nil
	}
	steps := make([]RecipeStep, 0, len(toolNames))
	for _, toolName := range toolNames {
		steps = append(steps, RecipeStep{
			Instruction: summarizeDecisionText(firstNonEmpty(decisionToolPurpose(toolName), "Use "+toolName), 120),
			ToolName:    strings.TrimSpace(toolName),
		})
	}
	return normalizeRecipeSteps(steps)
}

func decisionCollectValidationChecks(memos []DecisionMemo, limit int) []string {
	values := make([]string, 0, len(memos)*2)
	for _, memo := range memos {
		values = append(values, memo.ValidationChecks...)
	}
	return decisionTopStrings(values, limit, decisionMajorityCount(len(memos)))
}

func decisionCollectWarningPatterns(memos []DecisionMemo, limit int) []string {
	values := make([]string, 0, len(memos)*4)
	for _, memo := range memos {
		values = append(values, memo.AvoidPatterns...)
		values = append(values, memo.FailureReasons...)
		for _, need := range memo.NeedsHumanFor {
			values = append(values, "ask human early for "+need)
		}
	}
	return decisionTopStrings(values, limit, 1)
}

func decisionCollectStableGraphRefs(memos []DecisionMemo, limit int) []string {
	values := make([]string, 0, len(memos)*3)
	for _, memo := range memos {
		values = append(values, memo.GraphNodeRefs...)
		values = append(values, memo.GraphEdgeRefs...)
	}
	return decisionTopStrings(values, limit, decisionMajorityCount(len(memos)))
}

func decisionCollectStableAnchorKeys(memos []DecisionMemo, limit int) []string {
	values := make([]string, 0, len(memos)*3)
	for _, memo := range memos {
		values = append(values, memo.AnchorKeys...)
	}
	return decisionTopStrings(values, limit, decisionMajorityCount(len(memos)))
}

func decisionTopToolNames(memos []DecisionMemo, limit int) []string {
	values := make([]string, 0, len(memos)*3)
	for _, memo := range memos {
		values = append(values, decisionToolNamesFromUses(memo.ToolsUsed)...)
	}
	return decisionTopStrings(values, limit, 1)
}

func decisionTopStrings(values []string, limit int, minCount int) []string {
	if limit <= 0 || len(values) == 0 {
		return nil
	}
	counts := make(map[string]int, len(values))
	canonical := make(map[string]string, len(values))
	for _, raw := range values {
		trimmed := strings.TrimSpace(raw)
		lookup := decisionLookup(trimmed)
		if lookup == "" {
			continue
		}
		counts[lookup]++
		if existing, ok := canonical[lookup]; !ok || len(trimmed) < len(existing) {
			canonical[lookup] = trimmed
		}
	}
	if len(counts) == 0 {
		return nil
	}
	type entry struct {
		key   string
		count int
		text  string
	}
	entries := make([]entry, 0, len(counts))
	for key, count := range counts {
		if count < minCount {
			continue
		}
		entries = append(entries, entry{key: key, count: count, text: canonical[key]})
	}
	if len(entries) == 0 {
		return nil
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].count != entries[j].count {
			return entries[i].count > entries[j].count
		}
		if len(entries[i].text) != len(entries[j].text) {
			return len(entries[i].text) < len(entries[j].text)
		}
		return entries[i].key < entries[j].key
	})
	if len(entries) > limit {
		entries = entries[:limit]
	}
	out := make([]string, 0, len(entries))
	for _, item := range entries {
		out = append(out, item.text)
	}
	return uniqueStrings(out)
}

func decisionTopCountKey(counts map[string]int) (string, bool) {
	key, _, ok := decisionTopCountEntry(counts)
	return key, ok
}

func decisionTopCountEntry(counts map[string]int) (string, int, bool) {
	if len(counts) == 0 {
		return "", 0, false
	}
	bestKey := ""
	bestCount := 0
	for key, count := range counts {
		if count > bestCount || (count == bestCount && key < bestKey) {
			bestKey = key
			bestCount = count
		}
	}
	if bestKey == "" {
		return "", 0, false
	}
	return bestKey, bestCount, true
}

func decisionRecipeStepFingerprint(step RecipeStep) string {
	step = normalizeRecipeStep(step)
	return strings.Join([]string{step.Title, step.Instruction, step.ToolName, step.Validation}, "|")
}

func decisionRecipeConfidence(successRate float64, supportCount int, envStability float64) float64 {
	supportScore := clamp01(float64(supportCount) / float64(max(defaultDecisionRecipeMinSupport+2, 5)))
	return clamp01(0.45 + successRate*0.25 + supportScore*0.20 + envStability*0.10)
}

func decisionMajorityCount(total int) int {
	if total <= 1 {
		return total
	}
	return max(2, int(math.Ceil(float64(total)*0.66)))
}

func buildDecisionClusterID(namespace string, clusterKey string) string {
	hash := sha1.Sum([]byte(normalizeDecisionNamespace(namespace) + "|" + strings.TrimSpace(clusterKey)))
	return "cluster-" + hex.EncodeToString(hash[:8])
}

func buildDecisionRecipeID(namespace string, environmentKey string, intentKey string, triggerPhrases []string) string {
	seed := strings.Join([]string{
		normalizeDecisionNamespace(namespace),
		strings.TrimSpace(environmentKey),
		strings.TrimSpace(intentKey),
		strings.Join(triggerPhrases, "|"),
	}, "|")
	hash := sha1.Sum([]byte(seed))
	return "recipe-" + hex.EncodeToString(hash[:8])
}
