package memory

import (
	"log"
	"strings"
	"sync"
	"time"
)

func (s *QueryService) shouldRunShadow(query MemoryQuery) bool {
	if s == nil {
		return false
	}
	return s.shadowEnabled || query.IncludeVector || query.VectorDebug || query.IntentDebug || query.ShadowDebug
}

func (s *QueryService) runShadowRecall(query MemoryQuery, scope SessionScope, live MemoryQueryResult) (*QueryIntentPlan, []VectorHit, *ShadowRecallReport) {
	if !s.shouldRunShadow(query) {
		return nil, nil, nil
	}
	start := time.Now().UTC()
	plannerRequested := s.shadowEnabled || query.IntentDebug || query.ShadowDebug
	vectorRequested := s.shadowEnabled || query.IncludeVector || query.VectorDebug || query.ShadowDebug

	var (
		mu         sync.Mutex
		plan       *QueryIntentPlan
		vectorHits []VectorHit
		wg         sync.WaitGroup
	)

	if plannerRequested && s.planner != nil && s.planner.Enabled() {
		wg.Add(1)
		go func() {
			defer wg.Done()
			planned, err := s.planner.Plan(query, scope)
			if err != nil {
				if s.metrics != nil {
					s.metrics.plannerErrors.Add(1)
				}
				log.Printf("[MEMORY] intent planner failed, continuing with live recall only: err=%v", err)
				return
			}
			planned = normalizeQueryIntentPlan(planned)
			if !hasIntentPlan(planned) {
				return
			}
			mu.Lock()
			copyPlan := planned
			plan = &copyPlan
			mu.Unlock()
		}()
	}

	if vectorRequested && s.vector != nil && s.vector.Enabled() {
		wg.Add(1)
		go func() {
			defer wg.Done()
			hits, err := s.vector.Query(query, nil)
			if err != nil {
				log.Printf("[MEMORY] vector shadow recall failed, continuing with live recall only: err=%v", err)
				return
			}
			mu.Lock()
			vectorHits = append([]VectorHit(nil), hits...)
			mu.Unlock()
		}()
	}

	wg.Wait()
	report := &ShadowRecallReport{
		PlannerUsed: plannerRequested && plan != nil,
		VectorHits:  len(vectorHits),
		LatencyMs:   time.Since(start).Milliseconds(),
	}
	report.TopOverlap, report.ShadowOnlyCandidates, report.WouldPromote = shadowOverlapSummary(live.Entries, vectorHits, query)
	s.recordShadowMetrics(*report)

	var outPlan *QueryIntentPlan
	if plannerRequested && plan != nil {
		copyPlan := *plan
		outPlan = &copyPlan
	}
	var outHits []VectorHit
	if vectorRequested && len(vectorHits) > 0 {
		outHits = append([]VectorHit(nil), vectorHits...)
	}
	if !s.shadowEnabled && !query.ShadowDebug {
		report = nil
	}
	return outPlan, outHits, report
}

func (s *QueryService) recordShadowMetrics(report ShadowRecallReport) {
	if s == nil || s.metrics == nil {
		return
	}
	s.metrics.shadowQueries.Add(1)
	s.metrics.vectorShadowHits.Add(uint64(report.VectorHits))
	s.metrics.shadowOnlyCandidates.Add(uint64(len(report.ShadowOnlyCandidates)))
	if report.LatencyMs > 0 {
		s.metrics.shadowLatencyMs.Add(uint64(report.LatencyMs))
	}
	s.metrics.shadowOverlapMilli.Add(uint64(report.TopOverlap*1000 + 0.5))
	if report.WouldPromote {
		s.metrics.shadowWouldHelpCount.Add(1)
	}
}

func hasIntentPlan(plan QueryIntentPlan) bool {
	return strings.TrimSpace(plan.IntentKey) != "" || len(plan.Constraints) > 0 || len(plan.Entities) > 0 || len(plan.Environment) > 0 || len(plan.Risks) > 0 || len(plan.Terms) > 0
}

func shadowOverlapSummary(entries []MemoryEntry, hits []VectorHit, query MemoryQuery) (float64, []string, bool) {
	if len(hits) == 0 {
		return 0, nil, false
	}
	topN := len(hits)
	if len(entries) > 0 {
		topN = minInt(len(entries), len(hits))
	}
	if query.Limit > 0 {
		topN = minInt(topN, query.Limit)
	}
	if topN <= 0 {
		topN = len(hits)
	}
	liveFingerprints := liveEntryFingerprints(entries[:minInt(len(entries), topN)])
	matched := 0
	shadowOnly := make([]string, 0, topN)
	for _, hit := range hits[:minInt(len(hits), topN)] {
		if hitOverlapsLive(hit, liveFingerprints) {
			matched++
			continue
		}
		shadowOnly = append(shadowOnly, strings.TrimSpace(hit.ObjectID))
	}
	overlap := 0.0
	if topN > 0 {
		overlap = float64(matched) / float64(topN)
	}
	return overlap, shadowOnly, shadowWouldPromote(entries, hits, shadowOnly, topN)
}

func liveEntryFingerprints(entries []MemoryEntry) map[string]struct{} {
	out := make(map[string]struct{}, len(entries)*3)
	for _, entry := range entries {
		appendShadowFingerprint(out, normalizeRecallText(firstNonEmpty(entry.Summary, entry.Content)))
		appendShadowFingerprint(out, strings.TrimSpace(entry.ID))
		appendShadowFingerprint(out, metadataString(entry.Metadata, "memo_id"))
		appendShadowFingerprint(out, metadataString(entry.Metadata, "node_id"))
		for _, value := range metadataStrings(entry.Metadata, "source_ids") {
			appendShadowFingerprint(out, value)
		}
	}
	return out
}

func hitOverlapsLive(hit VectorHit, liveFingerprints map[string]struct{}) bool {
	for _, fingerprint := range vectorHitFingerprints(hit) {
		if _, ok := liveFingerprints[fingerprint]; ok {
			return true
		}
	}
	return false
}

func vectorHitFingerprints(hit VectorHit) []string {
	out := make([]string, 0, 2+len(hit.SourceRefs))
	out = append(out, normalizeRecallText(hit.Summary))
	out = append(out, strings.TrimSpace(hit.ObjectID))
	for _, ref := range hit.SourceRefs {
		appendShadowFingerprintSlice(&out, truthSourceRefFingerprint(ref))
		appendShadowFingerprintSlice(&out, strings.TrimSpace(ref.SourceID))
	}
	return uniqueStrings(out)
}

func shadowWouldPromote(entries []MemoryEntry, hits []VectorHit, shadowOnly []string, topN int) bool {
	if len(shadowOnly) == 0 {
		return false
	}
	if len(entries) < topN {
		return true
	}
	weakest := 0.0
	for index, entry := range entries[:minInt(len(entries), topN)] {
		score := maxFloat(clamp01(entry.Importance), clamp01(entry.Confidence))
		if index == 0 || score < weakest {
			weakest = score
		}
	}
	for _, hit := range hits[:minInt(len(hits), topN)] {
		if hit.Score > weakest && containsString(shadowOnly, hit.ObjectID) {
			return true
		}
	}
	return false
}

func appendShadowFingerprint(target map[string]struct{}, value string) {
	trimmed := normalizeRecallText(value)
	if trimmed == "" {
		return
	}
	target[trimmed] = struct{}{}
}

func appendShadowFingerprintSlice(target *[]string, value string) {
	trimmed := normalizeRecallText(value)
	if trimmed == "" {
		return
	}
	*target = append(*target, trimmed)
}

func metadataString(metadata map[string]any, key string) string {
	if len(metadata) == 0 {
		return ""
	}
	value, _ := metadata[key].(string)
	return strings.TrimSpace(value)
}

func metadataStrings(metadata map[string]any, key string) []string {
	if len(metadata) == 0 {
		return nil
	}
	raw, ok := metadata[key]
	if !ok {
		return nil
	}
	switch typed := raw.(type) {
	case []string:
		return uniqueStrings(typed)
	case []any:
		values := make([]string, 0, len(typed))
		for _, item := range typed {
			if value, ok := item.(string); ok {
				values = append(values, value)
			}
		}
		return uniqueStrings(values)
	default:
		return nil
	}
}

func containsString(values []string, target string) bool {
	trimmed := strings.TrimSpace(target)
	for _, value := range values {
		if strings.TrimSpace(value) == trimmed {
			return true
		}
	}
	return false
}
