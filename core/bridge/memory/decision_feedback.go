package memory

import (
	"crypto/sha1"
	"encoding/hex"
	"log"
	"strings"
	"time"

	"ghost-os/bridge/llm"
)

func (d *DecisionService) recordRecipeSelection(recipeID string, selectedAt time.Time) {
	if d == nil || d.store == nil || strings.TrimSpace(recipeID) == "" {
		return
	}
	recipe, ok := d.store.Recipe(recipeID)
	if !ok {
		return
	}
	selectedAt = selectedAt.UTC()
	if selectedAt.IsZero() {
		selectedAt = time.Now().UTC()
	}
	recipe.SelectedCount++
	recipe.LastSelectedAt = selectedAt
	recipe.UpdatedAt = latestDecisionTime(recipe.UpdatedAt, selectedAt)
	_, _ = d.store.UpsertRecipe(recipe)
	if d.metrics != nil {
		d.metrics.recipeSelections.Add(1)
	}
	if err := d.store.Persist(); err != nil && d.debugEnabled {
		logDecisionDebug("persist selected recipe stats", err)
	}
}

func (d *DecisionService) consumePendingRecipeRun(sessionID string, userMessage string) (RecipeRun, bool) {
	if d == nil {
		return RecipeRun{}, false
	}
	key := buildPendingRecipeRunKey(sessionID, userMessage)
	d.pendingMu.Lock()
	run, ok := d.pendingRecipeRuns[key]
	if ok {
		delete(d.pendingRecipeRuns, key)
	}
	d.pendingMu.Unlock()
	if !ok {
		return RecipeRun{}, false
	}
	return cloneRecipeRun(run), true
}

func (d *DecisionService) captureRecipeFeedback(input DecisionCaptureInput, memo DecisionMemo) error {
	if d == nil || d.store == nil || !d.recipeExecutionTrackingEnabled {
		return nil
	}
	run, ok := d.consumePendingRecipeRun(input.SessionID, input.UserMessage)
	if !ok {
		if input.RecipeRun == nil {
			return nil
		}
		run = cloneRecipeRun(*input.RecipeRun)
	}
	recipe, ok := d.store.Recipe(run.RecipeID)
	if !ok {
		return nil
	}
	actualTools := decisionArchiveToolNames(input.NewMessages)
	steps := buildRecipeRunSteps(recipe, actualTools, input.NewMessages, memo.Outcome)
	feedback := buildRecipeFeedback(memo, steps)
	if input.RecipeFeedback != nil {
		feedback = mergeRecipeFeedback(feedback, *input.RecipeFeedback)
	}
	run.ID = buildDecisionRecipeRunID(run, input, memo)
	run.Namespace = normalizeDecisionNamespace(firstNonEmpty(run.Namespace, memo.Namespace, input.Namespace))
	run.TraceID = strings.TrimSpace(firstNonEmpty(input.TraceID, memo.TraceID, run.TraceID))
	run.TurnID = strings.TrimSpace(firstNonEmpty(input.TurnID, memo.TurnID, run.TurnID))
	run.IntentKey = strings.TrimSpace(firstNonEmpty(run.IntentKey, recipe.IntentKey, memo.IntentKey))
	if run.SelectedAt.IsZero() {
		run.SelectedAt = effectiveDecisionTimestamp(input.TurnStartedAt, memo.CreatedAt)
	}
	run.CompletedAt = effectiveDecisionTimestamp(input.TurnFinishedAt, memo.LastUsedAt, memo.CreatedAt)
	if isZeroDecisionEnvFingerprint(run.EnvironmentFingerprint) {
		run.EnvironmentFingerprint = cloneDecisionEnvFingerprint(firstNonZeroEnv(run.EnvironmentFingerprint, memo.Environment, input.Environment))
	}
	run.Steps = normalizeRecipeRunSteps(steps)
	run.Feedback = normalizeRecipeFeedback(feedback)
	run.ActualTools = uniqueStrings(actualTools)
	if _, err := d.store.UpsertRecipeRun(run); err != nil {
		return err
	}
	if err := d.rebuildRecipeStatsFromRuns(run.Namespace); err != nil {
		return err
	}
	d.recordRecipeFeedbackMetrics(run.Feedback, run.Steps)
	d.maybeRedistillAfterRecipeFeedback(run.Namespace)
	return nil
}

func buildDecisionRecipeRunID(run RecipeRun, input DecisionCaptureInput, memo DecisionMemo) string {
	seed := strings.Join([]string{
		normalizeDecisionNamespace(firstNonEmpty(run.Namespace, memo.Namespace, input.Namespace)),
		strings.TrimSpace(run.RecipeID),
		strings.TrimSpace(firstNonEmpty(input.SessionID, memo.SessionID, run.SessionID)),
		strings.TrimSpace(firstNonEmpty(input.TurnID, memo.TurnID, run.TurnID)),
		strings.TrimSpace(firstNonEmpty(input.TraceID, memo.TraceID, run.TraceID)),
		strings.TrimSpace(memo.ID),
	}, "|")
	sum := sha1.Sum([]byte(seed))
	return "recipe-run-" + hex.EncodeToString(sum[:8])
}

func firstNonZeroEnv(values ...DecisionEnvFingerprint) DecisionEnvFingerprint {
	for _, value := range values {
		normalized := normalizeDecisionEnvFingerprint(value)
		if !isZeroDecisionEnvFingerprint(normalized) {
			return normalized
		}
	}
	return DecisionEnvFingerprint{}
}

func isZeroDecisionEnvFingerprint(env DecisionEnvFingerprint) bool {
	env = normalizeDecisionEnvFingerprint(env)
	return env.OS == "" &&
		env.Platform == "" &&
		env.Shell == "" &&
		env.WorkspaceRoot == "" &&
		env.Provider == "" &&
		env.Model == "" &&
		env.GraphNamespace == "" &&
		env.Domain == "" &&
		env.ToolsetSignature == "" &&
		env.TargetAppOrSite == "" &&
		len(env.PathHints) == 0 &&
		len(env.ToolNames) == 0 &&
		!env.NativePersistent
}

func buildRecipeRunSteps(recipe DecisionRecipe, actualTools []string, messages []llm.Message, outcome string) []RecipeRunStep {
	recipe = normalizeDecisionRecipe(recipe)
	if len(recipe.OrderedActions) == 0 && len(actualTools) == 0 {
		return nil
	}
	allText := strings.ToLower(renderDecisionMessages(messages))
	steps := make([]RecipeRunStep, 0, max(len(recipe.OrderedActions), len(actualTools)))
	usedActual := make([]bool, len(actualTools))
	for index, expected := range recipe.OrderedActions {
		step := RecipeRunStep{ExpectedStep: normalizeRecipeStep(expected), ValidationResult: RecipeValidationUnknown}
		expectedTool := decisionLookup(expected.ToolName)
		if index < len(actualTools) {
			step.ActualTool = strings.TrimSpace(actualTools[index])
		}
		matchedIndex := -1
		if expectedTool != "" {
			for actualIndex, actualTool := range actualTools {
				if decisionLookup(actualTool) == expectedTool {
					matchedIndex = actualIndex
					break
				}
			}
		}
		switch {
		case expectedTool == "" && step.ActualTool != "":
			step.Matched = true
			usedActual[index] = true
		case matchedIndex == index && matchedIndex >= 0:
			step.Matched = true
			usedActual[matchedIndex] = true
		case matchedIndex > index:
			step.ActualTool = actualTools[matchedIndex]
			step.Matched = true
			step.DeviationReason = "tool matched but execution order drifted"
			usedActual[matchedIndex] = true
		case matchedIndex >= 0:
			step.ActualTool = actualTools[matchedIndex]
			step.Matched = true
			step.DeviationReason = "tool repeated earlier than expected"
			usedActual[matchedIndex] = true
		case step.ActualTool == "":
			step.Skipped = true
			step.DeviationReason = "expected step was skipped"
		default:
			step.DeviationReason = "actual tool deviated from recipe"
		}
		step.ValidationResult = recipeStepValidationResult(step.ExpectedStep, allText, outcome)
		steps = append(steps, normalizeRecipeRunStep(step))
	}
	for index, actualTool := range actualTools {
		if index < len(usedActual) && usedActual[index] {
			continue
		}
		steps = append(steps, RecipeRunStep{
			ActualTool:       strings.TrimSpace(actualTool),
			DeviationReason:  "unexpected tool outside recipe",
			ValidationResult: RecipeValidationUnknown,
		})
	}
	return normalizeRecipeRunSteps(steps)
}

func recipeStepValidationResult(step RecipeStep, allText string, outcome string) string {
	validation := decisionLookup(step.Validation)
	if validation == "" {
		return RecipeValidationUnknown
	}
	if strings.Contains(allText, validation) {
		return RecipeValidationPassed
	}
	switch normalizeDecisionOutcome(outcome) {
	case DecisionOutcomeSuccess:
		return RecipeValidationPassed
	case DecisionOutcomeFailure, DecisionOutcomePartial:
		return RecipeValidationFailed
	default:
		return RecipeValidationUnknown
	}
}

func buildRecipeFeedback(memo DecisionMemo, steps []RecipeRunStep) RecipeFeedback {
	feedback := RecipeFeedback{
		Outcome:        normalizeDecisionOutcome(firstNonEmpty(memo.Outcome, DecisionOutcomeSuccess)),
		FailureReasons: append([]string(nil), memo.FailureReasons...),
		HumanBlocked:   memo.HumanBlocked || normalizeDecisionOutcome(memo.Outcome) == DecisionOutcomeAwaitingHuman,
	}
	matchedCount := 0
	for _, step := range steps {
		if step.Matched {
			matchedCount++
		}
		if step.ValidationResult == RecipeValidationFailed {
			feedback.ValidationFailures = append(feedback.ValidationFailures, firstNonEmpty(step.ExpectedStep.Validation, step.DeviationReason, step.ActualTool))
		}
		if step.DeviationReason != "" {
			feedback.FailureReasons = append(feedback.FailureReasons, step.DeviationReason)
		}
	}
	feedback.FailureReasons = uniqueStrings(summarizeDecisionTexts(feedback.FailureReasons, 160))
	feedback.ValidationFailures = uniqueStrings(summarizeDecisionTexts(feedback.ValidationFailures, 160))
	feedback.ReuseHelpful = !feedback.HumanBlocked && matchedCount > 0 && (feedback.Outcome == DecisionOutcomeSuccess || feedback.Outcome == DecisionOutcomePartial)
	return normalizeRecipeFeedback(feedback)
}

func mergeRecipeFeedback(base RecipeFeedback, override RecipeFeedback) RecipeFeedback {
	merged := normalizeRecipeFeedback(base)
	override = normalizeRecipeFeedback(override)
	merged.Outcome = firstNonEmpty(override.Outcome, merged.Outcome)
	merged.FailureReasons = uniqueStrings(append(merged.FailureReasons, override.FailureReasons...))
	merged.ValidationFailures = uniqueStrings(append(merged.ValidationFailures, override.ValidationFailures...))
	merged.HumanBlocked = merged.HumanBlocked || override.HumanBlocked
	merged.ReuseHelpful = merged.ReuseHelpful || override.ReuseHelpful
	return merged
}

func (d *DecisionService) rebuildRecipeStatsFromRuns(namespace string) error {
	if d == nil || d.store == nil {
		return nil
	}
	ns := normalizeDecisionNamespace(namespace)
	runs := d.store.ListRecipeRuns(ns)
	byRecipe := make(map[string][]RecipeRun, len(runs))
	for _, run := range runs {
		byRecipe[run.RecipeID] = append(byRecipe[run.RecipeID], run)
	}
	for _, recipe := range d.store.ListRecipes(ns) {
		recipe = normalizeDecisionRecipe(recipe)
		recipeRuns := byRecipe[recipe.ID]
		selectedCount := 0
		appliedCount := 0
		successCount := 0
		partialCount := 0
		failureCount := 0
		humanBlockedCount := 0
		deviationCount := 0
		avoidPatterns := append([]string(nil), recipe.AvoidPatterns...)
		validationChecklist := append([]string(nil), recipe.ValidationChecklist...)
		var lastSelectedAt time.Time
		var lastAppliedAt time.Time
		var lastOutcomeAt time.Time
		lastOutcome := recipe.LastOutcome
		for _, run := range recipeRuns {
			selectedCount++
			appliedCount++
			if !run.SelectedAt.IsZero() && (lastSelectedAt.IsZero() || run.SelectedAt.After(lastSelectedAt)) {
				lastSelectedAt = run.SelectedAt.UTC()
			}
			if !run.CompletedAt.IsZero() && (lastAppliedAt.IsZero() || run.CompletedAt.After(lastAppliedAt)) {
				lastAppliedAt = run.CompletedAt.UTC()
			}
			if !run.CompletedAt.IsZero() && (lastOutcomeAt.IsZero() || run.CompletedAt.After(lastOutcomeAt)) {
				lastOutcomeAt = run.CompletedAt.UTC()
				lastOutcome = run.Feedback.Outcome
			}
			switch normalizeDecisionOutcome(run.Feedback.Outcome) {
			case DecisionOutcomeSuccess:
				successCount++
			case DecisionOutcomePartial:
				partialCount++
			case DecisionOutcomeAwaitingHuman:
				humanBlockedCount++
			case DecisionOutcomeFailure, DecisionOutcomeCancelled:
				failureCount++
			}
			if run.Feedback.HumanBlocked && normalizeDecisionOutcome(run.Feedback.Outcome) != DecisionOutcomeAwaitingHuman {
				humanBlockedCount++
			}
			if recipeRunHasDeviation(run) {
				deviationCount++
			}
			avoidPatterns = append(avoidPatterns, run.Feedback.FailureReasons...)
			validationChecklist = append(validationChecklist, run.Feedback.ValidationFailures...)
			for _, step := range run.Steps {
				validationChecklist = append(validationChecklist, step.ExpectedStep.Validation)
			}
		}
		if appliedCount > 0 {
			recipe.SuccessRate = clamp01((float64(successCount) + float64(partialCount)*0.5) / float64(appliedCount))
		}
		recipe.SupportCount = max(recipe.SupportCount, successCount)
		recipe.SelectedCount = selectedCount
		recipe.AppliedCount = appliedCount
		recipe.SuccessCount = successCount
		recipe.PartialCount = partialCount
		recipe.FailureCount = failureCount
		recipe.HumanBlockedCount = humanBlockedCount
		recipe.DeviationCount = deviationCount
		recipe.LastSelectedAt = lastSelectedAt
		recipe.LastAppliedAt = lastAppliedAt
		recipe.LastOutcomeAt = lastOutcomeAt
		recipe.LastOutcome = normalizeDecisionOutcome(lastOutcome)
		recipe.AvoidPatterns = uniqueStrings(summarizeDecisionTexts(avoidPatterns, 140))
		recipe.ValidationChecklist = uniqueStrings(summarizeDecisionTexts(validationChecklist, 140))
		recipe.Status = recipeStatusFromRuns(recipe)
		recomputedConfidence := rebuildRecipeConfidence(recipe)
		if recipe.Status == RecipeStatusActive && recipe.FailureCount <= recipe.SuccessCount {
			recipe.Confidence = maxFloat(recipe.Confidence, recomputedConfidence)
		} else {
			recipe.Confidence = recomputedConfidence
		}
		recipe.UpdatedAt = latestDecisionTime(recipe.UpdatedAt, recipe.LastSelectedAt, recipe.LastAppliedAt, recipe.LastOutcomeAt)
		if _, err := d.store.UpsertRecipe(recipe); err != nil {
			return err
		}
	}
	return nil
}

func rebuildRecipeConfidence(recipe DecisionRecipe) float64 {
	supportScore := clamp01(float64(max(recipe.SupportCount, recipe.SuccessCount)) / float64(max(defaultDecisionRecipeMinSupport+2, 5)))
	deviationPenalty := 0.0
	if recipe.AppliedCount > 0 {
		deviationPenalty = clamp01(float64(recipe.DeviationCount) / float64(recipe.AppliedCount)) * 0.18
	}
	return clamp01(recipe.SuccessRate*0.50 + supportScore*0.25 + clamp01(float64(recipe.SelectedCount)/8.0)*0.10 + clamp01(float64(recipe.AppliedCount)/6.0)*0.10 - deviationPenalty)
}

func recipeStatusFromRuns(recipe DecisionRecipe) string {
	if recipe.AppliedCount >= 3 && recipe.SuccessCount == 0 && recipe.FailureCount >= 2 {
		return RecipeStatusDeprecated
	}
	if recipe.AppliedCount >= 3 && recipe.SuccessRate < 0.5 && recipe.FailureCount > recipe.SuccessCount {
		return RecipeStatusConflicted
	}
	return RecipeStatusActive
}

func recipeRunHasDeviation(run RecipeRun) bool {
	for _, step := range run.Steps {
		if strings.TrimSpace(step.DeviationReason) != "" || step.Skipped {
			return true
		}
	}
	return false
}

func (d *DecisionService) recordRecipeFeedbackMetrics(feedback RecipeFeedback, steps []RecipeRunStep) {
	if d == nil || d.metrics == nil {
		return
	}
	d.metrics.recipeApplied.Add(1)
	if hasRecipeDeviation(steps) {
		d.metrics.recipeDeviations.Add(1)
	}
	switch normalizeDecisionOutcome(feedback.Outcome) {
	case DecisionOutcomeSuccess:
		d.metrics.recipeSuccess.Add(1)
	case DecisionOutcomePartial:
		d.metrics.recipePartial.Add(1)
	case DecisionOutcomeAwaitingHuman:
		d.metrics.recipeHumanBlocked.Add(1)
	case DecisionOutcomeFailure, DecisionOutcomeCancelled:
		d.metrics.recipeFailure.Add(1)
	}
}

func hasRecipeDeviation(steps []RecipeRunStep) bool {
	for _, step := range steps {
		if strings.TrimSpace(step.DeviationReason) != "" || step.Skipped {
			return true
		}
	}
	return false
}

func (d *DecisionService) maybeRedistillAfterRecipeFeedback(namespace string) {
	if d == nil || d.distiller == nil || !d.recipeEnabled {
		return
	}
	stats := d.store.Stats(namespace)
	if stats.RecipeCount == 0 || stats.MemoCount == 0 {
		return
	}
	if stats.MemoCount%5 != 0 && stats.FailureCount < 2 {
		return
	}
	if _, err := d.distiller.DistillAll(namespace); err != nil && d.debugEnabled {
		logDecisionDebug("recipe re-distill after feedback", err)
	}
}

func renderDecisionMessages(messages []llm.Message) string {
	parts := make([]string, 0, len(messages))
	for _, message := range messages {
		text := strings.TrimSpace(messageToContent(message))
		if text == "" {
			continue
		}
		parts = append(parts, text)
	}
	return strings.Join(parts, "\n")
}

func logDecisionDebug(action string, err error) {
	if err == nil {
		return
	}
	// 保持短日志，避免在主链路里放大噪音。
	log.Printf("[MEMORY] %s: %v", strings.TrimSpace(action), err)
}
