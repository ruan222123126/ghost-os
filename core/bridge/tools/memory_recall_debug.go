package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"ghost-os/bridge/memoryaug"
)

type MemoryRecallDebugTool struct {
	service     memoryaug.RecallService
	userScopeID string
}

type memoryRecallDebugArgs struct {
	Query     *string `json:"query,omitempty"`
	SessionID *string `json:"session_id,omitempty"`
}

type memoryRecallDebugResult struct {
	Query       string                 `json:"query"`
	SessionID   string                 `json:"session_id"`
	UserScopeID string                 `json:"user_scope_id"`
	Items       []memoryaug.RecallItem `json:"items"`
	PromptBlock string                 `json:"prompt_block"`
}

func NewMemoryRecallDebugTool(service memoryaug.RecallService, userScopeID string) *MemoryRecallDebugTool {
	return &MemoryRecallDebugTool{
		service:     service,
		userScopeID: strings.TrimSpace(userScopeID),
	}
}

func (MemoryRecallDebugTool) Name() string {
	return "memory_recall_debug"
}

func (MemoryRecallDebugTool) Description() string {
	return "Read-only debug view for automatic memory recall hits, ranking reasons, and the injected prompt block."
}

func (MemoryRecallDebugTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type":"object",
		"properties":{
			"query":{"type":"string"},
			"session_id":{"type":"string"}
		},
		"required":["query"],
		"additionalProperties":false
	}`)
}

func (t *MemoryRecallDebugTool) Execute(ctx context.Context, argsJSON json.RawMessage, _ string) (string, error) {
	if t == nil || t.service == nil {
		return "", fmt.Errorf("memory recall service is not configured")
	}
	var args memoryRecallDebugArgs
	if err := decodeRecallDebugArgs(argsJSON, &args); err != nil {
		return "", err
	}
	query := strings.TrimSpace(optionalStringValue(args.Query))
	if query == "" {
		return "", fmt.Errorf("query is required")
	}
	sessionID := strings.TrimSpace(optionalStringValue(args.SessionID))
	if sessionID == "" {
		if sess := SessionFromContext(ctx); sess != nil {
			sessionID = strings.TrimSpace(sess.ID)
		}
	}
	items, err := t.service.Recall(ctx, memoryaug.RecallInput{
		SessionID: sessionID,
		UserScope: t.userScopeID,
		Query:     query,
	})
	if err != nil {
		return "", err
	}
	return marshalMemoryManageOutput(memoryRecallDebugResult{
		Query:       query,
		SessionID:   sessionID,
		UserScopeID: t.userScopeID,
		Items:       items,
		PromptBlock: memoryaug.FormatPromptBlock(items),
	})
}

func decodeRecallDebugArgs(raw json.RawMessage, target *memoryRecallDebugArgs) error {
	return decodeJSONArgs(raw, target)
}
