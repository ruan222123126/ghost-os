package memory

import (
	"testing"
	"time"

	"ghost-os/bridge/llm"
)

func TestExtractRuleAnchorsDetectsPreferenceAndAvoidance(t *testing.T) {
	now := time.Now().UTC()
	anchors := extractRuleAnchors([]llm.Message{
		{Role: llm.RoleUser, Text: "I prefer Go for backend services."},
		{Role: llm.RoleUser, Text: "不要动生产库。"},
	}, "session-anchor", now)

	if len(anchors) < 2 {
		t.Fatalf("expected at least 2 anchors, got %d", len(anchors))
	}

	var foundPreference bool
	var foundAvoidance bool
	for _, anchor := range anchors {
		switch anchor.Type {
		case MemoryAnchorPreference:
			if anchor.Key == "language" && anchor.Value == "Go" {
				foundPreference = true
			}
		case MemoryAnchorAvoidance, MemoryAnchorConstraint:
			if anchor.Key == "environment" || anchor.Key == "database" {
				foundAvoidance = true
			}
		}
	}
	if !foundPreference {
		t.Fatalf("expected Go preference anchor, got %+v", anchors)
	}
	if !foundAvoidance {
		t.Fatalf("expected production avoidance anchor, got %+v", anchors)
	}
}
