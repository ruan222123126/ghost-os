package decision

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"ghost-os/bridge/memory/internal/pathutil"
)

type decisionMemosPayload struct {
	Memos []DecisionMemo `json:"memos"`
}

type decisionRecipesPayload struct {
	Recipes []DecisionRecipe `json:"recipes"`
}

type decisionClustersPayload struct {
	Clusters []DecisionCluster `json:"clusters"`
}

type decisionRecipeRunsPayload struct {
	Runs []RecipeRun `json:"runs"`
}

// DecisionStore 维护 decision sidecar 的内存索引与文件快照。
type DecisionStore struct {
	baseDir      string
	memosPath    string
	recipesPath  string
	clustersPath string
	runsPath     string

	memoByID              map[string]DecisionMemo
	memoIDsByIntentKey    map[string][]string
	memoIDsByEvidenceID   map[string][]string
	memoIDsByClaimID      map[string][]string
	recipeByID            map[string]DecisionRecipe
	recipeIDsByIntentKey  map[string][]string
	recipeIDsByMemoID     map[string][]string
	recipeIDsByEvidenceID map[string][]string
	recipeIDsByClaimID    map[string][]string
	recipeRunByID         map[string]RecipeRun
	runIDsByRecipeID      map[string][]string
	runIDsByEvidenceID    map[string][]string
	runIDsByClaimID       map[string][]string
	memosByGraphNode      map[string][]string
	memosByAnchorKey      map[string][]string
	memosByToolName       map[string][]string
	clustersByID          map[string]DecisionCluster

	memos    []DecisionMemo
	recipes  []DecisionRecipe
	clusters []DecisionCluster
	runs     []RecipeRun
	mu       sync.RWMutex
}

// DecisionService 是后续 capture/query 的最薄 sidecar 壳。
type DecisionService struct {
	enabled                        bool
	captureOnTurn                  bool
	store                          *DecisionStore
	cold                           ColdStore
	truth                          TruthShadow
	workerExtractor                DecisionMemoExtractor
	maxHits                        int
	minConfidence                  float64
	minReuseScore                  float64
	recipeEnabled                  bool
	recipeInterval                 time.Duration
	recipeMinSupport               int
	recipeReuseEnabled             bool
	recipeExecutionTrackingEnabled bool
	recipeBackfillEnabled          bool
	recipeDefaultEnabled           bool
	recipeDefaultGrayPercent       int
	recipeMinSelectionConfidence   float64
	recipeMinSuccessRate           float64
	recipeBackfillBatchSize        int
	recipeBackfillInterval         time.Duration
	debugEnabled                   bool
	distiller                      *DecisionDistiller
	metrics                        *DecisionMetrics
	pendingMu                      sync.Mutex
	pendingRecipeRuns              map[string]RecipeRun
}

func NewDecisionStore(baseDir string) *DecisionStore {
	resolvedBaseDir := pathutil.Resolve(baseDir)
	store := &DecisionStore{
		baseDir:               resolvedBaseDir,
		memoByID:              make(map[string]DecisionMemo),
		memoIDsByIntentKey:    make(map[string][]string),
		memoIDsByEvidenceID:   make(map[string][]string),
		memoIDsByClaimID:      make(map[string][]string),
		recipeByID:            make(map[string]DecisionRecipe),
		recipeIDsByIntentKey:  make(map[string][]string),
		recipeIDsByMemoID:     make(map[string][]string),
		recipeIDsByEvidenceID: make(map[string][]string),
		recipeIDsByClaimID:    make(map[string][]string),
		recipeRunByID:         make(map[string]RecipeRun),
		runIDsByRecipeID:      make(map[string][]string),
		runIDsByEvidenceID:    make(map[string][]string),
		runIDsByClaimID:       make(map[string][]string),
		memosByGraphNode:      make(map[string][]string),
		memosByAnchorKey:      make(map[string][]string),
		memosByToolName:       make(map[string][]string),
		clustersByID:          make(map[string]DecisionCluster),
	}
	if resolvedBaseDir != "" {
		store.memosPath = filepath.Join(resolvedBaseDir, defaultDecisionMemosPathName)
		store.recipesPath = filepath.Join(resolvedBaseDir, defaultDecisionRecipesPathName)
		store.clustersPath = filepath.Join(resolvedBaseDir, defaultDecisionClustersPathName)
		store.runsPath = filepath.Join(resolvedBaseDir, defaultDecisionRecipeRunsPath)
	}
	return store
}

func NewDecisionService(config MemoryConfig, cold ColdStore) *DecisionService {
	workerExtractor := config.WorkerExtractor
	service := &DecisionService{
		enabled:                        config.DecisionEnabled,
		captureOnTurn:                  config.DecisionCaptureOnTurn,
		cold:                           cold,
		workerExtractor:                workerExtractor,
		maxHits:                        config.DecisionMaxHits,
		minConfidence:                  clamp01(maxFloat(config.DecisionMinConfidence, defaultDecisionMinConfidence)),
		minReuseScore:                  clamp01(maxFloat(config.DecisionMinReuseScore, defaultDecisionMinReuseScore)),
		recipeEnabled:                  config.DecisionRecipeEnabled,
		recipeInterval:                 config.DecisionRecipeInterval,
		recipeMinSupport:               config.DecisionRecipeMinSupport,
		recipeReuseEnabled:             config.RecipeReuseEnabled,
		recipeExecutionTrackingEnabled: config.RecipeExecutionTrackingEnabled,
		recipeBackfillEnabled:          config.RecipeBackfillEnabled,
		recipeDefaultEnabled:           config.RecipeDefaultEnabled,
		recipeDefaultGrayPercent:       config.RecipeDefaultGrayPercent,
		recipeMinSelectionConfidence:   clamp01(config.RecipeMinSelectionConfidence),
		recipeMinSuccessRate:           clamp01(config.RecipeMinSuccessRate),
		recipeBackfillBatchSize:        config.RecipeBackfillBatchSize,
		recipeBackfillInterval:         config.RecipeBackfillInterval,
		debugEnabled:                   config.DecisionDebugEnabled,
		metrics:                        nil,
		pendingRecipeRuns:              make(map[string]RecipeRun),
	}
	if service.maxHits <= 0 {
		service.maxHits = defaultDecisionMaxHits
	}
	if service.recipeInterval <= 0 {
		service.recipeInterval = defaultDecisionRecipeInterval
	}
	if service.recipeMinSupport <= 0 {
		service.recipeMinSupport = defaultDecisionRecipeMinSupport
	}
	if !config.DecisionCaptureOnTurnSet {
		service.captureOnTurn = true
	}
	if !service.enabled {
		return service
	}
	service.store = NewDecisionStore(config.DecisionPath)
	if err := service.store.Load(); err != nil {
		log.Printf("[MEMORY] decision sidecar load failed, fallback to empty store: %v", err)
		service.store = NewDecisionStore(config.DecisionPath)
	}
	if service.recipeEnabled {
		service.distiller = NewDecisionDistiller(service, service.recipeInterval, service.recipeMinSupport)
	}
	return service
}

func (d *DecisionService) Enabled() bool {
	return d != nil && d.enabled && d.store != nil
}

func (d *DecisionService) SetTruthShadow(shadow TruthShadow) {
	if d == nil {
		return
	}
	d.truth = shadow
}

func (d *DecisionService) SetMetrics(metrics *DecisionMetrics) {
	if d == nil {
		return
	}
	d.metrics = metrics
}

func (d *DecisionService) Stats(namespace string) DecisionStats {
	if d == nil || d.store == nil {
		return DecisionStats{Namespace: normalizeDecisionNamespace(namespace)}
	}
	return d.store.Stats(namespace)
}

func (d *DecisionService) RecordMemoAccess(ids []string, accessedAt time.Time) {
	if d == nil || d.store == nil {
		return
	}
	d.store.RecordMemoAccess(ids, accessedAt)
}

func (s *DecisionStore) enabled() bool {
	return s != nil && strings.TrimSpace(s.baseDir) != ""
}

func (s *DecisionStore) Load() error {
	if s == nil || !s.enabled() {
		return nil
	}

	memos, err := readDecisionMemos(s.memosPath)
	if err != nil {
		return err
	}
	recipes, err := readDecisionRecipes(s.recipesPath)
	if err != nil {
		return err
	}
	clusters, err := readDecisionClusters(s.clustersPath)
	if err != nil {
		return err
	}
	runs, err := readDecisionRecipeRuns(s.runsPath)
	if err != nil {
		return err
	}

	s.mu.Lock()
	s.memos = normalizeDecisionMemos(memos)
	s.recipes = normalizeDecisionRecipes(recipes)
	s.clusters = normalizeDecisionClusters(clusters)
	s.runs = normalizeRecipeRuns(runs)
	s.rebuildIndexesLocked()
	s.mu.Unlock()
	return nil
}

func readDecisionMemos(path string) ([]DecisionMemo, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read decision memos: %w", err)
	}
	var payload decisionMemosPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("decode decision memos: %w", err)
	}
	return payload.Memos, nil
}

func readDecisionRecipes(path string) ([]DecisionRecipe, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read decision recipes: %w", err)
	}
	var payload decisionRecipesPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("decode decision recipes: %w", err)
	}
	return payload.Recipes, nil
}

func readDecisionClusters(path string) ([]DecisionCluster, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read decision clusters: %w", err)
	}
	var payload decisionClustersPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("decode decision clusters: %w", err)
	}
	return payload.Clusters, nil
}

func readDecisionRecipeRuns(path string) ([]RecipeRun, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read decision recipe runs: %w", err)
	}
	var payload decisionRecipeRunsPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, fmt.Errorf("decode decision recipe runs: %w", err)
	}
	return payload.Runs, nil
}

func (s *DecisionStore) Persist() error {
	if s == nil || !s.enabled() {
		return nil
	}
	if err := os.MkdirAll(s.baseDir, 0o700); err != nil {
		return fmt.Errorf("create decision directory: %w", err)
	}

	s.mu.RLock()
	memos := cloneDecisionMemos(s.memos)
	recipes := cloneDecisionRecipes(s.recipes)
	clusters := cloneDecisionClusters(s.clusters)
	runs := cloneRecipeRuns(s.runs)
	s.mu.RUnlock()

	if memos == nil {
		memos = []DecisionMemo{}
	}
	if recipes == nil {
		recipes = []DecisionRecipe{}
	}
	if clusters == nil {
		clusters = []DecisionCluster{}
	}
	if runs == nil {
		runs = []RecipeRun{}
	}

	sort.Slice(memos, func(i, j int) bool { return memos[i].ID < memos[j].ID })
	sort.Slice(recipes, func(i, j int) bool { return recipes[i].ID < recipes[j].ID })
	sort.Slice(clusters, func(i, j int) bool { return clusters[i].ID < clusters[j].ID })
	sort.Slice(runs, func(i, j int) bool { return runs[i].ID < runs[j].ID })

	if err := writeDecisionJSON(s.memosPath, decisionMemosPayload{Memos: memos}); err != nil {
		return err
	}
	if err := writeDecisionJSON(s.recipesPath, decisionRecipesPayload{Recipes: recipes}); err != nil {
		return err
	}
	if err := writeDecisionJSON(s.clustersPath, decisionClustersPayload{Clusters: clusters}); err != nil {
		return err
	}
	if err := writeDecisionJSON(s.runsPath, decisionRecipeRunsPayload{Runs: runs}); err != nil {
		return err
	}
	return nil
}

func writeDecisionJSON(path string, payload any) error {
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal decision file %s: %w", filepath.Base(path), err)
	}
	data = append(data, '\n')
	tmpPath := fmt.Sprintf("%s.tmp-%d", path, time.Now().UTC().UnixNano())
	if err := os.WriteFile(tmpPath, data, 0o600); err != nil {
		return fmt.Errorf("write decision temp file %s: %w", filepath.Base(path), err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("replace decision file %s: %w", filepath.Base(path), err)
	}
	return nil
}

func (s *DecisionStore) Memo(id string) (DecisionMemo, bool) {
	if s == nil {
		return DecisionMemo{}, false
	}
	s.mu.RLock()
	memo, ok := s.memoByID[strings.TrimSpace(id)]
	s.mu.RUnlock()
	if !ok {
		return DecisionMemo{}, false
	}
	return cloneDecisionMemo(memo), true
}

func (s *DecisionStore) Recipe(id string) (DecisionRecipe, bool) {
	if s == nil {
		return DecisionRecipe{}, false
	}
	s.mu.RLock()
	recipe, ok := s.recipeByID[strings.TrimSpace(id)]
	s.mu.RUnlock()
	if !ok {
		return DecisionRecipe{}, false
	}
	return cloneDecisionRecipe(recipe), true
}

func (s *DecisionStore) Cluster(id string) (DecisionCluster, bool) {
	if s == nil {
		return DecisionCluster{}, false
	}
	s.mu.RLock()
	cluster, ok := s.clustersByID[strings.TrimSpace(id)]
	s.mu.RUnlock()
	if !ok {
		return DecisionCluster{}, false
	}
	return cloneDecisionCluster(cluster), true
}

func (s *DecisionStore) ListMemos(namespace string) []DecisionMemo {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	out := make([]DecisionMemo, 0, len(s.memos))
	ns := normalizeDecisionNamespace(namespace)
	for _, memo := range s.memos {
		if namespace != "" && memo.Namespace != ns {
			continue
		}
		out = append(out, cloneDecisionMemo(memo))
	}
	s.mu.RUnlock()
	sort.Slice(out, func(i, j int) bool {
		if out[i].LastUsedAt.Equal(out[j].LastUsedAt) {
			return out[i].ID < out[j].ID
		}
		return out[i].LastUsedAt.After(out[j].LastUsedAt)
	})
	return out
}

func (s *DecisionStore) ListRecipes(namespace string) []DecisionRecipe {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	out := make([]DecisionRecipe, 0, len(s.recipes))
	ns := normalizeDecisionNamespace(namespace)
	for _, recipe := range s.recipes {
		if namespace != "" && recipe.Namespace != ns {
			continue
		}
		out = append(out, cloneDecisionRecipe(recipe))
	}
	s.mu.RUnlock()
	sort.Slice(out, func(i, j int) bool {
		if out[i].UpdatedAt.Equal(out[j].UpdatedAt) {
			return out[i].ID < out[j].ID
		}
		return out[i].UpdatedAt.After(out[j].UpdatedAt)
	})
	return out
}

func (s *DecisionStore) ListClusters(namespace string) []DecisionCluster {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	out := make([]DecisionCluster, 0, len(s.clusters))
	ns := normalizeDecisionNamespace(namespace)
	for _, cluster := range s.clusters {
		if namespace != "" && cluster.Namespace != ns {
			continue
		}
		out = append(out, cloneDecisionCluster(cluster))
	}
	s.mu.RUnlock()
	sort.Slice(out, func(i, j int) bool {
		if out[i].UpdatedAt.Equal(out[j].UpdatedAt) {
			return out[i].ID < out[j].ID
		}
		return out[i].UpdatedAt.After(out[j].UpdatedAt)
	})
	return out
}

func (s *DecisionStore) ListRecipeRuns(namespace string) []RecipeRun {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	out := make([]RecipeRun, 0, len(s.runs))
	ns := normalizeDecisionNamespace(namespace)
	for _, run := range s.runs {
		if namespace != "" && run.Namespace != ns {
			continue
		}
		out = append(out, cloneRecipeRun(run))
	}
	s.mu.RUnlock()
	sort.Slice(out, func(i, j int) bool {
		if out[i].SelectedAt.Equal(out[j].SelectedAt) {
			return out[i].ID < out[j].ID
		}
		return out[i].SelectedAt.After(out[j].SelectedAt)
	})
	return out
}

func (s *DecisionStore) RecordMemoAccess(ids []string, accessedAt time.Time) {
	if s == nil || len(ids) == 0 {
		return
	}
	when := accessedAt.UTC()
	if when.IsZero() {
		when = time.Now().UTC()
	}

	s.mu.Lock()
	for _, rawID := range ids {
		id := strings.TrimSpace(rawID)
		if id == "" {
			continue
		}
		memo, ok := s.memoByID[id]
		if !ok {
			continue
		}
		memo.AccessCount++
		memo.LastUsedAt = when
		s.memoByID[id] = memo
		for i := range s.memos {
			if s.memos[i].ID == id {
				s.memos[i] = memo
				break
			}
		}
	}
	s.mu.Unlock()
}

func (s *DecisionStore) UpsertMemo(memo DecisionMemo) (bool, error) {
	if s == nil {
		return false, fmt.Errorf("decision store is nil")
	}
	normalized := normalizeDecisionMemo(memo)
	if normalized.ID == "" {
		return false, fmt.Errorf("decision memo id is required")
	}
	if err := validateDecisionMemoLineage(normalized); err != nil {
		return false, err
	}

	s.mu.Lock()
	_, exists := s.memoByID[normalized.ID]
	if exists {
		for i := range s.memos {
			if s.memos[i].ID == normalized.ID {
				s.memos[i] = normalized
				break
			}
		}
	} else {
		s.memos = append(s.memos, normalized)
	}
	s.rebuildIndexesLocked()
	s.mu.Unlock()
	return !exists, nil
}

func (s *DecisionStore) UpsertRecipe(recipe DecisionRecipe) (bool, error) {
	if s == nil {
		return false, fmt.Errorf("decision store is nil")
	}
	normalized := normalizeDecisionRecipe(recipe)
	if normalized.ID == "" {
		return false, fmt.Errorf("decision recipe id is required")
	}
	if err := validateDecisionRecipeLineage(normalized); err != nil {
		return false, err
	}

	s.mu.Lock()
	_, exists := s.recipeByID[normalized.ID]
	if exists {
		for i := range s.recipes {
			if s.recipes[i].ID == normalized.ID {
				s.recipes[i] = normalized
				break
			}
		}
	} else {
		s.recipes = append(s.recipes, normalized)
	}
	s.rebuildIndexesLocked()
	s.mu.Unlock()
	return !exists, nil
}

func (s *DecisionStore) UpsertCluster(cluster DecisionCluster) (bool, error) {
	if s == nil {
		return false, fmt.Errorf("decision store is nil")
	}
	normalized := normalizeDecisionCluster(cluster)
	if normalized.ID == "" {
		return false, fmt.Errorf("decision cluster id is required")
	}

	s.mu.Lock()
	_, exists := s.clustersByID[normalized.ID]
	if exists {
		for i := range s.clusters {
			if s.clusters[i].ID == normalized.ID {
				s.clusters[i] = normalized
				break
			}
		}
	} else {
		s.clusters = append(s.clusters, normalized)
	}
	s.rebuildIndexesLocked()
	s.mu.Unlock()
	return !exists, nil
}

func (s *DecisionStore) UpsertRecipeRun(run RecipeRun) (bool, error) {
	if s == nil {
		return false, fmt.Errorf("decision store is nil")
	}
	normalized := normalizeRecipeRun(run)
	if normalized.ID == "" {
		return false, fmt.Errorf("recipe run id is required")
	}
	if err := validateRecipeRunLineage(normalized); err != nil {
		return false, err
	}

	s.mu.Lock()
	_, exists := s.recipeRunByID[normalized.ID]
	if exists {
		for i := range s.runs {
			if s.runs[i].ID == normalized.ID {
				s.runs[i] = normalized
				break
			}
		}
	} else {
		s.runs = append(s.runs, normalized)
	}
	s.rebuildIndexesLocked()
	s.mu.Unlock()
	return !exists, nil
}

func (s *DecisionStore) ResetNamespace(namespace string) error {
	if s == nil {
		return nil
	}
	ns := normalizeDecisionNamespace(namespace)
	s.mu.Lock()
	s.memos = filterDecisionMemosByNamespace(s.memos, ns)
	s.recipes = filterDecisionRecipesByNamespace(s.recipes, ns)
	s.clusters = filterDecisionClustersByNamespace(s.clusters, ns)
	s.runs = filterRecipeRunsByNamespace(s.runs, ns)
	s.rebuildIndexesLocked()
	s.mu.Unlock()
	return s.Persist()
}

func (s *DecisionStore) Stats(namespace string) DecisionStats {
	if s == nil {
		return DecisionStats{Namespace: normalizeDecisionNamespace(namespace)}
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := DecisionStats{Namespace: normalizeDecisionNamespace(namespace)}
	for _, memo := range s.memos {
		if namespace != "" && memo.Namespace != stats.Namespace {
			continue
		}
		stats.MemoCount++
		if memo.HumanBlocked {
			stats.HumanBlockedCount++
		}
		switch memo.Outcome {
		case DecisionOutcomeSuccess:
			stats.SuccessCount++
		case DecisionOutcomePartial:
			stats.PartialCount++
		case DecisionOutcomeFailure:
			stats.FailureCount++
		case DecisionOutcomeAwaitingHuman:
			stats.AwaitingHumanCount++
		case DecisionOutcomeCancelled:
			stats.CancelledCount++
		}
	}
	for _, recipe := range s.recipes {
		if namespace != "" && recipe.Namespace != stats.Namespace {
			continue
		}
		stats.RecipeCount++
	}
	for _, cluster := range s.clusters {
		if namespace != "" && cluster.Namespace != stats.Namespace {
			continue
		}
		stats.ClusterCount++
	}
	return stats
}

func (s *DecisionStore) rebuildIndexes() {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.rebuildIndexesLocked()
	s.mu.Unlock()
}

func (s *DecisionStore) rebuildIndexesLocked() {
	s.memoByID = make(map[string]DecisionMemo, len(s.memos))
	s.memoIDsByIntentKey = make(map[string][]string)
	s.memoIDsByEvidenceID = make(map[string][]string)
	s.memoIDsByClaimID = make(map[string][]string)
	s.recipeByID = make(map[string]DecisionRecipe, len(s.recipes))
	s.recipeIDsByIntentKey = make(map[string][]string)
	s.recipeIDsByMemoID = make(map[string][]string)
	s.recipeIDsByEvidenceID = make(map[string][]string)
	s.recipeIDsByClaimID = make(map[string][]string)
	s.recipeRunByID = make(map[string]RecipeRun, len(s.runs))
	s.runIDsByRecipeID = make(map[string][]string)
	s.runIDsByEvidenceID = make(map[string][]string)
	s.runIDsByClaimID = make(map[string][]string)
	s.memosByGraphNode = make(map[string][]string)
	s.memosByAnchorKey = make(map[string][]string)
	s.memosByToolName = make(map[string][]string)
	s.clustersByID = make(map[string]DecisionCluster, len(s.clusters))

	for _, memo := range s.memos {
		normalized := normalizeDecisionMemo(memo)
		if normalized.ID == "" {
			continue
		}
		s.memoByID[normalized.ID] = normalized
		appendDecisionIndex(s.memoIDsByIntentKey, normalized.IntentKey, normalized.ID)
		for _, ref := range normalized.GraphNodeRefs {
			appendDecisionIndex(s.memosByGraphNode, ref, normalized.ID)
		}
		for _, key := range normalized.AnchorKeys {
			appendDecisionIndex(s.memosByAnchorKey, key, normalized.ID)
		}
		for _, tool := range normalized.ToolsUsed {
			appendDecisionIndex(s.memosByToolName, tool.Name, normalized.ID)
		}
		for _, evidenceID := range normalized.SourceEvidenceIDs {
			appendDecisionIndex(s.memoIDsByEvidenceID, evidenceID, normalized.ID)
		}
		for _, claimID := range append(append([]string(nil), normalized.SourceClaimIDs...), normalized.DerivedClaimIDs...) {
			appendDecisionIndex(s.memoIDsByClaimID, claimID, normalized.ID)
		}
	}
	for _, recipe := range s.recipes {
		normalized := normalizeDecisionRecipe(recipe)
		if normalized.ID == "" {
			continue
		}
		s.recipeByID[normalized.ID] = normalized
		appendDecisionIndex(s.recipeIDsByIntentKey, normalized.IntentKey, normalized.ID)
		for _, memoID := range normalized.SourceMemoIDs {
			appendDecisionIndex(s.recipeIDsByMemoID, memoID, normalized.ID)
		}
		for _, evidenceID := range normalized.SourceEvidenceIDs {
			appendDecisionIndex(s.recipeIDsByEvidenceID, evidenceID, normalized.ID)
		}
		for _, claimID := range append(append([]string(nil), normalized.SourceClaimIDs...), normalized.ContradictedClaimIDs...) {
			appendDecisionIndex(s.recipeIDsByClaimID, claimID, normalized.ID)
		}
	}
	for _, cluster := range s.clusters {
		normalized := normalizeDecisionCluster(cluster)
		if normalized.ID == "" {
			continue
		}
		s.clustersByID[normalized.ID] = normalized
	}
	for _, run := range s.runs {
		normalized := normalizeRecipeRun(run)
		if normalized.ID == "" {
			continue
		}
		s.recipeRunByID[normalized.ID] = normalized
		appendDecisionIndex(s.runIDsByRecipeID, normalized.RecipeID, normalized.ID)
		for _, evidenceID := range append(append([]string(nil), normalized.SelectionEvidenceIDs...), normalized.ExecutionEvidenceIDs...) {
			appendDecisionIndex(s.runIDsByEvidenceID, evidenceID, normalized.ID)
		}
		for _, claimID := range append(append(append([]string(nil), normalized.SelectionClaimIDs...), normalized.EmittedClaimIDs...), normalized.InvalidatedClaimIDs...) {
			appendDecisionIndex(s.runIDsByClaimID, claimID, normalized.ID)
		}
	}
	s.sortIndexValuesLocked()
	// 将归一化后的值回写到切片，避免索引与原始快照漂移。
	s.memos = normalizeDecisionMemos(s.memos)
	s.recipes = normalizeDecisionRecipes(s.recipes)
	s.clusters = normalizeDecisionClusters(s.clusters)
	s.runs = normalizeRecipeRuns(s.runs)
}

func (s *DecisionStore) sortIndexValuesLocked() {
	for key, ids := range s.memoIDsByIntentKey {
		sort.Strings(ids)
		s.memoIDsByIntentKey[key] = ids
	}
	for key, ids := range s.recipeIDsByIntentKey {
		sort.Strings(ids)
		s.recipeIDsByIntentKey[key] = ids
	}
	for key, ids := range s.memosByGraphNode {
		sort.Strings(ids)
		s.memosByGraphNode[key] = ids
	}
	for key, ids := range s.memosByAnchorKey {
		sort.Strings(ids)
		s.memosByAnchorKey[key] = ids
	}
	for key, ids := range s.memosByToolName {
		sort.Strings(ids)
		s.memosByToolName[key] = ids
	}
	for key, ids := range s.memoIDsByEvidenceID {
		sort.Strings(ids)
		s.memoIDsByEvidenceID[key] = ids
	}
	for key, ids := range s.memoIDsByClaimID {
		sort.Strings(ids)
		s.memoIDsByClaimID[key] = ids
	}
	for key, ids := range s.recipeIDsByMemoID {
		sort.Strings(ids)
		s.recipeIDsByMemoID[key] = ids
	}
	for key, ids := range s.recipeIDsByEvidenceID {
		sort.Strings(ids)
		s.recipeIDsByEvidenceID[key] = ids
	}
	for key, ids := range s.recipeIDsByClaimID {
		sort.Strings(ids)
		s.recipeIDsByClaimID[key] = ids
	}
	for key, ids := range s.runIDsByRecipeID {
		sort.Strings(ids)
		s.runIDsByRecipeID[key] = ids
	}
	for key, ids := range s.runIDsByEvidenceID {
		sort.Strings(ids)
		s.runIDsByEvidenceID[key] = ids
	}
	for key, ids := range s.runIDsByClaimID {
		sort.Strings(ids)
		s.runIDsByClaimID[key] = ids
	}
}

func appendDecisionIndex(index map[string][]string, key, id string) {
	trimmedKey := strings.TrimSpace(key)
	trimmedID := strings.TrimSpace(id)
	if trimmedKey == "" || trimmedID == "" {
		return
	}
	for _, existing := range index[trimmedKey] {
		if existing == trimmedID {
			return
		}
	}
	index[trimmedKey] = append(index[trimmedKey], trimmedID)
}

func validateDecisionMemoLineage(memo DecisionMemo) error {
	if len(memo.DerivedClaimIDs) > 0 && len(memo.SourceClaimIDs) == 0 && len(memo.SourceEvidenceIDs) == 0 && !memo.LineagePartial {
		return fmt.Errorf("decision memo lineage requires source_claim_ids or source_evidence_ids when derived_claim_ids are present: %s", memo.ID)
	}
	return nil
}

func validateDecisionRecipeLineage(recipe DecisionRecipe) error {
	if len(recipe.SourceMemoIDs) > 0 && len(recipe.SourceClaimIDs) == 0 && len(recipe.SourceEvidenceIDs) == 0 && !recipe.LineagePartial {
		return fmt.Errorf("decision recipe lineage requires source_claim_ids or source_evidence_ids when source_memo_ids are present: %s", recipe.ID)
	}
	return nil
}

func validateRecipeRunLineage(run RecipeRun) error {
	if run.RecipeID != "" && run.Selection.SelectedRecipeID != "" && len(run.SelectionClaimIDs) == 0 && len(run.SelectionEvidenceIDs) == 0 && !run.LineagePartial {
		return fmt.Errorf("recipe run lineage requires selection_claim_ids or selection_evidence_ids: %s", run.ID)
	}
	return nil
}

func (s *DecisionStore) ExplainMemoLineage(memoID string) (MemoLineageExplanation, bool) {
	if s == nil {
		return MemoLineageExplanation{}, false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	memo, ok := s.memoByID[strings.TrimSpace(memoID)]
	if !ok {
		return MemoLineageExplanation{}, false
	}
	return MemoLineageExplanation{
		Memo:             cloneDecisionMemo(memo),
		RelatedRecipeIDs: append([]string(nil), s.recipeIDsByMemoID[memo.ID]...),
		RelatedRunIDs:    relatedRunIDsForMemoLocked(s, memo),
	}, true
}

func (s *DecisionStore) ExplainRecipeLineage(recipeID string) (RecipeLineageExplanation, bool) {
	if s == nil {
		return RecipeLineageExplanation{}, false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	recipe, ok := s.recipeByID[strings.TrimSpace(recipeID)]
	if !ok {
		return RecipeLineageExplanation{}, false
	}
	sourceMemos := make([]DecisionMemo, 0, len(recipe.SourceMemoIDs))
	for _, memoID := range recipe.SourceMemoIDs {
		if memo, exists := s.memoByID[memoID]; exists {
			sourceMemos = append(sourceMemos, cloneDecisionMemo(memo))
		}
	}
	return RecipeLineageExplanation{
		Recipe:         cloneDecisionRecipe(recipe),
		SourceMemos:    sourceMemos,
		RelatedRunIDs:  append([]string(nil), s.runIDsByRecipeID[recipe.ID]...),
		RelatedMemoIDs: append([]string(nil), recipe.SourceMemoIDs...),
	}, true
}

func (s *DecisionStore) ExplainRecipeRunLineage(runID string) (RecipeRunLineageExplanation, bool) {
	if s == nil {
		return RecipeRunLineageExplanation{}, false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	run, ok := s.recipeRunByID[strings.TrimSpace(runID)]
	if !ok {
		return RecipeRunLineageExplanation{}, false
	}
	var recipe *DecisionRecipe
	if current, exists := s.recipeByID[run.RecipeID]; exists {
		cloned := cloneDecisionRecipe(current)
		recipe = &cloned
	}
	return RecipeRunLineageExplanation{
		Run:            cloneRecipeRun(run),
		Recipe:         recipe,
		RelatedMemoIDs: relatedMemoIDsForRunLocked(s, run),
	}, true
}

func relatedRunIDsForMemoLocked(s *DecisionStore, memo DecisionMemo) []string {
	seen := make(map[string]struct{})
	for _, claimID := range append(append([]string(nil), memo.SourceClaimIDs...), memo.DerivedClaimIDs...) {
		for _, runID := range s.runIDsByClaimID[claimID] {
			seen[runID] = struct{}{}
		}
	}
	for _, evidenceID := range memo.SourceEvidenceIDs {
		for _, runID := range s.runIDsByEvidenceID[evidenceID] {
			seen[runID] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for runID := range seen {
		out = append(out, runID)
	}
	sort.Strings(out)
	return out
}

func relatedMemoIDsForRunLocked(s *DecisionStore, run RecipeRun) []string {
	seen := make(map[string]struct{})
	for _, claimID := range append(append([]string(nil), run.SelectionClaimIDs...), run.EmittedClaimIDs...) {
		for _, memoID := range s.memoIDsByClaimID[claimID] {
			seen[memoID] = struct{}{}
		}
	}
	for _, evidenceID := range append(append([]string(nil), run.SelectionEvidenceIDs...), run.ExecutionEvidenceIDs...) {
		for _, memoID := range s.memoIDsByEvidenceID[evidenceID] {
			seen[memoID] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for memoID := range seen {
		out = append(out, memoID)
	}
	sort.Strings(out)
	return out
}

func filterDecisionMemosByNamespace(memos []DecisionMemo, namespace string) []DecisionMemo {
	if len(memos) == 0 {
		return nil
	}
	out := make([]DecisionMemo, 0, len(memos))
	for _, memo := range memos {
		if memo.Namespace == namespace {
			continue
		}
		out = append(out, memo)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func filterDecisionRecipesByNamespace(recipes []DecisionRecipe, namespace string) []DecisionRecipe {
	if len(recipes) == 0 {
		return nil
	}
	out := make([]DecisionRecipe, 0, len(recipes))
	for _, recipe := range recipes {
		if recipe.Namespace == namespace {
			continue
		}
		out = append(out, recipe)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func filterDecisionClustersByNamespace(clusters []DecisionCluster, namespace string) []DecisionCluster {
	if len(clusters) == 0 {
		return nil
	}
	out := make([]DecisionCluster, 0, len(clusters))
	for _, cluster := range clusters {
		if cluster.Namespace == namespace {
			continue
		}
		out = append(out, cluster)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func filterRecipeRunsByNamespace(runs []RecipeRun, namespace string) []RecipeRun {
	if len(runs) == 0 {
		return nil
	}
	out := make([]RecipeRun, 0, len(runs))
	for _, run := range runs {
		if run.Namespace == namespace {
			continue
		}
		out = append(out, run)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
