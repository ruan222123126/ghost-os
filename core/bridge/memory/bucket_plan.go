package memory

import (
	"math"
	"sort"
	"strings"
	"time"
)

type BucketPlanner struct {
	enabled             bool
	shadowCompare       bool
	ledger              *LedgerStore
	checkpointDir       string
	maxBuckets          int
	maxRecentBuckets    int
	maxHistoricalBuckets int
	sessionBudget       int
	debug               bool
	metrics             *memoryCounters
}

func NewBucketPlanner(config MemoryConfig, cold *ColdMemory, metrics *memoryCounters) *BucketPlanner {
	if cold == nil || cold.ledger == nil {
		return nil
	}
	return &BucketPlanner{
		enabled:              config.Recall.BucketReadEnabled,
		shadowCompare:        config.Recall.BucketShadowCompare,
		ledger:               cold.ledger,
		checkpointDir:        config.Index.CheckpointDir,
		maxBuckets:           config.Recall.MaxBuckets,
		maxRecentBuckets:     config.Recall.MaxRecentBuckets,
		maxHistoricalBuckets: config.Recall.MaxHistoricalBuckets,
		sessionBudget:        config.Recall.BucketSessionBudget,
		debug:                config.Recall.BucketDebug,
		metrics:              metrics,
	}
}

func (p *BucketPlanner) Enabled() bool {
	return p != nil && p.enabled && p.ledger != nil
}

func (p *BucketPlanner) ShadowCompareEnabled() bool {
	return p != nil && p.shadowCompare
}

func (p *BucketPlanner) Plan(query MemoryQuery, scope SessionScope, intent *QueryIntentPlan) *BucketPlan {
	if !p.Enabled() {
		return &BucketPlan{Mode: BucketModeLegacy, Reason: []string{"bucket planner disabled"}}
	}
	startedAt := time.Now().UTC()
	manifests, err := p.ledger.ListBucketManifests(p.checkpointDir)
	if err != nil || len(manifests) == 0 {
		return &BucketPlan{Mode: BucketModeLegacy, Reason: []string{"bucket manifests unavailable"}}
	}
	workspaceID := firstNonEmpty(strings.TrimSpace(query.WorkspaceID), metadataString(query.Metadata, "workspace_id"), p.ledger.WorkspaceID())
	namespace := firstNonEmpty(strings.TrimSpace(query.Namespace), metadataString(query.Metadata, "namespace"), p.ledger.Namespace())
	sessionHints := effectiveBucketSessionHints(query, scope)
	monthHints := effectiveMonthHints(query)
	terms := effectiveBucketTerms(query, intent)
	candidates := make([]BucketCandidate, 0, len(manifests))
	for _, manifest := range manifests {
		normalized := normalizeBucketManifest(manifest)
		score := 0.0
		reasons := make([]string, 0, 8)
		matchedSessions := matchedBucketSessions(normalized, sessionHints)
		matchedTerms := matchedBucketTerms(normalized, terms)
		if normalized.Key.Namespace == normalizeLedgerNamespace(namespace) {
			score += 1.2
			reasons = append(reasons, "namespace match")
		}
		if workspaceID != "" && normalized.Key.WorkspaceID == workspaceID {
			score += 2.4
			reasons = append(reasons, "workspace match")
		}
		if len(matchedSessions) > 0 {
			score += 4.0 + float64(len(matchedSessions))*0.3
			reasons = append(reasons, "session match")
		}
		if len(monthHints) > 0 && containsString(monthHints, normalized.Key.Month) {
			score += 1.6
			reasons = append(reasons, "month hint")
		}
		if query.TimeRange == nil || bucketManifestMatchesRange(normalized, query.TimeRange) {
			score += 1.4
			reasons = append(reasons, "time overlap")
		} else if len(matchedSessions) == 0 {
			continue
		}
		if len(matchedTerms) > 0 {
			score += math.Min(1.8, float64(len(matchedTerms))*0.45)
			reasons = append(reasons, "term overlap")
		}
		if recent := bucketRecencyScore(normalized, startedAt); recent > 0 {
			score += recent
			reasons = append(reasons, "recent bucket")
		}
		if layerSignal := bucketLayerSignal(normalized); layerSignal > 0 {
			score += layerSignal
			reasons = append(reasons, "layer signal")
		}
		if score <= 0 {
			continue
		}
		candidates = append(candidates, BucketCandidate{
			Bucket:          normalized,
			Score:           score,
			Reasons:         reasons,
			MatchedSessions: matchedSessions,
			MatchedTerms:    matchedTerms,
			EstimatedCost:   bucketEstimatedCost(normalized),
		})
	}
	sortBucketCandidates(candidates)
	selected, dropped, reasons := p.selectBuckets(candidates, sessionHints, workspaceID)
	plan := &BucketPlan{
		Mode:             effectiveBucketMode(query, p.shadowCompare),
		CandidateBuckets: cloneBucketCandidates(candidates),
		SelectedBuckets:  cloneBucketCandidates(selected),
		DroppedBuckets:   cloneBucketCandidates(dropped),
		Reason:           reasons,
	}
	for _, bucket := range selected {
		plan.EstimatedCost += bucket.EstimatedCost
	}
	if p.metrics != nil {
		p.metrics.bucketCandidatesTotal.Add(uint64(len(candidates)))
		p.metrics.bucketSelectedTotal.Add(uint64(len(selected)))
		p.metrics.bucketScanLatencyMs.Add(uint64(time.Since(startedAt).Milliseconds()))
		p.metrics.bucketScannedTotal.Add(uint64(len(selected)))
	}
	return plan
}

func (p *BucketPlanner) selectBuckets(candidates []BucketCandidate, sessionHints []string, workspaceID string) ([]BucketCandidate, []BucketCandidate, []string) {
	if len(candidates) == 0 {
		return nil, nil, []string{"no bucket candidates matched"}
	}
	maxBuckets := p.maxBuckets
	if maxBuckets <= 0 {
		maxBuckets = 6
	}
	sessionBudget := p.sessionBudget
	if sessionBudget <= 0 {
		sessionBudget = 1
	}
	recentBudget := p.maxRecentBuckets
	if recentBudget <= 0 {
		recentBudget = 2
	}
	historicalBudget := p.maxHistoricalBuckets
	if historicalBudget <= 0 {
		historicalBudget = 4
	}
	selected := make([]BucketCandidate, 0, minInt(len(candidates), maxBuckets))
	dropped := make([]BucketCandidate, 0, len(candidates))
	seen := make(map[string]struct{}, maxBuckets)
	reasons := make([]string, 0, 4)
	pick := func(candidate BucketCandidate, reason string) bool {
		if len(selected) >= maxBuckets {
			return false
		}
		key := candidate.Bucket.Key.String()
		if _, ok := seen[key]; ok {
			return false
		}
		candidate.Reasons = append(candidate.Reasons, reason)
		seen[key] = struct{}{}
		selected = append(selected, candidate)
		return true
	}
	for _, candidate := range candidates {
		if len(selected) >= maxBuckets || sessionBudget <= 0 {
			break
		}
		if len(candidate.MatchedSessions) == 0 {
			continue
		}
		if pick(candidate, "session budget") {
			sessionBudget--
		}
	}
	for _, candidate := range candidates {
		if len(selected) >= maxBuckets || recentBudget <= 0 {
			break
		}
		if _, ok := seen[candidate.Bucket.Key.String()]; ok {
			continue
		}
		if workspaceID != "" && candidate.Bucket.Key.WorkspaceID != workspaceID {
			continue
		}
		if pick(candidate, "recent workspace budget") {
			recentBudget--
		}
	}
	for _, candidate := range candidates {
		if len(selected) >= maxBuckets || historicalBudget <= 0 {
			break
		}
		if _, ok := seen[candidate.Bucket.Key.String()]; ok {
			continue
		}
		if workspaceID != "" && candidate.Bucket.Key.WorkspaceID != workspaceID {
			continue
		}
		if pick(candidate, "historical workspace budget") {
			historicalBudget--
		}
	}
	for _, candidate := range candidates {
		if len(selected) >= maxBuckets {
			candidate.Reasons = append(candidate.Reasons, "budget exhausted")
			dropped = append(dropped, candidate)
			continue
		}
		if pick(candidate, "remaining budget") {
			continue
		}
		candidate.Reasons = append(candidate.Reasons, "already selected")
		dropped = append(dropped, candidate)
	}
	if len(selected) == 0 && len(candidates) > 0 {
		selected = append(selected, candidates[0])
		reasons = append(reasons, "fallback to highest scored bucket")
	} else {
		reasons = append(reasons, "selected by session/workspace/time/term budget")
	}
	sortBucketCandidates(selected)
	return selected, dropped, reasons
}

func effectiveBucketMode(query MemoryQuery, shadowCompare bool) string {
	mode := strings.TrimSpace(query.BucketMode)
	switch mode {
	case BucketModeLegacy, BucketModeShadow, BucketModeBucketed:
		return mode
	}
	if shadowCompare {
		return BucketModeShadow
	}
	return BucketModeBucketed
}

func effectiveBucketSessionHints(query MemoryQuery, scope SessionScope) []string {
	values := append([]string(nil), query.SessionHints...)
	if sessionID := strings.TrimSpace(scope.SessionID); sessionID != "" {
		values = append(values, sessionID)
	}
	if sessionID := metadataString(query.Metadata, "session_id"); sessionID != "" {
		values = append(values, sessionID)
	}
	return uniqueStrings(values)
}

func effectiveMonthHints(query MemoryQuery) []string {
	values := append([]string(nil), query.MonthHints...)
	if query.TimeRange == nil {
		return uniqueStrings(values)
	}
	if month := bucketMonthFromTime(query.TimeRange.Start); month != "" {
		values = append(values, month)
	}
	if month := bucketMonthFromTime(query.TimeRange.End); month != "" {
		values = append(values, month)
	}
	return uniqueStrings(values)
}

func effectiveBucketTerms(query MemoryQuery, intent *QueryIntentPlan) []string {
	values := make([]string, 0, len(query.Keywords)+8)
	values = append(values, query.Keywords...)
	values = append(values, strings.Fields(strings.TrimSpace(query.SemanticQuery))...)
	if intent != nil {
		values = append(values, intent.Terms...)
		values = append(values, intent.Entities...)
		values = append(values, intent.Constraints...)
	}
	filtered := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.ToLower(strings.TrimSpace(value))
		if trimmed == "" || len(trimmed) < 3 {
			continue
		}
		filtered = append(filtered, trimmed)
	}
	return uniqueStrings(filtered)
}

func matchedBucketSessions(manifest BucketManifest, sessionHints []string) []string {
	if len(sessionHints) == 0 {
		return nil
	}
	matched := make([]string, 0, len(sessionHints))
	for _, sessionID := range sessionHints {
		if bucketManifestHasSession(manifest, sessionID) {
			matched = append(matched, sessionID)
		}
	}
	return uniqueStrings(matched)
}

func matchedBucketTerms(manifest BucketManifest, terms []string) []string {
	if len(terms) == 0 || len(manifest.TopTerms) == 0 {
		return nil
	}
	matched := make([]string, 0, len(terms))
	lookup := make(map[string]struct{}, len(manifest.TopTerms))
	for _, term := range manifest.TopTerms {
		lookup[strings.ToLower(strings.TrimSpace(term))] = struct{}{}
	}
	for _, term := range terms {
		if _, ok := lookup[strings.ToLower(strings.TrimSpace(term))]; ok {
			matched = append(matched, term)
		}
	}
	return uniqueStrings(matched)
}

func bucketManifestMatchesRange(manifest BucketManifest, timeRange *TimeRange) bool {
	if timeRange == nil {
		return true
	}
	if !timeRange.Start.IsZero() && !manifest.MaxOccurredAt.IsZero() && manifest.MaxOccurredAt.Before(timeRange.Start.UTC()) {
		return false
	}
	if !timeRange.End.IsZero() && !manifest.MinOccurredAt.IsZero() && manifest.MinOccurredAt.After(timeRange.End.UTC()) {
		return false
	}
	return true
}

func bucketManifestHasSession(manifest BucketManifest, sessionID string) bool {
	target := strings.TrimSpace(sessionID)
	if target == "" {
		return false
	}
	for _, current := range manifest.SessionIDs {
		if current == target {
			return true
		}
	}
	for _, session := range manifest.Sessions {
		if session.SessionID == target {
			return true
		}
	}
	return false
}

func bucketRecencyScore(manifest BucketManifest, now time.Time) float64 {
	if manifest.MaxOccurredAt.IsZero() {
		return 0
	}
	age := now.UTC().Sub(manifest.MaxOccurredAt.UTC())
	if age <= 0 {
		return 1.2
	}
	return clamp01(math.Exp2(-age.Hours()/(45*24))) * 1.2
}

func bucketLayerSignal(manifest BucketManifest) float64 {
	if len(manifest.Views) == 0 {
		return 0
	}
	signal := 0.0
	for _, view := range manifest.Views {
		if view.UpdatedAt.IsZero() {
			continue
		}
		signal += 0.15
		if view.EvidenceCount > 0 {
			signal += 0.05
		}
	}
	return signal
}

func bucketEstimatedCost(manifest BucketManifest) float64 {
	base := 1.0
	if manifest.Count > 0 {
		base += float64(minInt(manifest.Count, 1000)) / 250.0
	}
	if len(manifest.Views) > 0 {
		base += float64(len(manifest.Views)) * 0.2
	}
	return base
}

func cloneBucketCandidates(candidates []BucketCandidate) []BucketCandidate {
	if len(candidates) == 0 {
		return nil
	}
	out := make([]BucketCandidate, len(candidates))
	for i := range candidates {
		out[i] = candidates[i]
		out[i].Reasons = append([]string(nil), candidates[i].Reasons...)
		out[i].MatchedSessions = append([]string(nil), candidates[i].MatchedSessions...)
		out[i].MatchedTerms = append([]string(nil), candidates[i].MatchedTerms...)
	}
	return out
}

func bucketShadowOverlap(legacy []MemoryEntry, bucketed []MemoryEntry) (float64, []string) {
	if len(bucketed) == 0 {
		return 0, nil
	}
	topN := minInt(len(bucketed), len(legacy))
	if topN <= 0 {
		topN = len(bucketed)
	}
	live := make(map[string]struct{}, topN)
	for _, entry := range legacy[:minInt(len(legacy), topN)] {
		live[strings.TrimSpace(entry.ID)] = struct{}{}
	}
	matched := 0
	shadowOnly := make([]string, 0, topN)
	for _, entry := range bucketed[:minInt(len(bucketed), topN)] {
		id := strings.TrimSpace(entry.ID)
		if _, ok := live[id]; ok {
			matched++
			continue
		}
		shadowOnly = append(shadowOnly, id)
	}
	return float64(matched) / float64(topN), shadowOnly
}

func selectedBucketKeys(plan *BucketPlan) []string {
	if plan == nil || len(plan.SelectedBuckets) == 0 {
		return nil
	}
	keys := make([]string, 0, len(plan.SelectedBuckets))
	for _, bucket := range plan.SelectedBuckets {
		keys = append(keys, bucket.Bucket.Key.String())
	}
	sort.Strings(keys)
	return keys
}
