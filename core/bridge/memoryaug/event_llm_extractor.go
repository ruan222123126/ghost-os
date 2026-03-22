package memoryaug

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"ghost-os/bridge/llm"
)

type EventLLMExtractor struct {
	completer llm.Completer
}

func NewEventLLMExtractor(completer llm.Completer) *EventLLMExtractor {
	return &EventLLMExtractor{completer: completer}
}

func (e *EventLLMExtractor) Extract(ctx context.Context, input EventExtractInput) (EventExtractOutput, error) {
	if e == nil || e.completer == nil {
		return EventExtractOutput{}, fmt.Errorf("event memory extractor completer is not configured")
	}
	resp, err := e.completer.Complete(ctx, llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: llm.RoleSystem, Text: eventExtractorSystemPrompt()},
			{Role: llm.RoleUser, Text: eventExtractorUserPrompt(input)},
		},
	})
	if err != nil {
		return EventExtractOutput{}, err
	}
	raw := strings.TrimSpace(resp.Message.Text)
	items, err := parseEventExtractorItems(raw)
	if err != nil {
		return EventExtractOutput{}, err
	}
	return EventExtractOutput{RawJSON: raw, Items: items}, nil
}

func eventExtractorSystemPrompt() string {
	return `You extract durable event-scoped memories for Ghost-OS.

Rules:
- The current event is task/goal scoped, not step scoped.
- Learn only stable information that will help future turns on the same event.
- Prefer workflow, fact, profile, or event-local preference memories.
- Ignore greetings, tool noise, transient outputs, and one-off execution details.
- Do not emit global user preferences such as reply_language, response_style, or approval_style.
- Keep summary short and stable.
- Return JSON only.

Schema:
{"items":[{"memory_type":"preference|workflow|fact|profile","memory_key":"optional_stable_key","summary":"...","content":"...","confidence":0.0,"supersedes_ids":["..."],"reason":"..."}]}`
}

func eventExtractorUserPrompt(input EventExtractInput) string {
	payload := map[string]any{
		"session_id":       input.SessionID,
		"primary_event_id": input.PrimaryEventID,
		"active_event_ids": input.ActiveEventIDs,
		"transcript":       input.Transcript,
		"existing":         input.Existing,
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return `{"error":"marshal event extractor payload failed"}`
	}
	return string(encoded)
}

func parseEventExtractorItems(raw string) ([]EventMemoryCandidate, error) {
	var parsed struct {
		Items []EventMemoryCandidate `json:"items"`
	}
	if err := json.Unmarshal([]byte(extractJSONObject(raw)), &parsed); err != nil {
		return nil, fmt.Errorf("parse event memory extractor output: %w", err)
	}
	return parsed.Items, nil
}
