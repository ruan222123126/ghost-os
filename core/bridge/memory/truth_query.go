package memory

import (
	"fmt"
	"log"
	"math"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type truthQueryMatch struct {
	Object     MemoryObject
	Score      float64
	MatchedBy  []string
	WhyMatched string
}

type truthPrimaryAccumulator struct {
	Object             MemoryObject
	Score              float64
	MatchedBy          []string
	Reasons            []string
	MatchedClaimIDs    []string
	MatchedEvidenceIDs []string
	ConflictedClaimIDs []string
}

// TruthReader 基于 truth snapshot 构建 live 读侧索引。
type TruthReader struct {
	enabled        bool
	baseDir        string
	topK           int
	minSupportRefs int
	writer         *TruthWriter
	metrics        *memoryCounters

	mu      sync.RWMutex
	objects map[string]MemoryObject
	claims  map[string]MemoryClaim
	index   truthIndex
}

func NewTruthReader(config MemoryConfig, writer *TruthWriter, metrics *memoryCounters) *TruthReader {
	baseDir := strings.TrimSpace(config.Truth.BaseDir)
	if writer != nil && writer.Enabled() {
		baseDir = firstNonEmpty(writer.BaseDir(), baseDir)
	}
	baseDir = resolveMemoryPath(baseDir)
	if writer == nil && baseDir == "" {
		return nil
	}
	reader := &TruthReader{
		enabled:        true,
		baseDir:        baseDir,
		topK:           max(config.Truth.TopK, defaultVectorTopK),
		minSupportRefs: max(config.Truth.MinSupportRefs, 2),
		writer:         writer,
		metrics:        metrics,
		objects:        make(map[string]MemoryObject),
		claims:         make(map[string]MemoryClaim),
	}
	if err := reader.reload(); err != nil {
		log.Printf("[MEMORY] truth reader snapshot load failed, starting from empty index: base=%s err=%v", reader.baseDir, err)
	}
	return reader
}

func (r *TruthReader) Enabled() bool {
	return r != nil && r.enabled && (r.writer == nil || r.writer.Enabled() || strings.TrimSpace(r.baseDir) != "")
}

func (r *TruthReader) reload() error {
	if !r.Enabled() {
		return nil
	}
	objects, claims, err := r.loadSnapshots()
	if err != nil {
		return err
	}
	r.mu.Lock()
	r.objects = objects
	r.claims = claims
	r.index = buildTruthIndex(r.objects, r.claims)
	r.mu.Unlock()
	return nil
}

func (r *TruthReader) loadSnapshots() (map[string]MemoryObject, map[string]MemoryClaim, error) {
	if r.writer != nil && r.writer.Enabled() {
		objects, claims := r.writer.snapshotState()
		if len(objects) > 0 || len(claims) > 0 {
			if len(claims) == 0 {
				claims = truthClaimsFromObjects(objects)
			}
			return objects, claims, nil
		}
	}
	if strings.TrimSpace(r.baseDir) == "" {
		return nil, nil, nil
	}
	objectsList, err := readTruthObjectSnapshot(filepath.Join(r.baseDir, truthObjectsSnapshotFileName))
	if err != nil {
		return nil, nil, err
	}
	claimsList, err := readTruthClaimSnapshot(filepath.Join(r.baseDir, truthClaimsSnapshotFileName))
	if err != nil {
		return nil, nil, err
	}
	objects := make(map[string]MemoryObject, len(objectsList))
	for _, object := range objectsList {
		normalized := normalizeMemoryObject(object)
		if normalized.ObjectID == "" {
			continue
		}
		objects[normalized.ObjectID] = normalized
	}
	claims := make(map[string]MemoryClaim, len(claimsList))
	for _, claim := range claimsList {
		normalized := normalizeMemoryClaim(claim, claim.ObjectID)
		if normalized.ClaimID == "" {
			continue
		}
		claims[normalized.ClaimID] = normalized
	}
	if len(claims) == 0 {
		claims = truthClaimsFromObjects(objects)
	}
	return objects, claims, nil
}

func (r *TruthReader) SyncObject(object MemoryObject) error {
	if !r.Enabled() {
		return nil
	}
	normalized := normalizeMemoryObject(object)
	if normalized.ObjectID == "" {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.objects == nil {
		r.objects = make(map[string]MemoryObject)
	}
	if r.claims == nil {
		r.claims = make(map[string]MemoryClaim)
	}
	r.objects[normalized.ObjectID] = normalized
	for claimID, claim := range r.claims {
		if strings.TrimSpace(claim.ObjectID) == normalized.ObjectID {
			delete(r.claims, claimID)
		}
	}
	for _, claim := range normalized.Claims {
		normalizedClaim := normalizeMemoryClaim(claim, normalized.ObjectID)
		if normalizedClaim.ClaimID == "" {
			continue
		}
		r.claims[normalizedClaim.ClaimID] = normalizedClaim
	}
	r.index = buildTruthIndex(r.objects, r.claims)
	return nil
}

func (r *TruthReader) LookupObject(objectID string) (MemoryObject, bool) {
	if !r.Enabled() {
		return MemoryObject{}, false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	object, ok := r.index.objectsByID[strings.TrimSpace(objectID)]
	if !ok {
		return MemoryObject{}, false
	}
	return normalizeMemoryObject(object), true
}

func (r *TruthReader) LookupClaim(claimID string) (MemoryClaim, bool) {
	if !r.Enabled() {
		return MemoryClaim{}, false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	claim, ok := r.index.claimsByID[strings.TrimSpace(claimID)]
	if !ok {
		return MemoryClaim{}, false
	}
	return normalizeMemoryClaim(claim, claim.ObjectID), true
}

func (r *TruthReader) ResolveClaims(claimIDs []string) []MemoryClaim {
	if !r.Enabled() || len(claimIDs) == 0 {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]MemoryClaim, 0, len(claimIDs))
	seen := make(map[string]struct{}, len(claimIDs))
	for _, claimID := range claimIDs {
		trimmed := strings.TrimSpace(claimID)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		claim, ok := r.index.claimsByID[trimmed]
		if !ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, normalizeMemoryClaim(claim, claim.ObjectID))
	}
	return out
}

func (r *TruthReader) LookupEvidence(evidenceID string) (MemoryEvidence, bool) {
	if !r.Enabled() {
		return MemoryEvidence{}, false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	evidence, ok := r.index.evidenceByID[strings.TrimSpace(evidenceID)]
	if !ok {
		return MemoryEvidence{}, false
	}
	return normalizeMemoryEvidence(evidence, ""), true
}

func (r *TruthReader) ResolveEvidence(evidenceIDs []string) []MemoryEvidence {
	if !r.Enabled() || len(evidenceIDs) == 0 {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]MemoryEvidence, 0, len(evidenceIDs))
	seen := make(map[string]struct{}, len(evidenceIDs))
	for _, evidenceID := range evidenceIDs {
		trimmed := strings.TrimSpace(evidenceID)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		evidence, ok := r.index.evidenceByID[trimmed]
		if !ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, normalizeMemoryEvidence(evidence, ""))
	}
	return out
}

func (r *TruthReader) ResolveObjectIDsBySourceRef(ref SourceRef) []string {
	if !r.Enabled() {
		return nil
	}
	fingerprints := []string{truthSourceRefFingerprint(ref)}
	if trimmed := strings.TrimSpace(ref.SourceID); trimmed != "" && strings.TrimSpace(ref.SourceKind) == "" {
		fingerprints = append(fingerprints,
			truthSourceRefFingerprint(SourceRef{SessionID: ref.SessionID, SourceKind: truthSourceKindArchiveMessage, SourceID: trimmed}),
			truthSourceRefFingerprint(SourceRef{SessionID: ref.SessionID, SourceKind: truthSourceKindMarkdownSource, SourceID: trimmed}),
			truthSourceRefFingerprint(SourceRef{SessionID: ref.SessionID, SourceKind: truthSourceKindMarkdownNode, SourceID: trimmed}),
			truthSourceRefFingerprint(SourceRef{SessionID: ref.SessionID, SourceKind: truthSourceKindDecisionMemo, SourceID: trimmed}),
		)
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	seen := make(map[string]struct{})
	out := make([]string, 0, 2)
	for _, fingerprint := range fingerprints {
		for _, object := range r.index.objectsBySourceRef[fingerprint] {
			if _, ok := seen[object.ObjectID]; ok {
				continue
			}
			seen[object.ObjectID] = struct{}{}
			out = append(out, object.ObjectID)
		}
	}
	sort.Strings(out)
	return out
}

func (r *TruthReader) ResolvePrimaryObjectIDBySourceRef(ref SourceRef) string {
	ids := r.ResolveObjectIDsBySourceRef(ref)
	if len(ids) == 0 {
		return ""
	}
	return ids[0]
}

func (r *TruthReader) resolvePrimaryObjectIDByClaimID(claimID string) string {
	if !r.Enabled() {
		return ""
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	object, ok := r.index.objectsByClaimID[strings.TrimSpace(claimID)]
	if !ok {
		return ""
	}
	return strings.TrimSpace(object.ObjectID)
}

func (r *TruthReader) resolvePrimaryObjectIDByEvidenceID(evidenceID string) string {
	if !r.Enabled() {
		return ""
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	object, ok := r.index.objectsByEvidenceID[strings.TrimSpace(evidenceID)]
	if !ok {
		return ""
	}
	return strings.TrimSpace(object.ObjectID)
}

func (r *TruthReader) ConflictCount(objectID string) int {
	object, ok := r.LookupObject(objectID)
	if !ok {
		return 0
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return truthConflictCount(object, r.claims)
}

func (r *TruthReader) QueryPrimary(options TruthQueryOptions) []PrimaryCandidate {
	if !r.Enabled() {
		return nil
	}
	options = normalizeTruthQueryOptions(options)
	if options.Subject == "" && options.Predicate == "" && options.Object == "" && options.Value == "" {
		return nil
	}
	if r.metrics != nil {
		r.metrics.truthQueries.Add(1)
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	if len(r.index.objectsByID) == 0 || len(r.index.claimsByID) == 0 {
		return nil
	}

	seedClaims := r.primarySeedClaimsLocked(options)
	if len(seedClaims) == 0 {
		return nil
	}

	matched := make(map[string]*truthPrimaryAccumulator)
	for _, claim := range seedClaims {
		normalized := normalizeMemoryClaim(claim, claim.ObjectID)
		if !truthPrimaryClaimStatusAllowed(normalized, options) || !truthPrimaryClaimMatches(normalized, options) {
			continue
		}
		object, ok := r.index.objectsByClaimID[normalized.ClaimID]
		if !ok {
			object, ok = r.index.objectsByID[normalized.ObjectID]
		}
		if !ok || object.ObjectID == "" {
			continue
		}
		bucket, ok := matched[object.ObjectID]
		if !ok {
			bucket = &truthPrimaryAccumulator{Object: normalizeMemoryObject(object)}
			matched[object.ObjectID] = bucket
		}
		truthPrimaryAccumulateClaim(bucket, normalized, options, r.index.objectsByEvidenceID)
	}

	results := make([]PrimaryCandidate, 0, len(matched))
	for _, bucket := range matched {
		candidate := normalizePrimaryCandidate(PrimaryCandidate{
			ObjectID:           bucket.Object.ObjectID,
			MatchedClaimIDs:    bucket.MatchedClaimIDs,
			MatchedEvidenceIDs: bucket.MatchedEvidenceIDs,
			ConflictedClaimIDs: bucket.ConflictedClaimIDs,
			Score:              clamp01(bucket.Score),
			WhyMatched:         strings.Join(uniqueStrings(bucket.Reasons), "; "),
			Explain: TruthQueryExplain{
				MatchedClaimIDs:    bucket.MatchedClaimIDs,
				MatchedEvidenceIDs: bucket.MatchedEvidenceIDs,
				ConflictedClaimIDs: bucket.ConflictedClaimIDs,
				WhyMatched:         strings.Join(uniqueStrings(bucket.Reasons), "; "),
			},
		})
		results = append(results, candidate)
	}
	truthSortPrimaryCandidates(results, r.index.objectsByID)
	if r.topK > 0 && len(results) > r.topK {
		results = results[:r.topK]
	}
	if r.metrics != nil {
		r.metrics.truthHits.Add(uint64(len(results)))
		for _, item := range results {
			object, ok := r.index.objectsByID[item.ObjectID]
			if !ok {
				continue
			}
			status := truthObjectStatus(object, r.minSupportRefs, truthConflictCount(object, r.claims))
			switch status {
			case truthStatusVerified:
				r.metrics.truthVerifiedHits.Add(1)
			case truthStatusConflicted:
				r.metrics.truthConflictedHits.Add(1)
			}
		}
	}
	return results
}

func (r *TruthReader) Query(query MemoryQuery, plan *QueryIntentPlan) []truthQueryMatch {
	if !r.Enabled() {
		return nil
	}
	if r.metrics != nil {
		r.metrics.truthQueries.Add(1)
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	if len(r.index.objectsByID) == 0 {
		return nil
	}
	type accumulator struct {
		Object    MemoryObject
		Score     float64
		MatchedBy []string
		Reasons   []string
	}
	matched := make(map[string]*accumulator)
	add := func(object MemoryObject, score float64, matchedBy string, reason string) {
		if object.ObjectID == "" || score <= 0 {
			return
		}
		bucket, ok := matched[object.ObjectID]
		if !ok {
			bucket = &accumulator{Object: object}
			matched[object.ObjectID] = bucket
		}
		bucket.Score += score
		if marker := strings.TrimSpace(matchedBy); marker != "" {
			bucket.MatchedBy = append(bucket.MatchedBy, marker)
		}
		if text := strings.TrimSpace(reason); text != "" {
			bucket.Reasons = append(bucket.Reasons, text)
		}
	}

	for _, intentKey := range truthIntentKeys(query, plan) {
		for _, claim := range r.index.claimsByIntentKey[truthIndexKey(intentKey)] {
			object, ok := r.index.objectsByID[claim.ObjectID]
			if ok {
				add(object, 0.34, "intent_key", fmt.Sprintf("intent=%s", strings.TrimSpace(intentKey)))
			}
		}
	}
	for _, entityID := range truthEntityIDs(query, plan) {
		for _, claim := range r.index.claimsByEntityID[truthIndexKey(entityID)] {
			object, ok := r.index.objectsByID[claim.ObjectID]
			if ok {
				add(object, 0.2, "entity_id", fmt.Sprintf("entity=%s", strings.TrimSpace(entityID)))
			}
		}
	}
	for _, anchorKey := range truthAnchorKeys(query) {
		for _, claim := range r.index.claimsByAnchorKey[truthIndexKey(anchorKey)] {
			object, ok := r.index.objectsByID[claim.ObjectID]
			if ok {
				add(object, 0.14, "anchor_key", fmt.Sprintf("anchor=%s", strings.TrimSpace(anchorKey)))
			}
		}
	}
	for _, constraintType := range truthConstraintTypes(query, plan) {
		for _, claim := range r.index.claimsByConstraintType[truthIndexKey(constraintType)] {
			object, ok := r.index.objectsByID[claim.ObjectID]
			if ok {
				add(object, 0.18, "constraint_type", fmt.Sprintf("constraint=%s", strings.TrimSpace(constraintType)))
			}
		}
	}
	for _, riskType := range truthRiskTypes(query, plan) {
		for _, claim := range r.index.claimsByRiskType[truthIndexKey(riskType)] {
			object, ok := r.index.objectsByID[claim.ObjectID]
			if ok {
				add(object, 0.18, "risk_type", fmt.Sprintf("risk=%s", strings.TrimSpace(riskType)))
			}
		}
	}

	terms := queryTerms(query)
	terms = uniqueStrings(terms)
	for _, object := range r.index.objectsByID {
		if !truthObjectAllowed(object, query.Metadata) {
			continue
		}
		textScore := truthObjectTextScore(object, query.SemanticQuery, terms)
		if textScore > 0 {
			add(object, 0.24*textScore, "query_text", summarizeLine(truthObjectSummary(object), 120))
		}
	}

	results := make([]truthQueryMatch, 0, len(matched))
	for _, bucket := range matched {
		bucket.MatchedBy = uniqueStrings(bucket.MatchedBy)
		results = append(results, truthQueryMatch{
			Object:     normalizeMemoryObject(bucket.Object),
			Score:      clamp01(bucket.Score),
			MatchedBy:  bucket.MatchedBy,
			WhyMatched: strings.Join(uniqueStrings(bucket.Reasons), "; "),
		})
	}
	sort.SliceStable(results, func(i, j int) bool {
		if results[i].Score != results[j].Score {
			return results[i].Score > results[j].Score
		}
		if results[i].Object.Confidence != results[j].Object.Confidence {
			return results[i].Object.Confidence > results[j].Object.Confidence
		}
		tsi := effectiveDecisionTimestamp(results[i].Object.UpdatedAt, results[i].Object.CreatedAt)
		tsj := effectiveDecisionTimestamp(results[j].Object.UpdatedAt, results[j].Object.CreatedAt)
		if !tsi.Equal(tsj) {
			return tsi.After(tsj)
		}
		return results[i].Object.ObjectID < results[j].Object.ObjectID
	})
	limit := r.topK
	if query.Limit > 0 {
		limit = minInt(limit, query.Limit)
	}
	if limit > 0 && len(results) > limit {
		results = results[:limit]
	}
	if r.metrics != nil {
		r.metrics.truthHits.Add(uint64(len(results)))
		for _, item := range results {
			status := truthObjectStatus(item.Object, r.minSupportRefs, truthConflictCount(item.Object, r.claims))
			switch status {
			case truthStatusVerified:
				r.metrics.truthVerifiedHits.Add(1)
			case truthStatusConflicted:
				r.metrics.truthConflictedHits.Add(1)
			}
		}
	}
	return results
}

func (r *TruthReader) DebugHits(matches []truthQueryMatch) []TruthHit {
	if len(matches) == 0 {
		return nil
	}
	r.mu.RLock()
	claims := cloneTruthClaims(r.claims)
	r.mu.RUnlock()
	now := time.Now().UTC()
	hits := make([]TruthHit, 0, len(matches))
	for _, match := range matches {
		status := truthObjectStatus(match.Object, r.minSupportRefs, truthConflictCount(match.Object, claims))
		hits = append(hits, TruthHit{
			ObjectID:      match.Object.ObjectID,
			ObjectType:    match.Object.ObjectType,
			Score:         clamp01(match.Score),
			Confidence:    truthObjectConfidence(match.Object, status, r.minSupportRefs, claims),
			Freshness:     truthObjectFreshness(match.Object, now),
			EvidenceCount: len(match.Object.RawEvidence),
			SourceRefs:    truthObjectSourceRefs(match.Object),
			MatchedBy:     append([]string(nil), match.MatchedBy...),
			Status:        status,
		})
	}
	return hits
}

func (r *TruthReader) DebugHitsFromPrimaryCandidates(candidates []PrimaryCandidate) []TruthHit {
	if len(candidates) == 0 || !r.Enabled() {
		return nil
	}
	r.mu.RLock()
	claims := cloneTruthClaims(r.claims)
	r.mu.RUnlock()
	now := time.Now().UTC()
	hits := make([]TruthHit, 0, len(candidates))
	for _, candidate := range candidates {
		object, ok := r.LookupObject(candidate.ObjectID)
		if !ok {
			continue
		}
		status := truthObjectStatus(object, r.minSupportRefs, truthConflictCount(object, claims))
		hits = append(hits, TruthHit{
			ObjectID:      object.ObjectID,
			ClaimIDs:      append([]string(nil), candidate.MatchedClaimIDs...),
			ObjectType:    object.ObjectType,
			Score:         clamp01(maxFloat(candidate.Score, object.Confidence)),
			Confidence:    truthObjectConfidence(object, status, r.minSupportRefs, claims),
			Freshness:     truthObjectFreshness(object, now),
			EvidenceCount: max(len(candidate.MatchedEvidenceIDs), len(object.RawEvidence)),
			SourceRefs:    truthObjectSourceRefs(object),
			MatchedBy:     []string{"truth", "truth.primary"},
			Status:        status,
		})
	}
	return hits
}

func truthIntentKeys(query MemoryQuery, plan *QueryIntentPlan) []string {
	values := queryMetadataStrings(query.Metadata, "intent_key")
	if plan != nil && strings.TrimSpace(plan.IntentKey) != "" {
		values = append(values, plan.IntentKey)
	}
	return uniqueStrings(values)
}

func truthEntityIDs(query MemoryQuery, plan *QueryIntentPlan) []string {
	values := queryMetadataStrings(query.Metadata, "entity_id")
	if plan != nil {
		values = append(values, plan.Entities...)
	}
	return uniqueStrings(values)
}

func truthAnchorKeys(query MemoryQuery) []string {
	values := queryMetadataStrings(query.Metadata, "anchor_key")
	values = append(values, query.AnchorTypes...)
	return uniqueStrings(values)
}

func truthConstraintTypes(query MemoryQuery, plan *QueryIntentPlan) []string {
	values := queryMetadataStrings(query.Metadata, "constraint_type")
	if plan != nil {
		values = append(values, plan.Constraints...)
	}
	return uniqueStrings(values)
}

func truthRiskTypes(query MemoryQuery, plan *QueryIntentPlan) []string {
	values := queryMetadataStrings(query.Metadata, "risk_type")
	if plan != nil {
		values = append(values, plan.Risks...)
	}
	return uniqueStrings(values)
}

func truthObjectAllowed(object MemoryObject, metadata map[string]any) bool {
	allowed := queryMetadataStrings(metadata, "truth_object_types")
	if len(allowed) == 0 {
		return true
	}
	kind := truthIndexKey(object.ObjectType)
	for _, item := range allowed {
		if truthIndexKey(item) == kind {
			return true
		}
	}
	return false
}

func truthObjectSummary(object MemoryObject) string {
	return firstNonEmpty(strings.TrimSpace(object.Summary), summarizeLine(truthObjectText(object), 220))
}

func (r *TruthReader) primarySeedClaimsLocked(options TruthQueryOptions) []MemoryClaim {
	buckets := make([][]MemoryClaim, 0, 4)
	appendBucket := func(claims []MemoryClaim) {
		if len(claims) == 0 {
			return
		}
		buckets = append(buckets, claims)
	}
	if options.Subject != "" && options.Predicate != "" {
		appendBucket(truthIndexClaimsByKeys(r.index.claimsBySubjectPredicate, truthSubjectPredicateLookupKeys(options.Subject, options.Predicate)))
	}
	if options.Subject != "" {
		appendBucket(truthIndexClaimsByKeys(r.index.claimsBySubject, truthTermLookupKeys(options.Subject)))
	}
	if options.Predicate != "" {
		appendBucket(truthIndexClaimsByKeys(r.index.claimsByPredicate, []string{options.Predicate}))
	}
	if options.Object != "" {
		appendBucket(truthIndexClaimsByKeys(r.index.claimsByObject, truthTermLookupKeys(options.Object)))
		appendBucket(truthIndexClaimsByKeys(r.index.claimsByValueLookup, []string{options.Object}))
	}
	if options.Value != "" {
		appendBucket(truthIndexClaimsByKeys(r.index.claimsByValueLookup, []string{options.Value}))
	}
	if len(buckets) == 0 {
		return truthIndexAllClaims(r.index.claimsByID)
	}
	best := buckets[0]
	for _, bucket := range buckets[1:] {
		if len(bucket) == 0 {
			continue
		}
		if len(best) == 0 || len(bucket) < len(best) {
			best = bucket
		}
	}
	return best
}

func truthPrimaryAccumulateClaim(bucket *truthPrimaryAccumulator, claim MemoryClaim, options TruthQueryOptions, objectsByEvidenceID map[string]MemoryObject) {
	if bucket == nil || claim.ClaimID == "" {
		return
	}
	markers := truthPrimaryMatchedBy(claim, options)
	if len(markers) == 0 {
		return
	}
	bucket.MatchedBy = append(bucket.MatchedBy, markers...)
	bucket.MatchedClaimIDs = append(bucket.MatchedClaimIDs, claim.ClaimID)
	if normalizeTruthClaimStatus(claim.Status) == truthClaimStatusConflicted {
		bucket.ConflictedClaimIDs = append(bucket.ConflictedClaimIDs, claim.ClaimID)
	}
	for _, evidenceRef := range claim.EvidenceRefs {
		evidenceID := strings.TrimSpace(evidenceRef.EvidenceID)
		if evidenceID == "" {
			continue
		}
		if object, ok := objectsByEvidenceID[evidenceID]; ok && object.ObjectID != bucket.Object.ObjectID {
			continue
		}
		bucket.MatchedEvidenceIDs = append(bucket.MatchedEvidenceIDs, evidenceID)
	}
	bucket.Reasons = append(bucket.Reasons, truthPrimaryClaimReason(claim, markers))
	bucket.Score += truthPrimaryClaimScore(claim, options, markers)
	bucket.MatchedBy = uniqueStrings(bucket.MatchedBy)
	bucket.MatchedClaimIDs = uniqueStrings(bucket.MatchedClaimIDs)
	bucket.MatchedEvidenceIDs = uniqueStrings(bucket.MatchedEvidenceIDs)
	bucket.ConflictedClaimIDs = uniqueStrings(bucket.ConflictedClaimIDs)
	bucket.Reasons = uniqueStrings(bucket.Reasons)
	bucket.Score = clamp01(bucket.Score)
}

func truthIndexClaimsByKeys(index map[string][]MemoryClaim, keys []string) []MemoryClaim {
	if len(index) == 0 || len(keys) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(keys))
	out := make([]MemoryClaim, 0, len(keys))
	for _, key := range uniqueStrings(keys) {
		normalizedKey := truthIndexKey(key)
		if normalizedKey == "" {
			continue
		}
		for _, claim := range index[normalizedKey] {
			if claim.ClaimID == "" {
				continue
			}
			if _, ok := seen[claim.ClaimID]; ok {
				continue
			}
			seen[claim.ClaimID] = struct{}{}
			out = append(out, claim)
		}
	}
	return out
}

func truthIndexAllClaims(claimsByID map[string]MemoryClaim) []MemoryClaim {
	if len(claimsByID) == 0 {
		return nil
	}
	ids := make([]string, 0, len(claimsByID))
	for claimID := range claimsByID {
		ids = append(ids, claimID)
	}
	sort.Strings(ids)
	out := make([]MemoryClaim, 0, len(ids))
	for _, claimID := range ids {
		out = append(out, claimsByID[claimID])
	}
	return out
}

func truthTermLookupKeys(value string) []string {
	normalized := truthIndexKey(value)
	if normalized == "" {
		return nil
	}
	return uniqueStrings([]string{normalized, "entity:" + normalized, "subject:" + normalized, "object:" + normalized, "language:" + normalized})
}

func truthSubjectPredicateLookupKeys(subject string, predicate string) []string {
	predicateKey := truthIndexKey(predicate)
	if predicateKey == "" {
		return nil
	}
	keys := make([]string, 0, 4)
	for _, subjectKey := range truthTermLookupKeys(subject) {
		key := truthIndexKey(subjectKey)
		if key == "" {
			continue
		}
		keys = append(keys, key+"|"+predicateKey)
	}
	return uniqueStrings(keys)
}

func truthPrimaryClaimStatusAllowed(claim MemoryClaim, options TruthQueryOptions) bool {
	switch normalizeTruthClaimStatus(claim.Status) {
	case truthClaimStatusActive:
		return true
	case truthClaimStatusConflicted:
		return options.ExposeConflicts
	case truthClaimStatusSuperseded:
		return options.IncludeHistorical
	case truthClaimStatusUnverified:
		return !options.ActiveOnly || options.IncludeHistorical
	default:
		return !options.ActiveOnly
	}
}

func truthPrimaryClaimMatches(claim MemoryClaim, options TruthQueryOptions) bool {
	if options.Subject != "" && !truthClaimTermMatches(claim.Subject, options.Subject) {
		return false
	}
	if options.Predicate != "" && truthIndexKey(firstNonEmpty(claim.Predicate, legacyTruthPredicate(claim))) != truthIndexKey(options.Predicate) {
		return false
	}
	if options.Object != "" && !truthClaimTermMatches(claim.Object, options.Object) {
		return false
	}
	if options.Value != "" && !truthClaimValueMatches(claim, options.Value) {
		return false
	}
	return true
}

func truthClaimTermMatches(term ClaimTerm, query string) bool {
	term = normalizeClaimTerm(term)
	needle := truthIndexKey(query)
	if needle == "" {
		return false
	}
	for _, candidate := range []string{term.ID, term.Label, term.Kind + ":" + term.ID, term.Kind + ":" + term.Label} {
		if truthIndexKey(candidate) == needle {
			return true
		}
	}
	return false
}

func truthClaimValueMatches(claim MemoryClaim, value string) bool {
	needle := truthIndexKey(value)
	if needle == "" {
		return false
	}
	for _, candidate := range truthClaimValueLookupKeys(claim) {
		if truthIndexKey(candidate) == needle {
			return true
		}
	}
	return false
}

func truthPrimaryMatchedBy(claim MemoryClaim, options TruthQueryOptions) []string {
	matchedBy := make([]string, 0, 4)
	if options.Subject != "" && truthClaimTermMatches(claim.Subject, options.Subject) {
		if options.Predicate != "" && truthIndexKey(firstNonEmpty(claim.Predicate, legacyTruthPredicate(claim))) == truthIndexKey(options.Predicate) {
			matchedBy = append(matchedBy, "subject_predicate")
		} else {
			matchedBy = append(matchedBy, "subject")
		}
	}
	if options.Predicate != "" && truthIndexKey(firstNonEmpty(claim.Predicate, legacyTruthPredicate(claim))) == truthIndexKey(options.Predicate) {
		matchedBy = append(matchedBy, "predicate")
	}
	if options.Object != "" && truthClaimTermMatches(claim.Object, options.Object) {
		matchedBy = append(matchedBy, "object")
	}
	if options.Value != "" && truthClaimValueMatches(claim, options.Value) {
		matchedBy = append(matchedBy, "value")
	}
	return uniqueStrings(matchedBy)
}

func truthPrimaryClaimReason(claim MemoryClaim, matchedBy []string) string {
	parts := make([]string, 0, 5)
	if label := firstNonEmpty(claim.Subject.Label, claim.Subject.ID); label != "" {
		parts = append(parts, "subject="+label)
	}
	if predicate := firstNonEmpty(claim.Predicate, legacyTruthPredicate(claim)); predicate != "" {
		parts = append(parts, "predicate="+predicate)
	}
	if label := firstNonEmpty(claim.Object.Label, claim.Object.ID); label != "" {
		parts = append(parts, "object="+label)
	}
	if value := firstNonEmpty(claim.Value, claim.IntentKey, claim.AnchorKey, claim.EntityID, claim.ConstraintType, claim.RiskType); value != "" {
		parts = append(parts, "value="+value)
	}
	reason := fmt.Sprintf("claim=%s", claim.ClaimID)
	if len(parts) > 0 {
		reason += " (" + strings.Join(parts, ", ") + ")"
	}
	if len(matchedBy) > 0 {
		reason += " via " + strings.Join(uniqueStrings(matchedBy), "+")
	}
	return reason
}

func truthPrimaryClaimScore(claim MemoryClaim, options TruthQueryOptions, matchedBy []string) float64 {
	filters := 0
	for _, value := range []string{options.Subject, options.Predicate, options.Object, options.Value} {
		if strings.TrimSpace(value) != "" {
			filters++
		}
	}
	if filters == 0 {
		return 0
	}
	coverage := float64(len(uniqueStrings(matchedBy))) / float64(filters)
	strength := clamp01(truthClaimStrength(claim))
	bonus := 0.0
	if len(matchedBy) >= 2 {
		bonus += 0.08
	}
	if normalizeTruthClaimStatus(claim.Status) == truthClaimStatusActive {
		bonus += 0.04
	}
	return clamp01(0.55*coverage + 0.35*strength + bonus)
}

func truthSortPrimaryCandidates(candidates []PrimaryCandidate, objectsByID map[string]MemoryObject) {
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].Score != candidates[j].Score {
			return candidates[i].Score > candidates[j].Score
		}
		objectI := objectsByID[candidates[i].ObjectID]
		objectJ := objectsByID[candidates[j].ObjectID]
		if objectI.Confidence != objectJ.Confidence {
			return objectI.Confidence > objectJ.Confidence
		}
		tsi := effectiveDecisionTimestamp(objectI.UpdatedAt, objectI.CreatedAt)
		tsj := effectiveDecisionTimestamp(objectJ.UpdatedAt, objectJ.CreatedAt)
		if !tsi.Equal(tsj) {
			return tsi.After(tsj)
		}
		return candidates[i].ObjectID < candidates[j].ObjectID
	})
}

func truthObjectText(object MemoryObject) string {
	parts := make([]string, 0, 1+len(object.Claims)+len(object.RawEvidence))
	if summary := strings.TrimSpace(object.Summary); summary != "" {
		parts = append(parts, summary)
	}
	for _, claim := range object.Claims {
		parts = append(parts,
			strings.TrimSpace(claim.Value),
			strings.TrimSpace(claim.IntentKey),
			strings.TrimSpace(claim.AnchorKey),
			strings.TrimSpace(claim.EntityID),
			strings.TrimSpace(claim.ConstraintType),
			strings.TrimSpace(claim.RiskType),
		)
	}
	for _, evidence := range object.RawEvidence {
		parts = append(parts, strings.TrimSpace(firstNonEmpty(evidence.Summary, evidence.Text)))
	}
	return strings.TrimSpace(strings.Join(uniqueStrings(parts), " "))
}

func truthObjectTextScore(object MemoryObject, phrase string, terms []string) float64 {
	text := strings.ToLower(strings.TrimSpace(truthObjectText(object)))
	if text == "" {
		return 0
	}
	phrase = strings.ToLower(strings.TrimSpace(phrase))
	phraseScore := 0.0
	if phrase != "" && strings.Contains(text, phrase) {
		phraseScore = 1
	}
	matched := 0
	for _, term := range terms {
		trimmed := strings.ToLower(strings.TrimSpace(term))
		if trimmed == "" {
			continue
		}
		if strings.Contains(text, trimmed) {
			matched++
		}
	}
	termScore := 0.0
	if len(terms) > 0 {
		termScore = float64(matched) / float64(len(terms))
	}
	if phraseScore == 0 && termScore == 0 {
		return 0
	}
	return clamp01(termScore*0.7 + phraseScore*0.3)
}

func truthObjectFreshness(object MemoryObject, now time.Time) float64 {
	ts := effectiveDecisionTimestamp(object.UpdatedAt, object.CreatedAt)
	for _, evidence := range object.RawEvidence {
		ts = effectiveDecisionTimestamp(evidence.Timestamp, ts)
	}
	if ts.IsZero() {
		return 0
	}
	age := now.UTC().Sub(ts.UTC())
	if age < 0 {
		age = 0
	}
	return clamp01(math.Exp2(-age.Hours() / (30 * 24)))
}

func truthObjectSupportRefs(object MemoryObject) int {
	return len(truthObjectSourceRefs(object))
}

func truthObjectSupportCount(object MemoryObject) int {
	refs := truthObjectSupportRefs(object)
	evidence := len(object.RawEvidence)
	claims := len(object.Claims)
	if refs == 0 {
		return evidence + claims
	}
	return refs + minInt(claims, 2)
}

func truthObjectStatus(object MemoryObject, minSupportRefs int, conflictCount int) string {
	if conflictCount > 0 {
		return truthStatusConflicted
	}
	refs := truthObjectSupportRefs(object)
	supportCount := truthObjectSupportCount(object)
	if refs >= max(minSupportRefs, 2) || supportCount >= max(minSupportRefs+1, 3) {
		return truthStatusVerified
	}
	if object.ObjectID != "" {
		return truthStatusSupported
	}
	return truthStatusCandidate
}

func truthObjectConfidence(object MemoryObject, status string, minSupportRefs int, claims map[string]MemoryClaim) float64 {
	base := clamp01(object.Confidence)
	freshness := truthObjectFreshness(object, time.Now().UTC())
	supportRefs := truthObjectSupportRefs(object)
	supportCount := truthObjectSupportCount(object)
	conflictCount := truthConflictCount(object, claims)
	switch normalizeTruthStatus(status) {
	case truthStatusVerified:
		return clamp01(maxFloat(0.75, 0.68+float64(minInt(supportRefs, minSupportRefs+1))*0.05+freshness*0.06+base*0.08))
	case truthStatusSupported:
		return clamp01(min(0.8, maxFloat(0.55, 0.5+float64(minInt(supportCount, 3))*0.06+base*0.12+freshness*0.04)))
	case truthStatusConflicted:
		penalty := float64(max(conflictCount, 1)) * 0.08
		return clamp01(min(0.35, maxFloat(0.12, 0.34+base*0.06+freshness*0.04-penalty)))
	default:
		return clamp01(min(0.55, maxFloat(0.2, 0.22+base*0.18+freshness*0.05)))
	}
}

func truthConflictCount(object MemoryObject, claims map[string]MemoryClaim) int {
	if len(object.Claims) == 0 || len(claims) == 0 {
		return 0
	}
	seen := make(map[string]struct{})
	count := 0
	for _, own := range object.Claims {
		if !truthClaimStatusIsLive(own.Status) {
			continue
		}
		for _, other := range claims {
			if other.ObjectID == object.ObjectID || !truthClaimStatusIsLive(other.Status) {
				continue
			}
			if !truthClaimsConflict(own, other) {
				continue
			}
			key := other.ObjectID + ":" + other.ClaimID
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			count++
		}
	}
	return count
}

func truthClaimsConflict(left MemoryClaim, right MemoryClaim) bool {
	leftKey := truthConflictDimension(left)
	rightKey := truthConflictDimension(right)
	if leftKey == "" || leftKey != rightKey {
		return false
	}
	if normalizeTruthPredicateCardinality(truthPredicatePolicyFor(left.Predicate).Cardinality) != truthPredicateCardinalitySingleValue {
		return false
	}
	leftValue := truthClaimValueKey(left)
	rightValue := truthClaimValueKey(right)
	if leftValue == "" || rightValue == "" {
		return false
	}
	return leftValue != rightValue
}

func truthConflictDimension(claim MemoryClaim) string {
	if key := truthClaimDomainKey(claim); key != "" {
		return key
	}
	switch {
	case strings.TrimSpace(claim.EntityID) != "":
		return "entity:" + truthIndexKey(claim.EntityID) + ":" + truthIndexKey(firstNonEmpty(claim.Type, claim.ConstraintType, claim.RiskType))
	case strings.TrimSpace(claim.ConstraintType) != "":
		return "constraint:" + truthIndexKey(claim.ConstraintType)
	case strings.TrimSpace(claim.RiskType) != "":
		return "risk:" + truthIndexKey(claim.RiskType)
	case strings.TrimSpace(claim.IntentKey) != "":
		return "intent:" + truthIndexKey(claim.IntentKey) + ":" + truthIndexKey(claim.Type)
	default:
		return ""
	}
}

func queryMetadataStrings(metadata map[string]any, key string) []string {
	if len(metadata) == 0 {
		return nil
	}
	raw, ok := metadata[key]
	if !ok {
		return nil
	}
	switch typed := raw.(type) {
	case string:
		parts := strings.Split(typed, ",")
		return uniqueStrings(parts)
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
