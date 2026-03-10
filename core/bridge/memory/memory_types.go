package memory

import (
	"reflect"
	"strings"
	"time"
)

// MemoryType 表示统一记忆条目类型。
type MemoryType string

const (
	MemoryTypeMessage   MemoryType = "message"
	MemoryTypeSummary   MemoryType = "summary"
	MemoryTypeKnowledge MemoryType = "knowledge"
)

// MemoryEntry 是三层记忆统一数据结构。
type MemoryEntry struct {
	ID             string         `json:"id"`
	Content        string         `json:"content"`
	Type           MemoryType     `json:"type"`
	Timestamp      time.Time      `json:"timestamp"`
	AccessCount    int            `json:"access_count"`
	LastAccessedAt time.Time      `json:"last_accessed_at,omitempty"`
	Metadata       map[string]any `json:"metadata,omitempty"`
	Importance     float64        `json:"importance,omitempty"`
	ExpiresAt      time.Time      `json:"expires_at,omitempty"`
	RelatedTo      []string       `json:"related_to,omitempty"`
	Source         string         `json:"source,omitempty"`
	Summary        string         `json:"summary,omitempty"`
	Anchors        []MemoryAnchor `json:"anchors,omitempty"`
	Confidence     float64        `json:"confidence,omitempty"`
	Freshness      float64        `json:"freshness,omitempty"`
	EvidenceCount  int            `json:"evidence_count,omitempty"`
	SourceRefs     []SourceRef    `json:"source_refs,omitempty"`
	TruthStatus    string         `json:"truth_status,omitempty"`
	WhyMatched     string         `json:"why_matched,omitempty"`
	Explain        map[string]any `json:"explain,omitempty"`
	RerankScore    float64        `json:"rerank_score,omitempty"`
	FreshnessBoost float64        `json:"freshness_boost,omitempty"`
	// EmbeddingID 仅做占位；查询对外输出会清空，避免误认为已接入向量召回。
	EmbeddingID string `json:"embedding_id,omitempty"`
}

// TimeRange 表示查询时间范围（UTC）。
type TimeRange struct {
	Start time.Time `json:"start,omitempty"`
	End   time.Time `json:"end,omitempty"`
}

// Contains 判断时间是否落在区间内（包含端点）。
func (r *TimeRange) Contains(ts time.Time) bool {
	if r == nil {
		return true
	}
	t := ts.UTC()
	if !r.Start.IsZero() && t.Before(r.Start.UTC()) {
		return false
	}
	if !r.End.IsZero() && t.After(r.End.UTC()) {
		return false
	}
	return true
}

// MemoryQuery 是统一检索参数。
type MemoryQuery struct {
	TimeRange  *TimeRange     `json:"time_range,omitempty"`
	Limit      int            `json:"limit,omitempty"`
	Keywords   []string       `json:"keywords,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
	Explicit   bool           `json:"explicit,omitempty"`
	Debug      bool           `json:"debug,omitempty"`
	AutoInject bool           `json:"auto_inject,omitempty"`
	// HygieneMode 保留为兼容字段；新调用方优先使用 Explicit/Debug/AutoInject。
	HygieneMode       string                  `json:"hygiene_mode,omitempty"`
	IncludeQuarantine bool                    `json:"include_quarantine,omitempty"`
	Namespace         string                  `json:"namespace,omitempty"`
	WorkspaceID       string                  `json:"workspace_id,omitempty"`
	SessionHints      []string                `json:"session_hints,omitempty"`
	MonthHints        []string                `json:"month_hints,omitempty"`
	MaxBuckets        int                     `json:"max_buckets,omitempty"`
	BucketDebug       bool                    `json:"bucket_debug,omitempty"`
	BucketMode        string                  `json:"bucket_mode,omitempty"`
	IncludeMarkdown   bool                    `json:"include_markdown,omitempty"`
	IncludeGraph      bool                    `json:"include_graph,omitempty"`
	IncludeVector     bool                    `json:"include_vector,omitempty"`
	IncludeTruth      bool                    `json:"include_truth,omitempty"`
	SemanticQuery     string                  `json:"semantic_query,omitempty"`
	AnchorTypes       []string                `json:"anchor_types,omitempty"`
	MinConfidence     float64                 `json:"min_confidence,omitempty"`
	PreferRecent      bool                    `json:"prefer_recent,omitempty"`
	IncludeAnchors    bool                    `json:"include_anchors,omitempty"`
	GraphHops         int                     `json:"graph_hops,omitempty"`
	GraphPredicates   []string                `json:"graph_predicates,omitempty"`
	GraphDebug        bool                    `json:"graph_debug,omitempty"`
	IncludeDecision   bool                    `json:"include_decision,omitempty"`
	VectorDebug       bool                    `json:"vector_debug,omitempty"`
	IntentDebug       bool                    `json:"intent_debug,omitempty"`
	TruthDebug        bool                    `json:"truth_debug,omitempty"`
	RerankDebug       bool                    `json:"rerank_debug,omitempty"`
	ShadowDebug       bool                    `json:"shadow_debug,omitempty"`
	DecisionDebug     bool                    `json:"decision_debug,omitempty"`
	DecisionReuseOnly bool                    `json:"decision_reuse_only,omitempty"`
	DecisionTypes     []string                `json:"decision_types,omitempty"`
	EnvironmentStrict bool                    `json:"environment_strict,omitempty"`
	MinReuseScore     float64                 `json:"min_reuse_score,omitempty"`
	Environment       *DecisionEnvFingerprint `json:"-"`
}

// MemoryQueryResult 仅暴露稳定查询结果；调试视图降到独立 debug 面。
type MemoryQueryResult struct {
	Entries         []MemoryEntry          `json:"entries"`
	SelectedRecipe  *DecisionRecipe        `json:"selected_recipe,omitempty"`
	RecipeAdvisory  *RecipeAdvisory        `json:"recipe_advisory,omitempty"`
	RecipeSelection *RecipeSelectionReport `json:"recipe_selection,omitempty"`

	debug *MemoryQueryDebug `json:"-"`
}

// MemoryQueryDebug 描述 recall 主链的可选调试视图。
type MemoryQueryDebug struct {
	GraphHits      []GraphHit           `json:"graph_hits,omitempty"`
	DecisionHits   []DecisionHit        `json:"decision_hits,omitempty"`
	IntentPlan     *QueryIntentPlan     `json:"intent_plan,omitempty"`
	BucketPlan     *BucketPlan          `json:"bucket_plan,omitempty"`
	LayerFreshness map[string]time.Time `json:"layer_freshness,omitempty"`
	VectorHits     []VectorHit          `json:"vector_hits,omitempty"`
	TruthHits      []TruthHit           `json:"truth_hits,omitempty"`
	RerankReport   *HybridRerankReport  `json:"rerank_report,omitempty"`
}

func (r *MemoryQueryResult) ensureDebug() *MemoryQueryDebug {
	if r == nil {
		return nil
	}
	if r.debug == nil {
		r.debug = &MemoryQueryDebug{}
	}
	return r.debug
}

func (r MemoryQueryResult) Debug() *MemoryQueryDebug {
	if r.debug == nil {
		return nil
	}
	return cloneMemoryQueryDebug(*r.debug)
}

func cloneMemoryQueryDebug(debug MemoryQueryDebug) *MemoryQueryDebug {
	out := MemoryQueryDebug{
		GraphHits:      cloneGraphHits(debug.GraphHits),
		DecisionHits:   cloneDecisionHits(debug.DecisionHits),
		LayerFreshness: cloneTimeMap(debug.LayerFreshness),
		VectorHits:     append([]VectorHit(nil), debug.VectorHits...),
		TruthHits:      append([]TruthHit(nil), debug.TruthHits...),
	}
	if debug.IntentPlan != nil {
		cloned := cloneQueryIntentPlan(*debug.IntentPlan)
		out.IntentPlan = &cloned
	}
	if debug.BucketPlan != nil {
		cloned := cloneBucketPlan(*debug.BucketPlan)
		out.BucketPlan = &cloned
	}
	if debug.RerankReport != nil {
		cloned := cloneHybridRerankReport(*debug.RerankReport)
		out.RerankReport = &cloned
	}
	if !hasMemoryQueryDebug(out) {
		return nil
	}
	return &out
}

func cloneQueryIntentPlan(plan QueryIntentPlan) QueryIntentPlan {
	out := plan
	out.Constraints = append([]string(nil), plan.Constraints...)
	out.Entities = append([]string(nil), plan.Entities...)
	out.Environment = append([]string(nil), plan.Environment...)
	out.Risks = append([]string(nil), plan.Risks...)
	out.Hydration = append([]string(nil), plan.Hydration...)
	out.Truth = normalizeTruthQueryOptions(plan.Truth)
	out.Terms = append([]string(nil), plan.Terms...)
	return out
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

func cloneGraphHits(hits []GraphHit) []GraphHit {
	if len(hits) == 0 {
		return nil
	}
	out := make([]GraphHit, len(hits))
	for i := range hits {
		out[i] = cloneGraphHit(hits[i])
	}
	return out
}

func cloneGraphHit(hit GraphHit) GraphHit {
	out := hit
	out.SourceIDs = append([]string(nil), hit.SourceIDs...)
	out.SourceClaimIDs = append([]string(nil), hit.SourceClaimIDs...)
	out.SourceEvidenceIDs = append([]string(nil), hit.SourceEvidenceIDs...)
	out.Explain = cloneMetadata(hit.Explain)
	out.Evidence = append([]GraphEvidence(nil), hit.Evidence...)
	return out
}

func hasMemoryQueryDebug(debug MemoryQueryDebug) bool {
	return len(debug.GraphHits) > 0 ||
		len(debug.DecisionHits) > 0 ||
		debug.IntentPlan != nil ||
		debug.BucketPlan != nil ||
		len(debug.LayerFreshness) > 0 ||
		len(debug.VectorHits) > 0 ||
		len(debug.TruthHits) > 0 ||
		debug.RerankReport != nil
}

func cloneTimeMap(input map[string]time.Time) map[string]time.Time {
	if len(input) == 0 {
		return nil
	}
	out := make(map[string]time.Time, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func cloneBucketPlan(plan BucketPlan) BucketPlan {
	out := plan
	out.CandidateBuckets = cloneBucketCandidates(plan.CandidateBuckets)
	out.SelectedBuckets = cloneBucketCandidates(plan.SelectedBuckets)
	out.DroppedBuckets = cloneBucketCandidates(plan.DroppedBuckets)
	out.Reason = append([]string(nil), plan.Reason...)
	return out
}

func cloneHybridRerankReport(report HybridRerankReport) HybridRerankReport {
	out := report
	out.Candidates = append([]HybridRerankItem(nil), report.Candidates...)
	return out
}

// QueryIntentPlan 保存 query planner 对任务意图的结构化切面。
type QueryIntentPlan struct {
	IntentKey   string            `json:"intent_key,omitempty"`
	Constraints []string          `json:"constraints,omitempty"`
	Entities    []string          `json:"entities,omitempty"`
	Environment []string          `json:"environment,omitempty"`
	Risks       []string          `json:"risks,omitempty"`
	RecallMode  string            `json:"recall_mode,omitempty"`
	Hydration   []string          `json:"hydration,omitempty"`
	Truth       TruthQueryOptions `json:"truth,omitempty"`
	Terms       []string          `json:"terms,omitempty"`
	Confidence  float64           `json:"confidence,omitempty"`
}

// VectorHit 描述 vector sidecar 的单条 shadow 命中。
type VectorHit struct {
	ObjectID      string      `json:"object_id,omitempty"`
	ObjectType    string      `json:"object_type,omitempty"`
	Score         float64     `json:"score,omitempty"`
	Distance      float64     `json:"distance,omitempty"`
	Summary       string      `json:"summary,omitempty"`
	SourceRefs    []SourceRef `json:"source_refs,omitempty"`
	EvidenceCount int         `json:"evidence_count,omitempty"`
	Freshness     float64     `json:"freshness,omitempty"`
}

// ShadowRecallReport 描述 shadow planner/vector 与 live recall 的对比结果。
type ShadowRecallReport struct {
	PlannerUsed          bool     `json:"planner_used,omitempty"`
	VectorHits           int      `json:"vector_hits,omitempty"`
	TopOverlap           float64  `json:"top_overlap,omitempty"`
	ShadowOnlyCandidates []string `json:"shadow_only_candidates,omitempty"`
	WouldPromote         bool     `json:"would_promote,omitempty"`
	LatencyMs            int64    `json:"latency_ms,omitempty"`
}

const (
	BucketModeLegacy   = "legacy"
	BucketModeShadow   = "shadow"
	BucketModeBucketed = "bucketed"
)

func normalizeEntry(entry MemoryEntry) MemoryEntry {
	out := entry
	out.ID = strings.TrimSpace(out.ID)
	out.Content = strings.TrimSpace(out.Content)
	if out.Type == "" {
		out.Type = MemoryTypeMessage
	}
	if out.Timestamp.IsZero() {
		out.Timestamp = time.Now().UTC()
	} else {
		out.Timestamp = out.Timestamp.UTC()
	}
	if out.Metadata == nil {
		out.Metadata = make(map[string]any, 2)
	}
	if out.Importance < 0 {
		out.Importance = 0
	}
	if out.Importance > 1 {
		out.Importance = 1
	}
	if !out.ExpiresAt.IsZero() {
		out.ExpiresAt = out.ExpiresAt.UTC()
	}
	if !out.LastAccessedAt.IsZero() {
		out.LastAccessedAt = out.LastAccessedAt.UTC()
	}
	if len(out.RelatedTo) > 0 {
		seen := make(map[string]struct{}, len(out.RelatedTo))
		cleaned := make([]string, 0, len(out.RelatedTo))
		for _, related := range out.RelatedTo {
			id := strings.TrimSpace(related)
			if id == "" {
				continue
			}
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			cleaned = append(cleaned, id)
		}
		out.RelatedTo = cleaned
	}
	out.Source = strings.TrimSpace(out.Source)
	if out.Source == "" && out.Metadata != nil {
		if source, ok := out.Metadata["source"].(string); ok {
			out.Source = strings.TrimSpace(source)
		} else if layer, ok := out.Metadata["layer"].(string); ok {
			out.Source = strings.TrimSpace(layer)
		}
	}
	out.Summary = strings.TrimSpace(out.Summary)
	out.Confidence = clamp01(out.Confidence)
	out.Freshness = clamp01(out.Freshness)
	if out.EvidenceCount < 0 {
		out.EvidenceCount = 0
	}
	out.SourceRefs = normalizeSourceRefs(out.SourceRefs)
	out.TruthStatus = strings.TrimSpace(out.TruthStatus)
	out.WhyMatched = strings.TrimSpace(out.WhyMatched)
	out.RerankScore = clamp01(out.RerankScore)
	out.FreshnessBoost = clamp01(out.FreshnessBoost)
	out.Anchors = normalizeAnchors(out.Anchors)
	out.EmbeddingID = strings.TrimSpace(out.EmbeddingID)
	return out
}

func cloneEntry(entry MemoryEntry) MemoryEntry {
	out := entry
	if entry.Metadata != nil {
		out.Metadata = make(map[string]any, len(entry.Metadata))
		for k, v := range entry.Metadata {
			out.Metadata[k] = v
		}
	}
	if len(entry.RelatedTo) > 0 {
		out.RelatedTo = append([]string(nil), entry.RelatedTo...)
	}
	if len(entry.Anchors) > 0 {
		out.Anchors = cloneAnchors(entry.Anchors)
	}
	if len(entry.SourceRefs) > 0 {
		out.SourceRefs = append([]SourceRef(nil), entry.SourceRefs...)
	}
	return out
}

func cloneEntries(entries []MemoryEntry) []MemoryEntry {
	if len(entries) == 0 {
		return nil
	}
	out := make([]MemoryEntry, len(entries))
	for i := range entries {
		out[i] = cloneEntry(entries[i])
	}
	return out
}

func stripEmbeddingIDs(entries []MemoryEntry) []MemoryEntry {
	for i := range entries {
		entries[i].EmbeddingID = ""
	}
	return entries
}

func entryMatchesQuery(entry MemoryEntry, query MemoryQuery) bool {
	if query.TimeRange != nil && !query.TimeRange.Contains(entry.Timestamp) {
		return false
	}
	if namespace := strings.TrimSpace(query.Namespace); namespace != "" {
		if metadataString(entry.Metadata, "namespace") != "" && metadataString(entry.Metadata, "namespace") != namespace {
			return false
		}
	}
	if workspaceID := strings.TrimSpace(query.WorkspaceID); workspaceID != "" {
		if current := metadataString(entry.Metadata, "workspace_id"); current != "" && current != workspaceID {
			return false
		}
	}
	if len(query.SessionHints) > 0 {
		if sessionID := metadataString(entry.Metadata, "session_id"); sessionID != "" && !containsString(query.SessionHints, sessionID) {
			return false
		}
	}
	if len(query.MonthHints) > 0 {
		month := firstNonEmpty(metadataString(entry.Metadata, "bucket_month"), bucketMonthFromTime(entry.Timestamp))
		if month != "" && !containsString(query.MonthHints, month) {
			return false
		}
	}
	if query.MinConfidence > 0 && entry.Confidence > 0 && entry.Confidence < clamp01(query.MinConfidence) {
		return false
	}
	if len(query.AnchorTypes) > 0 && !entryHasAnchorTypes(entry, query.AnchorTypes, time.Now().UTC()) {
		return false
	}
	if !entryTextMatchesQuery(entry, query) {
		return false
	}
	if len(query.Metadata) == 0 {
		return true
	}
	for key, expected := range query.Metadata {
		actual, ok := entry.Metadata[key]
		if !ok {
			return false
		}
		if !reflect.DeepEqual(actual, expected) {
			return false
		}
	}
	return true
}

func entryHasAnchorTypes(entry MemoryEntry, anchorTypes []string, now time.Time) bool {
	if len(anchorTypes) == 0 {
		return true
	}
	anchors := activeAnchors(entry.Anchors, now)
	if len(anchors) == 0 {
		return false
	}
	allowed := make(map[string]struct{}, len(anchorTypes))
	for _, anchorType := range anchorTypes {
		trimmed := strings.ToLower(strings.TrimSpace(anchorType))
		if trimmed == "" {
			continue
		}
		allowed[trimmed] = struct{}{}
	}
	if len(allowed) == 0 {
		return true
	}
	for _, anchor := range anchors {
		if _, ok := allowed[anchor.Type]; ok {
			return true
		}
	}
	return false
}
