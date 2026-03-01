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
	ID          string         `json:"id"`
	Content     string         `json:"content"`
	Type        MemoryType     `json:"type"`
	Timestamp   time.Time      `json:"timestamp"`
	AccessCount int            `json:"access_count"`
	Metadata    map[string]any `json:"metadata,omitempty"`

	// 预留给后续时间衰减与优先级增强。
	DecayFactor float64 `json:"decay_factor,omitempty"`
	Priority    int     `json:"priority,omitempty"`
}

// MemoryLayer 定义三层存储统一操作接口。
type MemoryLayer interface {
	Store(entry MemoryEntry) error
	Retrieve(query MemoryQuery) ([]MemoryEntry, error)
	Delete(id string) error
	Clear() error
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
	TimeRange *TimeRange     `json:"time_range,omitempty"`
	Limit     int            `json:"limit,omitempty"`
	Keywords  []string       `json:"keywords,omitempty"`
	Metadata  map[string]any `json:"metadata,omitempty"`

	// 预留字段：后续支持时间衰减与优先级过滤。
	UseTimeDecay bool `json:"use_time_decay,omitempty"`
	MinPriority  int  `json:"min_priority,omitempty"`
}

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

func entryMatchesQuery(entry MemoryEntry, query MemoryQuery) bool {
	if query.TimeRange != nil && !query.TimeRange.Contains(entry.Timestamp) {
		return false
	}
	if query.MinPriority > 0 && entry.Priority < query.MinPriority {
		return false
	}
	if len(query.Keywords) > 0 {
		content := strings.ToLower(entry.Content)
		matched := false
		for _, keyword := range query.Keywords {
			k := strings.ToLower(strings.TrimSpace(keyword))
			if k == "" {
				continue
			}
			if strings.Contains(content, k) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
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
