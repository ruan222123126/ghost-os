package memory

import (
	"fmt"
	"log"
	"strings"
)

// VectorSidecar 负责 truth object 的索引、增量同步与 shadow 查询。
type VectorSidecar struct {
	enabled  bool
	truth    *TruthWriter
	store    *VectorStore
	metrics  *memoryCounters
	topK     int
	minScore float64
}

func NewVectorSidecar(config MemoryConfig, truth *TruthWriter, metrics *memoryCounters) *VectorSidecar {
	if !config.Vector.Enabled {
		return nil
	}
	store := NewVectorStore(config.Vector.Path, metrics)
	if store == nil || !store.Enabled() {
		return nil
	}
	if err := store.Load(); err != nil {
		log.Printf("[MEMORY] vector sidecar load failed, starting from empty index: path=%s err=%v", config.Vector.Path, err)
	}
	sidecar := &VectorSidecar{
		enabled:  true,
		truth:    truth,
		store:    store,
		metrics:  metrics,
		topK:     max(config.Vector.TopK, defaultVectorTopK),
		minScore: clamp01(maxFloat(config.Vector.MinScore, defaultVectorMinScore)),
	}
	if truth != nil && truth.Enabled() {
		if err := sidecar.RebuildFromTruthSnapshot(); err != nil {
			log.Printf("[MEMORY] vector sidecar rebuild failed: err=%v", err)
		}
	}
	return sidecar
}

func (v *VectorSidecar) Enabled() bool {
	return v != nil && v.enabled && v.store != nil && v.store.Enabled()
}

func (v *VectorSidecar) RebuildFromTruthSnapshot() error {
	if !v.Enabled() || v.truth == nil || !v.truth.Enabled() {
		return nil
	}
	objects, _ := v.truth.snapshotState()
	return v.store.Rebuild(buildVectorDocuments(objects))
}

func (v *VectorSidecar) SyncObject(object MemoryObject) error {
	if !v.Enabled() {
		return nil
	}
	normalized := normalizeMemoryObject(object)
	doc, ok := buildVectorDocument(normalized)
	if !ok {
		return v.store.Delete(normalized.ObjectID)
	}
	if err := v.store.Upsert(doc); err != nil {
		_ = v.updateTruthEmbeddingState(normalized, vectorEmbeddingStatusError, err)
		return err
	}
	return v.updateTruthEmbeddingState(normalized, vectorEmbeddingStatusIndexed, nil)
}

func (v *VectorSidecar) Query(query MemoryQuery, plan *QueryIntentPlan) ([]VectorHit, error) {
	if !v.Enabled() {
		return nil, nil
	}
	searchText := buildVectorQueryText(query, plan)
	if strings.TrimSpace(searchText) == "" {
		return nil, nil
	}
	return v.Search(searchText, VectorQueryOptions{
		TopK:               v.effectiveTopK(query),
		MinScore:           v.minScore,
		MinConfidence:      clamp01(query.MinConfidence),
		MinFreshness:       v.minFreshness(query),
		AllowedObjectTypes: vectorObjectTypesFromMetadata(query.Metadata),
		AllowedSessionIDs:  append([]string(nil), query.SessionHints...),
		AllowedMonths:      append([]string(nil), query.MonthHints...),
	}), nil
}

func (v *VectorSidecar) Search(text string, opts VectorQueryOptions) []VectorHit {
	if !v.Enabled() {
		return nil
	}
	scored := v.store.Query(text, opts)
	hits := make([]VectorHit, 0, len(scored))
	for _, item := range scored {
		hits = append(hits, VectorHit{
			ObjectID:      item.Document.ObjectID,
			ObjectType:    item.Document.ObjectType,
			Score:         clamp01(item.Score),
			Distance:      clamp01(item.Distance),
			Summary:       strings.TrimSpace(item.Document.Summary),
			SourceRefs:    append([]SourceRef(nil), item.Document.SourceRefs...),
			EvidenceCount: item.Document.EvidenceCount,
			Freshness:     clamp01(item.Freshness),
		})
	}
	return hits
}

func (v *VectorSidecar) DocumentCount() int {
	if !v.Enabled() {
		return 0
	}
	return v.store.DocumentCount()
}

func (v *VectorSidecar) updateTruthEmbeddingState(object MemoryObject, status string, err error) error {
	if v.truth == nil || !v.truth.Enabled() {
		return nil
	}
	errorText := ""
	if err != nil {
		errorText = err.Error()
	}
	return v.truth.UpdateEmbeddingRef(object.ObjectID, vectorEmbeddingRef(object, status, errorText))
}

func (v *VectorSidecar) effectiveTopK(query MemoryQuery) int {
	topK := v.topK
	if query.Limit > 0 {
		topK = minInt(topK, query.Limit)
	}
	if topK <= 0 {
		return defaultVectorTopK
	}
	return topK
}

func (v *VectorSidecar) minFreshness(query MemoryQuery) float64 {
	if query.PreferRecent {
		return defaultVectorRecentFreshness
	}
	return 0
}

func buildVectorQueryText(query MemoryQuery, plan *QueryIntentPlan) string {
	parts := make([]string, 0, 8)
	if value := strings.TrimSpace(query.SemanticQuery); value != "" {
		parts = append(parts, value)
	}
	parts = append(parts, query.Keywords...)
	if plan != nil {
		parts = append(parts, strings.TrimSpace(plan.RecallMode))
	}
	return strings.Join(uniqueStrings(parts), "\n")
}

func vectorObjectTypesFromMetadata(metadata map[string]any) []string {
	if len(metadata) == 0 {
		return nil
	}
	raw, ok := metadata["vector_object_types"]
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
	case string:
		parts := strings.Split(typed, ",")
		return uniqueStrings(parts)
	default:
		return nil
	}
}

func (v *VectorSidecar) String() string {
	if !v.Enabled() {
		return "vector:disabled"
	}
	return fmt.Sprintf("vector:path=%s top_k=%d min_score=%.2f", v.store.baseDir, v.topK, v.minScore)
}
