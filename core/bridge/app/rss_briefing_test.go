package app

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"ghost-os/bridge/llm"
)

func TestRSSInboxServiceBuildBriefingUsesAIHighlights(t *testing.T) {
	inboxStore, err := NewRSSInboxStore(filepath.Join(t.TempDir(), "inbox.json"))
	if err != nil {
		t.Fatalf("new inbox store: %v", err)
	}
	now := time.Date(2026, 3, 9, 12, 0, 0, 0, time.UTC)
	_, err = inboxStore.SaveItems([]RSSInboxItem{
		{
			FeedID:      "feed-ai",
			FeedURL:     "https://example.com/ai.xml",
			ItemTitle:   "OpenAI launches GPT-6 coding agent",
			AISummary:   "New coding-focused model rollout.",
			Tags:        []string{"ai", "openai", "launch"},
			Importance:  "high",
			PublishedAt: time.Date(2026, 3, 9, 10, 15, 0, 0, time.UTC),
			SavedAt:     time.Date(2026, 3, 9, 10, 20, 0, 0, time.UTC),
			ContentHash: "hash-1",
			DedupeKey:   "feed-ai::1",
		},
		{
			FeedID:      "feed-ai",
			FeedURL:     "https://example.com/ai.xml",
			ItemTitle:   "GPT-6 coding launch expands developer workflow",
			AISummary:   "Companion launch details for the same release.",
			Tags:        []string{"ai", "launch", "coding"},
			Importance:  "normal",
			PublishedAt: time.Date(2026, 3, 9, 11, 0, 0, 0, time.UTC),
			SavedAt:     time.Date(2026, 3, 9, 11, 5, 0, 0, time.UTC),
			ContentHash: "hash-2",
			DedupeKey:   "feed-ai::2",
		},
	})
	if err != nil {
		t.Fatalf("save items: %v", err)
	}

	fakeWorker := &fakeSelectorCompleter{response: &llm.CompletionResponse{
		Message: llm.Message{Role: llm.RoleAssistant, Text: `{
  "title": "AI Daily Brief",
  "summary": "One major AI release stands out in this window.",
  "highlights": [
    {
      "group_id": "rssg_ignored",
      "headline": "ignored invalid group"
    }
  ]
}`},
	}}
	service := &RSSInboxService{
		inboxStore: inboxStore,
		now:        func() time.Time { return now },
		briefingBuilder: &llmRSSBriefingBuilder{
			client:  fakeWorker,
			timeout: time.Second,
			cfg: Config{
				Provider: ProviderConfig{Model: "gpt-4o"},
				Worker:   WorkerConfig{Model: "gpt-4o-mini"},
			},
		},
	}

	result, err := service.BuildBriefing(context.Background(), RSSBriefingQuery{
		WindowHours:     24,
		GroupLimit:      5,
		ItemsPerGroup:   3,
		HighlightsLimit: 3,
		TraceID:         "trace-briefing-1",
	})
	if err != nil {
		t.Fatalf("build briefing returned error: %v", err)
	}
	if result.Title != "AI Daily Brief" {
		t.Fatalf("unexpected title: got %q", result.Title)
	}
	if len(result.Highlights) != 1 {
		t.Fatalf("expected fallback materialized one highlight, got %d", len(result.Highlights))
	}
	if result.Highlights[0].GroupID == "" {
		t.Fatalf("expected highlight group id, got %+v", result.Highlights[0])
	}
	if result.Highlights[0].SourceItemCount != 2 {
		t.Fatalf("unexpected source item count: got %d want %d", result.Highlights[0].SourceItemCount, 2)
	}
	if len(fakeWorker.requests) != 1 {
		t.Fatalf("expected one worker request, got %d", len(fakeWorker.requests))
	}
	prompt := fakeWorker.requests[0].Messages[1].Text
	if !strings.Contains(prompt, `"highlights_limit": 3`) {
		t.Fatalf("expected prompt to include highlights_limit, got %q", prompt)
	}
	if !strings.Contains(prompt, `"group_id"`) {
		t.Fatalf("expected prompt to include group payload, got %q", prompt)
	}
}
