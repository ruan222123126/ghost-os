package memory

import (
	"log"
	"strings"
	"time"
)

func (s *QueryService) queryResultHybridWithScope(query MemoryQuery, scope SessionScope) (MemoryQueryResult, error) {
	plan := s.queryIntentPlan(query, scope)
	bucketPlan := s.planBuckets(query, scope, plan)
	effectiveQuery := s.applyBucketQueryHints(query, scope, bucketPlan)
	if bucketPlan != nil && bucketPlan.Mode == BucketModeShadow {
		live, err := s.queryResultHybridLayers(effectiveQuery, scope, plan, bucketPlan, false)
		if err != nil {
			return MemoryQueryResult{}, err
		}
		shadow, shadowErr := s.queryResultHybridLayers(effectiveQuery, scope, plan, bucketPlan, true)
		if shadowErr == nil {
			s.compareBucketShadow(live, shadow, bucketPlan)
		}
		if debug := live.ensureDebug(); debug != nil {
			if bucketPlan != nil && query.BucketDebug {
				cloned := cloneBucketPlan(*bucketPlan)
				debug.BucketPlan = &cloned
				debug.LayerFreshness = s.bucketLayerFreshness(bucketPlan)
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
		debug.LayerFreshness = s.bucketLayerFreshness(bucketPlan)
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

func (s *QueryService) planBuckets(query MemoryQuery, scope SessionScope, intent *QueryIntentPlan) *BucketPlan {
	if s == nil || s.buckets == nil || !s.buckets.Enabled() {
		return nil
	}
	return s.buckets.Plan(query, scope, intent)
}

func (s *QueryService) applyBucketQueryHints(query MemoryQuery, scope SessionScope, plan *BucketPlan) MemoryQuery {
	out := query
	out.SessionHints = effectiveBucketSessionHints(out, scope)
	if plan != nil && len(plan.SelectedBuckets) > 0 {
		months := make([]string, 0, len(plan.SelectedBuckets))
		for _, bucket := range plan.SelectedBuckets {
			months = append(months, bucket.Bucket.Key.Month)
		}
		out.MonthHints = uniqueStrings(append(out.MonthHints, months...))
	}
	return out
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

func (s *QueryService) queryResultHybridLayers(query MemoryQuery, scope SessionScope, plan *QueryIntentPlan, bucketPlan *BucketPlan, useBuckets bool) (MemoryQueryResult, error) {
	now := time.Now().UTC()
	hydration := hydrationScopeFromPlan(plan)
	var hotEntries []MemoryEntry
	if hydration.hot {
		hotEntries = queryHot(scope, query)
	}
	if len(hotEntries) > 0 && s.metrics != nil {
		s.metrics.l1Hits.Add(uint64(len(hotEntries)))
	}
	warmQuery := query
	warmQuery.Limit = 0
	var (
		warmEntries []MemoryEntry
		err         error
	)
	if hydration.warm {
		warmEntries, err = s.warm.RetrieveCandidates(warmQuery)
		if err != nil {
			return MemoryQueryResult{}, err
		}
	}
	if len(warmEntries) > 0 && s.metrics != nil {
		s.metrics.l2Hits.Add(uint64(len(warmEntries)))
	}
	coldQuery := query
	coldQuery.Limit = 0
	var coldEntries []MemoryEntry
	if hydration.cold {
		if useBuckets {
			coldEntries, err = s.cold.RetrieveWithBucketPlan(coldQuery, bucketPlan)
		} else {
			coldEntries, err = s.cold.Retrieve(coldQuery)
		}
		if err != nil {
			return MemoryQueryResult{}, err
		}
	}
	if len(coldEntries) > 0 && s.metrics != nil {
		s.metrics.l3Hits.Add(uint64(len(coldEntries)))
	}
	var markdownEntries []MemoryEntry
	if hydration.markdown && query.IncludeMarkdown {
		markdownQuery := query
		markdownQuery.Limit = 0
		if useBuckets {
			markdownEntries, err = s.queryMarkdownWithPlan(markdownQuery, bucketPlan)
		} else {
			markdownEntries, err = s.queryMarkdown(markdownQuery)
		}
		if err != nil {
			return MemoryQueryResult{}, err
		}
		if len(markdownEntries) > 0 && s.metrics != nil {
			s.metrics.markdownHits.Add(uint64(len(markdownEntries)))
		}
	}
	var (
		decisionEntries []MemoryEntry
		decisionHits    []DecisionHit
	)
	if hydration.decision && s.decision != nil && query.IncludeDecision {
		decisionEntries, decisionHits, err = s.decision.Retrieve(query, scope)
		if err != nil {
			log.Printf("[MEMORY] decision recall failed, falling back to other layers: %v", err)
			decisionEntries = nil
			decisionHits = nil
		} else if useBuckets {
			decisionHits = filterDecisionHitsByBucketPlan(decisionHits, bucketPlan)
			decisionEntries = entriesFromDecisionHits(decisionHits, now)
		}
	}
	var truthMatches []truthQueryMatch
	truthPrimary := s.queryTruthPrimaryCandidates(query, plan)
	if s.truth != nil && s.truth.Enabled() && len(truthPrimary) == 0 && (s.hybridEnabled || s.truthReadRequested(query) || query.RerankDebug || s.rerankDebugEnabled) {
		truthMatches = s.truth.Query(query, plan)
	}
	var (
		graphEntries []MemoryEntry
		graphHits    []GraphHit
	)
	if hydration.graph && s.graph != nil && query.IncludeGraph {
		graphQuery := s.graphExplainQuery(query, truthPrimary, truthMatches, hotEntries, warmEntries, coldEntries, markdownEntries, decisionEntries)
		graphEntries, graphHits, err = s.graph.Retrieve(graphQuery, scope)
		if err != nil {
			log.Printf("[MEMORY] graph recall failed, falling back to other layers: %v", err)
			graphEntries = nil
			graphHits = nil
		} else {
			graphHits = s.enrichGraphHitsWithTruth(graphHits, truthPrimary, truthMatches)
			graphEntries = entriesFromGraphHits(graphHits, now)
			if useBuckets {
				graphHits = filterGraphHitsByBucketPlan(graphHits, bucketPlan)
				graphEntries = entriesFromGraphHits(graphHits, now)
			}
			if len(graphHits) > 0 && s.metrics != nil {
				s.metrics.graphHits.Add(uint64(len(graphHits)))
			}
		}
	}
	vectorHits := s.liveVectorHits(query, plan)
	primaryVectorHits := vectorHits
	if !s.useTruthPrimaryRecall(query) {
		primaryVectorHits = nil
	}
	if useBuckets {
		truthPrimary = s.filterPrimaryCandidatesByBucketPlan(truthPrimary, bucketPlan)
		truthMatches = filterTruthMatchesByBucketPlan(truthMatches, bucketPlan)
		vectorHits = filterVectorHitsByBucketPlan(vectorHits, bucketPlan)
		primaryVectorHits = filterVectorHitsByBucketPlan(primaryVectorHits, bucketPlan)
	}
	if !s.hybridEnabled && len(truthPrimary) == 0 && len(truthMatches) == 0 && len(primaryVectorHits) == 0 {
		results := make([]MemoryEntry, 0, len(hotEntries)+len(warmEntries)+len(decisionEntries)+len(graphEntries)+len(coldEntries)+len(markdownEntries))
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
		collect(hotEntries)
		collect(warmEntries)
		collectDistinct(decisionEntries)
		collectDistinct(graphEntries)
		collect(coldEntries)
		collect(markdownEntries)
		results = rankMemoryEntries(results, query, now, s.scoring)
		if query.Limit > 0 && len(results) > query.Limit {
			results = results[:query.Limit]
		}
		s.recordResultAccess(results, now)
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
	primaryCandidates = append(primaryCandidates, s.candidatesFromPrimaryCandidates(truthPrimary, query, now)...)
	if len(primaryCandidates) == 0 {
		primaryCandidates = append(primaryCandidates, s.candidatesFromTruthMatches(truthMatches, query, now)...)
	}
	primaryCandidates = append(primaryCandidates, s.candidatesFromVectorHits(primaryVectorHits, query, now)...)
	candidates := mergeRecallCandidates(primaryCandidates)
	hasPrimaryRecall := len(candidates) > 0
	candidates = appendHydratedRecallCandidates(candidates, s.candidatesFromEntries(hotEntries, "hot", query, now), true)
	candidates = appendHydratedRecallCandidates(candidates, s.candidatesFromEntries(warmEntries, "warm", query, now), true)
	candidates = appendHydratedRecallCandidates(candidates, s.candidatesFromEntries(coldEntries, "cold", query, now), !hasPrimaryRecall)
	candidates = appendHydratedRecallCandidates(candidates, s.candidatesFromEntries(markdownEntries, "markdown", query, now), !hasPrimaryRecall)
	candidates = appendHydratedRecallCandidates(candidates, s.candidatesFromDecisionHits(decisionHits, query, now), !hasPrimaryRecall)
	candidates = appendHydratedRecallCandidates(candidates, s.candidatesFromGraphHits(graphHits, query, now), !hasPrimaryRecall)
	candidates = mergeRecallCandidates(candidates)
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
	s.recordResultAccess(entries, now)
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

func (s *QueryService) bucketLayerFreshness(plan *BucketPlan) map[string]time.Time {
	if plan == nil || len(plan.SelectedBuckets) == 0 {
		return nil
	}
	freshness := make(map[string]time.Time)
	for _, selected := range plan.SelectedBuckets {
		if ts := selected.Bucket.UpdatedAt; !ts.IsZero() && ts.After(freshness["cold"]) {
			freshness["cold"] = ts
		}
		for layer, view := range selected.Bucket.Views {
			if view.UpdatedAt.After(freshness[layer]) {
				freshness[layer] = view.UpdatedAt
			}
		}
	}
	return freshness
}

func (s *QueryService) compareBucketShadow(live MemoryQueryResult, shadow MemoryQueryResult, plan *BucketPlan) *BucketShadowReport {
	overlap, shadowOnly := bucketShadowOverlap(live.Entries, shadow.Entries)
	report := &BucketShadowReport{
		Mode:             BucketModeShadow,
		LegacyEntryCount: len(live.Entries),
		BucketEntryCount: len(shadow.Entries),
		TopOverlap:       overlap,
		SelectedBuckets:  selectedBucketKeys(plan),
		ShadowOnly:       shadowOnly,
	}
	if s.metrics != nil {
		s.metrics.bucketShadowOverlapMilli.Add(uint64(overlap*1000 + 0.5))
		if len(shadowOnly) > 0 {
			s.metrics.bucketRecallOnlyHits.Add(uint64(len(shadowOnly)))
		}
	}
	return report
}

func filterDecisionHitsByBucketPlan(hits []DecisionHit, plan *BucketPlan) []DecisionHit {
	if plan == nil || len(plan.SelectedBuckets) == 0 || len(hits) == 0 {
		return hits
	}
	selectedSessions := selectedPlanSessions(plan)
	selectedMonths := selectedPlanMonths(plan)
	filtered := make([]DecisionHit, 0, len(hits))
	for _, hit := range hits {
		if len(selectedSessions) > 0 && hit.SessionID != "" {
			if _, ok := selectedSessions[strings.TrimSpace(hit.SessionID)]; ok {
				filtered = append(filtered, hit)
				continue
			}
		}
		if len(selectedMonths) > 0 && !hit.Timestamp.IsZero() {
			if _, ok := selectedMonths[bucketMonthFromTime(hit.Timestamp)]; ok {
				filtered = append(filtered, hit)
			}
		}
	}
	return filtered
}

func filterGraphHitsByBucketPlan(hits []GraphHit, plan *BucketPlan) []GraphHit {
	if plan == nil || len(plan.SelectedBuckets) == 0 || len(hits) == 0 {
		return hits
	}
	selectedSessions := selectedPlanSessions(plan)
	selectedMonths := selectedPlanMonths(plan)
	filtered := make([]GraphHit, 0, len(hits))
	for _, hit := range hits {
		matched := false
		for _, evidence := range hit.Evidence {
			if _, ok := selectedSessions[strings.TrimSpace(evidence.SessionID)]; ok {
				matched = true
				break
			}
			if _, ok := selectedMonths[bucketMonthFromTime(evidence.Timestamp)]; ok {
				matched = true
				break
			}
		}
		if matched || len(hit.Evidence) == 0 {
			filtered = append(filtered, hit)
		}
	}
	return filtered
}

func filterTruthMatchesByBucketPlan(matches []truthQueryMatch, plan *BucketPlan) []truthQueryMatch {
	if plan == nil || len(plan.SelectedBuckets) == 0 || len(matches) == 0 {
		return matches
	}
	filtered := make([]truthQueryMatch, 0, len(matches))
	for _, match := range matches {
		if bucketObjectMatchesPlan(match.Object, plan) {
			filtered = append(filtered, match)
		}
	}
	return filtered
}

func filterVectorHitsByBucketPlan(hits []VectorHit, plan *BucketPlan) []VectorHit {
	if plan == nil || len(plan.SelectedBuckets) == 0 || len(hits) == 0 {
		return hits
	}
	filtered := make([]VectorHit, 0, len(hits))
	for _, hit := range hits {
		if bucketSourceRefsMatchPlan(hit.SourceRefs, plan) {
			filtered = append(filtered, hit)
		}
	}
	return filtered
}

func selectedPlanSessions(plan *BucketPlan) map[string]struct{} {
	out := make(map[string]struct{})
	if plan == nil {
		return out
	}
	for _, bucket := range plan.SelectedBuckets {
		for _, sessionID := range bucket.Bucket.SessionIDs {
			out[strings.TrimSpace(sessionID)] = struct{}{}
		}
		for _, session := range bucket.Bucket.Sessions {
			out[strings.TrimSpace(session.SessionID)] = struct{}{}
		}
	}
	return out
}

func selectedPlanMonths(plan *BucketPlan) map[string]struct{} {
	out := make(map[string]struct{})
	if plan == nil {
		return out
	}
	for _, bucket := range plan.SelectedBuckets {
		out[strings.TrimSpace(bucket.Bucket.Key.Month)] = struct{}{}
	}
	return out
}

func bucketSourceRefsMatchPlan(refs []SourceRef, plan *BucketPlan) bool {
	if len(refs) == 0 {
		return true
	}
	sessions := selectedPlanSessions(plan)
	months := selectedPlanMonths(plan)
	for _, ref := range refs {
		if _, ok := sessions[strings.TrimSpace(ref.SessionID)]; ok {
			return true
		}
		if _, ok := months[strings.TrimSpace(ref.BucketMonth)]; ok {
			return true
		}
		if _, ok := months[bucketMonthFromTime(ref.OccurredAt)]; ok {
			return true
		}
	}
	return false
}

func bucketObjectMatchesPlan(object MemoryObject, plan *BucketPlan) bool {
	if bucketSourceRefsMatchPlan(object.SourceRefs, plan) {
		return true
	}
	for _, evidence := range object.RawEvidence {
		if bucketSourceRefsMatchPlan(evidence.SourceRefs, plan) {
			return true
		}
	}
	for _, claim := range object.Claims {
		if bucketSourceRefsMatchPlan(claim.SourceRefs, plan) {
			return true
		}
	}
	return false
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
