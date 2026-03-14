package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"ghost-os/bridge/memorystore"
)

type MemoryLearnedListTool struct {
	store *memorystore.Store
}

type memoryLearnedListArgs struct {
	ScopeType  *string `json:"scope_type,omitempty"`
	ScopeID    *string `json:"scope_id,omitempty"`
	MemoryType *string `json:"memory_type,omitempty"`
	Status     *string `json:"status,omitempty"`
	Query      *string `json:"query,omitempty"`
	Limit      *int    `json:"limit,omitempty"`
	Offset     *int    `json:"offset,omitempty"`
}

type memoryLearnedListResult struct {
	Items  []memorystore.MemoryEntry `json:"items"`
	Total  int                       `json:"total"`
	Limit  int                       `json:"limit"`
	Offset int                       `json:"offset"`
}

func NewMemoryLearnedListTool(store *memorystore.Store) *MemoryLearnedListTool {
	return &MemoryLearnedListTool{store: store}
}

func (MemoryLearnedListTool) Name() string {
	return "memory_learned_list"
}

func (MemoryLearnedListTool) Description() string {
	return "Read-only listing of automatically learned memories with optional scope, type, status, and query filters."
}

func (MemoryLearnedListTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type":"object",
		"properties":{
			"scope_type":{"type":"string","enum":["user","session"]},
			"scope_id":{"type":"string"},
			"memory_type":{"type":"string","enum":["profile","preference","workflow","fact"]},
			"status":{"type":"string","enum":["active","superseded","deleted"]},
			"query":{"type":"string"},
			"limit":{"type":"integer","minimum":1},
			"offset":{"type":"integer","minimum":0}
		},
		"additionalProperties":false
	}`)
}

func (t *MemoryLearnedListTool) Execute(ctx context.Context, argsJSON json.RawMessage, _ string) (string, error) {
	if t == nil || t.store == nil {
		return "", fmt.Errorf("memory store is not configured")
	}
	var args memoryLearnedListArgs
	if err := decodeLearnedListArgs(argsJSON, &args); err != nil {
		return "", err
	}
	limit, offset, err := resolveMemoryPaging(args.Limit, args.Offset)
	if err != nil {
		return "", err
	}
	items, total, err := t.store.ListLearned(ctx, memorystore.LearnedListFilter{
		ScopeType:  optionalStringValue(args.ScopeType),
		ScopeID:    optionalStringValue(args.ScopeID),
		MemoryType: optionalStringValue(args.MemoryType),
		Statuses:   resolveLearnedStatuses(args.Status),
		Query:      optionalStringValue(args.Query),
		Limit:      limit,
		Offset:     offset,
	})
	if err != nil {
		return "", err
	}
	return marshalMemoryManageOutput(memoryLearnedListResult{
		Items:  items,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	})
}

func decodeLearnedListArgs(raw json.RawMessage, target *memoryLearnedListArgs) error {
	return decodeJSONArgs(raw, target)
}

func resolveLearnedStatuses(status *string) []string {
	value := strings.TrimSpace(optionalStringValue(status))
	if value == "" {
		return []string{memorystore.MemoryStatusActive, memorystore.MemoryStatusSuperseded}
	}
	return []string{value}
}
