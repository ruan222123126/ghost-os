package app

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
)

func (t *transport) handleTasks(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		traceID := resolveTraceID("", r)
		payload, code, err := t.service.executeTaskListAction(taskListScopeUser, traceID)
		respondServiceResult(w, traceID, payload, code, err)
	case http.MethodPost:
		var req taskCreateParams
		if !decodeBodyOrWriteError(w, r, t.maxBodyBytes, &req) {
			return
		}
		traceID := resolveTraceID(req.TraceID, r)
		payload, code, err := t.service.executeTaskCreateAction(req, traceID)
		respondServiceResult(w, traceID, payload, code, err)
	default:
		writeMethodNotAllowed(w)
	}
}

func (t *transport) handleSystemTasks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}
	traceID := resolveTraceID("", r)
	payload, code, err := t.service.executeTaskListAction(taskListScopeSystem, traceID)
	respondServiceResult(w, traceID, payload, code, err)
}

func (t *transport) handleTaskByID(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, "/api/tasks/"))
	segments := strings.Split(path, "/")
	if len(segments) == 0 {
		writeError(w, http.StatusBadRequest, "task id is required", "")
		return
	}
	id := strings.TrimSpace(segments[0])
	if id == "" {
		writeError(w, http.StatusBadRequest, "task id is required", "")
		return
	}
	if len(segments) > 2 || (len(segments) == 2 && strings.TrimSpace(segments[1]) == "") {
		writeError(w, http.StatusBadRequest, "invalid task path", "")
		return
	}
	action := ""
	if len(segments) == 2 {
		action = strings.TrimSpace(segments[1])
	}
	traceID := resolveTraceID("", r)
	if action == "logs" {
		if r.Method != http.MethodGet {
			writeMethodNotAllowed(w)
			return
		}
		limit, err := parseTaskLogsLimit(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error(), traceID)
			return
		}
		payload, code, err := t.service.executeTaskLogsAction(taskLogsParams{ID: id, Limit: limit}, traceID)
		respondServiceResult(w, traceID, payload, code, err)
		return
	}
	if action == "run" {
		if r.Method != http.MethodPost {
			writeMethodNotAllowed(w)
			return
		}
		payload, code, err := t.service.executeTaskRunNowAction(taskIDParams{ID: id}, traceID)
		respondServiceResult(w, traceID, payload, code, err)
		return
	}
	if action != "" {
		writeError(w, http.StatusBadRequest, "invalid task path", traceID)
		return
	}
	params := taskIDParams{ID: id}
	switch r.Method {
	case http.MethodGet:
		payload, code, err := t.service.executeTaskGetAction(params, traceID)
		respondServiceResult(w, traceID, payload, code, err)
	case http.MethodPatch:
		var req taskUpdateParams
		if !decodeBodyOrWriteError(w, r, t.maxBodyBytes, &req) {
			return
		}
		req.ID = id
		traceID = resolveTraceID(req.TraceID, r)
		payload, code, err := t.service.executeTaskUpdateAction(req, traceID)
		respondServiceResult(w, traceID, payload, code, err)
	case http.MethodDelete:
		payload, code, err := t.service.executeTaskDeleteAction(params, traceID)
		respondServiceResult(w, traceID, payload, code, err)
	default:
		writeMethodNotAllowed(w)
	}
}

func parseTaskLogsLimit(r *http.Request) (int, error) {
	value := strings.TrimSpace(r.URL.Query().Get("limit"))
	if value == "" {
		return 0, nil
	}
	limit, err := strconv.Atoi(value)
	if err != nil || limit < 0 {
		return 0, errors.New("limit must be a non-negative integer")
	}
	return limit, nil
}
