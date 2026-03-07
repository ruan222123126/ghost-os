package memory

import (
	"sort"
	"strings"
	"time"
)

const (
	MemoryAnchorPreference = "preference"
	MemoryAnchorAvoidance  = "avoidance"
	MemoryAnchorEmotion    = "emotion"
	MemoryAnchorConstraint = "constraint"
	MemoryAnchorIdentity   = "identity"
)

// MemoryAnchor 表示可长期影响行为与召回排序的结构化锚点。
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

func normalizeAnchor(anchor MemoryAnchor) MemoryAnchor {
	out := anchor
	out.Type = strings.ToLower(strings.TrimSpace(out.Type))
	out.Key = strings.TrimSpace(out.Key)
	out.Value = strings.TrimSpace(out.Value)
	out.Reason = strings.TrimSpace(out.Reason)
	out.SessionID = strings.TrimSpace(out.SessionID)
	out.Weight = clamp01(out.Weight)
	if !out.DetectedAt.IsZero() {
		out.DetectedAt = out.DetectedAt.UTC()
	}
	if !out.ExpiresAt.IsZero() {
		out.ExpiresAt = out.ExpiresAt.UTC()
	}
	return out
}

func cloneAnchor(anchor MemoryAnchor) MemoryAnchor {
	return anchor
}

func cloneAnchors(anchors []MemoryAnchor) []MemoryAnchor {
	if len(anchors) == 0 {
		return nil
	}
	out := make([]MemoryAnchor, len(anchors))
	copy(out, anchors)
	return out
}

func normalizeAnchors(anchors []MemoryAnchor) []MemoryAnchor {
	if len(anchors) == 0 {
		return nil
	}

	merged := make(map[string]MemoryAnchor, len(anchors))
	for _, anchor := range anchors {
		normalized := normalizeAnchor(anchor)
		if normalized.Type == "" || normalized.Value == "" {
			continue
		}
		key := anchorFingerprint(normalized)
		existing, ok := merged[key]
		if !ok {
			merged[key] = normalized
			continue
		}
		merged[key] = mergeAnchor(existing, normalized)
	}
	if len(merged) == 0 {
		return nil
	}

	out := make([]MemoryAnchor, 0, len(merged))
	for _, anchor := range merged {
		out = append(out, anchor)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Weight != out[j].Weight {
			return out[i].Weight > out[j].Weight
		}
		if out[i].Type != out[j].Type {
			return out[i].Type < out[j].Type
		}
		if out[i].Key != out[j].Key {
			return out[i].Key < out[j].Key
		}
		return out[i].Value < out[j].Value
	})
	return out
}

func filterAnchorsByWeight(anchors []MemoryAnchor, minWeight float64) []MemoryAnchor {
	if len(anchors) == 0 {
		return nil
	}
	threshold := clamp01(minWeight)
	if threshold <= 0 {
		return normalizeAnchors(anchors)
	}
	out := make([]MemoryAnchor, 0, len(anchors))
	for _, anchor := range anchors {
		normalized := normalizeAnchor(anchor)
		if normalized.Weight < threshold {
			continue
		}
		out = append(out, normalized)
	}
	return normalizeAnchors(out)
}

func activeAnchors(anchors []MemoryAnchor, now time.Time) []MemoryAnchor {
	if len(anchors) == 0 {
		return nil
	}
	current := now.UTC()
	out := make([]MemoryAnchor, 0, len(anchors))
	for _, anchor := range anchors {
		normalized := normalizeAnchor(anchor)
		if normalized.Type == "" || normalized.Value == "" {
			continue
		}
		if !normalized.ExpiresAt.IsZero() && current.After(normalized.ExpiresAt) {
			continue
		}
		out = append(out, normalized)
	}
	return out
}

func anchorFingerprint(anchor MemoryAnchor) string {
	return strings.Join([]string{
		strings.ToLower(strings.TrimSpace(anchor.Type)),
		strings.ToLower(strings.TrimSpace(anchor.Key)),
		strings.ToLower(strings.TrimSpace(anchor.Value)),
		strings.ToLower(strings.TrimSpace(anchor.SessionID)),
	}, "|")
}

func mergeAnchor(primary MemoryAnchor, secondary MemoryAnchor) MemoryAnchor {
	out := primary
	if secondary.Weight > out.Weight {
		out.Weight = secondary.Weight
	}
	if strings.TrimSpace(out.Reason) == "" && strings.TrimSpace(secondary.Reason) != "" {
		out.Reason = secondary.Reason
	}
	if out.DetectedAt.IsZero() || (!secondary.DetectedAt.IsZero() && secondary.DetectedAt.Before(out.DetectedAt)) {
		out.DetectedAt = secondary.DetectedAt
	}
	if secondary.ExpiresAt.After(out.ExpiresAt) {
		out.ExpiresAt = secondary.ExpiresAt
	}
	if strings.TrimSpace(out.SessionID) == "" && strings.TrimSpace(secondary.SessionID) != "" {
		out.SessionID = secondary.SessionID
	}
	return normalizeAnchor(out)
}

func persistentAnchorType(anchorType string) bool {
	switch strings.ToLower(strings.TrimSpace(anchorType)) {
	case MemoryAnchorAvoidance, MemoryAnchorConstraint, MemoryAnchorIdentity:
		return true
	default:
		return false
	}
}
