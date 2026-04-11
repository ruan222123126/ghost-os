package rss

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	bridgeconfig "ghost-os/bridge/config"
)

// Action constants shared with transport layer.
const (
	ActionInboxPoll     = "RSS_INBOX_POLL"
	ActionInboxList     = "RSS_INBOX_LIST"
	ActionInboxGet      = "RSS_INBOX_GET"
	ActionInboxGroups   = "RSS_INBOX_GROUPS"
	ActionBriefingBuild = "RSS_BRIEFING_BUILD"
	ActionBriefingGet   = "RSS_BRIEFING_GET"
)

// API param types decoded from JSON request bodies.
type InboxPollParams struct {
	MaxItemsPerFeed int    `json:"max_items_per_feed,omitempty"`
	AIBatchSize     int    `json:"ai_batch_size,omitempty"`
	TraceID         string `json:"trace_id,omitempty"`
}

type InboxListParams struct {
	FeedID          string `json:"feed_id,omitempty"`
	Tag             string `json:"tag,omitempty"`
	Importance      string `json:"importance,omitempty"`
	SavedAfter      string `json:"saved_after,omitempty"`
	SavedBefore     string `json:"saved_before,omitempty"`
	PublishedAfter  string `json:"published_after,omitempty"`
	PublishedBefore string `json:"published_before,omitempty"`
	Limit           int    `json:"limit,omitempty"`
}

type InboxGetParams struct {
	ID string `json:"id"`
}

type InboxGroupsParams struct {
	FeedID        string `json:"feed_id,omitempty"`
	Tag           string `json:"tag,omitempty"`
	Importance    string `json:"importance,omitempty"`
	WindowHours   int    `json:"window_hours,omitempty"`
	Limit         int    `json:"limit,omitempty"`
	ItemLimit     int    `json:"item_limit,omitempty"`
	ItemsPerGroup int    `json:"items_per_group,omitempty"`
}

type BriefingParams struct {
	FeedID          string `json:"feed_id,omitempty"`
	Tag             string `json:"tag,omitempty"`
	Importance      string `json:"importance,omitempty"`
	WindowHours     int    `json:"window_hours,omitempty"`
	GroupLimit      int    `json:"group_limit,omitempty"`
	ItemLimit       int    `json:"item_limit,omitempty"`
	ItemsPerGroup   int    `json:"items_per_group,omitempty"`
	HighlightsLimit int    `json:"highlights_limit,omitempty"`
	TraceID         string `json:"trace_id,omitempty"`
	TaskID          string `json:"task_id,omitempty"`
}

var errInvalidRSSInboxParam = errors.New("invalid rss inbox params")

// LogFunc logs action lifecycle events.
type LogFunc func(traceID string, action string, status string, err error)

// ActionHandler implements all RSS action use cases.
type ActionHandler struct {
	inbox   *RSSInboxService
	initErr error
	log     LogFunc
}

// NewActionHandler creates an RSS action handler with the given service.
func NewActionHandler(inbox *RSSInboxService, initErr error, log LogFunc) *ActionHandler {
	return &ActionHandler{inbox: inbox, initErr: initErr, log: log}
}

// InitErr returns the initialization error, if any.
func (h *ActionHandler) InitErr() error {
	if h == nil {
		return nil
	}
	return h.initErr
}

// Reload reinitializes the RSS inbox service from config.
func (h *ActionHandler) Reload(store bridgeconfig.Store) error {
	if h == nil {
		return nil
	}
	service, err := NewRSSInboxServiceFromConfig(store)
	if err != nil {
		h.initErr = err
		h.inbox = nil
		return err
	}
	h.inbox = service
	h.initErr = nil
	return nil
}

func (h *ActionHandler) requireRSSInbox() (*RSSInboxService, int, error) {
	if h == nil || h.inbox == nil {
		if h != nil && h.initErr != nil {
			return nil, http.StatusInternalServerError, h.initErr
		}
		return nil, http.StatusInternalServerError, fmt.Errorf("rss inbox service is not configured")
	}
	return h.inbox, http.StatusOK, nil
}

func mapRSSInboxUsecaseError(err error) int {
	switch {
	case errors.Is(err, errInvalidRSSInboxParam):
		return http.StatusBadRequest
	case errors.Is(err, ErrRSSInboxItemNotFound), errors.Is(err, ErrRSSBriefingNotFound):
		return http.StatusNotFound
	default:
		return http.StatusInternalServerError
	}
}

// ExecuteInboxPollAction handles the RSS_INBOX_POLL action.
func (h *ActionHandler) ExecuteInboxPollAction(ctx context.Context, params InboxPollParams, traceID string) (any, int, error) {
	return h.ExecuteInboxPollUsecase(ctx, params, "", traceID)
}

// ExecuteInboxPollUsecase handles poll with optional taskID for system tasks.
func (h *ActionHandler) ExecuteInboxPollUsecase(ctx context.Context, params InboxPollParams, taskID string, traceID string) (RSSInboxPollResult, int, error) {
	inbox, code, err := h.requireRSSInbox()
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
		h.logAction(traceID, ActionInboxPoll, "error", err)
		return RSSInboxPollResult{}, mapRSSInboxUsecaseError(err), err
	}
	h.logAction(traceID, ActionInboxPoll, "success", nil)
	return result, http.StatusOK, nil
}

// ExecuteInboxListAction handles the RSS_INBOX_LIST action.
func (h *ActionHandler) ExecuteInboxListAction(params InboxListParams, traceID string) (any, int, error) {
	inbox, code, err := h.requireRSSInbox()
	if err != nil {
		return nil, code, err
	}
	filter, err := decodeInboxListFilter(params)
	if err != nil {
		return nil, mapRSSInboxUsecaseError(err), err
	}
	result, err := inbox.List(filter)
	if err != nil {
		h.logAction(traceID, ActionInboxList, "error", err)
		return nil, mapRSSInboxUsecaseError(err), err
	}
	h.logAction(traceID, ActionInboxList, "success", nil)
	return result, http.StatusOK, nil
}

// ExecuteInboxGetAction handles the RSS_INBOX_GET action.
func (h *ActionHandler) ExecuteInboxGetAction(params InboxGetParams, traceID string) (any, int, error) {
	inbox, code, err := h.requireRSSInbox()
	if err != nil {
		return nil, code, err
	}
	id := strings.TrimSpace(params.ID)
	if id == "" {
		return nil, http.StatusBadRequest, wrapRSSInboxParamError(errors.New("rss inbox item id is required"))
	}
	result, err := inbox.Get(id)
	if err != nil {
		h.logAction(traceID, ActionInboxGet, "error", err)
		return nil, mapRSSInboxUsecaseError(err), err
	}
	h.logAction(traceID, ActionInboxGet, "success", nil)
	return result, http.StatusOK, nil
}

// ExecuteInboxGroupsAction handles the RSS_INBOX_GROUPS action.
func (h *ActionHandler) ExecuteInboxGroupsAction(params InboxGroupsParams, traceID string) (any, int, error) {
	inbox, code, err := h.requireRSSInbox()
	if err != nil {
		return nil, code, err
	}
	result, err := inbox.Aggregate(RSSInboxGroupQuery{
		FeedID:        strings.TrimSpace(params.FeedID),
		Tag:           strings.TrimSpace(params.Tag),
		Importance:    strings.TrimSpace(params.Importance),
		WindowHours:   params.WindowHours,
		Limit:         params.Limit,
		ItemLimit:     params.ItemLimit,
		ItemsPerGroup: params.ItemsPerGroup,
	})
	if err != nil {
		h.logAction(traceID, ActionInboxGroups, "error", err)
		return nil, mapRSSInboxUsecaseError(err), err
	}
	h.logAction(traceID, ActionInboxGroups, "success", nil)
	return result, http.StatusOK, nil
}

// ExecuteBriefingBuildAction handles the RSS_BRIEFING_BUILD action.
func (h *ActionHandler) ExecuteBriefingBuildAction(ctx context.Context, params BriefingParams, traceID string) (any, int, error) {
	inbox, code, err := h.requireRSSInbox()
	if err != nil {
		return nil, code, err
	}
	result, err := inbox.BuildAndStoreBriefing(ctx, RSSBriefingQuery{
		FeedID:          strings.TrimSpace(params.FeedID),
		Tag:             strings.TrimSpace(params.Tag),
		Importance:      strings.TrimSpace(params.Importance),
		WindowHours:     params.WindowHours,
		GroupLimit:      params.GroupLimit,
		ItemLimit:       params.ItemLimit,
		ItemsPerGroup:   params.ItemsPerGroup,
		HighlightsLimit: params.HighlightsLimit,
		TraceID:         firstNonEmptyString(strings.TrimSpace(traceID), strings.TrimSpace(params.TraceID)),
		TaskID:          strings.TrimSpace(params.TaskID),
	})
	if err != nil {
		h.logAction(traceID, ActionBriefingBuild, "error", err)
		return nil, mapRSSInboxUsecaseError(err), err
	}
	h.logAction(traceID, ActionBriefingBuild, "success", nil)
	return result, http.StatusOK, nil
}

// ExecuteBriefingGetAction handles the RSS_BRIEFING_GET action.
func (h *ActionHandler) ExecuteBriefingGetAction(traceID string) (any, int, error) {
	inbox, code, err := h.requireRSSInbox()
	if err != nil {
		return nil, code, err
	}
	result, err := inbox.LatestBriefing()
	if err != nil {
		h.logAction(traceID, ActionBriefingGet, "error", err)
		return nil, mapRSSInboxUsecaseError(err), err
	}
	h.logAction(traceID, ActionBriefingGet, "success", nil)
	return result, http.StatusOK, nil
}

func (h *ActionHandler) logAction(traceID, action, status string, err error) {
	if h.log != nil {
		h.log(traceID, action, status, err)
	}
}
