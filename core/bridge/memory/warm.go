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
)

const (
	defaultWarmCapacity = 100
	defaultWarmTTL      = 24 * time.Hour
)

type warmStorePayload struct {
	Entries []MemoryEntry `json:"entries"`
}

// WarmMemory 是 L2 温数据层，维护最近 24 小时高频条目。
type WarmMemory struct {
	entries   []MemoryEntry
	index     map[string]int
	capacity  int
	ttl       time.Duration
	storePath string
	mu        sync.RWMutex
}

func NewWarmMemory(capacity int, storePath string) *WarmMemory {
	return NewWarmMemoryWithTTL(capacity, storePath, defaultWarmTTL)
}

func NewWarmMemoryWithTTL(capacity int, storePath string, ttl time.Duration) *WarmMemory {
	size := capacity
	if size <= 0 {
		size = defaultWarmCapacity
	}
	effectiveTTL := ttl
	if effectiveTTL <= 0 {
		effectiveTTL = defaultWarmTTL
	}
	return &WarmMemory{
		entries:   make([]MemoryEntry, 0, size),
		index:     make(map[string]int, size),
		capacity:  size,
		ttl:       effectiveTTL,
		storePath: resolveMemoryPath(storePath),
	}
}

// Store 写入或更新条目。命中容量上限时按 LRU 淘汰最旧项。
func (w *WarmMemory) Store(entry MemoryEntry) error {
	normalized := normalizeEntry(entry)
	if normalized.ID == "" {
		return fmt.Errorf("memory entry id is required")
	}
	if normalized.Importance <= 0 {
		normalized.Importance = calculateImportance(normalized)
	}
	if normalized.ExpiresAt.IsZero() {
		normalized.ExpiresAt = defaultEntryExpiry(normalized, time.Now().UTC(), w.ttl)
	}

	w.mu.Lock()
	w.pruneExpiredLocked(time.Now().UTC())

	if idx, ok := w.index[normalized.ID]; ok {
		w.entries[idx] = normalized
		w.touchLocked(normalized.ID)
	} else {
		if len(w.entries) >= w.capacity {
			w.evictOldestLocked()
		}
		w.entries = append(w.entries, normalized)
		w.index[normalized.ID] = len(w.entries) - 1
	}
	w.mu.Unlock()

	return w.Persist()
}

// Retrieve 按条件检索温数据，并更新命中条目的访问热度。
func (w *WarmMemory) Retrieve(query MemoryQuery) ([]MemoryEntry, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	now := time.Now().UTC()
	w.pruneExpiredLocked(now)

	type scored struct {
		Entry MemoryEntry
		Score float64
	}

	candidates := make([]scored, 0, len(w.entries))
	for _, entry := range w.entries {
		if !entryMatchesQuery(entry, query) {
			continue
		}
		candidates = append(candidates, scored{
			Entry: cloneEntry(entry),
			Score: warmEntryScore(entry, query, now),
		})
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].Score != candidates[j].Score {
			return candidates[i].Score > candidates[j].Score
		}
		return candidates[i].Entry.Timestamp.After(candidates[j].Entry.Timestamp)
	})

	limit := len(candidates)
	if query.Limit > 0 && query.Limit < limit {
		limit = query.Limit
	}
	selected := make([]MemoryEntry, 0, limit)
	for i := 0; i < limit; i++ {
		selected = append(selected, cloneEntry(candidates[i].Entry))
	}

	for _, entry := range selected {
		w.bumpAccessLocked(entry.ID)
	}

	return selected, nil
}

func (w *WarmMemory) Delete(id string) error {
	trimmed := strings.TrimSpace(id)
	if trimmed == "" {
		return fmt.Errorf("memory entry id is required")
	}

	w.mu.Lock()
	idx, ok := w.index[trimmed]
	if !ok {
		w.mu.Unlock()
		return nil
	}
	w.removeAtLocked(idx)
	w.mu.Unlock()

	return w.Persist()
}

// Snapshot 返回当前 warm 层快照，供后台演化流程使用。
func (w *WarmMemory) Snapshot() []MemoryEntry {
	w.mu.Lock()
	now := time.Now().UTC()
	w.pruneExpiredLocked(now)
	out := cloneEntries(w.entries)
	w.mu.Unlock()
	return out
}

// Persist 将温数据写入 JSON，便于进程重启后快速恢复。
func (w *WarmMemory) Persist() error {
	if w.storePath == "" {
		return nil
	}

	w.mu.RLock()
	payload := warmStorePayload{
		Entries: cloneEntries(w.entries),
	}
	w.mu.RUnlock()

	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal warm memory: %w", err)
	}
	data = append(data, '\n')

	if err := os.MkdirAll(filepath.Dir(w.storePath), 0o700); err != nil {
		return fmt.Errorf("create warm memory directory: %w", err)
	}

	tmpPath := fmt.Sprintf("%s.tmp-%d", w.storePath, time.Now().UnixNano())
	if err := os.WriteFile(tmpPath, data, 0o600); err != nil {
		return fmt.Errorf("write warm memory temp file: %w", err)
	}
	if err := os.Rename(tmpPath, w.storePath); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("replace warm memory file: %w", err)
	}
	return nil
}

// Load 从 JSON 载入温数据，并清理过期条目。
func (w *WarmMemory) Load() error {
	if w.storePath == "" {
		return nil
	}

	data, err := os.ReadFile(w.storePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read warm memory file: %w", err)
	}

	var payload warmStorePayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return fmt.Errorf("unmarshal warm memory: %w", err)
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	w.entries = w.entries[:0]
	w.index = make(map[string]int, w.capacity)

	for _, entry := range payload.Entries {
		normalized := normalizeEntry(entry)
		if normalized.ID == "" {
			continue
		}
		if len(w.entries) >= w.capacity {
			break
		}
		w.entries = append(w.entries, normalized)
		w.index[normalized.ID] = len(w.entries) - 1
	}

	w.pruneExpiredLocked(time.Now().UTC())
	return nil
}

func (w *WarmMemory) pruneExpiredLocked(now time.Time) {
	kept := w.entries[:0]
	for _, entry := range w.entries {
		if isEntryExpired(entry, now, w.ttl) {
			continue
		}
		kept = append(kept, entry)
	}
	w.entries = kept
	w.rebuildIndexLocked()
}

func (w *WarmMemory) evictOldestLocked() {
	if len(w.entries) == 0 {
		return
	}
	w.removeAtLocked(0)
}

func (w *WarmMemory) removeAtLocked(idx int) {
	if idx < 0 || idx >= len(w.entries) {
		return
	}
	w.entries = append(w.entries[:idx], w.entries[idx+1:]...)
	w.rebuildIndexLocked()
}

func (w *WarmMemory) rebuildIndexLocked() {
	w.index = make(map[string]int, max(w.capacity, len(w.entries)))
	for i := range w.entries {
		w.index[w.entries[i].ID] = i
	}
}

func (w *WarmMemory) touchLocked(id string) {
	idx, ok := w.index[id]
	if !ok {
		return
	}
	if idx == len(w.entries)-1 {
		return
	}

	entry := w.entries[idx]
	copy(w.entries[idx:], w.entries[idx+1:])
	w.entries[len(w.entries)-1] = entry
	w.rebuildIndexLocked()
}

func (w *WarmMemory) bumpAccessLocked(id string) {
	idx, ok := w.index[id]
	if !ok {
		return
	}
	w.entries[idx].AccessCount++
	w.touchLocked(id)
}

func warmEntryScore(entry MemoryEntry, query MemoryQuery, now time.Time) float64 {
	accessScore := float64(max(entry.AccessCount, 1))
	importance := entry.Importance
	if importance <= 0 {
		importance = 0.1
	}
	freshness := 1.0
	if query.UseTimeDecay || len(query.Keywords) > 0 || strings.TrimSpace(query.SemanticQuery) != "" {
		freshness = warmFreshnessScore(entry, now)
	}
	relevance := semanticRelevanceScore(entry, query)
	score := accessScore * importance * freshness * relevance
	if score <= 0 {
		return 0
	}
	return score
}

func warmFreshnessScore(entry MemoryEntry, now time.Time) float64 {
	ageHours := now.Sub(entry.Timestamp).Hours()
	if ageHours < 0 {
		ageHours = 0
	}
	decay := entry.DecayFactor
	if decay <= 0 {
		decay = 1 / (1 + ageHours)
	}
	return decay
}

func semanticRelevanceScore(entry MemoryEntry, query MemoryQuery) float64 {
	if len(query.Keywords) == 0 && strings.TrimSpace(query.SemanticQuery) == "" {
		return 1
	}

	content := strings.ToLower(entry.Content)
	if content == "" {
		return 0.1
	}

	keywords := make([]string, 0, len(query.Keywords)+1)
	keywords = append(keywords, query.Keywords...)
	if semantic := strings.TrimSpace(query.SemanticQuery); semantic != "" {
		keywords = append(keywords, strings.Fields(semantic)...)
	}

	matched := 0
	valid := 0
	for _, keyword := range keywords {
		k := strings.ToLower(strings.TrimSpace(keyword))
		if k == "" {
			continue
		}
		valid++
		if strings.Contains(content, k) {
			matched++
		}
	}
	if valid == 0 {
		return 1
	}
	ratio := float64(matched) / float64(valid)
	return 0.1 + ratio
}

func calculateImportance(entry MemoryEntry) float64 {
	role, _ := entry.Metadata["role"].(string)
	base := 0.4
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "tool":
		base = 0.8
	case "user":
		base = 0.6
	case "assistant":
		base = 0.4
	}

	if strings.TrimSpace(role) == "" && strings.Contains(entry.ID, "tool") {
		base = 0.8
	}
	length := len(strings.TrimSpace(entry.Content))
	penalty := 0.0
	switch {
	case length > 4000:
		penalty = 0.25
	case length > 2500:
		penalty = 0.2
	case length > 1200:
		penalty = 0.12
	case length > 600:
		penalty = 0.06
	}
	return clamp01(base - penalty)
}

func defaultEntryExpiry(entry MemoryEntry, now time.Time, baseTTL time.Duration) time.Time {
	ttl := baseTTL
	if ttl <= 0 {
		ttl = defaultWarmTTL
	}
	if entry.Importance >= 0.8 && ttl < 7*24*time.Hour {
		ttl = 7 * 24 * time.Hour
	}
	start := entry.Timestamp
	if start.IsZero() || start.After(now) {
		start = now
	}
	return start.Add(ttl).UTC()
}

func isEntryExpired(entry MemoryEntry, now time.Time, baseTTL time.Duration) bool {
	if !entry.ExpiresAt.IsZero() {
		return now.After(entry.ExpiresAt)
	}
	if entry.Timestamp.IsZero() {
		return false
	}
	ttl := baseTTL
	if ttl <= 0 {
		ttl = defaultWarmTTL
	}
	return now.Sub(entry.Timestamp) > ttl
}

func clamp01(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func max(a int, b int) int {
	if a >= b {
		return a
	}
	return b
}
