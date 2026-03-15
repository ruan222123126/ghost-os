package rss

import (
	"context"
	"strings"
	"time"

	"ghost-os/bridge/agent"
	"ghost-os/bridge/tools"
)

const (
	defaultRSSReportTimeout      = 60 * time.Second
	defaultRSSReportSummaryLimit = 400
)

type RSSReportQuery struct {
	TraceID     string
	TaskID      string
	DossierPath string
}

type RSSReportResult struct {
	ID             string    `json:"id,omitempty"`
	BriefingID     string    `json:"briefing_id,omitempty"`
	Title          string    `json:"title"`
	Summary        string    `json:"summary,omitempty"`
	GeneratedAt    time.Time `json:"generated_at"`
	SavedAt        time.Time `json:"saved_at,omitempty"`
	TraceID        string    `json:"trace_id,omitempty"`
	TaskID         string    `json:"task_id,omitempty"`
	MarkdownPath   string    `json:"markdown_path,omitempty"`
	HighlightCount int       `json:"highlight_count"`
	GroupCount     int       `json:"group_count"`
	SourceGroupIDs []string  `json:"source_group_ids,omitempty"`
}

type rssReportBuilder interface {
	Build(context.Context, RSSReportResult, RSSBriefingResult, []RSSInboxTopicGroup, RSSReportQuery) (string, error)
}

type agentRSSReportBuilder struct {
	store        *ConfigStore
	timeout      time.Duration
	buildRuntime func(*ConfigStore) (agentRuntimeDependencies, error)
	allowedTools []string
}

func (b *agentRSSReportBuilder) Build(
	ctx context.Context,
	report RSSReportResult,
	briefing RSSBriefingResult,
	groups []RSSInboxTopicGroup,
	query RSSReportQuery,
) (string, error) {
	deps, err := b.runtimeDependencies()
	if err != nil {
		return "", err
	}
	defer deps.Close()

	runCtx, cancel := context.WithTimeout(ctx, b.reportTimeout())
	defer cancel()

	scoped := tools.NewScopedCatalog(deps.registry, b.allowedToolNames())
	basePrompt, err := buildSystemPromptForCatalog(deps.cfg, scoped)
	if err != nil {
		return "", err
	}
	systemPrompt := basePrompt + "\n\n" + rssReportInvestigationSystemPrompt
	reportAgent := agent.NewAgent(deps.client, scoped, systemPrompt, deps.cfg.MaxTurns)
	toolGuidance := renderRSSReportToolGuidance(scoped)
	response, err := reportAgent.RunWithTraceID(
		runCtx,
		renderAgentRSSReportPrompt(report, briefing, groups, query, toolGuidance),
		query.TraceID,
	)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(stripMarkdownCodeFence(strings.TrimSpace(response))), nil
}

func (b *agentRSSReportBuilder) runtimeDependencies() (agentRuntimeDependencies, error) {
	if b != nil && b.buildRuntime != nil {
		return b.buildRuntime(b.store)
	}
	return newAgentRuntimeFactory().Build(b.store)
}

func (b *agentRSSReportBuilder) reportTimeout() time.Duration {
	if b != nil && b.timeout > 0 {
		return b.timeout
	}
	return defaultRSSReportTimeout
}

func (b *agentRSSReportBuilder) allowedToolNames() []string {
	if b != nil && len(b.allowedTools) > 0 {
		return append([]string(nil), b.allowedTools...)
	}
	return []string{"script_exec", "read_and_summarize", "web_search", "rss_fetch"}
}

func renderRSSReportToolGuidance(catalog tools.ToolCatalog) string {
	names := tools.CatalogToolNames(catalog)
	if len(names) == 0 {
		return ""
	}
	return "Use available tools when needed to validate important claims, inspect primary sources, and add missing context. Investigation tools for this run: " +
		formatRSSReportToolNames(names) + "."
}

func formatRSSReportToolNames(names []string) string {
	formatted := make([]string, 0, len(names))
	for _, name := range names {
		formatted = append(formatted, "`"+strings.TrimSpace(name)+"`")
	}
	return strings.Join(formatted, ", ")
}

func normalizeRSSReportQuery(query RSSReportQuery) RSSReportQuery {
	query.TraceID = strings.TrimSpace(query.TraceID)
	query.TaskID = strings.TrimSpace(query.TaskID)
	query.DossierPath = strings.TrimSpace(query.DossierPath)
	return query
}

func prepareRSSReportDraft(
	briefing RSSBriefingResult,
	groups []RSSInboxTopicGroup,
	query RSSReportQuery,
	now time.Time,
	rootDir string,
) (RSSReportResult, error) {
	report := RSSReportResult{
		BriefingID:     strings.TrimSpace(briefing.ID),
		Title:          rssReportTitleOrDefault(briefing.Title, now.UTC()),
		Summary:        truncateRunes(strings.TrimSpace(briefing.Summary), defaultRSSReportSummaryLimit),
		GeneratedAt:    now.UTC(),
		SavedAt:        now.UTC(),
		TraceID:        query.TraceID,
		TaskID:         query.TaskID,
		HighlightCount: briefing.HighlightCount,
		GroupCount:     len(groups),
		SourceGroupIDs: collectRSSReportSourceGroupIDs(briefing, groups),
	}
	return normalizeRSSReportResult(report, now.UTC(), rootDir)
}

func collectRSSReportSourceGroupIDs(briefing RSSBriefingResult, groups []RSSInboxTopicGroup) []string {
	seen := make(map[string]struct{}, len(groups))
	out := make([]string, 0, len(groups))
	appendGroupID := func(id string) {
		id = strings.TrimSpace(id)
		if id == "" {
			return
		}
		if _, exists := seen[id]; exists {
			return
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	for _, highlight := range briefing.Highlights {
		appendGroupID(highlight.GroupID)
	}
	for _, group := range groups {
		appendGroupID(group.ID)
	}
	return out
}

func rssReportTitleOrDefault(briefingTitle string, generatedAt time.Time) string {
	title := strings.TrimSpace(briefingTitle)
	if title == "" {
		title = "RSS Report"
	}
	suffix := "unknown-time"
	if !generatedAt.IsZero() {
		suffix = generatedAt.UTC().Format("2006-01-02 15:04 UTC")
	}
	return title + " Report - " + suffix
}

func stripMarkdownCodeFence(text string) string {
	trimmed := strings.TrimSpace(text)
	if !strings.HasPrefix(trimmed, "```") {
		return trimmed
	}
	lines := strings.Split(trimmed, "\n")
	if len(lines) == 0 {
		return trimmed
	}
	lines = lines[1:]
	if len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "```" {
		lines = lines[:len(lines)-1]
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}
