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
	Importance  float64        `json:"importance,omitempty"`
	ExpiresAt   time.Time      `json:"expires_at,omitempty"`
	RelatedTo   []string       `json:"related_to,omitempty"`
	// TODO(memory): 仅做字段透传，尚未接入向量索引/召回。
	EmbeddingID string `json:"embedding_id,omitempty"`

	// Deprecated: 预留字段，当前 API 不会写入该值，仅兼容历史/手工注入场景。
	DecayFactor float64 `json:"decay_factor,omitempty"`
	// Deprecated: 预留字段，当前 API 不会写入该值，仅兼容历史/手工注入场景。
	Priority int `json:"priority,omitempty"`
}

// Deprecated: 早期预留的抽象层接口，运行时已直接依赖具体实现。
// TODO(memory): 确认外部无依赖后移除该接口。
type MemoryLayer interface {
	Store(entry MemoryEntry) error
	Retrieve(query MemoryQuery) ([]MemoryEntry, error)
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
	TimeRange       *TimeRange     `json:"time_range,omitempty"`
	Limit           int            `json:"limit,omitempty"`
	Keywords        []string       `json:"keywords,omitempty"`
	Metadata        map[string]any `json:"metadata,omitempty"`
	IncludeMarkdown bool           `json:"include_markdown,omitempty"`
	SemanticQuery   string         `json:"semantic_query,omitempty"`

	// Deprecated: 预留字段，当前 API 未暴露该能力，仅兼容手工构造查询。
	UseTimeDecay bool `json:"use_time_decay,omitempty"`
	// Deprecated: 预留字段，当前 API 未暴露该能力，仅兼容手工构造查询。
	MinPriority int `json:"min_priority,omitempty"`
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
	if out.Importance < 0 {
		out.Importance = 0
	}
	if out.Importance > 1 {
		out.Importance = 1
	}
	if !out.ExpiresAt.IsZero() {
		out.ExpiresAt = out.ExpiresAt.UTC()
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
