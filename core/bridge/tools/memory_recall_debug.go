package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"ghost-os/bridge/memoryaug"
	"ghost-os/bridge/session"
)

type MemoryRecallDebugTool struct {
	planner     memoryaug.IntentPlanner
	service     memoryaug.RecallService
	userScopeID string
}

type memoryRecallDebugArgs struct {
	Query     *string `json:"query,omitempty"`
	SessionID *string `json:"session_id,omitempty"`
}

type memoryRecallDebugResult struct {
	Query           string                        `json:"query"`
	SessionID       string                        `json:"session_id"`
	UserScopeID     string                        `json:"user_scope_id"`
	PlannerDecision memoryaug.PlannerDecision     `json:"planner_decision"`
	ActiveNodes     []memoryRecallDebugActiveNode `json:"active_nodes"`
	PromptBlock     string                        `json:"prompt_block"`
}

type memoryRecallDebugActiveNode struct {
	EventID string `json:"event_id"`
	Title   string `json:"title"`
	Role    string `json:"role"`
	Reason  string `json:"reason"`
}

func NewMemoryRecallDebugTool(
	planner memoryaug.IntentPlanner,
	service memoryaug.RecallService,
	userScopeID string,
) *MemoryRecallDebugTool {
	return &MemoryRecallDebugTool{
		planner:     planner,
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
	if t == nil || t.planner == nil || t.service == nil {
		return "", fmt.Errorf("memory planner or recall service is not configured")
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
	plannerDecision, err := t.planner.Plan(ctx, memoryaug.PlannerInput{
		SessionID:      sessionID,
		UserScope:      t.userScopeID,
		UserMessage:    query,
		RecentMessages: debugPlannerMessages(SessionFromContext(ctx)),
	})
	if err != nil {
		return "", err
	}
	recall, err := t.service.Recall(ctx, memoryaug.RecallInput{
		SessionID:      sessionID,
		PrimaryEventID: plannerDecision.PrimaryEvent.EventID,
		ActiveEventIDs: plannerDecision.RecallPlan.EventIDs,
		FocusText:      query,
		RecallPlan:     plannerDecision.RecallPlan,
	})
	if err != nil {
		return "", err
	}
	return marshalMemoryManageOutput(memoryRecallDebugResult{
		Query:           query,
		SessionID:       sessionID,
		UserScopeID:     t.userScopeID,
		PlannerDecision: plannerDecision,
		ActiveNodes:     buildDebugActiveNodes(plannerDecision, recall),
		PromptBlock:     recall.PromptBlock,
	})
}

func decodeRecallDebugArgs(raw json.RawMessage, target *memoryRecallDebugArgs) error {
	return decodeJSONArgs(raw, target)
}

func debugPlannerMessages(sess *session.Session) []memoryaug.TurnMessage {
	if sess == nil {
		return nil
	}
	out := make([]memoryaug.TurnMessage, 0, 3)
	for i := len(sess.Messages) - 1; i >= 0 && len(out) < 3; i-- {
		message := sess.Messages[i]
		if message.Role != "user" && message.Role != "assistant" {
			continue
		}
		text := strings.TrimSpace(message.Text)
		if text == "" {
			continue
		}
		out = append([]memoryaug.TurnMessage{{
			Role: string(message.Role),
			Text: text,
		}}, out...)
	}
	return out
}

func buildDebugActiveNodes(
	decision memoryaug.PlannerDecision,
	output memoryaug.RecallOutput,
) []memoryRecallDebugActiveNode {
	nodes := make([]memoryRecallDebugActiveNode, 0, 3)
	if output.PrimaryEvent != nil {
		nodes = append(nodes, memoryRecallDebugActiveNode{
			EventID: output.PrimaryEvent.Event.ID,
			Title:   output.PrimaryEvent.Event.Title,
			Role:    output.PrimaryEvent.Role,
			Reason:  decision.PrimaryEvent.Reason,
		})
	}
	for idx, item := range output.AdjacentEvents {
		reason := item.Reason
		if idx < len(decision.AdjacentEvents) && strings.TrimSpace(decision.AdjacentEvents[idx].Reason) != "" {
			reason = decision.AdjacentEvents[idx].Reason
		}
		nodes = append(nodes, memoryRecallDebugActiveNode{
			EventID: item.Event.ID,
			Title:   item.Event.Title,
			Role:    item.Role,
			Reason:  reason,
		})
	}
	return nodes
}
