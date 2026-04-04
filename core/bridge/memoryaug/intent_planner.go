package memoryaug

import (
	"context"
	"fmt"
	"strings"

	"ghost-os/bridge/llm"
	"ghost-os/bridge/memorystore"
)

const plannerCandidateLimit = 12

type intentPlanner struct {
	settings  Settings
	store     plannerStore
	completer llm.Completer
}

type plannerContext struct {
	PreviousState memorystore.SessionEventState `json:"previous_state,omitempty"`
	Candidates    []memorystore.EventNode       `json:"candidates,omitempty"`
	Edges         []memorystore.EventEdge       `json:"edges,omitempty"`
}

func NewIntentPlanner(settings Settings, store plannerStore, completer llm.Completer) IntentPlanner {
	return &intentPlanner{
		settings:  normalizeSettings(settings),
		store:     store,
		completer: completer,
	}
}

func (p *intentPlanner) Plan(ctx context.Context, input PlannerInput) (PlannerDecision, error) {
	if p == nil || p.completer == nil {
		return PlannerDecision{}, fmt.Errorf("intent planner completer is not configured")
	}
	if p.store == nil {
		return PlannerDecision{}, fmt.Errorf("intent planner store is not configured")
	}
	normalized := normalizePlannerInput(input, p.settings)
	contextData, err := p.loadPlannerContext(ctx, normalized)
	if err != nil {
		return PlannerDecision{}, err
	}
	decision, err := p.completePlan(ctx, normalized, contextData)
	if err != nil {
		return PlannerDecision{}, err
	}
	return p.persistPlan(ctx, normalized, contextData, decision)
}

func (p *intentPlanner) loadPlannerContext(ctx context.Context, input PlannerInput) (plannerContext, error) {
	contextData := plannerContext{}
	state, err := p.store.LoadSessionEventState(ctx, input.SessionID)
	switch {
	case err == nil:
		contextData.PreviousState = state
	case err != nil && err != memorystore.ErrNotFound:
		return plannerContext{}, err
	}
	nodes, err := p.store.ListEventNodes(ctx, memorystore.EventNodeFilter{
		SessionID: input.SessionID,
		Statuses:  []string{memorystore.EventStatusActive, memorystore.EventStatusArchived},
		Query:     input.UserMessage,
		Limit:     plannerCandidateLimit,
	})
	if err != nil {
		return plannerContext{}, err
	}
	contextData.Candidates = nodes
	contextData.Edges, err = p.store.ListEventEdges(ctx, input.SessionID, eventNodeIDs(nodes))
	if err != nil {
		return plannerContext{}, err
	}
	return contextData, nil
}

func (p *intentPlanner) completePlan(
	ctx context.Context,
	input PlannerInput,
	contextData plannerContext,
) (PlannerDecision, error) {
	resp, err := p.completer.Complete(ctx, llm.CompletionRequest{
		Messages: []llm.Message{
			{Role: llm.RoleSystem, Text: plannerSystemPrompt()},
			{Role: llm.RoleUser, Text: plannerUserPrompt(input, contextData)},
		},
	})
	if err != nil {
		return PlannerDecision{}, err
	}
	return parsePlannerDecision(strings.TrimSpace(resp.Message.Text))
}

func (p *intentPlanner) persistPlan(
	ctx context.Context,
	input PlannerInput,
	contextData plannerContext,
	decision PlannerDecision,
) (PlannerDecision, error) {
	resolved, err := p.resolvePlannerDecision(ctx, input, contextData, decision)
	if err != nil {
		return PlannerDecision{}, err
	}
	if err := p.store.TouchEventNodes(ctx, resolved.RecallPlan.EventIDs); err != nil {
		return PlannerDecision{}, err
	}
	if err := p.applyEdgeUpdates(ctx, input.SessionID, resolved); err != nil {
		return PlannerDecision{}, err
	}
	if err := p.store.SaveSessionEventState(ctx, memorystore.SessionEventState{
		SessionID:       input.SessionID,
		PrimaryEventID:  resolved.PrimaryEvent.EventID,
		ActiveEventIDs:  resolved.RecallPlan.EventIDs,
		PlannerSnapshot: plannerSnapshotMap(resolved),
	}); err != nil {
		return PlannerDecision{}, err
	}
	return resolved, nil
}

func (p *intentPlanner) resolvePlannerDecision(
	ctx context.Context,
	input PlannerInput,
	_ plannerContext,
	decision PlannerDecision,
) (PlannerDecision, error) {
	if len(decision.AdjacentEvents) > 2 {
		return PlannerDecision{}, fmt.Errorf("planner adjacent_events cannot exceed 2")
	}
	resolved := decision
	primaryID, err := p.resolvePrimaryEventID(ctx, input.SessionID, decision)
	if err != nil {
		return PlannerDecision{}, err
	}
	resolved.PrimaryEvent.EventID = primaryID
	resolved.AdjacentEvents, err = p.resolveAdjacentEvents(ctx, decision.AdjacentEvents)
	if err != nil {
		return PlannerDecision{}, err
	}
	activeIDs := append([]string{primaryID}, plannerAdjacentIDs(resolved.AdjacentEvents)...)
	resolved.RecallPlan = normalizeRecallPlan(decision.RecallPlan, activeIDs)
	if len(resolved.RecallPlan.EventIDs) > 3 {
		return PlannerDecision{}, fmt.Errorf("planner recall_plan.event_ids cannot exceed 3")
	}
	return resolved, nil
}

func (p *intentPlanner) resolvePrimaryEventID(ctx context.Context, sessionID string, decision PlannerDecision) (string, error) {
	if decision.CreateNewEvent {
		return p.createPlannedEvent(ctx, sessionID, decision.NewEventTitle, decision.NewEventSummary)
	}
	eventID := strings.TrimSpace(decision.PrimaryEvent.EventID)
	if eventID == "" {
		return "", fmt.Errorf("planner primary_event.event_id is required")
	}
	if _, err := p.store.GetEventNode(ctx, eventID); err != nil {
		return "", err
	}
	return eventID, nil
}

func (p *intentPlanner) createPlannedEvent(ctx context.Context, sessionID string, title string, summary string) (string, error) {
	if strings.TrimSpace(title) == "" {
		return "", fmt.Errorf("planner new_event_title is required when create_new_event=true")
	}
	node, err := p.store.CreateEventNode(ctx, memorystore.EventNodeInput{
		SessionID: sessionID,
		Title:     title,
		Summary:   summary,
		Status:    memorystore.EventStatusActive,
	})
	if err != nil {
		return "", err
	}
	return node.ID, nil
}

func (p *intentPlanner) resolveAdjacentEvents(ctx context.Context, refs []PlannerEventRef) ([]PlannerEventRef, error) {
	out := make([]PlannerEventRef, 0, len(refs))
	for _, ref := range refs {
		eventID := strings.TrimSpace(ref.EventID)
		if eventID == "" {
			return nil, fmt.Errorf("planner adjacent_events.event_id is required")
		}
		if _, err := p.store.GetEventNode(ctx, eventID); err != nil {
			return nil, err
		}
		out = append(out, PlannerEventRef{EventID: eventID, Reason: strings.TrimSpace(ref.Reason)})
	}
	return out, nil
}

func (p *intentPlanner) applyEdgeUpdates(ctx context.Context, sessionID string, decision PlannerDecision) error {
	for _, update := range decision.EdgeUpdates {
		input, ok := resolvePlannerEdgeUpdate(sessionID, decision.PrimaryEvent.EventID, update)
		if !ok {
			continue
		}
		if _, err := p.store.UpsertEventEdge(ctx, input); err != nil {
			return err
		}
	}
	return nil
}

func normalizePlannerInput(input PlannerInput, settings Settings) PlannerInput {
	return PlannerInput{
		SessionID:      strings.TrimSpace(input.SessionID),
		UserScope:      strings.TrimSpace(firstNonEmpty(input.UserScope, settings.UserScopeID)),
		UserMessage:    strings.TrimSpace(input.UserMessage),
		RecentMessages: append([]TurnMessage(nil), input.RecentMessages...),
		ProjectRoot:    strings.TrimSpace(input.ProjectRoot),
	}
}
