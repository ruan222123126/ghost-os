package decision

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"ghost-os/bridge/agent"
	"ghost-os/bridge/llm"
)

const (
	decisionToolInputSummaryMaxLen  = 180
	decisionToolOutputSummaryMaxLen = 220
	decisionOutcomeSummaryMaxLen    = 220
	decisionFailureReasonMaxLen     = 180
	decisionQuestionSummaryMaxLen   = 160
	decisionValidationSummaryMaxLen = 160
	decisionWorkerHistoryMessageCap = 8
	decisionWorkerNewMessageCap     = 12
	decisionWorkerMessageSummaryLen = 220
)

type decisionWorkerDraft struct {
	IntentKey        string
	IntentSummary    string
	ProblemSummary   string
	ContextSummary   string
	Constraints      []string
	Assumptions      []string
	StrategySummary  string
	KeySteps         []DecisionStep
	AvoidPatterns    []string
	ValidationChecks []string
	NeedsHumanFor    []string
	Confidence       float64
	ReuseScore       float64
}

type decisionToolObservation struct {
	index int
	name  string
}

// CaptureTurn 在单轮消息已成功持久化后，提炼并落盘 decision memo。
func (d *DecisionService) CaptureTurn(input DecisionCaptureInput) error {
	if d == nil || !d.Enabled() || !d.captureOnTurn {
		return nil
	}

	memo := d.buildDecisionMemo(input)
	if _, err := d.store.UpsertMemo(memo); err != nil {
		return fmt.Errorf("upsert decision memo: %w", err)
	}
	if err := d.captureRecipeFeedback(input, memo); err != nil {
		return fmt.Errorf("capture recipe feedback: %w", err)
	}
	if err := d.store.Persist(); err != nil {
		return fmt.Errorf("persist decision memo: %w", err)
	}
	if d.truth != nil {
		if err := d.truth.WriteDecisionMemo(memo, input); err != nil {
			return err
		}
	}
	return nil
}

func (d *DecisionService) buildDecisionMemo(input DecisionCaptureInput) DecisionMemo {
	ruleMemo := d.captureRuleMemo(input)
	workerDraft, err := d.captureWorkerDraft(input)
	if err != nil && d.debugEnabled {
		log.Printf("[MEMORY] decision capture worker fallback to rule-only: session_id=%s trace_id=%s error=%v", strings.TrimSpace(input.SessionID), strings.TrimSpace(input.TraceID), err)
	}
	memo := mergeDecisionMemo(ruleMemo, workerDraft)
	memo.DecisionLineage = mergeDecisionLineage(memo.DecisionLineage, decisionCaptureLineage(input), DecisionLineage{
		DerivedClaimIDs: memo.DerivedClaimIDs,
		LineageSummary:  decisionLineageSummary("capture", memo.DerivedClaimIDs, memo.SourceEvidenceIDs, nil),
	})
	return normalizeDecisionMemo(memo)
}

func (d *DecisionService) captureRuleMemo(input DecisionCaptureInput) DecisionMemo {
	finishedAt := effectiveDecisionTimestamp(input.TurnFinishedAt, input.TurnStartedAt)
	memo := DecisionMemo{
		ID:              buildDecisionMemoID(input),
		Namespace:       normalizeDecisionNamespace(input.Namespace),
		SessionID:       strings.TrimSpace(input.SessionID),
		TraceID:         strings.TrimSpace(input.TraceID),
		TurnID:          strings.TrimSpace(input.TurnID),
		DecisionLineage: decisionCaptureLineage(input),
		IntentSummary:   summarizeDecisionText(input.UserMessage, decisionOutcomeSummaryMaxLen),
		ProblemSummary:  summarizeDecisionText(input.UserMessage, decisionOutcomeSummaryMaxLen),
		Outcome:         normalizeDecisionOutcome(input.Outcome),
		CreatedAt:       finishedAt,
		LastUsedAt:      finishedAt,
		Environment:     normalizeDecisionEnvFingerprint(input.Environment),
	}

	questions := make([]DecisionQuestion, 0, len(input.AnsweredQuestions)+1)
	answeredQuestions := make([]DecisionQuestion, 0, len(input.AnsweredQuestions))
	for _, item := range normalizeDecisionAnsweredQuestions(input.AnsweredQuestions) {
		answeredQuestions = append(answeredQuestions, DecisionQuestion{
			Question:   summarizeDecisionText(item.Prompt, decisionQuestionSummaryMaxLen),
			Answer:     summarizeDecisionText(item.Answer, decisionQuestionSummaryMaxLen),
			AskedAt:    item.AskedAt,
			AnsweredAt: item.AnsweredAt,
		})
	}

	failureReasons := make([]string, 0, 4)
	validationChecks := make([]string, 0, 4)
	needsHumanFor := make([]string, 0, 2)
	assistantSummaries := make([]string, 0, 4)
	toolErrors := 0
	humanBlocked := memo.Outcome == DecisionOutcomeAwaitingHuman
	toolObsByCallID := make(map[string]decisionToolObservation)

	for _, msg := range input.NewMessages {
		switch msg.Role {
		case llm.RoleAssistant:
			if summary := decisionAssistantSummary(msg.Text); summary != "" {
				assistantSummaries = append(assistantSummaries, summary)
			}
			for _, call := range msg.ToolCalls {
				toolName := strings.TrimSpace(call.Name)
				if toolName == "" {
					continue
				}
				memo.ToolsUsed = append(memo.ToolsUsed, DecisionToolUse{
					Name:         toolName,
					Purpose:      decisionToolPurpose(toolName),
					InputSummary: summarizeDecisionToolArgs(toolName, call.Arguments),
				})
				if callID := strings.TrimSpace(call.ID); callID != "" {
					toolObsByCallID[callID] = decisionToolObservation{index: len(memo.ToolsUsed) - 1, name: toolName}
				}
				validationChecks = append(validationChecks, detectValidationChecks(toolName, string(call.Arguments), "")...)
				if toolName == "ask_human" {
					prompt := extractAskHumanPrompt(call.Arguments)
					if prompt != "" {
						humanBlocked = true
						needsHumanFor = append(needsHumanFor, prompt)
						questions = append(questions, DecisionQuestion{
							Question: summarizeDecisionText(prompt, decisionQuestionSummaryMaxLen),
							AskedAt:  finishedAt,
						})
					}
				}
			}
		case llm.RoleTool:
			envelope, ok := agent.ParseToolResultEnvelope(msg.Text)
			if !ok {
				continue
			}
			toolName := strings.TrimSpace(envelope.Tool)
			obs, found := toolObsByCallID[strings.TrimSpace(msg.ToolCallID)]
			if found && obs.index >= 0 && obs.index < len(memo.ToolsUsed) {
				memo.ToolsUsed[obs.index].OutputSummary = summarizeDecisionToolResult(envelope)
			} else {
				memo.ToolsUsed = append(memo.ToolsUsed, DecisionToolUse{
					Name:          toolName,
					Purpose:       decisionToolPurpose(toolName),
					OutputSummary: summarizeDecisionToolResult(envelope),
				})
			}
			validationChecks = append(validationChecks, detectValidationChecks(toolName, "", envelope.Output)...)
			if strings.EqualFold(strings.TrimSpace(envelope.Status), "error") {
				toolErrors++
				failureReasons = append(failureReasons, summarizeDecisionText(firstNonEmpty(envelope.Error, envelope.Output), decisionFailureReasonMaxLen))
			}
		}
	}

	if memo.Outcome == "" {
		memo.Outcome = DecisionOutcomeSuccess
	}
	if memo.Outcome != DecisionOutcomeAwaitingHuman && toolErrors > 0 {
		memo.Outcome = DecisionOutcomePartial
	}
	if memo.Outcome == DecisionOutcomeAwaitingHuman {
		humanBlocked = true
	}

	memo.ToolsUsed = normalizeDecisionToolUses(memo.ToolsUsed)
	memo.QuestionsAsked = normalizeDecisionQuestions(append(questions, answeredQuestions...))
	memo.HumanBlocked = humanBlocked
	memo.NeedsHumanFor = uniqueStrings(summarizeDecisionTexts(needsHumanFor, decisionQuestionSummaryMaxLen))
	memo.FailureReasons = uniqueStrings(summarizeDecisionTexts(failureReasons, decisionFailureReasonMaxLen))
	memo.ValidationChecks = uniqueStrings(summarizeDecisionTexts(validationChecks, decisionValidationSummaryMaxLen))
	memo.OutcomeSummary = summarizeDecisionOutcome(memo.Outcome, assistantSummaries, memo.FailureReasons, memo.NeedsHumanFor, input.SessionEnded)
	memo.Confidence = decisionRuleConfidence(memo.Outcome)
	memo.ReuseScore = decisionRuleReuseScore(memo.Outcome)
	return normalizeDecisionMemo(memo)
}

func (d *DecisionService) captureWorkerDraft(input DecisionCaptureInput) (decisionWorkerDraft, error) {
	if d == nil || d.workerExtractor == nil {
		return decisionWorkerDraft{}, nil
	}

	memo, err := d.workerExtractor.ExtractDecisionMemo(input)
	if err != nil {
		return decisionWorkerDraft{}, err
	}
	return decisionWorkerDraft{
		IntentKey:        strings.TrimSpace(memo.IntentKey),
		IntentSummary:    summarizeDecisionText(memo.IntentSummary, decisionOutcomeSummaryMaxLen),
		ProblemSummary:   summarizeDecisionText(memo.ProblemSummary, decisionOutcomeSummaryMaxLen),
		ContextSummary:   summarizeDecisionText(memo.ContextSummary, decisionOutcomeSummaryMaxLen),
		Constraints:      uniqueStrings(memo.Constraints),
		Assumptions:      uniqueStrings(memo.Assumptions),
		StrategySummary:  summarizeDecisionText(memo.StrategySummary, decisionOutcomeSummaryMaxLen),
		KeySteps:         normalizeDecisionSteps(memo.KeySteps),
		AvoidPatterns:    uniqueStrings(memo.AvoidPatterns),
		ValidationChecks: uniqueStrings(summarizeDecisionTexts(memo.ValidationChecks, decisionValidationSummaryMaxLen)),
		NeedsHumanFor:    uniqueStrings(summarizeDecisionTexts(memo.NeedsHumanFor, decisionQuestionSummaryMaxLen)),
		Confidence:       clamp01(memo.Confidence),
		ReuseScore:       clamp01(memo.ReuseScore),
	}, nil
}

func mergeDecisionMemo(rule DecisionMemo, worker decisionWorkerDraft) DecisionMemo {
	merged := cloneDecisionMemo(rule)
	merged.IntentKey = firstNonEmpty(worker.IntentKey, merged.IntentKey)
	merged.IntentSummary = firstNonEmpty(worker.IntentSummary, merged.IntentSummary)
	merged.ProblemSummary = firstNonEmpty(worker.ProblemSummary, merged.ProblemSummary)
	merged.ContextSummary = firstNonEmpty(worker.ContextSummary, merged.ContextSummary)
	if len(worker.Constraints) > 0 {
		merged.Constraints = worker.Constraints
	}
	if len(worker.Assumptions) > 0 {
		merged.Assumptions = worker.Assumptions
	}
	merged.StrategySummary = firstNonEmpty(worker.StrategySummary, merged.StrategySummary)
	if len(worker.KeySteps) > 0 {
		merged.KeySteps = worker.KeySteps
	}
	if len(worker.AvoidPatterns) > 0 {
		merged.AvoidPatterns = worker.AvoidPatterns
	}
	merged.ValidationChecks = uniqueStrings(append(cloneStrings(merged.ValidationChecks), worker.ValidationChecks...))
	merged.NeedsHumanFor = uniqueStrings(append(cloneStrings(merged.NeedsHumanFor), worker.NeedsHumanFor...))
	if worker.Confidence > 0 {
		merged.Confidence = worker.Confidence
	}
	if worker.ReuseScore > 0 {
		merged.ReuseScore = worker.ReuseScore
	}
	return normalizeDecisionMemo(merged)
}

func buildDecisionMemoID(input DecisionCaptureInput) string {
	parts := []string{
		normalizeDecisionNamespace(input.Namespace),
		strings.TrimSpace(input.SessionID),
		strings.TrimSpace(input.TraceID),
		strings.TrimSpace(input.TurnID),
		summarizeDecisionText(input.UserMessage, 200),
	}
	hash := sha1.Sum([]byte(strings.Join(parts, "|")))
	return "memo-" + hex.EncodeToString(hash[:8])
}

func summarizeDecisionToolArgs(toolName string, raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var args map[string]any
	if err := json.Unmarshal(raw, &args); err != nil {
		return summarizeDecisionText(string(raw), decisionToolInputSummaryMaxLen)
	}
	preferredKeys := []string{"prompt", "path", "paths", "pattern", "query", "url", "command", "script", "diff", "target", "selector"}
	parts := make([]string, 0, 3)
	for _, key := range preferredKeys {
		value, ok := args[key]
		if !ok {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s=%s", key, summarizeDecisionText(renderDecisionValue(value), 80)))
		if len(parts) >= 3 {
			break
		}
	}
	if len(parts) == 0 {
		parts = append(parts, summarizeDecisionText(renderDecisionValue(args), decisionToolInputSummaryMaxLen))
	}
	return summarizeDecisionText(strings.Join(parts, ", "), decisionToolInputSummaryMaxLen)
}

func summarizeDecisionToolResult(envelope agent.ToolResultEnvelope) string {
	if strings.EqualFold(strings.TrimSpace(envelope.Status), "error") {
		return summarizeDecisionText("error: "+firstNonEmpty(envelope.Error, envelope.Output), decisionToolOutputSummaryMaxLen)
	}
	return summarizeDecisionText(envelope.Output, decisionToolOutputSummaryMaxLen)
}

func summarizeDecisionOutcome(outcome string, assistantSummaries []string, failureReasons []string, needsHumanFor []string, sessionEnded bool) string {
	base := ""
	switch outcome {
	case DecisionOutcomeAwaitingHuman:
		base = "Blocked waiting for human input"
		if len(needsHumanFor) > 0 {
			base += ": " + needsHumanFor[0]
		}
	case DecisionOutcomePartial:
		base = "Completed with tool issues"
		if len(failureReasons) > 0 {
			base += ": " + failureReasons[0]
		}
	case DecisionOutcomeSuccess:
		base = "Completed successfully"
	default:
		base = "Turn completed"
	}
	if len(assistantSummaries) > 0 {
		base += ". " + assistantSummaries[len(assistantSummaries)-1]
	}
	if sessionEnded {
		base += " Session ended."
	}
	return summarizeDecisionText(base, decisionOutcomeSummaryMaxLen)
}

func decisionAssistantSummary(text string) string {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return ""
	}
	var payload struct {
		Signal  string `json:"signal"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal([]byte(trimmed), &payload); err == nil {
		if strings.TrimSpace(payload.Signal) == "END_SESSION" && strings.TrimSpace(payload.Message) != "" {
			return summarizeDecisionText(payload.Message, decisionOutcomeSummaryMaxLen)
		}
	}
	return summarizeDecisionText(trimmed, decisionOutcomeSummaryMaxLen)
}

func decisionToolPurpose(toolName string) string {
	switch strings.TrimSpace(toolName) {
	case "list_files":
		return "inspect workspace structure"
	case "read_file", "read_and_summarize":
		return "inspect file contents"
	case "search_files":
		return "search code or text"
	case "apply_diff":
		return "edit workspace files"
	case "bash_exec":
		return "run shell command"
	case "script_exec":
		return "run script workflow"
	case "web_search":
		return "research the web"
	case "browser_action":
		return "interact with browser UI"
	case "ask_human":
		return "request user input"
	default:
		return ""
	}
}

func extractAskHumanPrompt(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var args struct {
		Prompt string `json:"prompt"`
	}
	if err := json.Unmarshal(raw, &args); err != nil {
		return ""
	}
	return summarizeDecisionText(args.Prompt, decisionQuestionSummaryMaxLen)
}

func detectValidationChecks(toolName string, inputSummary string, outputSummary string) []string {
	toolName = strings.ToLower(strings.TrimSpace(toolName))
	haystack := strings.ToLower(strings.Join([]string{toolName, inputSummary, outputSummary}, " "))
	if haystack == "" {
		return nil
	}
	labels := make([]string, 0, 2)
	appendLabel := func(keyword, label string) {
		if strings.Contains(haystack, keyword) {
			labels = append(labels, label)
		}
	}
	appendLabel("test", "test")
	appendLabel("build", "build")
	appendLabel("lint", "lint")
	appendLabel("check", "check")
	appendLabel("verify", "verify")
	if len(labels) == 0 {
		return nil
	}
	snippet := summarizeDecisionText(firstNonEmpty(inputSummary, outputSummary, toolName), 96)
	checks := make([]string, 0, len(labels))
	for _, label := range uniqueStrings(labels) {
		checks = append(checks, summarizeDecisionText(fmt.Sprintf("%s via %s: %s", label, toolName, snippet), decisionValidationSummaryMaxLen))
	}
	return checks
}

func summarizeDecisionText(text string, maxLen int) string {
	return summarizeLine(strings.Join(strings.Fields(strings.TrimSpace(text)), " "), maxLen)
}

func summarizeDecisionTexts(values []string, maxLen int) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if trimmed := summarizeDecisionText(value, maxLen); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func renderDecisionValue(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case []string:
		return strings.Join(typed, ", ")
	case []any:
		parts := make([]string, 0, len(typed))
		for _, item := range typed {
			parts = append(parts, renderDecisionValue(item))
		}
		return strings.Join(parts, ", ")
	default:
		encoded, err := json.Marshal(value)
		if err != nil {
			return ""
		}
		return string(encoded)
	}
}

func effectiveDecisionTimestamp(values ...time.Time) time.Time {
	for _, value := range values {
		if !value.IsZero() {
			return value.UTC()
		}
	}
	return time.Now().UTC()
}

func decisionRuleConfidence(outcome string) float64 {
	switch normalizeDecisionOutcome(outcome) {
	case DecisionOutcomeSuccess:
		return 0.78
	case DecisionOutcomeAwaitingHuman:
		return 0.74
	case DecisionOutcomePartial:
		return 0.68
	default:
		return 0.62
	}
}

func decisionRuleReuseScore(outcome string) float64 {
	switch normalizeDecisionOutcome(outcome) {
	case DecisionOutcomeSuccess:
		return 0.76
	case DecisionOutcomeAwaitingHuman:
		return 0.72
	case DecisionOutcomePartial:
		return 0.64
	default:
		return 0.58
	}
}

func cloneStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, len(values))
	copy(out, values)
	return out
}
