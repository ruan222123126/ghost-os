package decision

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"ghost-os/bridge/agent"
	"ghost-os/bridge/llm"
)

func (d *DecisionService) Rebuild(opts DecisionRebuildOptions) (DecisionRebuildStats, error) {
	stats := DecisionRebuildStats{Namespace: normalizeDecisionNamespace(opts.Namespace)}
	if d == nil || d.store == nil || d.cold == nil {
		return stats, nil
	}
	if opts.ResetNamespace && !opts.DryRun {
		if err := d.store.ResetNamespace(stats.Namespace); err != nil {
			return stats, err
		}
	}
	if !opts.RebuildMemos && !opts.IncludeRecipes {
		return stats, nil
	}

	rebuiltMemos := make([]DecisionMemo, 0, 32)
	if opts.RebuildMemos {
		archives, err := d.cold.ListArchives(opts.TimeRange)
		if err != nil {
			return stats, err
		}
		archives, stats.CursorCheckpoint = decisionArchiveWindow(archives, opts)
		for _, archive := range archives {
			stats.SessionsScanned++
			input, ok := d.captureInputFromArchive(stats.Namespace, archive)
			if !ok {
				stats.SkippedEntries++
				continue
			}
			memo := d.buildDecisionMemo(input)
			if memo.IntentSummary == "" {
				memo.IntentSummary = summarizeDecisionText(input.UserMessage, decisionOutcomeSummaryMaxLen)
				memo = normalizeDecisionMemo(memo)
			}
			rebuiltMemos = append(rebuiltMemos, memo)
			stats.MemosCaptured++
			if opts.DryRun {
				continue
			}
			if _, err := d.store.UpsertMemo(memo); err != nil {
				return stats, fmt.Errorf("upsert decision archive memo: %w", err)
			}
		}

		nodeIDs, err := d.cold.ListMarkdownNodes()
		if err != nil {
			return stats, err
		}
		for _, nodeID := range nodeIDs {
			node, err := d.cold.LoadMarkdownNode(nodeID)
			if err != nil {
				continue
			}
			ts := markdownNodeTimestamp(node)
			if opts.TimeRange != nil && !opts.TimeRange.Contains(ts) {
				continue
			}
			stats.MarkdownScanned++
			memo, ok := decisionMemoFromMarkdownNode(stats.Namespace, node)
			if !ok {
				stats.SkippedEntries++
				continue
			}
			rebuiltMemos = append(rebuiltMemos, memo)
			stats.MemosCaptured++
			if opts.DryRun {
				continue
			}
			if _, err := d.store.UpsertMemo(memo); err != nil {
				return stats, fmt.Errorf("upsert decision markdown memo: %w", err)
			}
		}

		if !opts.DryRun && stats.MemosCaptured > 0 {
			if err := d.store.Persist(); err != nil {
				return stats, fmt.Errorf("persist rebuilt decision memos: %w", err)
			}
		}
	}

	if !opts.IncludeRecipes {
		return stats, nil
	}
	if opts.DryRun {
		baseMemos := d.store.ListMemos(stats.Namespace)
		if opts.ResetNamespace {
			baseMemos = nil
		}
		if opts.RebuildMemos {
			baseMemos = normalizeDecisionMemos(append(baseMemos, rebuiltMemos...))
		}
		distiller := d.distiller
		if distiller == nil {
			distiller = NewDecisionDistiller(d, d.recipeInterval, d.recipeMinSupport)
		}
		recipes, clusters, _, err := distiller.DistillClusters(stats.Namespace, baseMemos)
		if err != nil {
			return stats, err
		}
		existingRecipes := map[string]struct{}{}
		if !opts.ResetNamespace {
			for _, recipe := range d.store.ListRecipes(stats.Namespace) {
				existingRecipes[recipe.ID] = struct{}{}
			}
		}
		for _, recipe := range recipes {
			if _, ok := existingRecipes[recipe.ID]; ok {
				stats.RecipesUpdated++
			} else {
				stats.RecipesCreated++
			}
		}
		stats.ClustersUpdated = len(clusters)
		stats.RecipeRunsCreated, stats.RecipeRunsUpdated, err = d.rebuildHistoricalRecipeRuns(stats.Namespace, baseMemos, true)
		if err != nil {
			return stats, err
		}
		if d.metrics != nil {
			d.metrics.recipeBackfillScanned.Add(uint64(stats.SessionsScanned))
			d.metrics.recipeBackfillCreated.Add(uint64(stats.RecipeRunsCreated))
			d.metrics.recipeBackfillUpdated.Add(uint64(stats.RecipeRunsUpdated))
		}
		return stats, nil
	}

	distiller := d.distiller
	if distiller == nil {
		distiller = NewDecisionDistiller(d, d.recipeInterval, d.recipeMinSupport)
	}
	distillStats, err := distiller.DistillAll(stats.Namespace)
	if err != nil {
		return stats, err
	}
	stats.RecipesCreated += distillStats.RecipesCreated
	stats.RecipesUpdated += distillStats.RecipesUpdated
	stats.ClustersUpdated += distillStats.ClustersBuilt
	stats.RecipeRunsCreated, stats.RecipeRunsUpdated, err = d.rebuildHistoricalRecipeRuns(stats.Namespace, d.store.ListMemos(stats.Namespace), false)
	if err != nil {
		return stats, err
	}
	if d.metrics != nil {
		d.metrics.recipeBackfillScanned.Add(uint64(stats.SessionsScanned))
		d.metrics.recipeBackfillCreated.Add(uint64(stats.RecipeRunsCreated))
		d.metrics.recipeBackfillUpdated.Add(uint64(stats.RecipeRunsUpdated))
	}
	return stats, nil
}

func (d *DecisionService) captureInputFromArchive(namespace string, archive ColdArchive) (DecisionCaptureInput, bool) {
	sessionID := strings.TrimSpace(archive.SessionID)
	userMessage := decisionArchiveUserMessage(archive.Messages)
	if sessionID == "" || userMessage == "" {
		return DecisionCaptureInput{}, false
	}
	toolNames := decisionArchiveToolNames(archive.Messages)
	finishedAt := archive.ArchivedAt.UTC()
	if finishedAt.IsZero() {
		finishedAt = time.Now().UTC()
	}
	return DecisionCaptureInput{
		Namespace:      normalizeDecisionNamespace(namespace),
		SessionID:      sessionID,
		TraceID:        "archive:" + sessionID,
		TurnID:         "archive:" + sessionID,
		UserMessage:    userMessage,
		NewMessages:    llm.CloneMessages(archive.Messages),
		Outcome:        inferDecisionArchiveOutcome(archive.Messages),
		SessionEnded:   decisionArchiveSessionEnded(archive.Messages),
		TurnStartedAt:  finishedAt,
		TurnFinishedAt: finishedAt,
		Environment: DecisionEnvFingerprint{
			GraphNamespace:   normalizeDecisionNamespace(namespace),
			ToolsetSignature: strings.Join(toolNames, ","),
			ToolNames:        toolNames,
		},
	}, true
}

func decisionArchiveUserMessage(messages []llm.Message) string {
	for _, msg := range messages {
		if msg.Role != llm.RoleUser {
			continue
		}
		if text := summarizeDecisionText(messageToContent(msg), decisionOutcomeSummaryMaxLen); text != "" {
			return text
		}
	}
	return ""
}

func decisionArchiveToolNames(messages []llm.Message) []string {
	out := make([]string, 0, 4)
	for _, msg := range messages {
		if msg.Role == llm.RoleAssistant {
			for _, call := range msg.ToolCalls {
				if name := strings.TrimSpace(call.Name); name != "" {
					out = append(out, name)
				}
			}
		}
		if msg.Role != llm.RoleTool {
			continue
		}
		envelope, ok := agent.ParseToolResultEnvelope(msg.Text)
		if !ok {
			continue
		}
		if name := strings.TrimSpace(envelope.Tool); name != "" {
			out = append(out, name)
		}
	}
	return uniqueStrings(out)
}

func inferDecisionArchiveOutcome(messages []llm.Message) string {
	askHumanPending := false
	toolErrors := 0
	for _, msg := range messages {
		switch msg.Role {
		case llm.RoleUser:
			askHumanPending = false
		case llm.RoleAssistant:
			for _, call := range msg.ToolCalls {
				if strings.TrimSpace(call.Name) == "ask_human" {
					askHumanPending = true
				}
			}
		case llm.RoleTool:
			envelope, ok := agent.ParseToolResultEnvelope(msg.Text)
			if ok && strings.EqualFold(strings.TrimSpace(envelope.Status), "error") {
				toolErrors++
			}
		}
	}
	if askHumanPending {
		return DecisionOutcomeAwaitingHuman
	}
	if toolErrors > 0 {
		return DecisionOutcomePartial
	}
	return DecisionOutcomeSuccess
}

func decisionArchiveSessionEnded(messages []llm.Message) bool {
	for _, msg := range messages {
		if msg.Role != llm.RoleAssistant {
			continue
		}
		var payload struct {
			Signal string `json:"signal"`
		}
		if err := json.Unmarshal([]byte(strings.TrimSpace(msg.Text)), &payload); err != nil {
			continue
		}
		if strings.TrimSpace(payload.Signal) == "END_SESSION" {
			return true
		}
	}
	return false
}

func decisionMemoFromMarkdownNode(namespace string, node MarkdownNode) (DecisionMemo, bool) {
	summary := summarizeDecisionText(firstNonEmpty(node.Summary, node.Content), decisionOutcomeSummaryMaxLen)
	if summary == "" && len(node.Anchors) == 0 && len(node.SourceIDs) == 0 {
		return DecisionMemo{}, false
	}
	anchorKeys := make([]string, 0, len(node.Anchors)+len(node.Tags))
	for _, anchor := range node.Anchors {
		anchorKeys = append(anchorKeys, anchor.Key)
	}
	anchorKeys = append(anchorKeys, node.Tags...)
	ts := markdownNodeTimestamp(node)
	memo := DecisionMemo{
		ID:               buildDecisionMarkdownMemoID(namespace, node.ID),
		Namespace:        normalizeDecisionNamespace(namespace),
		SessionID:        strings.TrimSpace(node.SessionID),
		TraceID:          "markdown:" + strings.TrimSpace(node.ID),
		TurnID:           "markdown:" + strings.TrimSpace(node.ID),
		IntentSummary:    summary,
		ProblemSummary:   summary,
		ContextSummary:   summarizeDecisionText(node.Content, decisionOutcomeSummaryMaxLen),
		StrategySummary:  summary,
		Outcome:          DecisionOutcomePartial,
		OutcomeSummary:   "Backfilled from markdown node; treat as weak evidence.",
		ValidationChecks: nil,
		GraphNodeRefs:    append([]string(nil), node.SourceIDs...),
		AnchorKeys:       uniqueStrings(anchorKeys),
		Confidence:       clamp01(maxFloat(node.Confidence*0.85, 0.52)),
		ReuseScore:       0.48,
		DecisionLineage: DecisionLineage{
			SourceEventIDs:    []string{"markdown:" + strings.TrimSpace(node.ID)},
			SourceEvidenceIDs: uniqueStrings(append([]string(nil), node.SourceIDs...)),
			LineageVersion:    decisionLineageVersion,
			LineagePartial:    true,
		},
		CreatedAt:  ts,
		LastUsedAt: ts,
		Environment: DecisionEnvFingerprint{
			GraphNamespace: normalizeDecisionNamespace(namespace),
		},
	}
	return normalizeDecisionMemo(memo), true
}

func buildDecisionMarkdownMemoID(namespace string, nodeID string) string {
	hash := sha1.Sum([]byte(normalizeDecisionNamespace(namespace) + "|markdown|" + strings.TrimSpace(nodeID)))
	return "memo-" + hex.EncodeToString(hash[:8])
}
