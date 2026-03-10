package memory

import (
	"log"
	"strings"
	"time"
)

func (s *QueryService) queryResultHybridWithScope(query MemoryQuery, scope SessionScope) (MemoryQueryResult, error) {
	plan := s.queryIntentPlan(query, scope)
	bucketPlan := s.planBuckets(query, scope, plan)
	effectiveQuery := applyBucketQueryHints(query, scope, bucketPlan)
	if bucketPlan != nil && bucketPlan.Mode == BucketModeShadow {
		live, err := s.queryResultHybridLayers(effectiveQuery, scope, plan, bucketPlan, false)
		if err != nil {
			return MemoryQueryResult{}, err
		}
		shadow, shadowErr := s.queryResultHybridLayers(effectiveQuery, scope, plan, bucketPlan, true)
		if shadowErr == nil {
			compareBucketShadow(live, shadow, bucketPlan, s.metrics)
		}
		if debug := live.ensureDebug(); debug != nil {
			if bucketPlan != nil && query.BucketDebug {
				cloned := cloneBucketPlan(*bucketPlan)
				debug.BucketPlan = &cloned
				debug.LayerFreshness = bucketLayerFreshness(bucketPlan)
			}
			if !hasMemoryQueryDebug(*debug) {
				live.debug = nil
			}
		}
		return live, nil
	}
	useBuckets := bucketPlan != nil && bucketPlan.Mode == BucketModeBucketed && len(bucketPlan.SelectedBuckets) > 0
	result, err := s.queryResultHybridLayers(effectiveQuery, scope, plan, bucketPlan, useBuckets)
	if err != nil {
		return MemoryQueryResult{}, err
	}
	if bucketPlan != nil && query.BucketDebug {
		debug := result.ensureDebug()
		cloned := cloneBucketPlan(*bucketPlan)
		debug.BucketPlan = &cloned
		debug.LayerFreshness = bucketLayerFreshness(bucketPlan)
	}
	return result, nil
}

func (s *QueryService) queryIntentPlan(query MemoryQuery, scope SessionScope) *QueryIntentPlan {
	if s.planner == nil || !s.planner.Enabled() {
		return nil
	}
	if !s.hybridEnabled && !query.IntentDebug && !query.RerankDebug && !query.BucketDebug && !s.truthReadRequested(query) {
		return nil
	}
	planned, err := s.planner.Plan(query, scope)
	if err != nil {
		if s.metrics != nil {
			s.metrics.plannerErrors.Add(1)
		}
		log.Printf("[MEMORY] hybrid planner failed, continuing without planner facets: err=%v", err)
		return nil
	}
	planned = normalizeQueryIntentPlan(planned)
	if !hasIntentPlan(planned) {
		return nil
	}
	return &planned
}

type queryHydrationScope struct {
	hot      bool
	warm     bool
	cold     bool
	markdown bool
	decision bool
	graph    bool
}

func hydrationScopeFromPlan(plan *QueryIntentPlan) queryHydrationScope {
	if plan == nil || len(plan.Hydration) == 0 {
		return queryHydrationScope{hot: true, warm: true, cold: true, markdown: true, decision: true, graph: true}
	}
	all := queryHydrationScope{}
	for _, layer := range plan.Hydration {
		switch strings.TrimSpace(layer) {
		case "hot":
			all.hot = true
		case "warm":
			all.warm = true
		case "cold":
			all.cold = true
		case "markdown":
			all.markdown = true
		case "decision":
			all.decision = true
		case "graph":
			all.graph = true
		}
	}
	return all
}

func (s *QueryService) effectiveHydrationScope(query MemoryQuery, plan *QueryIntentPlan) queryHydrationScope {
	if s == nil {
		return hydrationScopeFromPlan(plan)
	}
	if !s.hybridEnabled && !s.truthReadRequested(query) && !query.RerankDebug && !s.rerankDebugEnabled && !query.BucketDebug {
		return hydrationScopeFromPlan(nil)
	}
	return hydrationScopeFromPlan(plan)
}

type queryContext struct {
	query      MemoryQuery
	scope      SessionScope
	plan       *QueryIntentPlan
	bucketPlan *BucketPlan
	useBuckets bool
	now        time.Time
	hydration  queryHydrationScope
}

func newQueryContext(query MemoryQuery, scope SessionScope, plan *QueryIntentPlan, bucketPlan *BucketPlan, useBuckets bool, hydration queryHydrationScope) queryContext {
	return queryContext{
		query:      query,
		scope:      scope,
		plan:       plan,
		bucketPlan: bucketPlan,
		useBuckets: useBuckets,
		now:        time.Now().UTC(),
		hydration:  hydration,
	}
}

func (s *QueryService) hotEntries(ctx queryContext) []MemoryEntry {
	if !ctx.hydration.hot {
		return nil
	}
	return queryHot(ctx.scope, ctx.query)
}

func (s *QueryService) warmEntries(ctx queryContext) ([]MemoryEntry, error) {
	if !ctx.hydration.warm {
		return nil, nil
	}
	warmQuery := ctx.query
	warmQuery.Limit = 0
	return s.warm.RetrieveCandidates(warmQuery)
}

func (s *QueryService) coldEntries(ctx queryContext) ([]MemoryEntry, error) {
	if !ctx.hydration.cold {
		return nil, nil
	}
	coldQuery := ctx.query
	coldQuery.Limit = 0
	if ctx.useBuckets {
		return s.cold.RetrieveWithBucketPlan(coldQuery, ctx.bucketPlan)
	}
	return s.cold.Retrieve(coldQuery)
}

func (s *QueryService) markdownEntries(ctx queryContext) ([]MemoryEntry, error) {
	if !ctx.hydration.markdown || !ctx.query.IncludeMarkdown {
		return nil, nil
	}
	markdownQuery := ctx.query
	markdownQuery.Limit = 0
	if ctx.useBuckets {
		return s.queryMarkdownWithPlan(markdownQuery, ctx.bucketPlan)
	}
	return s.queryMarkdown(markdownQuery)
}

func (s *QueryService) decisionRecall(ctx queryContext) ([]MemoryEntry, []DecisionHit, error) {
	if !ctx.hydration.decision || s.decision == nil || !ctx.query.IncludeDecision {
		return nil, nil, nil
	}
	decisionEntries, decisionHits, err := s.decision.Retrieve(ctx.query, ctx.scope)
	if err != nil {
		log.Printf("[MEMORY] decision recall failed, falling back to other layers: %v", err)
		return nil, nil, nil
	}
	if ctx.useBuckets {
		decisionHits = filterDecisionHitsByBucketPlan(decisionHits, ctx.bucketPlan)
		decisionEntries = entriesFromDecisionHits(decisionHits, ctx.now)
	}
	return decisionEntries, decisionHits, nil
}

func (s *QueryService) truthRecall(ctx queryContext) ([]PrimaryCandidate, []truthQueryMatch) {
	truthPrimary := s.queryTruthPrimaryCandidates(ctx.query, ctx.plan)
	var truthMatches []truthQueryMatch
	if s.truth != nil && s.truth.Enabled() && len(truthPrimary) == 0 && (s.hybridEnabled || s.truthReadRequested(ctx.query) || ctx.query.RerankDebug || s.rerankDebugEnabled) {
		truthMatches = s.truth.Query(ctx.query, ctx.plan)
	}
	return truthPrimary, truthMatches
}

func (s *QueryService) graphRecall(ctx queryContext, truthPrimary []PrimaryCandidate, truthMatches []truthQueryMatch, entryGroups ...[]MemoryEntry) ([]MemoryEntry, []GraphHit, error) {
	if !ctx.hydration.graph || s.graph == nil || !ctx.query.IncludeGraph {
		return nil, nil, nil
	}
	graphQuery := s.graphExplainQuery(ctx.query, truthPrimary, truthMatches, entryGroups...)
	graphEntries, graphHits, err := s.graph.Retrieve(graphQuery, ctx.scope)
	if err != nil {
		log.Printf("[MEMORY] graph recall failed, falling back to other layers: %v", err)
		return nil, nil, nil
	}
	graphHits = s.enrichGraphHitsWithTruth(graphHits, truthPrimary, truthMatches)
	graphEntries = entriesFromGraphHits(graphHits, ctx.now)
	if ctx.useBuckets {
		graphHits = filterGraphHitsByBucketPlan(graphHits, ctx.bucketPlan)
		graphEntries = entriesFromGraphHits(graphHits, ctx.now)
	}
	if len(graphHits) > 0 && s.metrics != nil {
		s.metrics.graphHits.Add(uint64(len(graphHits)))
	}
	return graphEntries, graphHits, nil
}

func (s *QueryService) vectorRecall(ctx queryContext) ([]VectorHit, []VectorHit) {
	vectorHits := s.liveVectorHits(ctx.query, ctx.plan)
	primaryVectorHits := vectorHits
	if !s.useTruthPrimaryRecall(ctx.query) {
		primaryVectorHits = nil
	}
	return vectorHits, primaryVectorHits
}

func (s *QueryService) queryResultHybridLayers(query MemoryQuery, scope SessionScope, plan *QueryIntentPlan, bucketPlan *BucketPlan, useBuckets bool) (MemoryQueryResult, error) {
	ctx := newQueryContext(query, scope, plan, bucketPlan, useBuckets, s.effectiveHydrationScope(query, plan))
	hotEntries := s.hotEntries(ctx)
	hygieneEntries, err := s.queryHygieneCards(query, scope)
	if err != nil {
		return MemoryQueryResult{}, err
	}
	if len(hotEntries) > 0 && s.metrics != nil {
		s.metrics.l1Hits.Add(uint64(len(hotEntries)))
	}
	warmEntries, err := s.warmEntries(ctx)
	if err != nil {
		return MemoryQueryResult{}, err
	}
	if len(warmEntries) > 0 && s.metrics != nil {
		s.metrics.l2Hits.Add(uint64(len(warmEntries)))
	}
	coldEntries, err := s.coldEntries(ctx)
	if err != nil {
		return MemoryQueryResult{}, err
	}
	if len(coldEntries) > 0 && s.metrics != nil {
		s.metrics.l3Hits.Add(uint64(len(coldEntries)))
	}
	markdownEntries, err := s.markdownEntries(ctx)
	if err != nil {
		return MemoryQueryResult{}, err
	}
	if len(markdownEntries) > 0 && s.metrics != nil {
		s.metrics.markdownHits.Add(uint64(len(markdownEntries)))
	}
	decisionEntries, decisionHits, err := s.decisionRecall(ctx)
	if err != nil {
		return MemoryQueryResult{}, err
	}
	truthPrimary, truthMatches := s.truthRecall(ctx)
	graphEntries, graphHits, err := s.graphRecall(ctx, truthPrimary, truthMatches, hotEntries, warmEntries, coldEntries, markdownEntries, decisionEntries)
	if err != nil {
		return MemoryQueryResult{}, err
	}
	vectorHits, primaryVectorHits := s.vectorRecall(ctx)
	if useBuckets {
		truthPrimary = s.filterPrimaryCandidatesByBucketPlan(truthPrimary, bucketPlan)
		truthMatches = filterTruthMatchesByBucketPlan(truthMatches, bucketPlan)
		vectorHits = filterVectorHitsByBucketPlan(vectorHits, bucketPlan)
		primaryVectorHits = filterVectorHitsByBucketPlan(primaryVectorHits, bucketPlan)
	}
	if !s.hybridEnabled && len(truthPrimary) == 0 && len(truthMatches) == 0 && len(primaryVectorHits) == 0 {
		results := make([]MemoryEntry, 0, len(hygieneEntries)+len(hotEntries)+len(warmEntries)+len(decisionEntries)+len(graphEntries)+len(coldEntries)+len(markdownEntries))
		seenIDs := make(map[string]struct{}, 32)
		seenFingerprints := make(map[string]struct{}, 32)
		collect := func(entries []MemoryEntry) {
			for _, entry := range entries {
				if _, ok := seenIDs[entry.ID]; ok {
					continue
				}
				seenIDs[entry.ID] = struct{}{}
				fingerprint := normalizeRecallText(firstNonEmpty(entry.Summary, entry.Content))
				if fingerprint != "" {
					seenFingerprints[fingerprint] = struct{}{}
				}
				results = append(results, entry)
			}
		}
		collectDistinct := func(entries []MemoryEntry) {
			for _, entry := range entries {
				if _, ok := seenIDs[entry.ID]; ok {
					continue
				}
				fingerprint := normalizeRecallText(firstNonEmpty(entry.Summary, entry.Content))
				if fingerprint != "" {
					if _, ok := seenFingerprints[fingerprint]; ok {
						continue
					}
					seenFingerprints[fingerprint] = struct{}{}
				}
				seenIDs[entry.ID] = struct{}{}
				results = append(results, entry)
			}
		}
		collect(hygieneEntries)
		collect(hotEntries)
		collect(warmEntries)
		collectDistinct(decisionEntries)
		collectDistinct(graphEntries)
		collect(coldEntries)
		collect(markdownEntries)
		results = s.filterEntriesByHygiene(results, query)
		results = rankMemoryEntries(results, query, ctx.now, s.scoring)
		if query.Limit > 0 && len(results) > query.Limit {
			results = results[:query.Limit]
		}
		s.recordResultAccess(results, ctx.now)
		result := MemoryQueryResult{Entries: results}
		debug := result.ensureDebug()
		if query.GraphDebug && len(graphHits) > 0 {
			debug.GraphHits = append([]GraphHit(nil), graphHits...)
		}
		if (query.IncludeDecision || query.DecisionDebug) && len(decisionHits) > 0 {
			debug.DecisionHits = append([]DecisionHit(nil), decisionHits...)
		}
		if plan != nil && (query.IntentDebug || query.RerankDebug || query.BucketDebug) {
			cloned := cloneQueryIntentPlan(*plan)
			debug.IntentPlan = &cloned
		}
		if query.IncludeVector || query.VectorDebug || query.RerankDebug || s.rerankDebugEnabled {
			debug.VectorHits = append([]VectorHit(nil), vectorHits...)
		}
		if s.truth != nil && s.truth.Enabled() && (s.truthReadRequested(query) || query.RerankDebug || s.rerankDebugEnabled) {
			if len(truthPrimary) > 0 {
				debug.TruthHits = s.truth.DebugHitsFromPrimaryCandidates(truthPrimary)
			} else {
				debug.TruthHits = s.truth.DebugHits(truthMatches)
			}
		}
		if !hasMemoryQueryDebug(*debug) {
			result.debug = nil
		}
		return result, nil
	}
	primaryCandidates := make([]RecallCandidate, 0, len(truthPrimary)+len(truthMatches)+len(primaryVectorHits))
	primaryCandidates = append(primaryCandidates, s.candidatesFromPrimaryCandidates(truthPrimary, query, ctx.now)...)
	if len(primaryCandidates) == 0 {
		primaryCandidates = append(primaryCandidates, s.candidatesFromTruthMatches(truthMatches, query, ctx.now)...)
	}
	primaryCandidates = append(primaryCandidates, s.candidatesFromVectorHits(primaryVectorHits, query, ctx.now)...)
	candidates := mergeRecallCandidates(primaryCandidates)
	hasPrimaryRecall := len(candidates) > 0
	candidates = appendHydratedRecallCandidates(candidates, s.candidatesFromEntries(hygieneEntries, "hygiene", query, ctx.now), true)
	candidates = appendHydratedRecallCandidates(candidates, s.candidatesFromEntries(hotEntries, "hot", query, ctx.now), true)
	candidates = appendHydratedRecallCandidates(candidates, s.candidatesFromEntries(warmEntries, "warm", query, ctx.now), true)
	candidates = appendHydratedRecallCandidates(candidates, s.candidatesFromEntries(coldEntries, "cold", query, ctx.now), !hasPrimaryRecall)
	candidates = appendHydratedRecallCandidates(candidates, s.candidatesFromEntries(markdownEntries, "markdown", query, ctx.now), !hasPrimaryRecall)
	candidates = appendHydratedRecallCandidates(candidates, s.candidatesFromDecisionHits(decisionHits, query, ctx.now), !hasPrimaryRecall)
	candidates = appendHydratedRecallCandidates(candidates, s.candidatesFromGraphHits(graphHits, query, ctx.now), !hasPrimaryRecall)
	candidates = mergeRecallCandidates(candidates)
	candidates = s.filterRecallCandidatesByHygiene(candidates, query)
	candidates = verifyRecallCandidates(candidates, s.truth, recallVerifyConfig{
		minSupportRefs:  max(s.truthMinSupportRefs, 2),
		conflictPenalty: s.conflictPenalty,
	})
	ranked, report := rerankRecallCandidates(candidates, hybridRerankConfig{conflictPenalty: s.conflictPenalty})
	if query.MinConfidence > 0 {
		filtered := ranked[:0]
		for _, candidate := range ranked {
			if candidate.Entry.Confidence < clamp01(query.MinConfidence) {
				if s.metrics != nil {
					s.metrics.lowConfidenceFiltered.Add(1)
				}
				continue
			}
			filtered = append(filtered, candidate)
		}
		ranked = filtered
	}
	if query.Limit > 0 && len(ranked) > query.Limit {
		ranked = ranked[:query.Limit]
	}
	entries := make([]MemoryEntry, 0, len(ranked))
	for _, candidate := range ranked {
		entries = append(entries, finalizeCandidateEntry(candidate))
	}
	if s.metrics != nil {
		s.metrics.hybridRerankRuns.Add(1)
		for _, entry := range entries {
			s.metrics.resultConfidenceSumMilli.Add(uint64(entry.Confidence*1000 + 0.5))
			s.metrics.resultConfidenceCount.Add(1)
		}
	}
	s.recordResultAccess(entries, ctx.now)
	result := MemoryQueryResult{
		Entries: entries,
	}
	debug := result.ensureDebug()
	if query.GraphDebug && len(graphHits) > 0 {
		debug.GraphHits = append([]GraphHit(nil), graphHits...)
	}
	if (query.IncludeDecision || query.DecisionDebug) && len(decisionHits) > 0 {
		debug.DecisionHits = append([]DecisionHit(nil), decisionHits...)
	}
	if plan != nil && (query.IntentDebug || query.RerankDebug || query.BucketDebug) {
		cloned := cloneQueryIntentPlan(*plan)
		debug.IntentPlan = &cloned
	}
	if query.IncludeVector || query.VectorDebug || query.RerankDebug || s.rerankDebugEnabled {
		debug.VectorHits = append([]VectorHit(nil), vectorHits...)
	}
	if s.truth != nil && s.truth.Enabled() && (s.truthReadRequested(query) || query.RerankDebug || s.rerankDebugEnabled) {
		if len(truthPrimary) > 0 {
			debug.TruthHits = s.truth.DebugHitsFromPrimaryCandidates(truthPrimary)
		} else {
			debug.TruthHits = s.truth.DebugHits(truthMatches)
		}
	}
	if query.RerankDebug || s.rerankDebugEnabled {
		debug.RerankReport = report
	}
	if !hasMemoryQueryDebug(*debug) {
		result.debug = nil
	}
	return result, nil
}

func (s *QueryService) queryTruthPrimaryCandidates(query MemoryQuery, plan *QueryIntentPlan) []PrimaryCandidate {
	if s == nil || s.truth == nil || !s.truth.Enabled() {
		return nil
	}
	if !s.useTruthPrimaryRecall(query) {
		return nil
	}
	return s.truth.QueryPrimary(truthQueryOptionsFromMemoryQuery(query, plan))
}

func (s *QueryService) useTruthPrimaryRecall(query MemoryQuery) bool {
	if s == nil {
		return false
	}
	return s.hybridEnabled || s.truthReadRequested(query) || query.RerankDebug || s.rerankDebugEnabled
}

func (s *QueryService) filterPrimaryCandidatesByBucketPlan(candidates []PrimaryCandidate, plan *BucketPlan) []PrimaryCandidate {
	if s == nil || s.truth == nil || !s.truth.Enabled() || plan == nil || len(plan.SelectedBuckets) == 0 || len(candidates) == 0 {
		return candidates
	}
	filtered := make([]PrimaryCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		object, ok := s.truth.LookupObject(candidate.ObjectID)
		if !ok || !bucketObjectMatchesPlan(object, plan) {
			continue
		}
		filtered = append(filtered, candidate)
	}
	return filtered
}

func (s *QueryService) liveVectorHits(query MemoryQuery, plan *QueryIntentPlan) []VectorHit {
	if s.vector == nil || !s.vector.Enabled() {
		return nil
	}
	if !s.hybridEnabled && !query.IncludeVector && !query.VectorDebug && !query.RerankDebug && !s.rerankDebugEnabled {
		return nil
	}
	hits, err := s.vector.Query(query, plan)
	if err != nil {
		log.Printf("[MEMORY] vector live recall failed, continuing without vector promotions: err=%v", err)
		return nil
	}
	return hits
}

func hasIntentPlan(plan QueryIntentPlan) bool {
	return strings.TrimSpace(plan.IntentKey) != "" || len(plan.Constraints) > 0 || len(plan.Entities) > 0 || len(plan.Environment) > 0 || len(plan.Risks) > 0 || strings.TrimSpace(plan.RecallMode) != "" || len(plan.Hydration) > 0 || plan.Truth != (TruthQueryOptions{}) || len(plan.Terms) > 0
}
