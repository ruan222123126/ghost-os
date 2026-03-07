package memory

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestMemoryManagerQueryPrefersRecentMatches(t *testing.T) {
	baseDir := t.TempDir()
	manager := NewMemoryManager(MemoryConfig{
		WarmCapacity:          10,
		WarmPath:              filepath.Join(baseDir, "warm.json"),
		ColdBaseDir:           filepath.Join(baseDir, "cold"),
		WarmTTL:               365 * 24 * time.Hour,
		TemporalDecayEnabled:  true,
		TemporalDecayHalfLife: 72 * time.Hour,
	})

	now := time.Now().UTC()
	if err := manager.warm.Store(MemoryEntry{
		ID:         "old-match",
		Content:    "Ghost-OS backend plan for memory ranking",
		Summary:    "Old backend plan for memory ranking",
		Timestamp:  now.Add(-10 * 24 * time.Hour),
		Importance: 0.95,
		Metadata:   map[string]any{"layer": "warm", "role": "assistant"},
	}); err != nil {
		t.Fatalf("store old entry: %v", err)
	}
	if err := manager.warm.Store(MemoryEntry{
		ID:         "recent-match",
		Content:    "Ghost-OS backend plan for memory ranking",
		Summary:    "Recent backend plan for memory ranking",
		Timestamp:  now.Add(-2 * time.Hour),
		Importance: 0.7,
		Metadata:   map[string]any{"layer": "warm", "role": "assistant"},
	}); err != nil {
		t.Fatalf("store recent entry: %v", err)
	}

	entries, err := manager.Query(MemoryQuery{Keywords: []string{"backend", "ranking"}, Limit: 2})
	if err != nil {
		t.Fatalf("query entries: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("unexpected entry count: got %d want 2", len(entries))
	}
	if entries[0].ID != "recent-match" {
		t.Fatalf("expected recent match first, got %q", entries[0].ID)
	}
}

func TestMemoryManagerQueryBoostsAnchorMatches(t *testing.T) {
	baseDir := t.TempDir()
	manager := NewMemoryManager(MemoryConfig{
		WarmCapacity:          10,
		WarmPath:              filepath.Join(baseDir, "warm.json"),
		ColdBaseDir:           filepath.Join(baseDir, "cold"),
		WarmTTL:               365 * 24 * time.Hour,
		AnchorEnabled:         true,
		AnchorMinWeight:       0.65,
		TemporalDecayEnabled:  true,
		TemporalDecayHalfLife: 72 * time.Hour,
	})

	now := time.Now().UTC()
	if err := manager.warm.Store(MemoryEntry{
		ID:        "anchored-go",
		Content:   "Coding notes for services",
		Summary:   "User prefers Go services",
		Timestamp: now.Add(-24 * time.Hour),
		Anchors: []MemoryAnchor{{
			Type:       MemoryAnchorPreference,
			Key:        "language",
			Value:      "Go",
			Weight:     0.92,
			Reason:     "User prefers Go for backend services",
			DetectedAt: now.Add(-24 * time.Hour),
		}},
		Metadata: map[string]any{"layer": "warm", "role": "assistant"},
	}); err != nil {
		t.Fatalf("store anchored entry: %v", err)
	}
	if err := manager.warm.Store(MemoryEntry{
		ID:        "plain-go",
		Content:   "Go notes and snippets",
		Summary:   "Plain Go notes",
		Timestamp: now.Add(-6 * time.Hour),
		Metadata:  map[string]any{"layer": "warm", "role": "assistant"},
	}); err != nil {
		t.Fatalf("store plain entry: %v", err)
	}

	entries, err := manager.Query(MemoryQuery{SemanticQuery: "Go language preference", Limit: 2})
	if err != nil {
		t.Fatalf("query anchored entries: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("unexpected entry count: got %d want 2", len(entries))
	}
	if entries[0].ID != "anchored-go" {
		t.Fatalf("expected anchored entry first, got %q", entries[0].ID)
	}
}

func TestBuildContextWindowUsesCompactSummaryOnly(t *testing.T) {
	baseDir := t.TempDir()
	manager := NewMemoryManager(MemoryConfig{
		WarmCapacity:      10,
		WarmPath:          filepath.Join(baseDir, "warm.json"),
		ColdBaseDir:       filepath.Join(baseDir, "cold"),
		AutoRecallEnabled: true,
		AutoRecallLimit:   1,
	})

	now := time.Now().UTC()
	if err := manager.warm.Store(MemoryEntry{
		ID:        "summary-only",
		Content:   "## Anchors\n- `preference:language` Go (weight=0.92)\n- reason: user prefers Go for backend services",
		Summary:   "User prefers Go for backend services.",
		Timestamp: now,
		Anchors: []MemoryAnchor{{
			Type:       MemoryAnchorPreference,
			Key:        "language",
			Value:      "Go",
			Weight:     0.92,
			Reason:     "User prefers Go for backend services",
			DetectedAt: now,
		}},
		Metadata: map[string]any{"layer": "warm", "session_id": "summary-session", "role": "assistant"},
	}); err != nil {
		t.Fatalf("store summary entry: %v", err)
	}

	window, err := manager.BuildContextWindow("summary-session", "What language should we use?")
	if err != nil {
		t.Fatalf("build context window: %v", err)
	}
	if len(window) != 1 {
		t.Fatalf("unexpected context window size: got %d want 1", len(window))
	}
	if !strings.Contains(window[0].Text, "User prefers Go for backend services") {
		t.Fatalf("expected summary in recall window, got %q", window[0].Text)
	}
	if strings.Contains(window[0].Text, "weight=0.92") || strings.Contains(window[0].Text, "preference:language") {
		t.Fatalf("expected compact recall only, got %q", window[0].Text)
	}
}

func TestMemoryManagerQueryKeepsOldConstraintsAheadOfPlainMatches(t *testing.T) {
	baseDir := t.TempDir()
	manager := NewMemoryManager(MemoryConfig{
		WarmCapacity:          10,
		WarmPath:              filepath.Join(baseDir, "warm.json"),
		ColdBaseDir:           filepath.Join(baseDir, "cold"),
		WarmTTL:               365 * 24 * time.Hour,
		AnchorEnabled:         true,
		AnchorMinWeight:       0.65,
		TemporalDecayEnabled:  true,
		TemporalDecayHalfLife: 72 * time.Hour,
	})

	now := time.Now().UTC()
	if err := manager.warm.Store(MemoryEntry{
		ID:        "old-constraint",
		Content:   "Production database handling rule",
		Summary:   "Do not touch the production database directly.",
		Timestamp: now.Add(-30 * 24 * time.Hour),
		Anchors: []MemoryAnchor{{
			Type:       MemoryAnchorConstraint,
			Key:        "database",
			Value:      "production database",
			Weight:     0.96,
			Reason:     "Do not touch the production database directly.",
			DetectedAt: now.Add(-30 * 24 * time.Hour),
		}},
		Metadata: map[string]any{"layer": "warm", "role": "assistant"},
	}); err != nil {
		t.Fatalf("store constraint entry: %v", err)
	}
	if err := manager.warm.Store(MemoryEntry{
		ID:         "plain-match",
		Content:    "Production database checklist",
		Summary:    "Production database checklist",
		Timestamp:  now.Add(-4 * time.Hour),
		Importance: 0.9,
		Metadata:   map[string]any{"layer": "warm", "role": "assistant"},
	}); err != nil {
		t.Fatalf("store plain match: %v", err)
	}

	entries, err := manager.Query(MemoryQuery{SemanticQuery: "production database", Limit: 2})
	if err != nil {
		t.Fatalf("query constraint entries: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("unexpected entry count: got %d want 2", len(entries))
	}
	if entries[0].ID != "old-constraint" {
		t.Fatalf("expected constraint entry first, got %q", entries[0].ID)
	}
}
