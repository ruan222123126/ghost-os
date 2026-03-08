package indexer

import (
	"sort"
	"strings"
	"time"
)

// SessionManifest 保存物理 month bucket 内的逻辑 session 子桶摘要。
type SessionManifest struct {
	SessionID      string    `json:"session_id,omitempty"`
	MinOccurredAt  time.Time `json:"min_occurred_at,omitempty"`
	MaxOccurredAt  time.Time `json:"max_occurred_at,omitempty"`
	FirstOffset    int64     `json:"first_offset,omitempty"`
	LastOffset     int64     `json:"last_offset,omitempty"`
	Count          int       `json:"count,omitempty"`
	TraceCount     int       `json:"trace_count,omitempty"`
	UserTurns      int       `json:"user_turns,omitempty"`
	AssistantTurns int       `json:"assistant_turns,omitempty"`
	Kinds          []string  `json:"kinds,omitempty"`
	TopTerms       []string  `json:"top_terms,omitempty"`
}

func NormalizeSessionManifest(manifest SessionManifest) SessionManifest {
	out := manifest
	out.SessionID = strings.TrimSpace(out.SessionID)
	out.MinOccurredAt = out.MinOccurredAt.UTC()
	out.MaxOccurredAt = out.MaxOccurredAt.UTC()
	if out.Count < 0 {
		out.Count = 0
	}
	if out.TraceCount < 0 {
		out.TraceCount = 0
	}
	if out.UserTurns < 0 {
		out.UserTurns = 0
	}
	if out.AssistantTurns < 0 {
		out.AssistantTurns = 0
	}
	out.Kinds = uniqueSessionStrings(out.Kinds)
	out.TopTerms = uniqueSessionStrings(out.TopTerms)
	return out
}

func NormalizeSessionManifests(manifests []SessionManifest) []SessionManifest {
	if len(manifests) == 0 {
		return nil
	}
	out := make([]SessionManifest, 0, len(manifests))
	for _, manifest := range manifests {
		normalized := NormalizeSessionManifest(manifest)
		if normalized.SessionID == "" {
			continue
		}
		out = append(out, normalized)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].SessionID != out[j].SessionID {
			return out[i].SessionID < out[j].SessionID
		}
		if !out[i].MaxOccurredAt.Equal(out[j].MaxOccurredAt) {
			return out[i].MaxOccurredAt.Before(out[j].MaxOccurredAt)
		}
		return out[i].FirstOffset < out[j].FirstOffset
	})
	return out
}

func uniqueSessionStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}
