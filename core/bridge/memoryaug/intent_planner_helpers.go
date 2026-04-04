package memoryaug

import (
	"encoding/json"
	"fmt"
	"strings"

	"ghost-os/bridge/memorystore"
)

func plannerSystemPrompt() string {
	return `You are the Ghost-OS intent planner for event-graph memory.

Rules:
- Event granularity is task/goal level, never step level.
- Select exactly one primary_event.
- adjacent_events can contain at most 2 events.
- reuse_existing=true when the user is clearly continuing the same task.
- create_new_event=true when the user clearly starts a different task/goal.
- Long-term user preferences are global and must not become events.
- edge_updates may only use edge_type values: parent_of, blocks, related_to, same_goal.
- edge_updates may use "primary" to refer to the current primary event.
- recall_plan.event_ids must only reference the primary event and adjacent events.
- Return JSON only.

Schema:
{"primary_event":{"event_id":"...","reason":"..."},"adjacent_events":[{"event_id":"...","reason":"..."}],"reuse_existing":true,"create_new_event":false,"new_event_title":"","new_event_summary":"","edge_updates":[{"from_event_id":"primary","to_event_id":"...","edge_type":"related_to","confidence":0.8}],"recall_plan":{"event_ids":["..."],"include_node_summary":true,"include_workflow":true,"include_preference":true,"include_fact":true,"include_profile":false,"allow_learning":true}}`
}

func plannerUserPrompt(input PlannerInput, contextData plannerContext) string {
	payload := map[string]any{
		"session_id":       input.SessionID,
		"user_scope_id":    input.UserScope,
		"user_message":     input.UserMessage,
		"recent_messages":  input.RecentMessages,
		"project_root":     input.ProjectRoot,
		"previous_state":   contextData.PreviousState,
		"candidate_events": contextData.Candidates,
		"candidate_edges":  contextData.Edges,
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return `{"error":"marshal planner payload failed"}`
	}
	return string(encoded)
}

func parsePlannerDecision(raw string) (PlannerDecision, error) {
	var decision PlannerDecision
	trimmed := extractJSONObject(raw)
	if err := json.Unmarshal([]byte(trimmed), &decision); err != nil {
		return PlannerDecision{}, fmt.Errorf("parse intent planner output: %w", err)
	}
	decision.RawJSON = trimmed
	return decision, nil
}

func plannerSnapshotMap(decision PlannerDecision) map[string]any {
	decoded := make(map[string]any)
	if err := json.Unmarshal([]byte(decision.RawJSON), &decoded); err == nil {
		return decoded
	}
	return map[string]any{
		"primary_event":    decision.PrimaryEvent,
		"adjacent_events":  decision.AdjacentEvents,
		"reuse_existing":   decision.ReuseExisting,
		"create_new_event": decision.CreateNewEvent,
		"recall_plan":      decision.RecallPlan,
	}
}

func plannerAdjacentIDs(refs []PlannerEventRef) []string {
	out := make([]string, 0, len(refs))
	for _, ref := range refs {
		out = append(out, ref.EventID)
	}
	return normalizeIDs(out)
}

func resolvePlannerEdgeUpdate(
	sessionID string,
	primaryEventID string,
	update PlannerEdgeUpdate,
) (memorystore.EventEdgeInput, bool) {
	fromEventID := resolvePlannerEdgeRef(primaryEventID, update.FromEventID)
	toEventID := resolvePlannerEdgeRef(primaryEventID, update.ToEventID)
	if fromEventID == "" || toEventID == "" {
		return memorystore.EventEdgeInput{}, false
	}
	return memorystore.EventEdgeInput{
		SessionID:   sessionID,
		FromEventID: fromEventID,
		ToEventID:   toEventID,
		EdgeType:    update.EdgeType,
		Confidence:  update.Confidence,
	}, true
}

func resolvePlannerEdgeRef(primaryEventID string, raw string) string {
	switch strings.TrimSpace(raw) {
	case "", "none":
		return ""
	case "primary":
		return primaryEventID
	default:
		return strings.TrimSpace(raw)
	}
}

func eventNodeIDs(nodes []memorystore.EventNode) []string {
	out := make([]string, 0, len(nodes))
	for _, node := range nodes {
		out = append(out, node.ID)
	}
	return normalizeIDs(out)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}
