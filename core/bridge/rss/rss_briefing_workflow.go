package rss

import (
	"context"
	"fmt"
	"log"
	"time"
)

type rssBriefingAggregator func(RSSInboxGroupQuery) (RSSInboxGroupResult, error)

type rssBriefingStoreContract interface {
	Save(RSSBriefingResult) (RSSBriefingResult, error)
	Latest() (RSSBriefingResult, error)
}

type rssBriefingReportCoordinator interface {
	BuildAndStore(context.Context, RSSBriefingResult, []RSSInboxTopicGroup, RSSReportQuery) (RSSReportResult, error)
}

type rssBriefingWorkflow struct {
	aggregate rssBriefingAggregator
	store     rssBriefingStoreContract
	builder   rssBriefingBuilder
	reporter  rssBriefingReportCoordinator
	now       func() time.Time
}

func (s *RSSInboxService) BuildBriefing(ctx context.Context, query RSSBriefingQuery) (RSSBriefingResult, error) {
	result, _, err := s.briefingWorkflow().Build(ctx, query)
	return result, err
}

func (s *RSSInboxService) BuildAndStoreBriefing(ctx context.Context, query RSSBriefingQuery) (RSSBriefingResult, error) {
	return s.briefingWorkflow().BuildAndStore(ctx, query)
}

func (s *RSSInboxService) LatestBriefing() (RSSBriefingResult, error) {
	return s.briefingWorkflow().Latest()
}

func (s *RSSInboxService) briefingWorkflow() rssBriefingWorkflow {
	var reporter rssBriefingReportCoordinator
	if s != nil && s.reportStore != nil && s.reportBuilder != nil {
		reporter = s.reportWorkflow()
	}
	return rssBriefingWorkflow{
		aggregate: s.Aggregate,
		store:     s.briefingStore,
		builder:   s.briefingBuilder,
		reporter:  reporter,
		now:       s.currentTime,
	}
}

func (w rssBriefingWorkflow) Build(
	ctx context.Context,
	query RSSBriefingQuery,
) (RSSBriefingResult, []RSSInboxTopicGroup, error) {
	if err := w.validateBuilder(); err != nil {
		return RSSBriefingResult{}, nil, err
	}
	query = normalizeRSSBriefingQuery(query)
	aggregate, err := w.aggregateGroups(query)
	if err != nil {
		return RSSBriefingResult{}, nil, err
	}
	result := newRSSBriefingResult(query, aggregate.WindowHours, w.currentTime())
	if len(aggregate.Groups) == 0 {
		result.Summary = "No notable items matched the current RSS briefing window."
		return result, aggregate.Groups, nil
	}
	draft, err := w.builder.Build(ctx, aggregate.Groups, query)
	if err != nil {
		return RSSBriefingResult{}, nil, err
	}
	return finalizeRSSBriefingResult(result, draft, aggregate.Groups, query.HighlightsLimit), aggregate.Groups, nil
}

func (w rssBriefingWorkflow) BuildAndStore(ctx context.Context, query RSSBriefingQuery) (RSSBriefingResult, error) {
	if err := w.validateStore(); err != nil {
		return RSSBriefingResult{}, err
	}
	result, groups, err := w.Build(ctx, query)
	if err != nil {
		return RSSBriefingResult{}, err
	}
	result.SavedAt = w.currentTime().UTC()
	saved, err := w.store.Save(result)
	if err != nil {
		return RSSBriefingResult{}, err
	}
	return w.attachReport(ctx, saved, groups)
}

func (w rssBriefingWorkflow) Latest() (RSSBriefingResult, error) {
	if err := w.validateStore(); err != nil {
		return RSSBriefingResult{}, err
	}
	return w.store.Latest()
}

func (w rssBriefingWorkflow) aggregateGroups(query RSSBriefingQuery) (RSSInboxGroupResult, error) {
	if w.aggregate == nil {
		return RSSInboxGroupResult{}, fmt.Errorf("rss inbox service is not configured")
	}
	return w.aggregate(RSSInboxGroupQuery{
		FeedID:        query.FeedID,
		Tag:           query.Tag,
		Importance:    query.Importance,
		WindowHours:   query.WindowHours,
		Limit:         query.GroupLimit,
		ItemLimit:     query.ItemLimit,
		ItemsPerGroup: query.ItemsPerGroup,
	})
}

func (w rssBriefingWorkflow) attachReport(
	ctx context.Context,
	briefing RSSBriefingResult,
	groups []RSSInboxTopicGroup,
) (RSSBriefingResult, error) {
	if w.reporter == nil {
		return briefing, nil
	}
	report, err := w.reporter.BuildAndStore(ctx, briefing, groups, RSSReportQuery{
		TraceID: briefing.TraceID,
		TaskID:  briefing.TaskID,
	})
	if err != nil {
		briefing.ReportError = err.Error()
		log.Printf("rss report build skipped: briefing_id=%s trace_id=%s error=%v", briefing.ID, briefing.TraceID, err)
		return briefing, nil
	}
	briefing.Report = &report
	return briefing, nil
}

func (w rssBriefingWorkflow) validateBuilder() error {
	if w.builder == nil {
		return fmt.Errorf("rss briefing builder is not configured")
	}
	return nil
}

func (w rssBriefingWorkflow) validateStore() error {
	if w.store == nil {
		return fmt.Errorf("rss briefing store is not configured")
	}
	return nil
}

func (w rssBriefingWorkflow) currentTime() time.Time {
	if w.now != nil {
		return w.now()
	}
	return time.Now()
}
