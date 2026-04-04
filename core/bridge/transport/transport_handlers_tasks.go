package transport

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
		payload, code, err := t.service.ExecuteTaskListAction(taskListScopeUser, traceID)
		respondServiceResult(w, traceID, payload, code, err)
	case http.MethodPost:
		var req taskCreateParams
		if !decodeBodyOrWriteError(w, r, t.maxBodyBytes, &req) {
			return
		}
		traceID := resolveTraceID(req.TraceID, r)
		payload, code, err := t.service.ExecuteTaskCreateAction(req, traceID)
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
	payload, code, err := t.service.ExecuteTaskListAction(taskListScopeSystem, traceID)
	respondServiceResult(w, traceID, payload, code, err)
}

func (t *transport) handleTaskByID(w http.ResponseWriter, r *http.Request) {
	id, action, err := parseTaskPath(r.URL.Path)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error(), "")
		return
	}
	traceID := resolveTraceID("", r)
	if action != "" {
		t.handleTaskSubresource(w, r, traceID, id, action)
		return
	}
	t.handleTaskResource(w, r, traceID, id)
}

func parseTaskPath(rawPath string) (string, string, error) {
	path := strings.TrimSpace(strings.TrimPrefix(rawPath, "/api/tasks/"))
	segments := strings.Split(path, "/")
	if len(segments) == 0 {
		return "", "", errors.New("task id is required")
	}
	id := strings.TrimSpace(segments[0])
	if id == "" {
		return "", "", errors.New("task id is required")
	}
	if len(segments) > 2 || (len(segments) == 2 && strings.TrimSpace(segments[1]) == "") {
		return "", "", errors.New("invalid task path")
	}
	if len(segments) == 1 {
		return id, "", nil
	}
	return id, strings.TrimSpace(segments[1]), nil
}

func (t *transport) handleTaskSubresource(
	w http.ResponseWriter,
	r *http.Request,
	traceID string,
	id string,
	action string,
) {
	switch action {
	case "logs":
		t.handleTaskLogs(w, r, traceID, id)
	case "run":
		t.handleTaskRun(w, r, traceID, id)
	default:
		writeError(w, http.StatusBadRequest, "invalid task path", traceID)
	}
}

func (t *transport) handleTaskLogs(
	w http.ResponseWriter,
	r *http.Request,
	traceID string,
	id string,
) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}
	limit, err := parseTaskLogsLimit(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error(), traceID)
		return
	}
	payload, code, err := t.service.ExecuteTaskLogsAction(taskLogsParams{ID: id, Limit: limit}, traceID)
	respondServiceResult(w, traceID, payload, code, err)
}

func (t *transport) handleTaskRun(
	w http.ResponseWriter,
	r *http.Request,
	traceID string,
	id string,
) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w)
		return
	}
	payload, code, err := t.service.ExecuteTaskRunNowAction(taskIDParams{ID: id}, traceID)
	respondServiceResult(w, traceID, payload, code, err)
}

func (t *transport) handleTaskResource(
	w http.ResponseWriter,
	r *http.Request,
	traceID string,
	id string,
) {
	params := taskIDParams{ID: id}
	switch r.Method {
	case http.MethodGet:
		payload, code, err := t.service.ExecuteTaskGetAction(params, traceID)
		respondServiceResult(w, traceID, payload, code, err)
	case http.MethodPatch:
		var req taskUpdateParams
		if !decodeBodyOrWriteError(w, r, t.maxBodyBytes, &req) {
			return
		}
		req.ID = id
		traceID = resolveTraceID(req.TraceID, r)
		payload, code, err := t.service.ExecuteTaskUpdateAction(req, traceID)
		respondServiceResult(w, traceID, payload, code, err)
	case http.MethodDelete:
		payload, code, err := t.service.ExecuteTaskDeleteAction(params, traceID)
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
