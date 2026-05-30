package transport

import (
	"encoding/json"
	"errors"
	bridgeorchestration "ghost-os/bridge/orchestration"
	"net/http"
	"strconv"
	"strings"
)

func (t *transport) handleTasks(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		traceID := resolveTraceID("", r)
		t.dispatchActionObject(w, r, actionTaskList, map[string]any{"scope": bridgeorchestration.TaskListScopeUser}, traceID)
	case http.MethodPost:
		var req bridgeorchestration.TaskCreateParams
		if !decodeBodyOrWriteError(w, r, t.maxBodyBytes, &req) {
			return
		}
		req.Scope = bridgeorchestration.TaskListScopeUser
		traceID := resolveTraceID(req.TraceID, r)
		t.dispatchScopedTaskAction(w, r, actionTaskCreate, req, req.Scope, traceID)
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
	t.dispatchActionObject(w, r, actionTaskList, map[string]any{"scope": bridgeorchestration.TaskListScopeSystem}, traceID)
}

func (t *transport) handleTaskByID(w http.ResponseWriter, r *http.Request) {
	id, action, err := parseTaskPath(r.URL.Path)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error(), "")
		return
	}
	traceID := resolveTraceID("", r)
	if action != "" {
		t.handleTaskSubresource(w, r, traceID, id, bridgeorchestration.TaskListScopeUser, action)
		return
	}
	t.handleTaskResource(w, r, traceID, id, bridgeorchestration.TaskListScopeUser)
}

func parseTaskPath(rawPath string) (string, string, error) {
	return parseScopedTaskPath(rawPath, "/api/tasks/")
}

func (t *transport) handleTaskSubresource(
	w http.ResponseWriter,
	r *http.Request,
	traceID string,
	id string,
	scope string,
	action string,
) {
	switch action {
	case "logs":
		t.handleTaskLogs(w, r, traceID, id, scope)
	case "run":
		t.handleTaskRun(w, r, traceID, id, scope)
	default:
		writeError(w, http.StatusBadRequest, "invalid task path", traceID)
	}
}

func (t *transport) handleTaskLogs(
	w http.ResponseWriter,
	r *http.Request,
	traceID string,
	id string,
	scope string,
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
	t.dispatchScopedTaskAction(
		w,
		r,
		actionTaskLogs,
		bridgeorchestration.TaskLogsParams{ID: id, Limit: limit, Scope: scope},
		scope,
		traceID,
	)
}

func (t *transport) handleTaskRun(
	w http.ResponseWriter,
	r *http.Request,
	traceID string,
	id string,
	scope string,
) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w)
		return
	}
	t.dispatchScopedTaskAction(
		w,
		r,
		actionTaskRunNow,
		bridgeorchestration.TaskIDParams{
			ID:        id,
			Scope:     scope,
			StartOnly: taskRunStartOnly(r),
		},
		scope,
		traceID,
	)
}

func (t *transport) handleTaskResource(
	w http.ResponseWriter,
	r *http.Request,
	traceID string,
	id string,
	scope string,
) {
	params := bridgeorchestration.TaskIDParams{ID: id, Scope: scope}
	switch r.Method {
	case http.MethodGet:
		t.dispatchScopedTaskAction(w, r, actionTaskGet, params, scope, traceID)
	case http.MethodPatch:
		var req bridgeorchestration.TaskUpdateParams
		if !decodeBodyOrWriteError(w, r, t.maxBodyBytes, &req) {
			return
		}
		req.ID = id
		req.Scope = scope
		traceID = resolveTraceID(req.TraceID, r)
		t.dispatchScopedTaskAction(w, r, actionTaskUpdate, req, scope, traceID)
	case http.MethodDelete:
		t.dispatchScopedTaskAction(w, r, actionTaskDelete, params, scope, traceID)
	default:
		writeMethodNotAllowed(w)
	}
}

func (t *transport) dispatchScopedTaskAction(
	w http.ResponseWriter,
	r *http.Request,
	action string,
	params any,
	scope string,
	traceID string,
) bool {
	scopedParams, err := scopedTaskActionParams(params, scope)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to encode action params", traceID)
		return false
	}
	return t.dispatchActionObject(w, r, action, scopedParams, traceID)
}

func scopedTaskActionParams(params any, scope string) (map[string]any, error) {
	raw, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}
	payload := map[string]any{}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, err
	}
	payload["scope"] = strings.TrimSpace(scope)
	return payload, nil
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

func taskRunStartOnly(r *http.Request) bool {
	value := strings.TrimSpace(r.URL.Query().Get("start_only"))
	return value == "1" || strings.EqualFold(value, "true")
}
