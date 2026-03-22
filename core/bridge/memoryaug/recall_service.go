package memoryaug

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"ghost-os/bridge/memorystore"
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
	primaryLimit, adjacentLimit := s.recallItemBudget(len(input.ActiveEventIDs) - 1)
	recallEventMemories := planIncludesEventMemory(input.RecallPlan)
	recallPrimaryMemories := recallEventMemories &&
		primaryLimit > 0 &&
		planRecallsEvent(input.RecallPlan, input.PrimaryEventID)
	needsPrimaryNode := input.RecallPlan.IncludeNodeSummary || recallPrimaryMemories
	if !needsPrimaryNode {
		if len(input.ActiveEventIDs) == 1 || (adjacentLimit == 0 && !input.RecallPlan.IncludeNodeSummary) {
			return RecallOutput{}, nil
		}
		return s.loadAdjacentRecall(ctx, input, adjacentLimit)
	}
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
	if recallPrimaryMemories {
		output.PrimaryMemories, err = s.loadEventMemories(ctx, primaryNode, input.FocusText, input.RecallPlan, primaryLimit)
		if err != nil {
			return RecallOutput{}, err
		}
	}
	if len(input.ActiveEventIDs) == 1 || (adjacentLimit == 0 && !input.RecallPlan.IncludeNodeSummary) {
		return output, nil
	}
	adjacentOutput, err := s.loadAdjacentRecall(ctx, input, adjacentLimit)
	if err != nil {
		return RecallOutput{}, err
	}
	output.AdjacentEvents = adjacentOutput.AdjacentEvents
	output.AdjacentMemories = adjacentOutput.AdjacentMemories
	return output, nil
}

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
	if len(activeIDs) > maxActiveEventCount {
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

func planIncludesEventMemory(plan RecallPlan) bool {
	return plan.IncludeWorkflow || plan.IncludePreference || plan.IncludeFact || plan.IncludeProfile
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

func roundShare(total int, weight int, totalWeight int) int {
	if total <= 0 || weight <= 0 || totalWeight <= 0 {
		return 0
	}
	return (total*weight + totalWeight/2) / totalWeight
}
