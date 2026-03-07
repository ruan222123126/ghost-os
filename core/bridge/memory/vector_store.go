package memory

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"
)

const (
	vectorIndexSchemaVersion       = 1
	vectorIndexFileName            = "index.snapshot.json"
	defaultVectorTopK              = 6
	defaultVectorMinScore          = 0.34
	defaultVectorFreshnessHalfLife = 21 * 24 * time.Hour
	defaultVectorRecentFreshness   = 0.08
	vectorEvidenceMinConfidence    = 0.78
	vectorEmbeddingSource          = "vector_sidecar"
	vectorEmbeddingStatusPending   = "pending"
	vectorEmbeddingStatusIndexed   = "indexed"
	vectorEmbeddingStatusError     = "error"
)

type VectorDocument struct {
	ObjectID      string             `json:"object_id"`
	ObjectType    string             `json:"object_type,omitempty"`
	Summary       string             `json:"summary,omitempty"`
	Text          string             `json:"text,omitempty"`
	Terms         []string           `json:"terms,omitempty"`
	Weights       map[string]float64 `json:"weights,omitempty"`
	Norm          float64            `json:"norm,omitempty"`
	SourceRefs    []SourceRef        `json:"source_refs,omitempty"`
	EvidenceCount int                `json:"evidence_count,omitempty"`
	Confidence    float64            `json:"confidence,omitempty"`
	CreatedAt     time.Time          `json:"created_at,omitempty"`
	UpdatedAt     time.Time          `json:"updated_at,omitempty"`
	EmbeddingRef  EmbeddingRef       `json:"embedding_ref,omitempty"`
}

type vectorStoreSnapshot struct {
	SchemaVersion int              `json:"schema_version"`
	UpdatedAt     time.Time        `json:"updated_at,omitempty"`
	Documents     []VectorDocument `json:"documents,omitempty"`
}

type VectorQueryOptions struct {
	TopK               int
	MinScore           float64
	MinConfidence      float64
	MinFreshness       float64
	AllowedObjectTypes []string
}

type vectorScoredDocument struct {
	Document  VectorDocument
	Score     float64
	Distance  float64
	Freshness float64
}

type VectorStore struct {
	enabled   bool
	baseDir   string
	indexPath string
	metrics   *memoryCounters

	mu        sync.RWMutex
	documents map[string]VectorDocument
	inverted  map[string]map[string]float64
	updatedAt time.Time
}

func NewVectorStore(baseDir string, metrics *memoryCounters) *VectorStore {
	resolved := resolveMemoryPath(baseDir)
	if strings.TrimSpace(resolved) == "" {
		return nil
	}
	return &VectorStore{
		enabled:   true,
		baseDir:   resolved,
		indexPath: filepath.Join(resolved, vectorIndexFileName),
		metrics:   metrics,
		documents: make(map[string]VectorDocument),
		inverted:  make(map[string]map[string]float64),
	}
}

func (s *VectorStore) Enabled() bool {
	return s != nil && s.enabled && strings.TrimSpace(s.baseDir) != ""
}

func (s *VectorStore) Load() error {
	if !s.Enabled() {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	data, err := os.ReadFile(s.indexPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read vector snapshot: %w", err)
	}
	var snapshot vectorStoreSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return fmt.Errorf("decode vector snapshot: %w", err)
	}
	s.updatedAt = snapshot.UpdatedAt.UTC()
	s.documents = make(map[string]VectorDocument, len(snapshot.Documents))
	for _, doc := range snapshot.Documents {
		normalized := normalizeVectorDocument(doc)
		if normalized.ObjectID == "" {
			continue
		}
		s.documents[normalized.ObjectID] = normalized
	}
	s.rebuildInvertedLocked()
	return nil
}

func (s *VectorStore) Rebuild(documents []VectorDocument) error {
	if !s.Enabled() {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.MkdirAll(s.baseDir, 0o755); err != nil {
		return fmt.Errorf("create vector sidecar dir: %w", err)
	}
	s.documents = make(map[string]VectorDocument, len(documents))
	for _, doc := range documents {
		normalized := normalizeVectorDocument(doc)
		if normalized.ObjectID == "" {
			continue
		}
		s.documents[normalized.ObjectID] = normalized
	}
	s.updatedAt = time.Now().UTC()
	s.rebuildInvertedLocked()
	if err := s.persistLocked(); err != nil {
		return err
	}
	if s.metrics != nil {
		s.metrics.vectorDocsIndexed.Add(uint64(len(s.documents)))
	}
	return nil
}

func (s *VectorStore) Upsert(doc VectorDocument) error {
	if !s.Enabled() {
		return nil
	}
	normalized := normalizeVectorDocument(doc)
	if normalized.ObjectID == "" {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.MkdirAll(s.baseDir, 0o755); err != nil {
		return fmt.Errorf("create vector sidecar dir: %w", err)
	}
	s.documents[normalized.ObjectID] = normalized
	s.updatedAt = time.Now().UTC()
	s.rebuildInvertedLocked()
	if err := s.persistLocked(); err != nil {
		return err
	}
	if s.metrics != nil {
		s.metrics.vectorDocsIndexed.Add(1)
	}
	return nil
}

func (s *VectorStore) Delete(objectID string) error {
	if !s.Enabled() {
		return nil
	}
	trimmed := strings.TrimSpace(objectID)
	if trimmed == "" {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.documents[trimmed]; !ok {
		return nil
	}
	delete(s.documents, trimmed)
	s.updatedAt = time.Now().UTC()
	s.rebuildInvertedLocked()
	return s.persistLocked()
}

func (s *VectorStore) Query(text string, opts VectorQueryOptions) []vectorScoredDocument {
	if !s.Enabled() {
		return nil
	}
	weights, norm := buildVectorWeights(text)
	if len(weights) == 0 || norm == 0 {
		return nil
	}
	allowed := make(map[string]struct{}, len(opts.AllowedObjectTypes))
	for _, kind := range opts.AllowedObjectTypes {
		trimmed := strings.TrimSpace(kind)
		if trimmed == "" {
			continue
		}
		allowed[trimmed] = struct{}{}
	}
	minScore := clamp01(maxFloat(opts.MinScore, 0))
	minConfidence := clamp01(opts.MinConfidence)
	minFreshness := clamp01(opts.MinFreshness)
	topK := opts.TopK
	if topK <= 0 {
		topK = defaultVectorTopK
	}

	now := time.Now().UTC()
	s.mu.RLock()
	defer s.mu.RUnlock()
	candidateIDs := s.candidateIDsLocked(weights)
	if len(candidateIDs) == 0 {
		candidateIDs = make([]string, 0, len(s.documents))
		for objectID := range s.documents {
			candidateIDs = append(candidateIDs, objectID)
		}
	}
	hits := make([]vectorScoredDocument, 0, len(candidateIDs))
	for _, objectID := range candidateIDs {
		doc, ok := s.documents[objectID]
		if !ok {
			continue
		}
		if len(allowed) > 0 {
			if _, ok := allowed[doc.ObjectType]; !ok {
				continue
			}
		}
		if minConfidence > 0 && doc.Confidence > 0 && doc.Confidence < minConfidence {
			continue
		}
		freshness := vectorDocumentFreshness(doc, now)
		if minFreshness > 0 && freshness < minFreshness {
			continue
		}
		score := cosineVectorScore(weights, norm, doc.Weights, doc.Norm)
		if score < minScore {
			continue
		}
		hits = append(hits, vectorScoredDocument{
			Document:  cloneVectorDocument(doc),
			Score:     clamp01(score),
			Distance:  clamp01(1 - score),
			Freshness: freshness,
		})
	}
	sort.SliceStable(hits, func(i, j int) bool {
		if hits[i].Score != hits[j].Score {
			return hits[i].Score > hits[j].Score
		}
		if hits[i].Freshness != hits[j].Freshness {
			return hits[i].Freshness > hits[j].Freshness
		}
		if !hits[i].Document.UpdatedAt.Equal(hits[j].Document.UpdatedAt) {
			return hits[i].Document.UpdatedAt.After(hits[j].Document.UpdatedAt)
		}
		return hits[i].Document.ObjectID < hits[j].Document.ObjectID
	})
	if len(hits) > topK {
		hits = hits[:topK]
	}
	return hits
}

func (s *VectorStore) DocumentCount() int {
	if !s.Enabled() {
		return 0
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.documents)
}

func (s *VectorStore) persistLocked() error {
	snapshot := vectorStoreSnapshot{
		SchemaVersion: vectorIndexSchemaVersion,
		UpdatedAt:     s.updatedAt,
		Documents:     make([]VectorDocument, 0, len(s.documents)),
	}
	for _, doc := range s.documents {
		snapshot.Documents = append(snapshot.Documents, normalizeVectorDocument(doc))
	}
	sort.SliceStable(snapshot.Documents, func(i, j int) bool {
		return snapshot.Documents[i].ObjectID < snapshot.Documents[j].ObjectID
	})
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("encode vector snapshot: %w", err)
	}
	if err := os.WriteFile(s.indexPath, data, 0o644); err != nil {
		return fmt.Errorf("write vector snapshot: %w", err)
	}
	return nil
}

func (s *VectorStore) rebuildInvertedLocked() {
	s.inverted = make(map[string]map[string]float64, len(s.documents))
	for objectID, doc := range s.documents {
		for term, weight := range doc.Weights {
			bucket := s.inverted[term]
			if bucket == nil {
				bucket = make(map[string]float64)
				s.inverted[term] = bucket
			}
			bucket[objectID] = weight
		}
	}
}

func (s *VectorStore) candidateIDsLocked(weights map[string]float64) []string {
	if len(weights) == 0 || len(s.inverted) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(weights)*2)
	out := make([]string, 0, len(weights)*2)
	for term := range weights {
		for objectID := range s.inverted[term] {
			if _, ok := seen[objectID]; ok {
				continue
			}
			seen[objectID] = struct{}{}
			out = append(out, objectID)
		}
	}
	return out
}

func normalizeVectorDocument(doc VectorDocument) VectorDocument {
	out := doc
	out.ObjectID = strings.TrimSpace(out.ObjectID)
	out.ObjectType = strings.TrimSpace(out.ObjectType)
	out.Summary = strings.TrimSpace(out.Summary)
	out.Text = strings.TrimSpace(out.Text)
	out.Terms = uniqueStrings(out.Terms)
	out.SourceRefs = normalizeSourceRefs(out.SourceRefs)
	out.Confidence = clamp01(out.Confidence)
	if !out.CreatedAt.IsZero() {
		out.CreatedAt = out.CreatedAt.UTC()
	}
	if !out.UpdatedAt.IsZero() {
		out.UpdatedAt = out.UpdatedAt.UTC()
	}
	out.EmbeddingRef = normalizeEmbeddingRef(out.EmbeddingRef, out.ObjectID)
	if len(out.Weights) == 0 && len(out.Terms) > 0 {
		out.Weights, out.Norm = buildVectorWeights(strings.Join(out.Terms, " "))
	} else {
		cleaned := make(map[string]float64, len(out.Weights))
		for term, weight := range out.Weights {
			trimmed := strings.TrimSpace(strings.ToLower(term))
			if trimmed == "" || weight <= 0 {
				continue
			}
			cleaned[trimmed] = weight
		}
		out.Weights = cleaned
		if out.Norm <= 0 {
			out.Norm = vectorWeightsNorm(cleaned)
		}
	}
	return out
}

func cloneVectorDocument(doc VectorDocument) VectorDocument {
	out := doc
	out.Terms = append([]string(nil), doc.Terms...)
	out.SourceRefs = append([]SourceRef(nil), doc.SourceRefs...)
	if doc.Weights != nil {
		out.Weights = make(map[string]float64, len(doc.Weights))
		for term, weight := range doc.Weights {
			out.Weights[term] = weight
		}
	}
	return out
}

func buildVectorWeights(text string) (map[string]float64, float64) {
	terms := vectorTokenize(text)
	if len(terms) == 0 {
		return nil, 0
	}
	weights := make(map[string]float64, len(terms))
	for _, term := range terms {
		weights[term]++
	}
	return weights, vectorWeightsNorm(weights)
}

func vectorWeightsNorm(weights map[string]float64) float64 {
	if len(weights) == 0 {
		return 0
	}
	total := 0.0
	for _, weight := range weights {
		total += weight * weight
	}
	if total <= 0 {
		return 0
	}
	return math.Sqrt(total)
}

func cosineVectorScore(queryWeights map[string]float64, queryNorm float64, docWeights map[string]float64, docNorm float64) float64 {
	if len(queryWeights) == 0 || len(docWeights) == 0 || queryNorm == 0 || docNorm == 0 {
		return 0
	}
	dot := 0.0
	for term, qWeight := range queryWeights {
		dot += qWeight * docWeights[term]
	}
	if dot <= 0 {
		return 0
	}
	return clamp01(dot / (queryNorm * docNorm))
}

func vectorTokenize(text string) []string {
	normalized := strings.ToLower(strings.TrimSpace(text))
	if normalized == "" {
		return nil
	}
	parts := strings.FieldsFunc(normalized, func(r rune) bool {
		return unicode.IsSpace(r) || strings.ContainsRune(",.!?:;()[]{}\"'`|", r)
	})
	out := make([]string, 0, len(parts)*2)
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		out = append(out, trimmed)
		for _, token := range strings.FieldsFunc(trimmed, func(r rune) bool {
			return r == '/' || r == '\\' || r == '_' || r == '-' || r == '.' || r == ':'
		}) {
			if strings.TrimSpace(token) == "" {
				continue
			}
			out = append(out, token)
		}
	}
	return uniqueStrings(out)
}

func vectorDocumentFreshness(doc VectorDocument, now time.Time) float64 {
	updatedAt := effectiveDecisionTimestamp(doc.UpdatedAt, doc.CreatedAt)
	if updatedAt.IsZero() {
		return 0
	}
	age := now.UTC().Sub(updatedAt.UTC())
	if age <= 0 {
		return 1
	}
	return clamp01(math.Exp2(-age.Hours() / defaultVectorFreshnessHalfLife.Hours()))
}

func defaultVectorBaseDir(baseDir string) string {
	trimmed := strings.TrimSpace(baseDir)
	if trimmed == "" {
		return ""
	}
	resolved := filepath.Clean(trimmed)
	parent := filepath.Dir(resolved)
	name := filepath.Base(resolved)
	if strings.TrimSpace(name) == "" || name == "." || name == string(filepath.Separator) {
		name = "memory"
	}
	return filepath.Join(parent, name+"-vector-sidecar")
}
