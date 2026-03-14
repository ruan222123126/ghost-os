package memoryaug

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"ghost-os/bridge/llm"
)

type LLMExtractor struct {
	completer llm.Completer
}

func NewLLMExtractor(completer llm.Completer) *LLMExtractor {
	return &LLMExtractor{completer: completer}
}

func (e *LLMExtractor) Extract(ctx context.Context, input ExtractInput) (ExtractOutput, error) {
	if e == nil || e.completer == nil {
		return ExtractOutput{}, fmt.Errorf("memory extractor completer is not configured")
	}
	resp, err := e.completer.Complete(ctx, llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: llm.RoleSystem, Text: extractorSystemPrompt()},
			{Role: llm.RoleUser, Text: extractorUserPrompt(input)},
		},
	})
	if err != nil {
		return ExtractOutput{}, err
	}
	raw := strings.TrimSpace(resp.Message.Text)
	items, err := parseExtractorItems(raw)
	if err != nil {
		return ExtractOutput{}, err
	}
	return ExtractOutput{RawJSON: raw, Items: items}, nil
}

func extractorSystemPrompt() string {
	return `You extract durable conversation memories for Ghost-OS.

Rules:
- Learn only durable profile, preference, workflow, or fact memories.
- Ignore greetings, low-information confirmations, tool noise, one-off tasks, and temporary execution results.
- Prefer session scope unless the information is clearly stable across sessions.
- Never emit memories that duplicate explicit memories already provided.
- Return JSON only.

Schema:
{"items":[{"memory_type":"profile|preference|workflow|fact","summary":"...","content":"...","scope_type":"user|session","scope_id":"...","confidence":0.0,"supersedes_ids":["..."],"reason":"..."}]}`
}

func extractorUserPrompt(input ExtractInput) string {
	payload := map[string]any{
		"session_id":        input.SessionID,
		"user_scope_id":     input.UserScopeID,
		"transcript":        input.Transcript,
		"existing_explicit": input.ExistingExplicit,
		"existing_learned":  input.ExistingLearned,
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return `{"error":"marshal extractor payload failed"}`
	}
	return string(encoded)
}

func parseExtractorItems(raw string) ([]Candidate, error) {
	var parsed struct {
		Items []Candidate `json:"items"`
	}
	if err := json.Unmarshal([]byte(extractJSONObject(raw)), &parsed); err != nil {
		return nil, fmt.Errorf("parse memory extractor output: %w", err)
	}
	return parsed.Items, nil
}

func extractJSONObject(raw string) string {
	trimmed := strings.TrimSpace(raw)
	start := strings.Index(trimmed, "{")
	end := strings.LastIndex(trimmed, "}")
	if start < 0 || end < start {
		return trimmed
	}
	return trimmed[start : end+1]
}
