package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	busActionRSSInboxPoll = "RSS_INBOX_POLL"
	busActionRSSInboxList = "RSS_INBOX_LIST"
	busActionRSSInboxGet  = "RSS_INBOX_GET"
)

type rssInboxPollParams struct {
	MaxItemsPerFeed int    `json:"max_items_per_feed,omitempty"`
	AIBatchSize     int    `json:"ai_batch_size,omitempty"`
	TraceID         string `json:"trace_id,omitempty"`
}

type rssInboxListParams struct {
	FeedID          string `json:"feed_id,omitempty"`
	Tag             string `json:"tag,omitempty"`
	Importance      string `json:"importance,omitempty"`
	SavedAfter      string `json:"saved_after,omitempty"`
	SavedBefore     string `json:"saved_before,omitempty"`
	PublishedAfter  string `json:"published_after,omitempty"`
	PublishedBefore string `json:"published_before,omitempty"`
	Limit           int    `json:"limit,omitempty"`
}

type rssInboxGetParams struct {
	ID string `json:"id"`
}

func (s *bridgeService) requireRSSInbox() (*RSSInboxService, int, error) {
	if s == nil || s.rssInbox == nil {
		if s != nil && s.rssInitErr != nil {
			return nil, http.StatusInternalServerError, s.rssInitErr
		}
		return nil, http.StatusInternalServerError, fmt.Errorf("rss inbox service is not configured")
	}
	return s.rssInbox, http.StatusOK, nil
}

func (s *bridgeService) executeRSSInboxPollAction(ctx context.Context, params rssInboxPollParams, traceID string) (any, int, error) {
	return s.executeRSSInboxPollUsecase(ctx, params, "", traceID)
}

func (s *bridgeService) executeRSSInboxPollUsecase(ctx context.Context, params rssInboxPollParams, taskID string, traceID string) (RSSInboxPollResult, int, error) {
	inbox, code, err := s.requireRSSInbox()
	if err != nil {
		return RSSInboxPollResult{}, code, err
	}
	result, err := inbox.Poll(ctx, RSSInboxPollOptions{
		MaxItemsPerFeed: params.MaxItemsPerFeed,
		AIBatchSize:     params.AIBatchSize,
		TraceID:         strings.TrimSpace(traceID),
		TaskID:          strings.TrimSpace(taskID),
	})
	if err != nil {
		logAction(traceID, busActionRSSInboxPoll, "error", err)
		return RSSInboxPollResult{}, http.StatusInternalServerError, err
	}
	logAction(traceID, busActionRSSInboxPoll, "success", nil)
	return result, http.StatusOK, nil
}

func (s *bridgeService) executeRSSInboxListAction(params rssInboxListParams, traceID string) (any, int, error) {
	inbox, code, err := s.requireRSSInbox()
	if err != nil {
		return nil, code, err
	}
	filter, err := decodeRSSInboxListFilter(params)
	if err != nil {
		logAction(traceID, busActionRSSInboxList, "error", err)
		return nil, http.StatusBadRequest, err
	}
	items, err := inbox.List(filter)
	if err != nil {
		logAction(traceID, busActionRSSInboxList, "error", err)
		return nil, http.StatusInternalServerError, err
	}
	logAction(traceID, busActionRSSInboxList, "success", nil)
	return items, http.StatusOK, nil
}

func (s *bridgeService) executeRSSInboxGetAction(params rssInboxGetParams, traceID string) (any, int, error) {
	inbox, code, err := s.requireRSSInbox()
	if err != nil {
		return nil, code, err
	}
	id := strings.TrimSpace(params.ID)
	if id == "" {
		return nil, http.StatusBadRequest, fmt.Errorf("rss inbox item id is required")
	}
	item, err := inbox.Get(id)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, ErrRSSInboxItemNotFound) {
			status = http.StatusNotFound
		}
		logAction(traceID, busActionRSSInboxGet, "error", err)
		return nil, status, err
	}
	logAction(traceID, busActionRSSInboxGet, "success", nil)
	return item, http.StatusOK, nil
}

func decodeRSSInboxListFilter(params rssInboxListParams) (RSSInboxListFilter, error) {
	filter := RSSInboxListFilter{
		FeedID:     strings.TrimSpace(params.FeedID),
		Tag:        strings.TrimSpace(params.Tag),
		Importance: strings.TrimSpace(params.Importance),
		Limit:      params.Limit,
	}
	var err error
	if filter.SavedAfter, err = parseOptionalTimeParam(params.SavedAfter, "saved_after"); err != nil {
		return RSSInboxListFilter{}, err
	}
	if filter.SavedBefore, err = parseOptionalTimeParam(params.SavedBefore, "saved_before"); err != nil {
		return RSSInboxListFilter{}, err
	}
	if filter.PublishedAfter, err = parseOptionalTimeParam(params.PublishedAfter, "published_after"); err != nil {
		return RSSInboxListFilter{}, err
	}
	if filter.PublishedBefore, err = parseOptionalTimeParam(params.PublishedBefore, "published_before"); err != nil {
		return RSSInboxListFilter{}, err
	}
	return filter, nil
}

func parseOptionalTimeParam(raw string, field string) (time.Time, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return time.Time{}, nil
	}
	parsed, err := time.Parse(time.RFC3339, trimmed)
	if err != nil {
		return time.Time{}, fmt.Errorf("%s must use RFC3339", field)
	}
	return parsed.UTC(), nil
}

func (s *bridgeService) ensureRSSPollTask() error {
	if s == nil || s.taskStore == nil || s.taskScheduler == nil {
		return nil
	}
	if s.rssInitErr != nil {
		return s.rssInitErr
	}
	var (
		cfg Config
		err error
	)
	if s.configStore != nil {
		cfg, err = loadConfigWithRuntime(s.configStore.RuntimeConfig())
	} else {
		cfg, err = LoadConfig()
	}
	if err != nil {
		return err
	}
	if !cfg.RSSPollEnabled {
		_ = s.taskScheduler.Unregister(defaultRSSPollTaskID)
		if err := s.taskStore.DeleteTask(defaultRSSPollTaskID); err != nil && !errors.Is(err, ErrTaskNotFound) {
			return err
		}
		return nil
	}
	task, err := buildRSSPollScheduledTask(cfg)
	if err != nil {
		return err
	}
	if err := s.taskStore.SaveTask(&task); err != nil {
		return err
	}
	return s.taskScheduler.Upsert(task)
}

func buildRSSPollScheduledTask(cfg Config) (ScheduledTask, error) {
	interval := cfg.RSSPollInterval
	if interval <= 0 {
		interval = defaultRSSPollInterval
	}
	task := ScheduledTask{
		ID:           defaultRSSPollTaskID,
		TaskKind:     taskKindSystemAction,
		Action:       busActionRSSInboxPoll,
		ActionParams: rssInboxPollParamsToMap(rssInboxPollParams{MaxItemsPerFeed: cfg.RSSPollMaxItemsPerFeed, AIBatchSize: cfg.RSSAIBatchSize}),
		ScheduleType: taskScheduleTypeInterval,
		Enabled:      cfg.RSSPollEnabled,
		CreatedAt:    time.Now().UTC(),
	}
	task.IntervalSeconds = int(interval / time.Second)
	if task.IntervalSeconds <= 0 {
		task.IntervalSeconds = int(defaultRSSPollInterval / time.Second)
	}
	if task.Enabled {
		nextRun, err := nextTaskRunAt(task, time.Now().UTC())
		if err != nil {
			return ScheduledTask{}, err
		}
		task.NextRunAt = nextRun
	}
	if err := validateTaskDefinition(&task); err != nil {
		return ScheduledTask{}, err
	}
	return task, nil
}

func decodeRSSInboxPollParams(input map[string]any) (rssInboxPollParams, error) {
	params, err := decodeActionParamsMap[rssInboxPollParams](input)
	if err != nil {
		return rssInboxPollParams{}, err
	}
	if params.MaxItemsPerFeed < 0 {
		return rssInboxPollParams{}, fmt.Errorf("max_items_per_feed must be >= 0")
	}
	if params.AIBatchSize < 0 {
		return rssInboxPollParams{}, fmt.Errorf("ai_batch_size must be >= 0")
	}
	return params, nil
}

func rssInboxPollParamsToMap(params rssInboxPollParams) map[string]any {
	out := map[string]any{}
	if params.MaxItemsPerFeed > 0 {
		out["max_items_per_feed"] = params.MaxItemsPerFeed
	}
	if params.AIBatchSize > 0 {
		out["ai_batch_size"] = params.AIBatchSize
	}
	return out
}

func formatRSSInboxPollPreview(result RSSInboxPollResult) string {
	return fmt.Sprintf(
		"rss poll feeds=%d failed=%d saved=%d discarded=%d",
		result.FeedsScanned,
		result.FeedsFailed,
		result.ItemsSaved,
		result.ItemsDiscarded,
	)
}
