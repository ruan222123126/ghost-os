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
			shadowReport := s.compareBucketShadow(live, shadow, bucketPlan)
			live.ShadowRead = shadowReport
		}
		live.BucketPlan = bucketPlan
		live.LayerFreshness = s.bucketLayerFreshness(bucketPlan)
		return live, nil
	}
	useBuckets := bucketPlan != nil && bucketPlan.Mode == BucketModeBucketed && len(bucketPlan.SelectedBuckets) > 0
	result, err := s.queryResultHybridLayers(effectiveQuery, scope, plan, bucketPlan, useBuckets)
	if err != nil {
		return MemoryQueryResult{}, err
	}
	result.BucketPlan = bucketPlan
	result.LayerFreshness = s.bucketLayerFreshness(bucketPlan)
	return result, nil
}

func (s *QueryService) queryIntentPlan(query MemoryQuery, scope SessionScope) *QueryIntentPlan {
	if s.planner == nil || !s.planner.Enabled() {
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
		return &BucketPlan{Mode: BucketModeLegacy, Reason: []string{"bucket read disabled"}}
	}
	return s.buckets.Plan(query, scope, intent)
}

func (s *QueryService) applyBucketQueryHints(query MemoryQuery, scope SessionScope, plan *BucketPlan) MemoryQuery {
	out := query
	if out.Namespace == "" && s.buckets != nil && s.buckets.ledger != nil {
		out.Namespace = s.buckets.ledger.Namespace()
	}
	if out.WorkspaceID == "" && s.buckets != nil && s.buckets.ledger != nil {
		out.WorkspaceID = s.buckets.ledger.WorkspaceID()
	}
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

func (s *QueryService) queryResultHybridLayers(query MemoryQuery, scope SessionScope, plan *QueryIntentPlan, bucketPlan *BucketPlan, useBuckets bool) (MemoryQueryResult, error) {
	now := time.Now().UTC()
	hotEntries := queryHot(scope, query)
	if len(hotEntries) > 0 && s.metrics != nil {
		s.metrics.l1Hits.Add(uint64(len(hotEntries)))
	}
	warmQuery := query
	warmQuery.Limit = 0
	warmEntries, err := s.warm.RetrieveCandidates(warmQuery)
	if err != nil {
		return MemoryQueryResult{}, err
	}
	if len(warmEntries) > 0 && s.metrics != nil {
		s.metrics.l2Hits.Add(uint64(len(warmEntries)))
	}
	coldQuery := query
	coldQuery.Limit = 0
	var coldEntries []MemoryEntry
	if useBuckets {
		coldEntries, err = s.cold.RetrieveWithBucketPlan(coldQuery, bucketPlan)
	} else {
		coldEntries, err = s.cold.Retrieve(coldQuery)
	}
	if err != nil {
		return MemoryQueryResult{}, err
	}
	if len(coldEntries) > 0 && s.metrics != nil {
		s.metrics.l3Hits.Add(uint64(len(coldEntries)))
	}
	var markdownEntries []MemoryEntry
	if query.IncludeMarkdown {
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
	var decisionHits []DecisionHit
	if s.decision != nil && query.IncludeDecision {
		_, decisionHits, err = s.decision.Retrieve(query, scope)
		if err != nil {
			log.Printf("[MEMORY] decision recall failed, falling back to other layers: %v", err)
			decisionHits = nil
		} else if useBuckets {
			decisionHits = filterDecisionHitsByBucketPlan(decisionHits, bucketPlan)
		}
	}
	var graphHits []GraphHit
	if s.graph != nil && query.IncludeGraph {
		_, graphHits, err = s.graph.Retrieve(query, scope)
		if err != nil {
			log.Printf("[MEMORY] graph recall failed, falling back to other layers: %v", err)
			graphHits = nil
		} else {
			if useBuckets {
				graphHits = filterGraphHitsByBucketPlan(graphHits, bucketPlan)
			}
			if len(graphHits) > 0 && s.metrics != nil {
				s.metrics.graphHits.Add(uint64(len(graphHits)))
			}
		}
	}
	truthMatches := s.truth.Query(query, plan)
	vectorHits := s.liveVectorHits(query, plan)
	if useBuckets {
		truthMatches = filterTruthMatchesByBucketPlan(truthMatches, bucketPlan)
		vectorHits = filterVectorHitsByBucketPlan(vectorHits, bucketPlan)
	}
	candidates := make([]RecallCandidate, 0, len(hotEntries)+len(warmEntries)+len(coldEntries)+len(markdownEntries)+len(decisionHits)+len(graphHits)+len(truthMatches)+len(vectorHits))
	candidates = append(candidates, s.candidatesFromEntries(hotEntries, "hot", query, now)...)
	candidates = append(candidates, s.candidatesFromEntries(warmEntries, "warm", query, now)...)
	candidates = append(candidates, s.candidatesFromEntries(coldEntries, "cold", query, now)...)
	candidates = append(candidates, s.candidatesFromEntries(markdownEntries, "markdown", query, now)...)
	candidates = append(candidates, s.candidatesFromDecisionHits(decisionHits, query, now)...)
	candidates = append(candidates, s.candidatesFromGraphHits(graphHits, query, now)...)
	candidates = append(candidates, s.candidatesFromTruthMatches(truthMatches, query, now)...)
	candidates = append(candidates, s.candidatesFromVectorHits(vectorHits, query, now)...)
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
		Entries:      entries,
		GraphHits:    graphHits,
		DecisionHits: decisionHits,
		IntentPlan:   plan,
	}
	if query.IncludeVector || query.VectorDebug || query.RerankDebug || s.rerankDebugEnabled {
		result.VectorHits = append([]VectorHit(nil), vectorHits...)
	}
	if s.truthReadRequested(query) || query.RerankDebug || s.rerankDebugEnabled {
		result.TruthHits = s.truth.DebugHits(truthMatches)
	}
	if query.RerankDebug || s.rerankDebugEnabled || s.hybridEnabled {
		result.RerankReport = report
	}
	return result, nil
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
	hits, err := s.vector.Query(query, plan)
	if err != nil {
		log.Printf("[MEMORY] vector live recall failed, continuing without vector promotions: err=%v", err)
		return nil
	}
	return hits
}
