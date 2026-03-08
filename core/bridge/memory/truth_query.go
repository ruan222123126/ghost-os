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

func (r *TruthReader) ConflictCount(objectID string) int {
	object, ok := r.LookupObject(objectID)
	if !ok {
		return 0
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return truthConflictCount(object, r.claims)
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
	if plan != nil {
		terms = append(terms, plan.Terms...)
	}
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
		for _, other := range claims {
			if other.ObjectID == object.ObjectID {
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
	leftValue := truthIndexKey(firstNonEmpty(left.Value, left.IntentKey, left.EntityID, left.ConstraintType, left.RiskType))
	rightValue := truthIndexKey(firstNonEmpty(right.Value, right.IntentKey, right.EntityID, right.ConstraintType, right.RiskType))
	if leftValue == "" || rightValue == "" {
		return false
	}
	return leftValue != rightValue
}

func truthConflictDimension(claim MemoryClaim) string {
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
