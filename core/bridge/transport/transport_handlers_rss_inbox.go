package transport

import (
	"net/http"
	"strconv"
	"strings"
)

func (t *transport) handleRSSBriefing(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		traceID := resolveTraceID("", r)
		t.dispatchActionObject(w, r, actionRSSBriefingGet, map[string]any{}, traceID)
	case http.MethodPost:
		var req rssBriefingParams
		if !decodeBodyOrWriteError(w, r, t.maxBodyBytes, &req) {
			return
		}
		traceID := resolveTraceID(req.TraceID, r)
		t.dispatchActionObject(w, r, actionRSSBriefingBuild, req, traceID)
	default:
		writeMethodNotAllowed(w)
	}
}

func (t *transport) handleRSSInboxGroups(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	traceID := resolveTraceID("", r)
	t.dispatchActionObject(w, r, actionRSSInboxGroups, rssInboxGroupsParams{
		FeedID:        strings.TrimSpace(r.URL.Query().Get("feed_id")),
		Tag:           strings.TrimSpace(r.URL.Query().Get("tag")),
		Importance:    strings.TrimSpace(r.URL.Query().Get("importance")),
		WindowHours:   parseOptionalIntQuery(r, "window_hours"),
		Limit:         parseRSSInboxLimit(r),
		ItemLimit:     parseOptionalIntQuery(r, "item_limit"),
		ItemsPerGroup: parseOptionalIntQuery(r, "items_per_group"),
	}, traceID)
}

func (t *transport) handleRSSInbox(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		traceID := resolveTraceID("", r)
		t.dispatchActionObject(w, r, actionRSSInboxList, rssInboxListParams{
			FeedID:          strings.TrimSpace(r.URL.Query().Get("feed_id")),
			Tag:             strings.TrimSpace(r.URL.Query().Get("tag")),
			Importance:      strings.TrimSpace(r.URL.Query().Get("importance")),
			SavedAfter:      strings.TrimSpace(r.URL.Query().Get("saved_after")),
			SavedBefore:     strings.TrimSpace(r.URL.Query().Get("saved_before")),
			PublishedAfter:  strings.TrimSpace(r.URL.Query().Get("published_after")),
			PublishedBefore: strings.TrimSpace(r.URL.Query().Get("published_before")),
			Limit:           parseRSSInboxLimit(r),
		}, traceID)
	case http.MethodPost:
		var req rssInboxPollParams
		if !decodeBodyOrWriteError(w, r, t.maxBodyBytes, &req) {
			return
		}
		traceID := resolveTraceID(req.TraceID, r)
		t.dispatchActionObject(w, r, actionRSSInboxPoll, req, traceID)
	default:
		writeMethodNotAllowed(w)
	}
}

func (t *transport) handleRSSInboxByID(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, "/api/rss/inbox/"))
	if id == "" || strings.Contains(id, "/") {
		writeError(w, http.StatusBadRequest, "rss inbox item id is required", "")
		return
	}
	if !requireMethod(w, r, http.MethodGet) {
		return
	}
	traceID := resolveTraceID("", r)
	t.dispatchActionObject(w, r, actionRSSInboxGet, rssInboxGetParams{ID: id}, traceID)
}

func parseRSSInboxLimit(r *http.Request) int {
	return parseOptionalIntQuery(r, "limit")
}

func parseOptionalIntQuery(r *http.Request, key string) int {
	value := strings.TrimSpace(r.URL.Query().Get("limit"))
	if key != "" {
		value = strings.TrimSpace(r.URL.Query().Get(key))
	}
	if value == "" {
		return 0
	}
	limit, err := strconv.Atoi(value)
	if err != nil || limit < 0 {
		return 0
	}
	return limit
}
