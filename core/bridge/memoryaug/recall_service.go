package memoryaug

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"ghost-os/bridge/memorystore"
)

const (
	primaryMemoryRecallLimit  = 3
	adjacentMemoryRecallLimit = 2
	eventMemoryCandidateLimit = 12
)

type recallService struct {
	settings Settings
	store    recallStore
}

type scoredEventMemory struct {
	hit       RecallMemoryHit
	textScore float64
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
	primaryNode, err := s.store.GetEventNode(ctx, normalized.PrimaryEventID)
	if err != nil {
		return RecallOutput{}, err
	}
	output := RecallOutput{
		PrimaryEvent: &RecallEventHit{
			Event:  primaryNode,
			Role:   "primary",
			Reason: "planner primary_event",
		},
	}
	adjacentNodes, err := s.loadAdjacentNodes(ctx, normalized.ActiveEventIDs[1:])
	if err != nil {
		return RecallOutput{}, err
	}
	output.AdjacentEvents = adjacentNodes
	if planRecallsEvent(normalized.RecallPlan, primaryNode.ID) {
		primaryMemories, err := s.loadEventMemories(ctx, primaryNode, normalized.FocusText, normalized.RecallPlan, primaryMemoryRecallLimit)
		if err != nil {
			return RecallOutput{}, err
		}
		output.PrimaryMemories = primaryMemories
	}
	adjacentMemories, err := s.loadAdjacentMemories(ctx, adjacentNodes, normalized.FocusText, normalized.RecallPlan)
	if err != nil {
		return RecallOutput{}, err
	}
	output.AdjacentMemories = adjacentMemories
	preferences, err := s.store.ListGlobalPreferences(ctx, globalPreferenceKeys())
	if err != nil {
		return RecallOutput{}, err
	}
	output.GlobalPreferences = preferences
	if err := s.touchRecallSelection(ctx, output); err != nil {
		return RecallOutput{}, err
	}
	output.PromptBlock = FormatPromptBlock(output)
	return output, nil
}

func (s *recallService) loadAdjacentNodes(ctx context.Context, ids []string) ([]RecallEventHit, error) {
	out := make([]RecallEventHit, 0, len(ids))
	for _, id := range normalizeIDs(ids) {
		node, err := s.store.GetEventNode(ctx, id)
		if err != nil {
			return nil, err
		}
		out = append(out, RecallEventHit{
			Event:  node,
			Role:   "adjacent",
			Reason: "planner adjacent_event",
		})
	}
	return out, nil
}

func (s *recallService) loadEventMemories(
	ctx context.Context,
	node memorystore.EventNode,
	focusText string,
	plan RecallPlan,
	limit int,
) ([]RecallMemoryHit, error) {
	items, _, err := s.store.ListEventMemories(ctx, memorystore.EventMemoryListFilter{
		EventID:  node.ID,
		Statuses: []string{memorystore.MemoryStatusActive},
		Limit:    eventMemoryCandidateLimit,
	})
	if err != nil {
		return nil, err
	}
	scored := make([]scoredEventMemory, 0, len(items))
	for _, item := range items {
		if !allowsMemoryType(plan, item.MemoryType) {
			continue
		}
		score := computeEventMemoryScore(focusText, item)
		scored = append(scored, scoredEventMemory{
			hit: RecallMemoryHit{
				Event: node,
				Entry: item,
				Reason: fmt.Sprintf(
					"event=%s score=%.2f confidence=%.2f",
					node.ID,
					score,
					item.Confidence,
				),
			},
			textScore: score,
		})
	}
	sort.SliceStable(scored, func(i int, j int) bool {
		return compareScoredEventMemory(scored[i], scored[j])
	})
	out := make([]RecallMemoryHit, 0, minInt(limit, len(scored)))
	for _, item := range scored {
		out = append(out, item.hit)
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (s *recallService) loadAdjacentMemories(
	ctx context.Context,
	nodes []RecallEventHit,
	focusText string,
	plan RecallPlan,
) ([]RecallMemoryHit, error) {
	scored := make([]scoredEventMemory, 0, len(nodes)*adjacentMemoryRecallLimit)
	for _, node := range nodes {
		if !planRecallsEvent(plan, node.Event.ID) {
			continue
		}
		items, err := s.loadEventMemories(ctx, node.Event, focusText, plan, adjacentMemoryRecallLimit)
		if err != nil {
			return nil, err
		}
		for _, item := range items {
			scored = append(scored, scoredEventMemory{
				hit:       item,
				textScore: computeEventMemoryScore(focusText, item.Entry),
			})
		}
	}
	sort.SliceStable(scored, func(i int, j int) bool {
		return compareScoredEventMemory(scored[i], scored[j])
	})
	out := make([]RecallMemoryHit, 0, minInt(adjacentMemoryRecallLimit, len(scored)))
	for _, item := range scored {
		out = append(out, item.hit)
		if len(out) >= adjacentMemoryRecallLimit {
			break
		}
	}
	return out, nil
}

func (s *recallService) touchRecallSelection(ctx context.Context, output RecallOutput) error {
	eventMemoryIDs := make([]string, 0, len(output.PrimaryMemories)+len(output.AdjacentMemories))
	for _, hit := range output.PrimaryMemories {
		eventMemoryIDs = append(eventMemoryIDs, hit.Entry.ID)
	}
	for _, hit := range output.AdjacentMemories {
		eventMemoryIDs = append(eventMemoryIDs, hit.Entry.ID)
	}
	if err := s.store.TouchEventMemories(ctx, eventMemoryIDs); err != nil {
		return err
	}
	explicitIDs := make([]string, 0, len(output.GlobalPreferences))
	learnedIDs := make([]string, 0, len(output.GlobalPreferences))
	for _, item := range output.GlobalPreferences {
		if item.SourceKind == memorystore.SourceKindExplicit {
			explicitIDs = append(explicitIDs, strings.TrimPrefix(item.ID, "explicit:"))
			continue
		}
		learnedIDs = append(learnedIDs, item.ID)
	}
	if err := s.store.TouchExplicitRecords(ctx, explicitIDs); err != nil {
		return err
	}
	return s.store.TouchLearned(ctx, learnedIDs)
}

func normalizeEventRecallInput(input RecallInput) (RecallInput, error) {
	activeIDs := normalizeIDs(input.ActiveEventIDs)
	primaryEventID := strings.TrimSpace(input.PrimaryEventID)
	if primaryEventID == "" {
		return RecallInput{}, fmt.Errorf("primary_event_id is required")
	}
	if len(activeIDs) == 0 {
		activeIDs = []string{primaryEventID}
	}
	if activeIDs[0] != primaryEventID {
		activeIDs = append([]string{primaryEventID}, filterEventIDs(activeIDs, primaryEventID)...)
	}
	if len(activeIDs) > 3 {
		return RecallInput{}, fmt.Errorf("active_event_ids cannot exceed 3")
	}
	return RecallInput{
		SessionID:      strings.TrimSpace(input.SessionID),
		PrimaryEventID: primaryEventID,
		ActiveEventIDs: activeIDs,
		FocusText:      strings.TrimSpace(input.FocusText),
		RecallPlan:     normalizeRecallPlan(input.RecallPlan, activeIDs),
	}, nil
}

func normalizeRecallPlan(plan RecallPlan, activeIDs []string) RecallPlan {
	normalized := plan
	normalized.EventIDs = normalizeIDs(plan.EventIDs)
	if len(normalized.EventIDs) == 0 {
		normalized.EventIDs = append([]string(nil), activeIDs...)
	}
	return normalized
}

func planRecallsEvent(plan RecallPlan, eventID string) bool {
	if len(plan.EventIDs) == 0 {
		return true
	}
	for _, id := range plan.EventIDs {
		if id == eventID {
			return true
		}
	}
	return false
}

func allowsMemoryType(plan RecallPlan, memoryType string) bool {
	switch memoryType {
	case memorystore.MemoryTypeWorkflow:
		return plan.IncludeWorkflow
	case memorystore.MemoryTypePreference:
		return plan.IncludePreference
	case memorystore.MemoryTypeProfile:
		return plan.IncludeProfile
	case memorystore.MemoryTypeFact:
		return plan.IncludeFact
	default:
		return false
	}
}

func computeEventMemoryScore(query string, entry memorystore.EventMemory) float64 {
	memoryEntry := memorystore.MemoryEntry{
		ID:         entry.ID,
		MemoryType: entry.MemoryType,
		MemoryKey:  entry.MemoryKey,
		Content:    entry.Content,
		Summary:    entry.Summary,
	}
	return computeTextScore(query, memoryEntry)
}

func compareScoredEventMemory(left scoredEventMemory, right scoredEventMemory) bool {
	if left.textScore != right.textScore {
		return left.textScore > right.textScore
	}
	if left.hit.Entry.Confidence != right.hit.Entry.Confidence {
		return left.hit.Entry.Confidence > right.hit.Entry.Confidence
	}
	if !left.hit.Entry.LastUsedAt.Equal(right.hit.Entry.LastUsedAt) {
		return left.hit.Entry.LastUsedAt.After(right.hit.Entry.LastUsedAt)
	}
	if !left.hit.Entry.UpdatedAt.Equal(right.hit.Entry.UpdatedAt) {
		return left.hit.Entry.UpdatedAt.After(right.hit.Entry.UpdatedAt)
	}
	return left.hit.Entry.ID < right.hit.Entry.ID
}

func filterEventIDs(ids []string, excluded string) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if id == excluded {
			continue
		}
		out = append(out, id)
	}
	return out
}

func minInt(left int, right int) int {
	if left < right {
		return left
	}
	return right
}
