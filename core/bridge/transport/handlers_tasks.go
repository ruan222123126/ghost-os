package transport

import (
	"context"
	"errors"
	bridgeorchestration "ghost-os/bridge/orchestration"
	"net/http"
	"strconv"
	"strings"
)

type taskEndpointUsecase struct {
	get    func(id string, traceID string) (bridgeorchestration.ServiceResult, error)
	update func(id string, req bridgeorchestration.TaskUpdateParams, traceID string) (bridgeorchestration.ServiceResult, error)
	delete func(id string, traceID string) (bridgeorchestration.ServiceResult, error)
	run    func(id string, startOnly *bool, traceID string) (bridgeorchestration.ServiceResult, error)
	stop   func(ctx context.Context, id string, req bridgeorchestration.TaskStopParams, traceID string) (bridgeorchestration.ServiceResult, error)
	logs   func(id string, limit *int, traceID string) (bridgeorchestration.ServiceResult, error)
}

func (t *transport) handleTasks(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		traceID := resolveTraceID("", r)
		result, err := t.usecases.tasks.ListUserTasks(traceID)
		respondServiceContractResult(w, traceID, result, err)
	case http.MethodPost:
		var req bridgeorchestration.TaskCreateParams
		if !decodeBodyOrWriteError(w, r, t.maxBodyBytes, &req) {
			return
		}
		traceID := resolveTraceID(req.TraceID, r)
		result, err := t.usecases.tasks.CreateUserTask(req, traceID)
		respondServiceContractResult(w, traceID, result, err)
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
	result, err := t.usecases.tasks.ListSystemTasks(traceID)
	respondServiceContractResult(w, traceID, result, err)
}

func (t *transport) handleOrchestrations(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		traceID := resolveTraceID("", r)
		result, err := t.usecases.tasks.ListOrchestrations(traceID)
		respondServiceContractResult(w, traceID, result, err)
	case http.MethodPost:
		var req bridgeorchestration.TaskCreateParams
		if !decodeBodyOrWriteError(w, r, t.maxBodyBytes, &req) {
			return
		}
		traceID := resolveTraceID(req.TraceID, r)
		result, err := t.usecases.tasks.CreateOrchestration(req, traceID)
		respondServiceContractResult(w, traceID, result, err)
	default:
		writeMethodNotAllowed(w)
	}
}

func (t *transport) handleOrchestrationByID(w http.ResponseWriter, r *http.Request) {
	id, action, err := parseTaskResourcePath(r.URL.Path, "/api/orchestrations/")
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error(), "")
		return
	}
	traceID := resolveTraceID("", r)
	endpoint := orchestrationTaskEndpoint(t.usecases.tasks)
	if action != "" {
		t.handleTaskSubresource(w, r, traceID, id, action, endpoint)
		return
	}
	t.handleTaskResource(w, r, traceID, id, endpoint)
}

func (t *transport) handleTaskByID(w http.ResponseWriter, r *http.Request) {
	id, action, err := parseTaskPath(r.URL.Path)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error(), "")
		return
	}
	traceID := resolveTraceID("", r)
	endpoint := userTaskEndpoint(t.usecases.tasks)
	if action != "" {
		t.handleTaskSubresource(w, r, traceID, id, action, endpoint)
		return
	}
	t.handleTaskResource(w, r, traceID, id, endpoint)
}

func parseTaskPath(rawPath string) (string, string, error) {
	return parseTaskResourcePath(rawPath, "/api/tasks/")
}

func parseTaskResourcePath(rawPath string, prefix string) (string, string, error) {
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

func userTaskEndpoint(usecase TaskUsecase) taskEndpointUsecase {
	return taskEndpointUsecase{
		get:    usecase.GetUserTask,
		update: usecase.UpdateUserTask,
		delete: usecase.DeleteUserTask,
		run:    usecase.RunUserTask,
		stop:   usecase.StopUserTask,
		logs:   usecase.UserTaskLogs,
	}
}

func orchestrationTaskEndpoint(usecase TaskUsecase) taskEndpointUsecase {
	return taskEndpointUsecase{
		get:    usecase.GetOrchestrationTask,
		update: usecase.UpdateOrchestrationTask,
		delete: usecase.DeleteOrchestrationTask,
		run:    usecase.RunOrchestrationTask,
		stop:   usecase.StopOrchestrationTask,
		logs:   usecase.OrchestrationTaskLogs,
	}
}

func (t *transport) handleTaskSubresource(
	w http.ResponseWriter,
	r *http.Request,
	traceID string,
	id string,
	action string,
	endpoint taskEndpointUsecase,
) {
	switch action {
	case "logs":
		t.handleTaskLogs(w, r, traceID, id, endpoint)
	case "run":
		t.handleTaskRun(w, r, traceID, id, endpoint)
	case "stop":
		t.handleTaskStop(w, r, traceID, id, endpoint)
	default:
		writeError(w, http.StatusBadRequest, "invalid task path", traceID)
	}
}

func (t *transport) handleTaskLogs(
	w http.ResponseWriter,
	r *http.Request,
	traceID string,
	id string,
	endpoint taskEndpointUsecase,
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
	result, err := endpoint.logs(id, limit, traceID)
	respondServiceContractResult(w, traceID, result, err)
}

func (t *transport) handleTaskRun(
	w http.ResponseWriter,
	r *http.Request,
	traceID string,
	id string,
	endpoint taskEndpointUsecase,
) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w)
		return
	}
	startOnly, err := parseTaskRunStartOnly(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error(), traceID)
		return
	}
	result, err := endpoint.run(id, startOnly, traceID)
	respondServiceContractResult(w, traceID, result, err)
}

func (t *transport) handleTaskStop(
	w http.ResponseWriter,
	r *http.Request,
	traceID string,
	id string,
	endpoint taskEndpointUsecase,
) {
	if r.Method != http.MethodPost {
		writeMethodNotAllowed(w)
		return
	}
	var params bridgeorchestration.TaskStopParams
	if !decodeBodyOrWriteError(w, r, t.maxBodyBytes, &params) {
		return
	}
	result, err := endpoint.stop(r.Context(), id, params, traceID)
	respondServiceContractResult(w, traceID, result, err)
}

func (t *transport) handleTaskResource(
	w http.ResponseWriter,
	r *http.Request,
	traceID string,
	id string,
	endpoint taskEndpointUsecase,
) {
	switch r.Method {
	case http.MethodGet:
		result, err := endpoint.get(id, traceID)
		respondServiceContractResult(w, traceID, result, err)
	case http.MethodPatch:
		var req bridgeorchestration.TaskUpdateParams
		if !decodeBodyOrWriteError(w, r, t.maxBodyBytes, &req) {
			return
		}
		traceID = resolveTraceID(req.TraceID, r)
		result, err := endpoint.update(id, req, traceID)
		respondServiceContractResult(w, traceID, result, err)
	case http.MethodDelete:
		result, err := endpoint.delete(id, traceID)
		respondServiceContractResult(w, traceID, result, err)
	default:
		writeMethodNotAllowed(w)
	}
}

func parseTaskLogsLimit(r *http.Request) (*int, error) {
	value := strings.TrimSpace(r.URL.Query().Get("limit"))
	if value == "" {
		return nil, nil
	}
	limit, err := strconv.Atoi(value)
	if err != nil {
		return nil, errors.New("limit must be an integer")
	}
	return &limit, nil
}

func parseTaskRunStartOnly(r *http.Request) (*bool, error) {
	value := strings.TrimSpace(r.URL.Query().Get("start_only"))
	if value == "" {
		return nil, nil
	}
	switch {
	case value == "1", strings.EqualFold(value, "true"):
		parsed := true
		return &parsed, nil
	case value == "0", strings.EqualFold(value, "false"):
		parsed := false
		return &parsed, nil
	default:
		return nil, errors.New("start_only must be a boolean")
	}
}
