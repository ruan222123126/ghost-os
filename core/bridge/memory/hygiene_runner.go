package memory

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

const (
	defaultHygieneMinConfidence = 0.5
	defaultHygieneVotesPerRun   = 1
)

type hygieneCandidate struct {
	Target       HygieneTarget
	Kind         string
	Source       string
	Role         string
	Text         string
	Confidence   float64
	AccessCount  int
	LastActivity time.Time
	Timestamp    time.Time
}

type hygieneScoredCandidate struct {
	Candidate  hygieneCandidate
	Score      float64
	Reasons    []HygieneReason
	TargetKey  string
	NextRecord HygieneRecord
}

func (m *MemoryManager) RunHygiene(opts HygieneRunOptions) (HygieneRunResult, error) {
	normalized, err := normalizeHygieneRunOptions(opts)
	if err != nil {
		return HygieneRunResult{}, err
	}
	candidates, err := m.collectHygieneCandidates(normalized.Scope, normalized.Limit)
	if err != nil {
		return HygieneRunResult{}, err
	}
	result := HygieneRunResult{
		Scanned: len(candidates),
		DryRun:  normalized.DryRun || normalized.Scope == HygieneScopeReport,
		TraceID: normalized.TraceID,
	}
	if len(candidates) == 0 {
		return result, nil
	}

	deduped := buildHygieneDuplicateIndex(candidates)
	targets := make([]HygieneTarget, 0, len(candidates))
	for _, candidate := range candidates {
		targets = append(targets, candidate.Target)
	}
	existing := map[string]HygieneRecord(nil)
	if m != nil && m.hygiene != nil {
		existing = m.hygiene.BatchGet(targets)
	}
	scored := make([]hygieneScoredCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		item, ok := m.scoreHygieneCandidate(candidate, deduped, normalized, existing)
		if !ok {
			continue
		}
		scored = append(scored, item)
	}
	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].Score != scored[j].Score {
			return scored[i].Score > scored[j].Score
		}
		left := firstNonZeroTime(scored[i].Candidate.LastActivity, scored[i].Candidate.Timestamp)
		right := firstNonZeroTime(scored[j].Candidate.LastActivity, scored[j].Candidate.Timestamp)
		if !left.Equal(right) {
			return left.Before(right)
		}
		return scored[i].TargetKey < scored[j].TargetKey
	})
	budget := normalized.MaxVotesPerRun
	if budget > len(scored) {
		budget = len(scored)
	}
	for i := 0; i < budget; i++ {
		item := scored[i]
		result.Scored++
		result.SuppressedCandidates++
		if item.NextRecord.QuarantineLevel != QuarantineLevelNone {
			result.QuarantinedCandidates++
		}
		if result.DryRun {
			continue
		}
		if err := m.ScoreGarbage(HygieneAssessmentInput{
			Target:    item.Candidate.Target,
			VoteDelta: 1,
			Reasons:   item.Reasons,
			TraceID:   normalized.TraceID,
			TaskID:    normalized.TaskID,
			Scope:     normalized.Scope,
		}); err != nil {
			return result, err
		}
	}
	return result, nil
}

func normalizeHygieneRunOptions(opts HygieneRunOptions) (HygieneRunOptions, error) {
	normalized := opts
	normalized.Scope = strings.TrimSpace(normalized.Scope)
	if normalized.Scope == "" {
		normalized.Scope = HygieneScopeWarm
	}
	switch normalized.Scope {
	case HygieneScopeWarm, HygieneScopeProjection, HygieneScopeReport:
	default:
		return HygieneRunOptions{}, fmt.Errorf("unsupported hygiene scope %q", normalized.Scope)
	}
	if normalized.Limit <= 0 {
		return HygieneRunOptions{}, fmt.Errorf("hygiene limit must be > 0")
	}
	normalized.MinConfidence = clamp01(maxFloat(normalized.MinConfidence, defaultHygieneMinConfidence))
	if normalized.MaxVotesPerRun <= 0 {
		normalized.MaxVotesPerRun = defaultHygieneVotesPerRun
	}
	normalized.TraceID = strings.TrimSpace(normalized.TraceID)
	normalized.TaskID = strings.TrimSpace(normalized.TaskID)
	normalized.ReasonCodes = uniqueStrings(normalized.ReasonCodes)
	return normalized, nil
}

func (m *MemoryManager) collectHygieneCandidates(scope string, limit int) ([]hygieneCandidate, error) {
	switch scope {
	case HygieneScopeWarm:
		return m.collectWarmHygieneCandidates(limit), nil
	case HygieneScopeProjection:
		return m.collectProjectionHygieneCandidates(limit)
	case HygieneScopeReport:
		warmLimit := max(1, limit/2)
		projectionLimit := max(0, limit-warmLimit)
		warm := m.collectWarmHygieneCandidates(warmLimit)
		projection, err := m.collectProjectionHygieneCandidates(projectionLimit)
		if err != nil {
			return nil, err
		}
		return append(warm, projection...), nil
	default:
		return nil, fmt.Errorf("unsupported hygiene scope %q", scope)
	}
}

func (m *MemoryManager) collectWarmHygieneCandidates(limit int) []hygieneCandidate {
	if m == nil || m.warm == nil || limit <= 0 {
		return nil
	}
	entries := m.warm.Snapshot()
	if len(entries) > limit {
		entries = entries[:limit]
	}
	out := make([]hygieneCandidate, 0, len(entries))
	for _, entry := range entries {
		target := HygieneTarget{EntryID: strings.TrimSpace(entry.ID)}
		if m.query != nil {
			if objectID := strings.TrimSpace(m.query.objectIDFromEntry(entry)); objectID != "" {
				target.ObjectID = objectID
			}
		}
		out = append(out, hygieneCandidate{
			Target:       target,
			Kind:         HygieneScopeWarm,
			Source:       firstNonEmpty(entry.Source, metadataString(entry.Metadata, "source")),
			Role:         metadataString(entry.Metadata, "role"),
			Text:         firstNonEmpty(entry.Summary, entry.Content),
			Confidence:   clamp01(entry.Confidence),
			AccessCount:  entry.AccessCount,
			LastActivity: firstNonZeroTime(entry.LastAccessedAt, entry.Timestamp),
			Timestamp:    entry.Timestamp.UTC(),
		})
	}
	return out
}

func (m *MemoryManager) collectProjectionHygieneCandidates(limit int) ([]hygieneCandidate, error) {
	if m == nil || limit <= 0 {
		return nil, nil
	}
	out := make([]hygieneCandidate, 0, limit)
	if m.cold != nil && len(out) < limit {
		ids, err := m.cold.ListMarkdownNodes()
		if err != nil {
			return nil, err
		}
		for _, id := range ids {
			if len(out) >= limit {
				break
			}
			node, err := m.cold.LoadMarkdownNode(id)
			if err != nil {
				continue
			}
			out = append(out, hygieneCandidate{
				Target:       HygieneTarget{EntryID: "markdown:" + strings.TrimSpace(node.ID)},
				Kind:         "markdown",
				Source:       "markdown",
				Text:         firstNonEmpty(node.Summary, node.Content),
				Confidence:   clamp01(node.Confidence),
				LastActivity: firstNonZeroTime(node.LastSeenAt, node.CreatedAt),
				Timestamp:    node.CreatedAt.UTC(),
			})
		}
	}
	if m.decision != nil && len(out) < limit {
		for _, memo := range m.decision.ListMemos("") {
			if len(out) >= limit {
				break
			}
			out = append(out, hygieneCandidate{
				Target:       HygieneTarget{EntryID: "decision.memo:" + strings.TrimSpace(memo.ID)},
				Kind:         "decision.memo",
				Source:       "decision",
				Text:         firstNonEmpty(memo.OutcomeSummary, memo.StrategySummary, memo.ContextSummary, memo.IntentSummary, memo.ProblemSummary),
				Confidence:   clamp01(maxFloat(memo.Confidence, memo.ReuseScore)),
				AccessCount:  memo.AccessCount,
				LastActivity: firstNonZeroTime(memo.LastUsedAt, memo.CreatedAt),
				Timestamp:    memo.CreatedAt.UTC(),
			})
		}
		for _, recipe := range m.decision.ListRecipes("") {
			if len(out) >= limit {
				break
			}
			out = append(out, hygieneCandidate{
				Target:       HygieneTarget{EntryID: "decision.recipe:" + strings.TrimSpace(recipe.ID)},
				Kind:         "decision.recipe",
				Source:       "decision",
				Text:         firstNonEmpty(recipe.StrategySummary, strings.Join(recipe.ValidationChecklist, " "), strings.Join(recipe.AvoidPatterns, " ")),
				Confidence:   clamp01(maxFloat(recipe.Confidence, recipe.SuccessRate)),
				AccessCount:  max(recipe.SelectedCount, recipe.AppliedCount),
				LastActivity: firstNonZeroTime(recipe.LastAppliedAt, recipe.LastSelectedAt, recipe.LastOutcomeAt, recipe.UpdatedAt, recipe.CreatedAt),
				Timestamp:    firstNonZeroTime(recipe.CreatedAt, recipe.UpdatedAt),
			})
		}
	}
	return out, nil
}

func (m *MemoryManager) scoreHygieneCandidate(candidate hygieneCandidate, duplicates map[string]time.Time, opts HygieneRunOptions, existing map[string]HygieneRecord) (hygieneScoredCandidate, bool) {
	targetKey, ok := candidate.Target.Key()
	if !ok {
		return hygieneScoredCandidate{}, false
	}
	reasons := make([]HygieneReason, 0, 4)
	lowActivity := hygieneLowActivity(candidate)
	text := strings.TrimSpace(candidate.Text)
	if lowActivity && hygieneLowSignal(text) {
		reasons = append(reasons, HygieneReason{Code: HygieneReasonLowSignalSummary, Label: "Low signal", Weight: 0.35, Evidence: summarizeLine(text, 120)})
	}
	if lowActivity && hygieneLooksLikeToolNoise(candidate) {
		reasons = append(reasons, HygieneReason{Code: HygieneReasonToolNoise, Label: "Tool noise", Weight: 0.45, Evidence: summarizeLine(text, 120)})
	}
	if lowActivity && hygieneLooksLikeStalePlan(candidate) {
		reasons = append(reasons, HygieneReason{Code: HygieneReasonStalePlan, Label: "Stale plan", Weight: 0.35, Evidence: summarizeLine(text, 120)})
	}
	if candidate.Kind == "decision.memo" && lowActivity && candidate.Confidence < 0.45 {
		reasons = append(reasons, HygieneReason{Code: HygieneReasonRedundantDecisionMemo, Label: "Redundant decision memo", Weight: 0.4, Evidence: summarizeLine(text, 120)})
	}
	if hygieneIsDuplicate(candidate, duplicates) {
		reasons = append(reasons, HygieneReason{Code: HygieneReasonDuplicateSummary, Label: "Duplicate summary", Weight: 0.45, Evidence: summarizeLine(text, 120)})
	}
	reasons = hygieneFilterReasonCodes(reasons, opts.ReasonCodes)
	if len(reasons) == 0 {
		return hygieneScoredCandidate{}, false
	}
	score := 0.0
	for _, reason := range reasons {
		score += reason.Weight
	}
	score = clamp01(score)
	if score < opts.MinConfidence {
		return hygieneScoredCandidate{}, false
	}
	next := HygieneRecord{QuarantineLevel: QuarantineLevelNone}
	if record, ok := existing[targetKey]; ok {
		next = record
	}
	next.Key = targetKey
	next.ObjectID = candidate.Target.ObjectID
	next.EntryID = candidate.Target.EntryID
	next.GarbageVotes++
	next.GarbageScore = hygieneScoreFromVotes(next.GarbageVotes)
	next.QuarantineLevel = hygieneQuarantineLevelFromVotes(next.GarbageVotes)
	return hygieneScoredCandidate{Candidate: candidate, Score: score, Reasons: reasons, TargetKey: targetKey, NextRecord: next}, true
}

func buildHygieneDuplicateIndex(candidates []hygieneCandidate) map[string]time.Time {
	latest := make(map[string]time.Time, len(candidates))
	counts := make(map[string]int, len(candidates))
	for _, candidate := range candidates {
		key := hygieneTextFingerprint(candidate.Text)
		if key == "" {
			continue
		}
		counts[key]++
		ts := firstNonZeroTime(candidate.LastActivity, candidate.Timestamp)
		if ts.After(latest[key]) {
			latest[key] = ts
		}
	}
	for key, count := range counts {
		if count < 2 {
			delete(latest, key)
		}
	}
	return latest
}

func hygieneFilterReasonCodes(reasons []HygieneReason, allow []string) []HygieneReason {
	if len(reasons) == 0 {
		return nil
	}
	if len(allow) == 0 {
		return reasons
	}
	out := make([]HygieneReason, 0, len(reasons))
	for _, reason := range reasons {
		if containsString(allow, reason.Code) {
			out = append(out, reason)
		}
	}
	return out
}

func hygieneLowActivity(candidate hygieneCandidate) bool {
	activity := firstNonZeroTime(candidate.LastActivity, candidate.Timestamp)
	if candidate.AccessCount > 1 {
		return false
	}
	if activity.IsZero() {
		return true
	}
	if candidate.Kind == HygieneScopeWarm {
		return time.Since(activity) > 24*time.Hour
	}
	return time.Since(activity) > 72*time.Hour
}

func hygieneLowSignal(text string) bool {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return true
	}
	return len([]rune(trimmed)) < 80 || len(strings.Fields(trimmed)) < 12
}

func hygieneLooksLikeToolNoise(candidate hygieneCandidate) bool {
	text := strings.ToLower(strings.TrimSpace(candidate.Text))
	if candidate.Role == "tool" {
		return true
	}
	for _, marker := range []string{"stdout", "stderr", "exit code", "tool", "trace", "command output"} {
		if strings.Contains(text, marker) {
			return true
		}
	}
	return false
}

func hygieneLooksLikeStalePlan(candidate hygieneCandidate) bool {
	activity := firstNonZeroTime(candidate.LastActivity, candidate.Timestamp)
	if activity.IsZero() || time.Since(activity) < 48*time.Hour {
		return false
	}
	text := strings.ToLower(strings.TrimSpace(candidate.Text))
	for _, marker := range []string{"plan", "next step", "todo", "follow up", "later", "will do"} {
		if strings.Contains(text, marker) {
			return true
		}
	}
	return false
}

func hygieneIsDuplicate(candidate hygieneCandidate, duplicates map[string]time.Time) bool {
	key := hygieneTextFingerprint(candidate.Text)
	if key == "" {
		return false
	}
	latest, ok := duplicates[key]
	if !ok {
		return false
	}
	activity := firstNonZeroTime(candidate.LastActivity, candidate.Timestamp)
	return !activity.IsZero() && activity.Before(latest)
}

func hygieneTextFingerprint(text string) string {
	trimmed := strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(text)), " "))
	if len(trimmed) > 160 {
		trimmed = trimmed[:160]
	}
	return trimmed
}
