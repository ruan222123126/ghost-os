// Session list/detail HTTP handlers.

package transport

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	bridgeorchestration "ghost-os/bridge/orchestration"
)

type sessionRouteKind string

const (
	sessionRouteDetail   sessionRouteKind = "detail"
	sessionRouteEvents   sessionRouteKind = "events"
	sessionRouteArtifact sessionRouteKind = "artifact"
)

type sessionRoute struct {
	kind       sessionRouteKind
	sessionID  string
	artifactID string
}

// handleSessionsList 列出会话概要，供 Web/CLI 构建侧边栏或历史视图。
func (t *transport) handleSessionsList(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}

	traceID := resolveTraceID("", r)
	result, err := t.usecases.sessions.List(traceID)
	respondServiceContractResult(w, traceID, result, err)
}

func (t *transport) handleSessionsSearch(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}

	traceID := resolveTraceID("", r)
	query, limit, code, err := parseSessionSearchParams(r)
	if err != nil {
		writeError(w, code, err.Error(), traceID)
		return
	}
	result, err := t.usecases.sessions.Search(query, limit, traceID)
	respondServiceContractResult(w, traceID, result, err)
}

func (t *transport) handleSessionSources(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}

	traceID := resolveTraceID("", r)
	result, err := t.usecases.sessions.Sources(traceID)
	respondServiceContractResult(w, traceID, result, err)
}

func (t *transport) handleSessionPartitions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		traceID := resolveTraceID("", r)
		result, err := t.usecases.sessions.ListPartitions(traceID)
		respondServiceContractResult(w, traceID, result, err)
	case http.MethodPut:
		var req bridgeorchestration.SessionSidebarPartitionPutRequest
		if !decodeBodyOrWriteError(w, r, t.maxBodyBytes, &req) {
			return
		}
		traceID := resolveTraceID(req.TraceID, r)
		result, err := t.usecases.sessions.SavePartitions(req, traceID)
		respondServiceContractResult(w, traceID, result, err)
	default:
		writeMethodNotAllowed(w)
	}
}

// handleSessionByID 处理单会话查询与删除，并在路径层面做 session id 基本校验。
func (t *transport) handleSessionByID(w http.ResponseWriter, r *http.Request) {
	rawPath := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, "/api/sessions/"))
	route, code, err := parseSessionRoute(rawPath)
	if err != nil {
		writeError(w, code, err.Error(), "")
		return
	}
	switch route.kind {
	case sessionRouteEvents:
		t.handleSessionEvents(w, r, route.sessionID)
	case sessionRouteArtifact:
		t.handleSessionArtifactDownload(w, r, route.sessionID, route.artifactID)
	default:
		t.handleSessionResource(w, r, route.sessionID)
	}
}

func parseSessionRoute(rawPath string) (sessionRoute, int, error) {
	if rawPath == "" {
		return sessionRoute{}, http.StatusBadRequest, errors.New("session id is required")
	}
	if strings.HasSuffix(rawPath, "/events") {
		sessionID := strings.TrimSpace(strings.TrimSuffix(rawPath, "/events"))
		if sessionID == "" || strings.Contains(sessionID, "/") {
			return sessionRoute{}, http.StatusBadRequest, errors.New("session id is required")
		}
		return sessionRoute{kind: sessionRouteEvents, sessionID: sessionID}, http.StatusOK, nil
	}
	if strings.Contains(rawPath, "/artifacts/") {
		sessionID, artifactID, ok := parseSessionArtifactPath(rawPath)
		if !ok {
			return sessionRoute{}, http.StatusBadRequest, errors.New("invalid artifact path")
		}
		return sessionRoute{
			kind:       sessionRouteArtifact,
			sessionID:  sessionID,
			artifactID: artifactID,
		}, http.StatusOK, nil
	}
	if strings.Contains(rawPath, "/") {
		return sessionRoute{}, http.StatusBadRequest, errors.New("session id is required")
	}
	return sessionRoute{kind: sessionRouteDetail, sessionID: rawPath}, http.StatusOK, nil
}

func (t *transport) handleSessionResource(w http.ResponseWriter, r *http.Request, sessionID string) {
	traceID := resolveTraceID("", r)
	switch r.Method {
	case http.MethodGet:
		params, code, err := parseSessionGetParams(sessionID, r)
		if err != nil {
			writeError(w, code, err.Error(), traceID)
			return
		}
		result, err := t.usecases.sessions.Get(params, traceID)
		respondServiceContractResult(w, traceID, result, err)
	case http.MethodDelete:
		params := bridgeorchestration.SessionIDParams{ID: sessionID}
		result, err := t.usecases.sessions.Delete(params, traceID)
		respondServiceContractResult(w, traceID, result, err)
	default:
		writeMethodNotAllowed(w)
	}
}

func parseSessionArtifactPath(rawPath string) (string, string, bool) {
	parts := strings.SplitN(rawPath, "/artifacts/", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	sessionID := strings.TrimSpace(parts[0])
	artifactID := strings.TrimSpace(parts[1])
	if sessionID == "" || artifactID == "" || strings.Contains(sessionID, "/") || strings.Contains(artifactID, "/") {
		return "", "", false
	}
	return sessionID, artifactID, true
}

func parseSessionGetParams(rawPath string, r *http.Request) (bridgeorchestration.SessionGetParams, int, error) {
	params := bridgeorchestration.SessionGetParams{
		ID: rawPath,
	}
	query := r.URL.Query()
	if query.Has("limit") {
		limit, err := strconv.Atoi(strings.TrimSpace(query.Get("limit")))
		if err != nil {
			return bridgeorchestration.SessionGetParams{}, http.StatusBadRequest, errors.New("limit must be an integer")
		}
		params.Limit = &limit
	}
	if query.Has("before") {
		before, err := strconv.Atoi(strings.TrimSpace(query.Get("before")))
		if err != nil {
			return bridgeorchestration.SessionGetParams{}, http.StatusBadRequest, errors.New("before must be an integer")
		}
		params.Before = &before
	}
	return params, http.StatusOK, nil
}

func parseSessionSearchParams(r *http.Request) (string, int, int, error) {
	query := r.URL.Query()
	searchQuery := strings.TrimSpace(query.Get("q"))
	if searchQuery == "" {
		searchQuery = strings.TrimSpace(query.Get("query"))
	}

	if !query.Has("limit") {
		return searchQuery, 0, http.StatusOK, nil
	}

	limit, err := strconv.Atoi(strings.TrimSpace(query.Get("limit")))
	if err != nil || limit <= 0 {
		return "", 0, http.StatusBadRequest, errors.New("limit must be a positive integer")
	}
	return searchQuery, limit, http.StatusOK, nil
}

func (t *transport) handleSessionArtifactDownload(w http.ResponseWriter, r *http.Request, sessionID string, artifactID string) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}

	traceID := resolveTraceID("", r)
	download, err := t.usecases.sessions.OpenArtifactDownload(sessionID, artifactID)
	if err != nil {
		writeError(w, httpStatusFromServiceError(err), err.Error(), traceID)
		return
	}
	defer download.Reader.Close()

	writeBinaryDownload(w, traceID, download, resolveArtifactDisposition(r), true)
}

func resolveArtifactDisposition(r *http.Request) string {
	disposition := strings.TrimSpace(r.URL.Query().Get("disposition"))
	if disposition == "inline" {
		return "inline"
	}
	return "attachment"
}
