package rss

import (
	"context"
	"fmt"
	"strings"
	"time"
)

type rssReportStoreContract interface {
	Save(RSSReportResult, string) (RSSReportResult, error)
	RootDir() string
}

type rssReportWorkflow struct {
	store   rssReportStoreContract
	builder rssReportBuilder
	now     func() time.Time
}

func (s *RSSInboxService) reportWorkflow() rssReportWorkflow {
	return rssReportWorkflow{
		store:   s.reportStore,
		builder: s.reportBuilder,
		now:     s.currentTime,
	}
}

func (w rssReportWorkflow) BuildAndStore(
	ctx context.Context,
	briefing RSSBriefingResult,
	groups []RSSInboxTopicGroup,
	query RSSReportQuery,
) (RSSReportResult, error) {
	if err := w.validate(); err != nil {
		return RSSReportResult{}, err
	}
	query = normalizeRSSReportQuery(query)
	report, query, err := w.prepareDraft(briefing, groups, query)
	if err != nil {
		return RSSReportResult{}, err
	}
	if err := writeRSSReportDossier(query.DossierPath, renderRSSReportDossier(report, briefing, groups)); err != nil {
		return RSSReportResult{}, err
	}
	markdown, err := w.buildMarkdown(ctx, report, briefing, groups, query)
	if err != nil {
		return RSSReportResult{}, err
	}
	return w.store.Save(report, finalizeRSSReportMarkdown(markdown, briefing, groups))
}

func (w rssReportWorkflow) prepareDraft(
	briefing RSSBriefingResult,
	groups []RSSInboxTopicGroup,
	query RSSReportQuery,
) (RSSReportResult, RSSReportQuery, error) {
	report, err := prepareRSSReportDraft(briefing, groups, query, w.currentTime().UTC(), w.store.RootDir())
	if err != nil {
		return RSSReportResult{}, RSSReportQuery{}, err
	}
	if query.DossierPath == "" {
		query.DossierPath = rssReportDossierPath(w.store.RootDir(), report.ID)
	}
	return report, query, nil
}

func (w rssReportWorkflow) buildMarkdown(
	ctx context.Context,
	report RSSReportResult,
	briefing RSSBriefingResult,
	groups []RSSInboxTopicGroup,
	query RSSReportQuery,
) (string, error) {
	markdown, err := w.builder.Build(ctx, report, briefing, groups, query)
	if strings.TrimSpace(markdown) != "" {
		return markdown, nil
	}
	if err != nil {
		return "", err
	}
	return "", fmt.Errorf("rss report builder returned empty content")
}

func (w rssReportWorkflow) validate() error {
	if w.store == nil {
		return fmt.Errorf("rss report store is not configured")
	}
	if w.builder == nil {
		return fmt.Errorf("rss report builder is not configured")
	}
	return nil
}

func (w rssReportWorkflow) currentTime() time.Time {
	if w.now != nil {
		return w.now()
	}
	return time.Now()
}
