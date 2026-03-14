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
	return `You extract durable slot memories for Ghost-OS.

Rules:
- Learn only durable preferences or workflows that match one of the provided known_slots.
- Ignore greetings, low-information confirmations, tool noise, one-off tasks, and temporary execution results.
- Never emit memories that duplicate explicit memories already provided.
- Only emit memory_key values that exactly match a known_slots key.
- Fill value with the normalized slot value. Use enum values exactly when the slot defines them.
- Keep summary and content stable for the slot. If the transcript does not match any known slot, return {"items":[]}.
- Return JSON only.

Schema:
{"items":[{"memory_type":"preference|workflow","memory_key":"known_slot_key","value":"normalized slot value","summary":"...","content":"...","scope_type":"user|session","scope_id":"...","confidence":0.0,"supersedes_ids":["..."],"reason":"..."}]}`
}

func extractorUserPrompt(input ExtractInput) string {
	payload := map[string]any{
		"session_id":        input.SessionID,
		"user_scope_id":     input.UserScopeID,
		"known_slots":       extractorKnownSlots(),
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
