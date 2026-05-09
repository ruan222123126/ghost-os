package transport

import (
	"errors"
	bridgeorchestration "ghost-os/bridge/orchestration"
	"net/http"
	"strings"
)

func (t *transport) handleOrchestrations(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		traceID := resolveTraceID("", r)
		t.dispatchActionObject(w, r, actionTaskList, map[string]any{"scope": bridgeorchestration.TaskListScopeOrchestration}, traceID)
	case http.MethodPost:
		var req bridgeorchestration.TaskCreateParams
		if !decodeBodyOrWriteError(w, r, t.maxBodyBytes, &req) {
			return
		}
		req.Scope = bridgeorchestration.TaskListScopeOrchestration
		traceID := resolveTraceID(req.TraceID, r)
		t.dispatchScopedTaskAction(w, r, actionTaskCreate, req, req.Scope, traceID)
	default:
		writeMethodNotAllowed(w)
	}
}

func (t *transport) handleOrchestrationByID(w http.ResponseWriter, r *http.Request) {
	id, action, err := parseScopedTaskPath(r.URL.Path, "/api/orchestrations/")
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error(), "")
		return
	}
	traceID := resolveTraceID("", r)
	scope := bridgeorchestration.TaskListScopeOrchestration
	if action != "" {
		t.handleTaskSubresource(w, r, traceID, id, scope, action)
		return
	}
	t.handleTaskResource(w, r, traceID, id, scope)
}

func parseScopedTaskPath(rawPath string, prefix string) (string, string, error) {
	path := strings.TrimSpace(strings.TrimPrefix(rawPath, prefix))
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
