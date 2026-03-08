package memory

import (
	"crypto/sha1"
	"encoding/hex"
	"strings"
	"time"
)

const decisionRecipeMinSuccessRate = 0.66

func distillStatsNamespace(namespace string) string {
	return normalizeDecisionNamespace(namespace)
}

func decisionClusterEnvironmentKey(env DecisionEnvFingerprint, graphRefs []string, anchorKeys []string) string {
	parts := []string{
		strings.TrimSpace(env.WorkspaceRoot),
		strings.TrimSpace(env.Platform),
		strings.TrimSpace(env.ToolsetSignature),
		strings.TrimSpace(env.GraphNamespace),
		strings.Join(uniqueStrings(env.ToolNames), ","),
		strings.Join(uniqueStrings(graphRefs), ","),
		strings.Join(uniqueStrings(anchorKeys), ","),
	}
	sum := sha1.Sum([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(sum[:8])
}

func decisionHitFromMemo(memo DecisionMemo, score float64, whyMatched string, caution string) DecisionHit {
	memo = normalizeDecisionMemo(memo)
	return normalizeDecisionHit(DecisionHit{
		Type:       DecisionHitTypeMemo,
		MemoID:     memo.ID,
		SessionID:  memo.SessionID,
		IntentKey:  memo.IntentKey,
		Summary:    firstNonEmpty(memo.StrategySummary, memo.IntentSummary, memo.OutcomeSummary),
		Reason:     memo.IntentSummary,
		Caution:    caution,
		Outcome:    memo.Outcome,
		Score:      clamp01(score),
		ReuseScore: clamp01(memo.ReuseScore),
		Confidence: clamp01(memo.Confidence),
		Timestamp:  effectiveDecisionTimestamp(memo.LastUsedAt, memo.CreatedAt),
		WhyMatched: strings.TrimSpace(whyMatched),
		GraphRefs:  append([]string(nil), memo.GraphNodeRefs...),
		AnchorKeys: append([]string(nil), memo.AnchorKeys...),
	})
}

func effectiveDecisionTimestamp(values ...time.Time) time.Time {
	for _, value := range values {
		if !value.IsZero() {
			return value.UTC()
		}
	}
	return time.Now().UTC()
}

func recipeRunHasDeviation(run RecipeRun) bool {
	for _, step := range run.Steps {
		if step.Skipped || !step.Matched || strings.TrimSpace(step.DeviationReason) != "" {
			return true
		}
	}
	return false
}

func selectorHintClause(text string) string {
	trimmed := strings.TrimSpace(text)
	trimmed = strings.TrimRight(trimmed, ".;:! ")
	return trimmed
}

func buildDecisionMemoID(input DecisionCaptureInput) string {
	hashInput := strings.Join([]string{
		normalizeDecisionNamespace(input.Namespace),
		strings.TrimSpace(input.SessionID),
		strings.TrimSpace(input.TraceID),
		strings.TrimSpace(input.TurnID),
		strings.TrimSpace(input.UserMessage),
		normalizeDecisionOutcome(input.Outcome),
		effectiveDecisionTimestamp(input.TurnFinishedAt, input.TurnStartedAt).Format(time.RFC3339Nano),
	}, "|")
	sum := sha1.Sum([]byte(hashInput))
	return "memo-" + hex.EncodeToString(sum[:8])
}
