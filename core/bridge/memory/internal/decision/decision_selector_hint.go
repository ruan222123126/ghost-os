package decision

import (
	"strings"
)

const (
	selectorHintMaxChars = 400
	selectorHintMaxLines = 3
)

func (d *DecisionService) formatSelectorHint(hits []DecisionHit) string {
	if d == nil || len(hits) == 0 {
		return ""
	}

	toolNames := make([]string, 0, 6)
	startHints := make([]string, 0, 3)
	avoidHints := make([]string, 0, 3)
	askHumanHints := make([]string, 0, 2)
	seenTools := make(map[string]struct{}, 8)
	seenStarts := make(map[string]struct{}, 4)
	seenAvoids := make(map[string]struct{}, 4)
	seenAskHuman := make(map[string]struct{}, 4)

	appendUnique := func(dst *[]string, seen map[string]struct{}, value string) {
		trimmed := strings.TrimSpace(value)
		lookup := decisionLookup(trimmed)
		if lookup == "" {
			return
		}
		if _, ok := seen[lookup]; ok {
			return
		}
		seen[lookup] = struct{}{}
		*dst = append(*dst, trimmed)
	}

	for _, rawHit := range hits {
		hit := normalizeDecisionHit(rawHit)
		switch hit.Type {
		case DecisionHitTypeRecipe:
			recipe, ok := d.store.Recipe(hit.RecipeID)
			if !ok {
				appendUnique(&startHints, seenStarts, d.selectorHintLine(hit))
				continue
			}
			for _, name := range recipe.RecommendedTools {
				appendUnique(&toolNames, seenTools, name)
			}
			for _, step := range recipe.OrderedActions {
				appendUnique(&toolNames, seenTools, step.ToolName)
			}
			appendUnique(&startHints, seenStarts, decisionRecipeFirstAction(recipe))
			appendUnique(&startHints, seenStarts, recipe.StrategySummary)
			appendUnique(&avoidHints, seenAvoids, firstNonEmptySlice(recipe.AvoidPatterns))
		case DecisionHitTypeMemo:
			memo, ok := d.store.Memo(hit.MemoID)
			if !ok {
				appendUnique(&startHints, seenStarts, d.selectorHintLine(hit))
				continue
			}
			for _, name := range decisionToolNamesFromUses(memo.ToolsUsed) {
				appendUnique(&toolNames, seenTools, name)
			}
			appendUnique(&startHints, seenStarts, memo.StrategySummary)
			appendUnique(&avoidHints, seenAvoids, firstNonEmptySlice(memo.AvoidPatterns))
			appendUnique(&askHumanHints, seenAskHuman, firstNonEmptySlice(memo.NeedsHumanFor))
		case DecisionHitTypeWarning:
			caution := strings.TrimSpace(hit.Caution)
			lower := strings.ToLower(caution)
			if hit.Outcome == DecisionOutcomeAwaitingHuman || strings.Contains(lower, "human") || strings.Contains(lower, "approval") || strings.Contains(lower, "confirm") || strings.Contains(lower, "credential") {
				appendUnique(&askHumanHints, seenAskHuman, caution)
				continue
			}
			appendUnique(&avoidHints, seenAvoids, caution)
		default:
			appendUnique(&startHints, seenStarts, d.selectorHintLine(hit))
		}
	}

	lines := make([]string, 0, selectorHintMaxLines)
	if len(toolNames) > 0 {
		lines = append(lines, summarizeLine("Similar successful cases used: "+strings.Join(toolNames[:minInt(len(toolNames), 5)], ", "), 140))
	} else if selectorHasReusableCase(hits) {
		lines = append(lines, "Similar successful cases used: prior decision memo")
	}
	if len(startHints) > 0 {
		lines = append(lines, summarizeLine("Usually start with: "+selectorHintClause(startHints[0]), 150))
	}
	cautionParts := make([]string, 0, 2)
	if len(avoidHints) > 0 {
		cautionParts = append(cautionParts, "Avoid: "+selectorHintClause(avoidHints[0]))
	}
	if len(askHumanHints) > 0 {
		cautionParts = append(cautionParts, "Ask human early if "+selectorHintClause(askHumanHints[0]))
	}
	if len(cautionParts) > 0 {
		lines = append(lines, summarizeLine(strings.Join(cautionParts, ". "), 170))
	}
	if len(lines) == 0 {
		for _, hit := range hits {
			if line := summarizeLine(d.selectorHintLine(hit), 150); line != "" {
				lines = append(lines, line)
				break
			}
		}
	}
	if len(lines) == 0 {
		return ""
	}
	return trimSelectorHint(lines)
}

func selectorHasReusableCase(hits []DecisionHit) bool {
	for _, hit := range hits {
		normalized := normalizeDecisionHit(hit)
		if normalized.Type == DecisionHitTypeMemo || normalized.Type == DecisionHitTypeRecipe {
			return true
		}
	}
	return false
}

func (d *DecisionService) formatSelectorHintSelection(advisory *RecipeAdvisory, report *RecipeSelectionReport, hits []DecisionHit) string {
	lines := make([]string, 0, selectorHintMaxLines)
	if advisory != nil {
		prefix := "Selected recipe"
		if advisory.LowConfidenceOnly {
			prefix = "Advisory recipe"
		}
		if start := selectorHintClause(advisory.StartWith); start != "" {
			lines = append(lines, summarizeLine(prefix+": start with "+start, 150))
		}
		parts := make([]string, 0, 2)
		if len(advisory.Avoid) > 0 {
			parts = append(parts, "Avoid: "+selectorHintClause(advisory.Avoid[0]))
		}
		if len(advisory.Validate) > 0 {
			parts = append(parts, "Validate: "+selectorHintClause(advisory.Validate[0]))
		}
		if len(parts) > 0 {
			lines = append(lines, summarizeLine(strings.Join(parts, ". "), 170))
		}
		if len(advisory.AskHumanIf) > 0 {
			lines = append(lines, summarizeLine("Ask human early if "+selectorHintClause(advisory.AskHumanIf[0]), 170))
		}
	}
	if fallback := d.fallbackSelectorHit(hits); fallback != nil {
		line := d.selectorHintLine(*fallback)
		if trimmed := selectorHintClause(line); trimmed != "" {
			label := "Fallback memo"
			if normalizeDecisionHitType(fallback.Type) == DecisionHitTypeWarning {
				label = "Fallback caution"
			}
			lines = append(lines, summarizeLine(label+": "+trimmed, 170))
		}
	}
	if len(lines) == 0 {
		if report != nil && strings.TrimSpace(report.FallbackReason) != "" {
			return summarizeLine("Fallback: "+selectorHintClause(report.FallbackReason), selectorHintMaxChars)
		}
		return d.formatSelectorHint(hits)
	}
	return trimSelectorHint(lines)
}

func (d *DecisionService) fallbackSelectorHit(hits []DecisionHit) *DecisionHit {
	for _, hit := range hits {
		normalized := normalizeDecisionHit(hit)
		if normalized.Type == DecisionHitTypeMemo {
			return &normalized
		}
	}
	for _, hit := range hits {
		normalized := normalizeDecisionHit(hit)
		if normalized.Type == DecisionHitTypeWarning {
			return &normalized
		}
	}
	return nil
}

func (d *DecisionService) selectorHintLine(hit DecisionHit) string {
	hit = normalizeDecisionHit(hit)
	if hit.Type == DecisionHitTypeWarning {
		return selectorHintFromWarning(hit)
	}
	if d != nil && d.store != nil {
		if hit.RecipeID != "" {
			if recipe, ok := d.store.Recipe(hit.RecipeID); ok {
				return d.selectorHintFromRecipe(hit, recipe)
			}
		}
		if hit.MemoID != "" {
			if memo, ok := d.store.Memo(hit.MemoID); ok {
				return d.selectorHintFromMemo(hit, memo)
			}
		}
	}
	return firstNonEmpty(hit.Summary, hit.Caution, hit.WhyMatched, hit.Reason)
}

func (d *DecisionService) selectorHintFromMemo(hit DecisionHit, memo DecisionMemo) string {
	parts := make([]string, 0, 3)
	if names := decisionToolNamesFromUses(memo.ToolsUsed); len(names) > 0 {
		names = uniqueStrings(names)
		parts = append(parts, "used "+strings.Join(names[:minInt(len(names), 4)], ", "))
	}
	if strategy := selectorHintClause(memo.StrategySummary); strategy != "" {
		parts = append(parts, "start with "+strategy)
	}
	if caution := selectorHintClause(firstNonEmptySlice(memo.AvoidPatterns)); caution != "" {
		parts = append(parts, "avoid "+caution)
	}
	if len(parts) == 0 {
		return firstNonEmpty(hit.Summary, hit.WhyMatched, hit.Reason)
	}
	return summarizeLine(strings.Join(parts, "; "), 160)
}

func (d *DecisionService) selectorHintFromRecipe(hit DecisionHit, recipe DecisionRecipe) string {
	parts := make([]string, 0, 3)
	if len(recipe.RecommendedTools) > 0 {
		parts = append(parts, "used "+strings.Join(recipe.RecommendedTools[:minInt(len(recipe.RecommendedTools), 4)], ", "))
	}
	if action := selectorHintClause(decisionRecipeFirstAction(recipe)); action != "" {
		parts = append(parts, "start with "+action)
	} else if strategy := selectorHintClause(recipe.StrategySummary); strategy != "" {
		parts = append(parts, "start with "+strategy)
	}
	if caution := selectorHintClause(firstNonEmptySlice(recipe.AvoidPatterns)); caution != "" {
		parts = append(parts, "avoid "+caution)
	}
	if len(parts) == 0 {
		return firstNonEmpty(hit.Summary, hit.WhyMatched, hit.Reason)
	}
	return summarizeLine(strings.Join(parts, "; "), 160)
}

func selectorHintFromWarning(hit DecisionHit) string {
	hit = normalizeDecisionHit(hit)
	clause := selectorHintClause(firstNonEmpty(hit.Caution, hit.Summary, hit.WhyMatched, hit.Reason))
	if clause == "" {
		clause = "the scope or prior failure is unclear"
	}
	lower := strings.ToLower(clause)
	if hit.Outcome == DecisionOutcomeAwaitingHuman || strings.Contains(lower, "human") || strings.Contains(lower, "approval") || strings.Contains(lower, "confirm") || strings.Contains(lower, "credential") {
		return summarizeLine("Ask human early if "+clause, 150)
	}
	return summarizeLine("Avoid: "+clause, 150)
}

func selectorHintClause(text string) string {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return ""
	}
	trimmed = trimDecisionSuffix(trimmed)
	trimmed = strings.ReplaceAll(trimmed, "\n", " ")
	trimmed = strings.Join(strings.Fields(trimmed), " ")
	return lowerDecisionClause(trimmed)
}

func trimSelectorHint(lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	if len(lines) > selectorHintMaxLines {
		lines = lines[:selectorHintMaxLines]
	}
	joined := strings.Join(lines, "\n")
	if len([]rune(joined)) <= selectorHintMaxChars {
		return joined
	}
	trimmedLines := append([]string(nil), lines...)
	for len([]rune(strings.Join(trimmedLines, "\n"))) > selectorHintMaxChars && len(trimmedLines) > 0 {
		last := len(trimmedLines) - 1
		trimmedLines[last] = summarizeLine(trimmedLines[last], max(48, selectorHintMaxChars/len(trimmedLines)))
		joined = strings.Join(trimmedLines, "\n")
		if len([]rune(joined)) <= selectorHintMaxChars {
			return joined
		}
		if len(trimmedLines) == 1 {
			break
		}
		trimmedLines = trimmedLines[:last]
	}
	return summarizeLine(strings.Join(trimmedLines, " "), selectorHintMaxChars)
}
