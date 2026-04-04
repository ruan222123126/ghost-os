package memoryaug

import (
	"context"
	"fmt"
)

const (
	eventMemoryCandidateLimit = 12
	maxActiveEventCount       = 3
	primaryRecallWeight       = 3
	adjacentRecallWeight      = 2
	totalRecallWeight         = primaryRecallWeight + adjacentRecallWeight
)

type recallService struct {
	settings Settings
	store    recallStore
}

type scoredEventMemory struct {
	hit       RecallMemoryHit
	textScore float64
}

type sessionRecallNeeds struct {
	primaryLimit          int
	adjacentLimit         int
	recallPrimaryMemories bool
	needsPrimaryNode      bool
	needsAdjacent         bool
}

func NewRecallService(settings Settings, store recallStore) RecallService {
	return &recallService{
		settings: normalizeSettings(settings),
		store:    store,
	}
}

func (s *recallService) Recall(ctx context.Context, input RecallInput) (RecallOutput, error) {
	if !s.settings.Enabled || !s.settings.RecallEnabled {
		return RecallOutput{}, nil
	}
	if s.store == nil {
		return RecallOutput{}, fmt.Errorf("memory recall store is not configured")
	}
	normalized, err := normalizeEventRecallInput(input)
	if err != nil {
		return RecallOutput{}, err
	}
	output := RecallOutput{}
	if s.settings.SessionScopeEnabled {
		sessionOutput, err := s.loadSessionRecall(ctx, normalized)
		if err != nil {
			return RecallOutput{}, err
		}
		output.PrimaryEvent = sessionOutput.PrimaryEvent
		output.AdjacentEvents = sessionOutput.AdjacentEvents
		output.PrimaryMemories = sessionOutput.PrimaryMemories
		output.AdjacentMemories = sessionOutput.AdjacentMemories
	}
	if s.shouldRecallGlobalPreferences(normalized.RecallPlan) {
		preferences, err := s.store.ListGlobalPreferences(ctx, globalPreferenceKeys())
		if err != nil {
			return RecallOutput{}, err
		}
		output.GlobalPreferences = preferences
	}
	if err := s.touchRecallSelection(ctx, output); err != nil {
		return RecallOutput{}, err
	}
	output.PromptBlock = FormatPromptBlock(output)
	return output, nil
}

func (s *recallService) loadSessionRecall(ctx context.Context, input RecallInput) (RecallOutput, error) {
	needs := s.planSessionRecall(input)
	if !needs.needsPrimaryNode {
		if !needs.needsAdjacent {
			return RecallOutput{}, nil
		}
		return s.loadAdjacentRecall(ctx, input, needs.adjacentLimit)
	}

	output, err := s.loadPrimaryRecall(ctx, input, needs)
	if err != nil {
		return RecallOutput{}, err
	}
	if !needs.needsAdjacent {
		return output, nil
	}

	adjacentOutput, err := s.loadAdjacentRecall(ctx, input, needs.adjacentLimit)
	if err != nil {
		return RecallOutput{}, err
	}
	output.AdjacentEvents = adjacentOutput.AdjacentEvents
	output.AdjacentMemories = adjacentOutput.AdjacentMemories
	return output, nil
}

func (s *recallService) planSessionRecall(input RecallInput) sessionRecallNeeds {
	primaryLimit, adjacentLimit := s.recallItemBudget(len(input.ActiveEventIDs) - 1)
	recallEventMemories := planIncludesEventMemory(input.RecallPlan)
	recallPrimaryMemories := recallEventMemories &&
		primaryLimit > 0 &&
		planRecallsEvent(input.RecallPlan, input.PrimaryEventID)
	needsPrimaryNode := input.RecallPlan.IncludeNodeSummary || recallPrimaryMemories
	needsAdjacent := len(input.ActiveEventIDs) > 1 &&
		(input.RecallPlan.IncludeNodeSummary || (recallEventMemories && adjacentLimit > 0))
	return sessionRecallNeeds{
		primaryLimit:          primaryLimit,
		adjacentLimit:         adjacentLimit,
		recallPrimaryMemories: recallPrimaryMemories,
		needsPrimaryNode:      needsPrimaryNode,
		needsAdjacent:         needsAdjacent,
	}
}

func (s *recallService) loadPrimaryRecall(
	ctx context.Context,
	input RecallInput,
	needs sessionRecallNeeds,
) (RecallOutput, error) {
	primaryNode, err := s.store.GetEventNode(ctx, input.PrimaryEventID)
	if err != nil {
		return RecallOutput{}, err
	}
	output := RecallOutput{}
	if input.RecallPlan.IncludeNodeSummary {
		output.PrimaryEvent = &RecallEventHit{
			Event:  primaryNode,
			Role:   "primary",
			Reason: "planner primary_event",
		}
	}
	if !needs.recallPrimaryMemories {
		return output, nil
	}
	output.PrimaryMemories, err = s.loadEventMemories(
		ctx,
		primaryNode,
		input.FocusText,
		input.RecallPlan,
		needs.primaryLimit,
	)
	if err != nil {
		return RecallOutput{}, err
	}
	return output, nil
}
