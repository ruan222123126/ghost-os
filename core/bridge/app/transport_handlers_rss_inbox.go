package app

import (
	"net/http"
	"strconv"
	"strings"
)

func (t *transport) handleRSSInbox(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		traceID := resolveTraceID("", r)
		payload, code, err := t.service.executeRSSInboxListAction(rssInboxListParams{
			FeedID:          strings.TrimSpace(r.URL.Query().Get("feed_id")),
			Tag:             strings.TrimSpace(r.URL.Query().Get("tag")),
			Importance:      strings.TrimSpace(r.URL.Query().Get("importance")),
			SavedAfter:      strings.TrimSpace(r.URL.Query().Get("saved_after")),
			SavedBefore:     strings.TrimSpace(r.URL.Query().Get("saved_before")),
			PublishedAfter:  strings.TrimSpace(r.URL.Query().Get("published_after")),
			PublishedBefore: strings.TrimSpace(r.URL.Query().Get("published_before")),
			Limit:           parseRSSInboxLimit(r),
		}, traceID)
		respondServiceResult(w, traceID, payload, code, err)
	case http.MethodPost:
		var req rssInboxPollParams
		if !decodeBodyOrWriteError(w, r, t.maxBodyBytes, &req) {
			return
		}
		traceID := resolveTraceID(req.TraceID, r)
		payload, code, err := t.service.executeRSSInboxPollAction(r.Context(), req, traceID)
		respondServiceResult(w, traceID, payload, code, err)
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
	payload, code, err := t.service.executeRSSInboxGetAction(rssInboxGetParams{ID: id}, traceID)
	respondServiceResult(w, traceID, payload, code, err)
}

func parseRSSInboxLimit(r *http.Request) int {
	value := strings.TrimSpace(r.URL.Query().Get("limit"))
	if value == "" {
		return 0
	}
	limit, err := strconv.Atoi(value)
	if err != nil || limit < 0 {
		return 0
	}
	return limit
}
