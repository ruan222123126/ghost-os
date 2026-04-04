package memoryaug

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"ghost-os/bridge/memorystore"
)

func (s *recallService) loadAdjacentRecall(ctx context.Context, input RecallInput, adjacentLimit int) (RecallOutput, error) {
	if len(input.ActiveEventIDs) == 1 {
		return RecallOutput{}, nil
	}
	recallEventMemories := planIncludesEventMemory(input.RecallPlan)
	if !input.RecallPlan.IncludeNodeSummary && (!recallEventMemories || adjacentLimit == 0) {
		return RecallOutput{}, nil
	}
	adjacentNodes, err := s.loadAdjacentNodes(ctx, input.ActiveEventIDs[1:])
	if err != nil {
		return RecallOutput{}, err
	}
	output := RecallOutput{}
	if input.RecallPlan.IncludeNodeSummary {
		output.AdjacentEvents = adjacentNodes
	}
	if !recallEventMemories || adjacentLimit == 0 {
		return output, nil
	}
	output.AdjacentMemories, err = s.loadAdjacentMemories(
		ctx,
		adjacentNodes,
		input.FocusText,
		input.RecallPlan,
		adjacentLimit,
	)
	if err != nil {
		return RecallOutput{}, err
	}
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
	limit int,
) ([]RecallMemoryHit, error) {
	if limit <= 0 {
		return nil, nil
	}
	perNodeLimit := minInt(limit, eventMemoryCandidateLimit)
	scored := make([]scoredEventMemory, 0, len(nodes)*perNodeLimit)
	for _, node := range nodes {
		if !planRecallsEvent(plan, node.Event.ID) {
			continue
		}
		items, err := s.loadEventMemories(ctx, node.Event, focusText, plan, perNodeLimit)
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
	out := make([]RecallMemoryHit, 0, minInt(limit, len(scored)))
	for _, item := range scored {
		out = append(out, item.hit)
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (s *recallService) recallItemBudget(adjacentCount int) (int, int) {
	total := s.settings.MaxRecallItems
	if total <= 0 {
		return 0, 0
	}
	if adjacentCount <= 0 {
		return total, 0
	}
	adjacentLimit := roundShare(total, adjacentRecallWeight, totalRecallWeight)
	if adjacentLimit >= total {
		adjacentLimit = total - 1
	}
	return total - adjacentLimit, adjacentLimit
}

func (s *recallService) shouldRecallGlobalPreferences(plan RecallPlan) bool {
	return s.settings.UserScopeEnabled && plan.IncludePreference
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
