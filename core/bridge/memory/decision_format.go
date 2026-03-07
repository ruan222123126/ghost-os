package memory

import (
	"strings"
	"time"
)

const decisionRecallLineMaxLen = 180

func memoryEntryFromDecisionHit(hit DecisionHit) MemoryEntry {
	hit = normalizeDecisionHit(hit)
	line := formatDecisionHitLine(hit)
	metadata := map[string]any{
		"layer":                "decision",
		"source":               "decision",
		"decision_hit_type":    hit.Type,
		"decision_outcome":     hit.Outcome,
		"decision_score":       hit.Score,
		"decision_why_matched": hit.WhyMatched,
		"decision_caution":     hit.Caution,
	}
	if hit.MemoID != "" {
		metadata["memo_id"] = hit.MemoID
	}
	if hit.RecipeID != "" {
		metadata["recipe_id"] = hit.RecipeID
	}
	if hit.SessionID != "" {
		metadata["session_id"] = hit.SessionID
	}

	relatedTo := make([]string, 0, 2)
	if hit.MemoID != "" {
		relatedTo = append(relatedTo, hit.MemoID)
	}
	if hit.RecipeID != "" {
		relatedTo = append(relatedTo, hit.RecipeID)
	}

	return normalizeEntry(MemoryEntry{
		ID:         decisionEntryID(hit),
		Content:    line,
		Summary:    line,
		Type:       MemoryTypeKnowledge,
		Timestamp:  decisionHitTimestamp(hit),
		Importance: clamp01(hit.Score*0.75 + hit.ReuseScore*0.25),
		RelatedTo:  relatedTo,
		Source:     "decision",
		Confidence: hit.Confidence,
		Metadata:   metadata,
	})
}

func formatDecisionRecallLine(hit DecisionHit) string {
	hit = normalizeDecisionHit(hit)
	base := firstNonEmpty(hit.Summary, hit.WhyMatched, hit.Reason)
	if base == "" {
		base = "similar approach proved reliable before"
	}
	line := "Prior similar experience: " + trimDecisionSuffix(base)
	if hit.Outcome == DecisionOutcomePartial && hit.Caution != "" {
		line += "; watch out for " + lowerDecisionClause(hit.Caution)
	}
	return summarizeLine(line, decisionRecallLineMaxLen)
}

func formatDecisionWarningLine(hit DecisionHit) string {
	hit = normalizeDecisionHit(hit)
	clause := firstNonEmpty(hit.Caution, hit.Summary, hit.WhyMatched, hit.Reason)
	if clause == "" {
		clause = "a similar past attempt went poorly"
	}
	lowerClause := strings.ToLower(strings.TrimSpace(clause))
	if hit.Outcome == DecisionOutcomeAwaitingHuman || strings.Contains(lowerClause, "human") || strings.Contains(lowerClause, "approval") || strings.Contains(lowerClause, "confirm") {
		return summarizeLine("Ask human early if "+lowerDecisionClause(clause), decisionRecallLineMaxLen)
	}
	return summarizeLine("Caution: "+trimDecisionSuffix(clause), decisionRecallLineMaxLen)
}

func formatDecisionHitLine(hit DecisionHit) string {
	if normalizeDecisionHitType(hit.Type) == DecisionHitTypeWarning {
		return formatDecisionWarningLine(hit)
	}
	return formatDecisionRecallLine(hit)
}

func decisionEntryID(hit DecisionHit) string {
	base := firstNonEmpty(hit.MemoID, hit.RecipeID)
	if base == "" {
		base = strings.ReplaceAll(normalizeRecallText(firstNonEmpty(hit.Summary, hit.Caution, hit.WhyMatched)), " ", "-")
	}
	if base == "" {
		base = "hit"
	}
	return "decision:" + firstNonEmpty(hit.Type, "hit") + ":" + base
}

func decisionHitTimestamp(hit DecisionHit) time.Time {
	if !hit.Timestamp.IsZero() {
		return hit.Timestamp.UTC()
	}
	return time.Now().UTC()
}

func trimDecisionSuffix(text string) string {
	return strings.TrimRight(strings.TrimSpace(text), ".;:! ")
}

func lowerDecisionClause(text string) string {
	trimmed := trimDecisionSuffix(text)
	if trimmed == "" {
		return "approval or context is missing"
	}
	if len(trimmed) == 1 {
		return strings.ToLower(trimmed)
	}
	return strings.ToLower(trimmed[:1]) + trimmed[1:]
}
