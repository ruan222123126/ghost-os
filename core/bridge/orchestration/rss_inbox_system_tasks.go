package orchestration

import (
	"errors"
	"fmt"
	"strings"
	"time"

	bridgeconfig "ghost-os/bridge/config"
)

type rssSystemTaskCoordinator struct {
	configStore   *ConfigStore
	taskStore     *TaskStore
	taskScheduler *TaskScheduler
	rssInitErr    error
}

func newRSSSystemTaskCoordinator(service *bridgeService) rssSystemTaskCoordinator {
	if service == nil {
		return rssSystemTaskCoordinator{}
	}
	return rssSystemTaskCoordinator{
		configStore:   service.configStore,
		taskStore:     service.taskStore,
		taskScheduler: service.taskScheduler,
		rssInitErr:    service.rssInitErr,
	}
}

func (c rssSystemTaskCoordinator) syncPollTask() error {
	task, enabled, err := c.planPollTask()
	if err != nil {
		return err
	}
	return c.syncTask(defaultRSSPollTaskID, task, enabled)
}

func (c rssSystemTaskCoordinator) syncBriefingTask() error {
	task, enabled, err := c.planBriefingTask()
	if err != nil {
		return err
	}
	return c.syncTask(defaultRSSBriefingTaskID, task, enabled)
}

func (c rssSystemTaskCoordinator) planPollTask() (ScheduledTask, bool, error) {
	cfg, err := c.loadConfig()
	if err != nil {
		return ScheduledTask{}, false, err
	}
	if !cfg.RSS.PollEnabled {
		return ScheduledTask{}, false, nil
	}
	task, err := buildRSSPollScheduledTask(cfg.RSS)
	return task, true, err
}

func (c rssSystemTaskCoordinator) planBriefingTask() (ScheduledTask, bool, error) {
	cfg, err := c.loadConfig()
	if err != nil {
		return ScheduledTask{}, false, err
	}
	if !cfg.RSS.BriefingEnabled {
		return ScheduledTask{}, false, nil
	}
	task, err := buildRSSBriefingScheduledTask(cfg.RSS)
	return task, true, err
}

func (c rssSystemTaskCoordinator) loadConfig() (Config, error) {
	if c.taskStore == nil || c.taskScheduler == nil {
		return Config{}, nil
	}
	if c.rssInitErr != nil {
		return Config{}, c.rssInitErr
	}
	if c.configStore != nil {
		return c.configStore.Config()
	}
	return bridgeconfig.Load()
}

func (c rssSystemTaskCoordinator) syncTask(id string, task ScheduledTask, enabled bool) error {
	if c.taskStore == nil || c.taskScheduler == nil {
		return nil
	}
	if !enabled {
		_ = c.taskScheduler.Unregister(id)
		if err := c.taskStore.DeleteTask(id); err != nil && !errors.Is(err, ErrTaskNotFound) {
			return err
		}
		return nil
	}
	if err := c.taskStore.SaveTask(&task); err != nil {
		return err
	}
	return c.taskScheduler.Upsert(task)
}

func buildRSSPollScheduledTask(cfg RSSConfig) (ScheduledTask, error) {
	interval := cfg.PollInterval
	if interval <= 0 {
		interval = defaultRSSPollInterval
	}
	task := ScheduledTask{
		ID:           defaultRSSPollTaskID,
		TaskKind:     taskKindSystemAction,
		Action:       busActionRSSInboxPoll,
		ActionParams: rssInboxPollParamsToMap(rssInboxPollParams{MaxItemsPerFeed: cfg.PollMaxItemsPerFeed, AIBatchSize: cfg.AIBatchSize}),
		ScheduleType: taskScheduleTypeInterval,
		Enabled:      cfg.PollEnabled,
		CreatedAt:    time.Now().UTC(),
	}
	return finalizeRSSSystemTask(task, interval, defaultRSSPollInterval)
}

func buildRSSBriefingScheduledTask(cfg RSSConfig) (ScheduledTask, error) {
	interval := cfg.BriefingInterval
	if interval <= 0 {
		interval = defaultRSSBriefingInterval
	}
	task := ScheduledTask{
		ID:       defaultRSSBriefingTaskID,
		TaskKind: taskKindSystemAction,
		Action:   busActionRSSBriefingBuild,
		ActionParams: rssBriefingParamsToMap(rssBriefingParams{
			WindowHours:     defaultRSSAggregateWindowHours,
			GroupLimit:      defaultRSSBriefingGroupLimit,
			ItemLimit:       defaultRSSAggregateItemLimit,
			ItemsPerGroup:   3,
			HighlightsLimit: defaultRSSBriefingHighlightsLimit,
		}),
		ScheduleType: taskScheduleTypeInterval,
		Enabled:      cfg.BriefingEnabled,
		CreatedAt:    time.Now().UTC(),
	}
	return finalizeRSSSystemTask(task, interval, defaultRSSBriefingInterval)
}

func finalizeRSSSystemTask(task ScheduledTask, interval time.Duration, fallback time.Duration) (ScheduledTask, error) {
	task.IntervalSeconds = int(interval / time.Second)
	if task.IntervalSeconds <= 0 {
		task.IntervalSeconds = int(fallback / time.Second)
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

func decodeRSSBriefingParams(input map[string]any) (rssBriefingParams, error) {
	params, err := decodeActionParamsMap[rssBriefingParams](input)
	if err != nil {
		return rssBriefingParams{}, err
	}
	if params.WindowHours < 0 || params.GroupLimit < 0 || params.ItemLimit < 0 {
		return rssBriefingParams{}, fmt.Errorf("briefing limits must be >= 0")
	}
	if params.ItemsPerGroup < 0 || params.HighlightsLimit < 0 {
		return rssBriefingParams{}, fmt.Errorf("briefing limits must be >= 0")
	}
	return params, nil
}

func rssBriefingParamsToMap(params rssBriefingParams) map[string]any {
	out := map[string]any{}
	if params.FeedID != "" {
		out["feed_id"] = strings.TrimSpace(params.FeedID)
	}
	if params.Tag != "" {
		out["tag"] = strings.TrimSpace(params.Tag)
	}
	if params.Importance != "" {
		out["importance"] = strings.TrimSpace(params.Importance)
	}
	if params.WindowHours > 0 {
		out["window_hours"] = params.WindowHours
	}
	if params.GroupLimit > 0 {
		out["group_limit"] = params.GroupLimit
	}
	if params.ItemLimit > 0 {
		out["item_limit"] = params.ItemLimit
	}
	if params.ItemsPerGroup > 0 {
		out["items_per_group"] = params.ItemsPerGroup
	}
	if params.HighlightsLimit > 0 {
		out["highlights_limit"] = params.HighlightsLimit
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

func formatRSSBriefingPreview(result RSSBriefingResult) string {
	preview := fmt.Sprintf(
		"rss briefing highlights=%d groups=%d title=%s",
		result.HighlightCount,
		result.ScannedGroups,
		truncateRunes(strings.TrimSpace(result.Title), 80),
	)
	if result.Report != nil && strings.TrimSpace(result.Report.MarkdownPath) != "" {
		preview += " report=" + truncateRunes(strings.TrimSpace(result.Report.MarkdownPath), 120)
	}
	if strings.TrimSpace(result.ReportError) != "" {
		preview += " report_error=" + truncateRunes(strings.TrimSpace(result.ReportError), 80)
	}
	return preview
}
