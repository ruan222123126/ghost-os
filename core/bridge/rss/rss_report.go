package rss

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"ghost-os/bridge/agent"
	"ghost-os/bridge/tools"
)

const defaultRSSReportTimeout = 60 * time.Second

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

func (s *RSSInboxService) BuildAndStoreReport(ctx context.Context, briefing RSSBriefingResult, groups []RSSInboxTopicGroup, query RSSReportQuery) (RSSReportResult, error) {
	if s == nil || s.reportStore == nil {
		return RSSReportResult{}, fmt.Errorf("rss report store is not configured")
	}
	if s.reportBuilder == nil {
		return RSSReportResult{}, fmt.Errorf("rss report builder is not configured")
	}

	query = normalizeRSSReportQuery(query)
	report, err := prepareRSSReportDraft(s.reportStore, briefing, groups, query, s.currentTime().UTC())
	if err != nil {
		return RSSReportResult{}, err
	}
	if query.DossierPath == "" {
		query.DossierPath = rssReportDossierPath(filepath.Dir(s.reportStore.path), report.ID)
	}
	dossier := renderRSSReportDossier(report, briefing, groups)
	if err := writeRSSReportDossier(query.DossierPath, dossier); err != nil {
		return RSSReportResult{}, err
	}
	markdown, err := s.reportBuilder.Build(ctx, report, briefing, groups, query)
	if strings.TrimSpace(markdown) == "" {
		markdown = fallbackRSSReportMarkdown(report, briefing, groups)
	}
	if strings.TrimSpace(markdown) == "" {
		if err != nil {
			return RSSReportResult{}, err
		}
		return RSSReportResult{}, fmt.Errorf("rss report builder returned empty content")
	}
	markdown = finalizeRSSReportMarkdown(markdown, briefing, groups)

	return s.reportStore.Save(report, markdown)
}

func (b *agentRSSReportBuilder) Build(ctx context.Context, report RSSReportResult, briefing RSSBriefingResult, groups []RSSInboxTopicGroup, query RSSReportQuery) (string, error) {
	deps, err := b.runtimeDependencies()
	if err != nil {
		return "", err
	}
	defer deps.Close()

	timeout := b.timeout
	if timeout <= 0 {
		timeout = defaultRSSReportTimeout
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	scoped := tools.NewScopedCatalog(deps.registry, b.allowedToolNames())
	basePrompt, err := buildSystemPromptForCatalog(deps.cfg, scoped)
	if err != nil {
		return "", err
	}
	systemPrompt := basePrompt + "\n\n" + rssReportInvestigationSystemPrompt
	reportAgent := agent.NewAgent(deps.client, scoped, systemPrompt, deps.cfg.MaxTurns)
	response, err := reportAgent.RunWithTraceID(runCtx, renderAgentRSSReportPrompt(report, briefing, groups, query), query.TraceID)
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

func (b *agentRSSReportBuilder) allowedToolNames() []string {
	if b != nil && len(b.allowedTools) > 0 {
		return append([]string(nil), b.allowedTools...)
	}
	return []string{
		"script_exec",
		"read_and_summarize",
		"web_search",
		"rss_fetch",
	}
}

func normalizeRSSReportQuery(query RSSReportQuery) RSSReportQuery {
	query.TraceID = strings.TrimSpace(query.TraceID)
	query.TaskID = strings.TrimSpace(query.TaskID)
	query.DossierPath = strings.TrimSpace(query.DossierPath)
	return query
}

func renderRSSReportPrompt(briefing RSSBriefingResult, groups []RSSInboxTopicGroup, query RSSReportQuery, cfg Config) string {
	type promptItem struct {
		SourceTitle string `json:"source_title,omitempty"`
		SourceHost  string `json:"source_host,omitempty"`
		Title       string `json:"title,omitempty"`
		Link        string `json:"link,omitempty"`
		Summary     string `json:"summary,omitempty"`
		Importance  string `json:"importance,omitempty"`
		PublishedAt string `json:"published_at,omitempty"`
	}
	type promptGroup struct {
		GroupID          string       `json:"group_id"`
		TopicLabel       string       `json:"topic_label,omitempty"`
		Headline         string       `json:"headline,omitempty"`
		Summary          string       `json:"summary,omitempty"`
		Importance       string       `json:"importance,omitempty"`
		ItemCount        int          `json:"item_count"`
		FeedCount        int          `json:"feed_count"`
		LatestActivityAt string       `json:"latest_activity_at,omitempty"`
		Tags             []string     `json:"tags,omitempty"`
		Items            []promptItem `json:"items,omitempty"`
	}
	payload := struct {
		WorkerModel string            `json:"worker_model"`
		TraceID     string            `json:"trace_id,omitempty"`
		TaskID      string            `json:"task_id,omitempty"`
		Briefing    RSSBriefingResult `json:"briefing"`
		Groups      []promptGroup     `json:"groups"`
	}{
		WorkerModel: effectiveWorkerModel(cfg),
		TraceID:     query.TraceID,
		TaskID:      query.TaskID,
		Briefing: RSSBriefingResult{
			ID:             briefing.ID,
			Title:          briefing.Title,
			Summary:        briefing.Summary,
			GeneratedAt:    briefing.GeneratedAt,
			WindowHours:    briefing.WindowHours,
			ScannedGroups:  briefing.ScannedGroups,
			HighlightCount: briefing.HighlightCount,
			TraceID:        briefing.TraceID,
			TaskID:         briefing.TaskID,
			Highlights:     cloneRSSBriefingHighlights(briefing.Highlights),
		},
		Groups: make([]promptGroup, 0, len(groups)),
	}
	for i := range payload.Briefing.Highlights {
		payload.Briefing.Highlights[i].Tags = append([]string(nil), payload.Briefing.Highlights[i].Tags...)
	}
	for _, group := range groups {
		entry := promptGroup{
			GroupID:    group.ID,
			TopicLabel: group.TopicLabel,
			Headline:   group.Headline,
			Summary:    group.Summary,
			Importance: group.Importance,
			ItemCount:  group.ItemCount,
			FeedCount:  group.FeedCount,
			Tags:       append([]string(nil), group.Tags...),
			Items:      make([]promptItem, 0, len(group.Items)),
		}
		if !group.LatestActivityAt.IsZero() {
			entry.LatestActivityAt = group.LatestActivityAt.UTC().Format(time.RFC3339)
		}
		for _, item := range group.Items {
			prompt := promptItem{
				SourceTitle: rssReportItemSourceTitle(item),
				SourceHost:  rssReportItemSourceHost(item.ItemLink),
				Title:       item.ItemTitle,
				Link:        item.ItemLink,
				Summary:     rssInboxItemSummary(item),
				Importance:  item.Importance,
			}
			if !item.PublishedAt.IsZero() {
				prompt.PublishedAt = item.PublishedAt.UTC().Format(time.RFC3339)
			}
			entry.Items = append(entry.Items, prompt)
		}
		payload.Groups = append(payload.Groups, entry)
	}
	encoded, _ := json.MarshalIndent(payload, "", "  ")
	return string(encoded)
}

func fallbackRSSReportMarkdown(report RSSReportResult, briefing RSSBriefingResult, groups []RSSInboxTopicGroup) string {
	var builder strings.Builder
	title := strings.TrimSpace(report.Title)
	if title == "" {
		title = rssReportTitleOrDefault(briefing.Title, report.GeneratedAt)
	}
	builder.WriteString("# ")
	builder.WriteString(title)
	builder.WriteString("\n\n")
	if !briefing.GeneratedAt.IsZero() {
		builder.WriteString("_Generated: ")
		builder.WriteString(briefing.GeneratedAt.UTC().Format(time.RFC3339))
		builder.WriteString("_\n\n")
	}

	builder.WriteString("## 发生了什么\n\n")
	if summary := strings.TrimSpace(briefing.Summary); summary != "" {
		builder.WriteString(summary)
		builder.WriteString("\n\n")
	}
	for _, highlight := range briefing.Highlights {
		builder.WriteString("- **")
		builder.WriteString(firstNonEmptyString(highlight.Headline, highlight.TopicLabel, "Untitled signal"))
		builder.WriteString("**")
		if summary := firstNonEmptyString(highlight.Summary, highlight.WhyItMatters); summary != "" {
			builder.WriteString(": ")
			builder.WriteString(summary)
		}
		builder.WriteString("\n")
	}

	builder.WriteString("\n## 为什么重要\n\n")
	for _, highlight := range briefing.Highlights {
		builder.WriteString("- **")
		builder.WriteString(firstNonEmptyString(highlight.TopicLabel, highlight.Headline, "Signal"))
		builder.WriteString("**: ")
		builder.WriteString(firstNonEmptyString(highlight.WhyItMatters, highlight.Summary, "这个主题在当前 RSS 时间窗内出现了足够多的重复信号，值得重点关注。"))
		builder.WriteString("\n")
	}

	builder.WriteString("\n## 机会点\n\n")
	for _, highlight := range briefing.Highlights {
		builder.WriteString("- 跟踪 **")
		builder.WriteString(firstNonEmptyString(highlight.TopicLabel, highlight.Headline, "this topic"))
		builder.WriteString("** 是否会在近期带来产品、合作或分发层面的机会。\n")
	}

	builder.WriteString("\n## 风险与约束\n\n")
	builder.WriteString("- 本报告基于当前时间窗内的 RSS 覆盖生成，可能遗漏后续更正或站外上下文。\n")
	builder.WriteString("- 任何业务、运营或投资动作仍应回到一手来源做确认。\n")
	for _, highlight := range briefing.Highlights {
		builder.WriteString("- **")
		builder.WriteString(firstNonEmptyString(highlight.TopicLabel, highlight.Headline, "Signal"))
		builder.WriteString("** 仍可能处于早期信号阶段，需要继续等待更多来源确认。\n")
	}

	builder.WriteString("\n## 接下来可能会怎样\n\n")
	for _, highlight := range briefing.Highlights {
		builder.WriteString("- 预计 **")
		builder.WriteString(firstNonEmptyString(highlight.TopicLabel, highlight.Headline, "this topic"))
		builder.WriteString("** 如果继续累积信号，下一轮里大概率会出现后续报道。\n")
	}

	builder.WriteString("\n## 值得持续关注的具体信号\n\n")
	for _, group := range groups {
		builder.WriteString("### ")
		builder.WriteString(firstNonEmptyString(group.Headline, group.TopicLabel, group.ID))
		builder.WriteString("\n")
		for _, item := range group.Items {
			builder.WriteString("- ")
			if strings.TrimSpace(item.ItemLink) != "" {
				builder.WriteString("[")
				builder.WriteString(firstNonEmptyString(item.ItemTitle, item.ItemLink))
				builder.WriteString("](")
				builder.WriteString(strings.TrimSpace(item.ItemLink))
				builder.WriteString(")")
			} else {
				builder.WriteString(firstNonEmptyString(item.ItemTitle, "Untitled source"))
			}
			if summary := rssInboxItemSummary(item); summary != "" {
				builder.WriteString(": ")
				builder.WriteString(summary)
			}
			builder.WriteString("\n")
		}
		builder.WriteString("\n")
	}
	return strings.TrimSpace(builder.String())
}

func collectRSSReportSourceGroupIDs(briefing RSSBriefingResult, groups []RSSInboxTopicGroup) []string {
	seen := make(map[string]struct{}, len(groups))
	out := make([]string, 0, len(groups))
	for _, highlight := range briefing.Highlights {
		id := strings.TrimSpace(highlight.GroupID)
		if id == "" {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	for _, group := range groups {
		id := strings.TrimSpace(group.ID)
		if id == "" {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
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

const rssReportInvestigationSystemPrompt = `You are Ghost-OS RSS report investigation agent.

You are producing an end-user markdown report from an RSS briefing dossier.

Workflow requirements:
1. Read the dossier file first.
2. Use tools when needed to validate important claims, inspect linked sources, and gather missing context.
3. Prefer source-backed claims over speculation.
4. Return Markdown only.
5. The entire report must be written in Simplified Chinese.

The final report must cover:
- What happened
- Why it matters
- Opportunities
- Risks / constraints
- What may happen next
- Concrete signals to watch
- Event sources. In the "What happened" section, every major event/highlight must end with its corresponding "出处：..." line mapped to concrete source items.

Use the exact report title provided in the prompt as the first Markdown H1.

Do not mention your tool usage, internal workflow, or chain-of-thought.`

func prepareRSSReportDraft(store *RSSReportStore, briefing RSSBriefingResult, groups []RSSInboxTopicGroup, query RSSReportQuery, now time.Time) (RSSReportResult, error) {
	if store == nil {
		return RSSReportResult{}, fmt.Errorf("rss report store is nil")
	}
	report := RSSReportResult{
		BriefingID:     strings.TrimSpace(briefing.ID),
		Title:          rssReportTitleOrDefault(briefing.Title, now.UTC()),
		Summary:        truncateRunes(strings.TrimSpace(briefing.Summary), 400),
		GeneratedAt:    now.UTC(),
		SavedAt:        now.UTC(),
		TraceID:        query.TraceID,
		TaskID:         query.TaskID,
		HighlightCount: briefing.HighlightCount,
		GroupCount:     len(groups),
		SourceGroupIDs: collectRSSReportSourceGroupIDs(briefing, groups),
	}
	normalized, err := normalizeRSSReportResult(report, now.UTC(), filepath.Dir(store.path))
	if err != nil {
		return RSSReportResult{}, err
	}
	return normalized, nil
}

func renderAgentRSSReportPrompt(report RSSReportResult, briefing RSSBriefingResult, groups []RSSInboxTopicGroup, query RSSReportQuery) string {
	return strings.TrimSpace(fmt.Sprintf(`Read the dossier file at %s first.

You may use script_exec (with tools.read_file/tools.search_files), read_and_summarize, web_search, and rss_fetch to investigate the topics, validate important claims, inspect primary sources, and add missing context.

Return the final report in Markdown only.
Write the entire report in Simplified Chinese.
Use this exact H1 title: %s
Inside the "## 发生了什么" section, every major event/highlight must be followed by its corresponding "出处：..." line with concrete source items and links. Do not move sources into a separate appendix section.

Report context:
- report_title: %s
- report_id: %s
- briefing_id: %s
- highlight_count: %d
- group_count: %d
- trace_id: %s
- task_id: %s`, query.DossierPath, report.Title, report.Title, report.ID, briefing.ID, len(briefing.Highlights), len(groups), query.TraceID, query.TaskID))
}

func renderRSSReportDossier(report RSSReportResult, briefing RSSBriefingResult, groups []RSSInboxTopicGroup) string {
	var builder strings.Builder
	builder.WriteString("# RSS Report Dossier\n\n")
	builder.WriteString("## Report Context\n\n")
	builder.WriteString("- Report ID: ")
	builder.WriteString(report.ID)
	builder.WriteString("\n- Briefing ID: ")
	builder.WriteString(briefing.ID)
	builder.WriteString("\n- Title: ")
	builder.WriteString(firstNonEmptyString(briefing.Title, report.Title))
	builder.WriteString("\n- Summary: ")
	builder.WriteString(firstNonEmptyString(briefing.Summary, report.Summary))
	builder.WriteString("\n")
	if !briefing.GeneratedAt.IsZero() {
		builder.WriteString("- Briefing generated at: ")
		builder.WriteString(briefing.GeneratedAt.UTC().Format(time.RFC3339))
		builder.WriteString("\n")
	}
	builder.WriteString("- Highlight count: ")
	builder.WriteString(fmt.Sprintf("%d", len(briefing.Highlights)))
	builder.WriteString("\n- Group count: ")
	builder.WriteString(fmt.Sprintf("%d", len(groups)))
	builder.WriteString("\n\n## Highlights\n\n")
	if len(briefing.Highlights) == 0 {
		builder.WriteString("- No highlights were generated.\n")
	}
	for _, highlight := range briefing.Highlights {
		builder.WriteString("### ")
		builder.WriteString(firstNonEmptyString(highlight.Headline, highlight.TopicLabel, highlight.GroupID))
		builder.WriteString("\n")
		builder.WriteString("- Group ID: ")
		builder.WriteString(highlight.GroupID)
		builder.WriteString("\n- Topic: ")
		builder.WriteString(firstNonEmptyString(highlight.TopicLabel, "n/a"))
		builder.WriteString("\n- Importance: ")
		builder.WriteString(firstNonEmptyString(highlight.Importance, "normal"))
		builder.WriteString("\n- Summary: ")
		builder.WriteString(firstNonEmptyString(highlight.Summary, "n/a"))
		builder.WriteString("\n- Why it matters: ")
		builder.WriteString(firstNonEmptyString(highlight.WhyItMatters, "n/a"))
		builder.WriteString("\n- Source item count: ")
		builder.WriteString(fmt.Sprintf("%d", highlight.SourceItemCount))
		builder.WriteString("\n- Source feed count: ")
		builder.WriteString(fmt.Sprintf("%d", highlight.SourceFeedCount))
		if len(highlight.Tags) > 0 {
			builder.WriteString("\n- Tags: ")
			builder.WriteString(strings.Join(highlight.Tags, ", "))
		}
		builder.WriteString("\n\n")
	}
	builder.WriteString("## Aggregated Groups\n\n")
	if len(groups) == 0 {
		builder.WriteString("- No groups were available.\n")
	}
	for _, group := range groups {
		builder.WriteString("### ")
		builder.WriteString(firstNonEmptyString(group.Headline, group.TopicLabel, group.ID))
		builder.WriteString("\n")
		builder.WriteString("- Group ID: ")
		builder.WriteString(group.ID)
		builder.WriteString("\n- Topic label: ")
		builder.WriteString(firstNonEmptyString(group.TopicLabel, "n/a"))
		builder.WriteString("\n- Importance: ")
		builder.WriteString(firstNonEmptyString(group.Importance, "normal"))
		builder.WriteString("\n- Summary: ")
		builder.WriteString(firstNonEmptyString(group.Summary, "n/a"))
		builder.WriteString("\n- Item count: ")
		builder.WriteString(fmt.Sprintf("%d", group.ItemCount))
		builder.WriteString("\n- Feed count: ")
		builder.WriteString(fmt.Sprintf("%d", group.FeedCount))
		if len(group.Tags) > 0 {
			builder.WriteString("\n- Tags: ")
			builder.WriteString(strings.Join(group.Tags, ", "))
		}
		if !group.LatestActivityAt.IsZero() {
			builder.WriteString("\n- Latest activity at: ")
			builder.WriteString(group.LatestActivityAt.UTC().Format(time.RFC3339))
		}
		builder.WriteString("\n\n#### Source Items\n\n")
		if len(group.Items) == 0 {
			builder.WriteString("- No items.\n\n")
			continue
		}
		for _, item := range group.Items {
			builder.WriteString("- Source: ")
			builder.WriteString(rssReportItemSourceTitle(item))
			builder.WriteString("\n  Title: ")
			builder.WriteString(firstNonEmptyString(item.ItemTitle, "Untitled"))
			if link := strings.TrimSpace(item.ItemLink); link != "" {
				builder.WriteString("\n  Link: ")
				builder.WriteString(link)
			}
			if summary := rssInboxItemSummary(item); summary != "" {
				builder.WriteString("\n  Summary: ")
				builder.WriteString(summary)
			}
			if !item.PublishedAt.IsZero() {
				builder.WriteString("\n  Published at: ")
				builder.WriteString(item.PublishedAt.UTC().Format(time.RFC3339))
			}
			builder.WriteString("\n")
		}
		builder.WriteString("\n")
	}
	return strings.TrimSpace(builder.String())
}

func finalizeRSSReportMarkdown(markdown string, briefing RSSBriefingResult, groups []RSSInboxTopicGroup) string {
	trimmed := strings.TrimSpace(markdown)
	if trimmed == "" {
		return ""
	}
	return injectRSSReportSourcesIntoWhatHappened(trimmed, briefing, groups)
}

func injectRSSReportSourcesIntoWhatHappened(markdown string, briefing RSSBriefingResult, groups []RSSInboxTopicGroup) string {
	if rssReportWhatHappenedHasInlineSources(markdown) {
		return markdown
	}
	sectionStart, sectionEnd, sectionText, ok := findRSSReportSection(markdown, "发生了什么", "What happened")
	if !ok {
		return markdown
	}
	groupByID := make(map[string]RSSInboxTopicGroup, len(groups))
	for _, group := range groups {
		groupByID[group.ID] = group
	}
	events := make([]string, 0, len(briefing.Highlights))
	for _, highlight := range briefing.Highlights {
		group, ok := groupByID[strings.TrimSpace(highlight.GroupID)]
		if !ok {
			continue
		}
		if line := buildRSSReportSourceLine(group); line != "" {
			events = append(events, line)
		}
	}
	if len(events) == 0 {
		return markdown
	}

	updated := injectRSSReportSourceLinesIntoSection(sectionText, briefing, events)
	if strings.TrimSpace(updated) == strings.TrimSpace(sectionText) {
		return markdown
	}
	return markdown[:sectionStart] + updated + markdown[sectionEnd:]
}

func rssReportWhatHappenedHasInlineSources(markdown string) bool {
	_, _, section, ok := findRSSReportSection(markdown, "发生了什么", "What happened")
	if !ok {
		return false
	}
	return strings.Contains(section, "出处：")
}

func findRSSReportSection(markdown string, names ...string) (int, int, string, bool) {
	lines := strings.SplitAfter(markdown, "\n")
	offset := 0
	start := -1
	end := len(markdown)
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if start < 0 {
			for _, name := range names {
				if trimmed == "## "+name {
					start = offset
					break
				}
			}
			offset += len(line)
			continue
		}
		if strings.HasPrefix(trimmed, "## ") {
			end = offset
			return start, end, markdown[start:end], true
		}
		offset += len(line)
		if i == len(lines)-1 {
			end = len(markdown)
		}
	}
	if start < 0 {
		return 0, 0, "", false
	}
	return start, end, markdown[start:end], true
}

func injectRSSReportSourceLinesIntoSection(section string, briefing RSSBriefingResult, sourceLines []string) string {
	lines := strings.Split(section, "\n")
	if len(lines) == 0 {
		return section
	}
	inserts := make(map[int]string, len(sourceLines))
	eventStarts := make([]int, 0, len(sourceLines))
	for i := 1; i < len(lines); i++ {
		if isRSSReportEventBlockStart(lines[i]) {
			eventStarts = append(eventStarts, i)
		}
	}
	nextSource := 0
	for idx, start := range eventStarts {
		if nextSource >= len(sourceLines) {
			break
		}
		end := len(lines) - 1
		if idx+1 < len(eventStarts) {
			end = eventStarts[idx+1] - 1
		}
		insertAt := rssReportEventInsertIndex(lines, start, end)
		inserts[insertAt] = sourceLines[nextSource]
		nextSource++
	}

	var builder strings.Builder
	for i, line := range lines {
		if i > 0 {
			builder.WriteString("\n")
		}
		builder.WriteString(line)
		if source, ok := inserts[i]; ok {
			if strings.TrimSpace(line) != "" {
				builder.WriteString("\n")
			}
			builder.WriteString(source)
		}
	}
	if nextSource < len(sourceLines) {
		tail := buildRSSReportFallbackEventBlocks(briefing, sourceLines[nextSource:])
		if strings.TrimSpace(tail) != "" {
			builder.WriteString("\n\n")
			builder.WriteString(tail)
		}
	}
	return strings.TrimSpace(builder.String())
}

func rssReportEventInsertIndex(lines []string, start int, end int) int {
	for i := end; i >= start; i-- {
		if strings.TrimSpace(lines[i]) != "" {
			return i
		}
	}
	return start
}

func isRSSReportEventBlockStart(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return false
	}
	if strings.HasPrefix(trimmed, "**") && strings.HasSuffix(trimmed, "**") {
		return true
	}
	if strings.HasPrefix(trimmed, "- **") {
		return true
	}
	return false
}

func buildRSSReportFallbackEventBlocks(briefing RSSBriefingResult, sourceLines []string) string {
	if len(sourceLines) == 0 {
		return ""
	}
	var builder strings.Builder
	for i, sourceLine := range sourceLines {
		title := fmt.Sprintf("事件 %d", i+1)
		if i < len(briefing.Highlights) {
			title = firstNonEmptyString(briefing.Highlights[i].Headline, briefing.Highlights[i].TopicLabel, title)
		}
		builder.WriteString("- **")
		builder.WriteString(title)
		builder.WriteString("**\n")
		builder.WriteString("  ")
		builder.WriteString(sourceLine)
		if i+1 < len(sourceLines) {
			builder.WriteString("\n")
		}
	}
	return builder.String()
}

func buildRSSReportSourceLine(group RSSInboxTopicGroup) string {
	items := dedupeRSSReportSourceItems(group.Items)
	if len(items) == 0 {
		return ""
	}
	if len(items) > 3 {
		items = items[:3]
	}
	parts := make([]string, 0, len(items))
	for _, item := range items {
		parts = append(parts, formatRSSReportSourceItem(item))
	}
	if len(parts) == 0 {
		return ""
	}
	return "出处：" + strings.Join(parts, "；")
}

func dedupeRSSReportSourceItems(items []RSSInboxItem) []RSSInboxItem {
	if len(items) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(items))
	out := make([]RSSInboxItem, 0, len(items))
	for _, item := range items {
		key := rssReportSourceItemKey(item)
		if key == "" {
			key = fmt.Sprintf("untitled:%d", len(out))
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, item)
	}
	return out
}

func rssReportSourceItemKey(item RSSInboxItem) string {
	if link := strings.TrimSpace(item.ItemLink); link != "" {
		return link
	}
	title := strings.TrimSpace(item.ItemTitle)
	source := rssReportItemSourceTitle(item)
	if title == "" && source == "" {
		return ""
	}
	return title + "::" + source
}

func rssReportItemSourceTitle(item RSSInboxItem) string {
	return firstNonEmptyString(
		strings.TrimSpace(item.SourceTitle),
		rssReportItemSourceHost(item.ItemLink),
		strings.TrimSpace(item.SourceFeedID),
		strings.TrimSpace(item.FeedID),
		"unknown-source",
	)
}

func formatRSSReportSourceItem(item RSSInboxItem) string {
	var builder strings.Builder
	if link := strings.TrimSpace(item.ItemLink); link != "" {
		builder.WriteString("[")
		builder.WriteString(firstNonEmptyString(item.ItemTitle, link))
		builder.WriteString("](")
		builder.WriteString(link)
		builder.WriteString(")")
	} else {
		builder.WriteString(firstNonEmptyString(item.ItemTitle, "Untitled source"))
	}
	builder.WriteString("（")
	builder.WriteString(rssReportItemSourceTitle(item))
	if !item.PublishedAt.IsZero() {
		builder.WriteString("，")
		builder.WriteString(item.PublishedAt.UTC().Format("2006-01-02"))
	}
	builder.WriteString("）")
	return builder.String()
}

func rssReportItemSourceHost(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return ""
	}
	host := strings.TrimSpace(parsed.Hostname())
	if host == "" {
		return ""
	}
	return strings.TrimPrefix(strings.ToLower(host), "www.")
}

func writeRSSReportDossier(path string, dossier string) error {
	resolved := strings.TrimSpace(path)
	if resolved == "" {
		return fmt.Errorf("rss report dossier path is required")
	}
	if err := os.MkdirAll(filepath.Dir(resolved), 0o700); err != nil {
		return fmt.Errorf("create rss report dossier directory %q: %w", filepath.Dir(resolved), err)
	}
	body := strings.TrimSpace(dossier) + "\n"
	tempPath := fmt.Sprintf("%s.tmp-%d", resolved, time.Now().UnixNano())
	if err := os.WriteFile(tempPath, []byte(body), 0o600); err != nil {
		return fmt.Errorf("write temp rss report dossier %q: %w", tempPath, err)
	}
	if err := os.Rename(tempPath, resolved); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("replace rss report dossier %q: %w", resolved, err)
	}
	return nil
}

func rssReportDossierPath(rootDir string, reportID string) string {
	return filepath.Join(rootDir, "dossiers", strings.TrimSpace(reportID)+".source.md")
}
