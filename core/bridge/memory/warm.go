package memory

import (
	"encoding/json"
	"fmt"
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
	storePath string
	mu        sync.RWMutex
}

func NewWarmMemory(capacity int, storePath string) *WarmMemory {
	size := capacity
	if size <= 0 {
		size = defaultWarmCapacity
	}
	return &WarmMemory{
		entries:   make([]MemoryEntry, 0, size),
		index:     make(map[string]int, size),
		capacity:  size,
		storePath: resolveMemoryPath(storePath),
	}
}

// Store 写入或更新条目。命中容量上限时按 LRU 淘汰最旧项。
func (w *WarmMemory) Store(entry MemoryEntry) error {
	normalized := normalizeEntry(entry)
	if normalized.ID == "" {
		return fmt.Errorf("memory entry id is required")
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

	w.pruneExpiredLocked(time.Now().UTC())

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
			Score: warmEntryScore(entry, query.UseTimeDecay),
		})
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		if query.UseTimeDecay && candidates[i].Score != candidates[j].Score {
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

func (w *WarmMemory) Clear() error {
	w.mu.Lock()
	w.entries = w.entries[:0]
	w.index = make(map[string]int, w.capacity)
	w.mu.Unlock()

	return w.Persist()
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
	cutoff := now.Add(-defaultWarmTTL)
	kept := w.entries[:0]
	for _, entry := range w.entries {
		if entry.Timestamp.Before(cutoff) {
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

func warmEntryScore(entry MemoryEntry, useTimeDecay bool) float64 {
	accessScore := float64(max(entry.AccessCount, 1))
	if !useTimeDecay {
		return accessScore
	}

	ageHours := time.Since(entry.Timestamp).Hours()
	if ageHours < 0 {
		ageHours = 0
	}
	decay := entry.DecayFactor
	if decay <= 0 {
		decay = 1 / (1 + ageHours)
	}
	return accessScore * decay
}

func max(a int, b int) int {
	if a >= b {
		return a
	}
	return b
}
