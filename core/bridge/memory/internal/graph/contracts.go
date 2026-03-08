package graph

import (
	"strings"
	"time"

	"ghost-os/bridge/agent"
	"ghost-os/bridge/llm"
)

type MemoryType string

const (
	MemoryTypeMessage   MemoryType = "message"
	MemoryTypeSummary   MemoryType = "summary"
	MemoryTypeKnowledge MemoryType = "knowledge"
)

type MemoryAnchor struct {
	Type       string    `json:"type,omitempty" yaml:"type,omitempty"`
	Key        string    `json:"key,omitempty" yaml:"key,omitempty"`
	Value      string    `json:"value,omitempty" yaml:"value,omitempty"`
	Weight     float64   `json:"weight,omitempty" yaml:"weight,omitempty"`
	Reason     string    `json:"reason,omitempty" yaml:"reason,omitempty"`
	DetectedAt time.Time `json:"detected_at,omitempty" yaml:"detected_at,omitempty"`
	ExpiresAt  time.Time `json:"expires_at,omitempty" yaml:"expires_at,omitempty"`
	SessionID  string    `json:"session_id,omitempty" yaml:"session_id,omitempty"`
}

type MemoryEntry struct {
	ID         string         `json:"id"`
	Content    string         `json:"content"`
	Type       MemoryType     `json:"type"`
	Timestamp  time.Time      `json:"timestamp"`
	Metadata   map[string]any `json:"metadata,omitempty"`
	Importance float64        `json:"importance,omitempty"`
	RelatedTo  []string       `json:"related_to,omitempty"`
	Source     string         `json:"source,omitempty"`
	Summary    string         `json:"summary,omitempty"`
	Confidence float64        `json:"confidence,omitempty"`
}

type TimeRange struct {
	Start time.Time `json:"start,omitempty"`
	End   time.Time `json:"end,omitempty"`
}

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

type MemoryQuery struct {
	TimeRange       *TimeRange     `json:"time_range,omitempty"`
	Limit           int            `json:"limit,omitempty"`
	Keywords        []string       `json:"keywords,omitempty"`
	Metadata        map[string]any `json:"metadata,omitempty"`
	IncludeGraph    bool           `json:"include_graph,omitempty"`
	SemanticQuery   string         `json:"semantic_query,omitempty"`
	GraphHops       int            `json:"graph_hops,omitempty"`
	GraphPredicates []string       `json:"graph_predicates,omitempty"`
	GraphDebug      bool           `json:"graph_debug,omitempty"`
}

type SessionScope struct {
	SessionID string
	History   *agent.History
}

type MemoryConfig struct {
	GraphEnabled          bool
	GraphPath             string
	GraphExtractOnArchive bool
	GraphExtractOnEvolve  bool
	GraphMaxHops          int
	GraphMaxHits          int
	GraphMinConfidence    float64
	GraphNamespace        string
	GraphDebugEnabled     bool
	EvolutionUseWorker    bool
}

type Summarizer interface {
	Summarize(messages []llm.Message) (string, error)
}

type GraphFactExtractor interface {
	ExtractGraphFacts(messages []llm.Message) ([]GraphFact, error)
}

type MarkdownNode struct {
	ID         string         `yaml:"id"`
	Importance float64        `yaml:"importance"`
	CreatedAt  time.Time      `yaml:"created_at"`
	RelatedTo  []string       `yaml:"related_to,omitempty"`
	Tags       []string       `yaml:"tags,omitempty"`
	SessionID  string         `yaml:"session_id,omitempty"`
	Summary    string         `yaml:"summary,omitempty"`
	Anchors    []MemoryAnchor `yaml:"anchors,omitempty"`
	SourceIDs  []string       `yaml:"source_ids,omitempty"`
	Confidence float64        `yaml:"confidence,omitempty"`
	LastSeenAt time.Time      `yaml:"last_seen_at,omitempty"`
	Content    string         `yaml:"-"`
}

type ColdArchive struct {
	SessionID  string        `json:"session_id"`
	ArchivedAt time.Time     `json:"archived_at"`
	Messages   []llm.Message `json:"messages"`
}

type ColdStore interface {
	ListArchives(timeRange *TimeRange) ([]ColdArchive, error)
	ListMarkdownNodes() ([]string, error)
	LoadMarkdownNode(id string) (MarkdownNode, error)
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
	out.Source = strings.TrimSpace(out.Source)
	out.Summary = strings.TrimSpace(out.Summary)
	out.Confidence = clamp01(out.Confidence)
	out.RelatedTo = uniqueStrings(out.RelatedTo)
	return out
}

func messageToContent(msg llm.Message) string {
	if text := strings.TrimSpace(msg.Text); text != "" {
		return text
	}
	return ""
}

func evolutionMessages(entries []MemoryEntry) []llm.Message {
	messages := make([]llm.Message, 0, len(entries))
	for _, entry := range entries {
		content := strings.TrimSpace(firstNonEmpty(entry.Content, entry.Summary))
		if content == "" {
			continue
		}
		messages = append(messages, llm.Message{Role: llm.RoleAssistant, Text: content})
	}
	return messages
}

func latestEntryTimestamp(entries []MemoryEntry) time.Time {
	latest := time.Time{}
	for _, entry := range entries {
		ts := entry.Timestamp.UTC()
		if ts.After(latest) {
			latest = ts
		}
	}
	if latest.IsZero() {
		return time.Now().UTC()
	}
	return latest
}

func markdownNodeTimestamp(node MarkdownNode) time.Time {
	if !node.LastSeenAt.IsZero() {
		return node.LastSeenAt.UTC()
	}
	if !node.CreatedAt.IsZero() {
		return node.CreatedAt.UTC()
	}
	return time.Now().UTC()
}
