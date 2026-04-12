// Session list/detail HTTP handlers.

package transport

import (
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"ghost-os/bridge/artifacts"
	bridgeorchestration "ghost-os/bridge/orchestration"
	"ghost-os/bridge/session"
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

type artifactDownloadRequest struct {
	sessionID  string
	artifactID string
}

type artifactDownloadPayload struct {
	file     *os.File
	info     os.FileInfo
	artifact *artifacts.SessionFileArtifact
}

// handleSessionsList 列出会话概要，供 Web/CLI 构建侧边栏或历史视图。
func (t *transport) handleSessionsList(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}

	traceID := resolveTraceID("", r)
	result, err := t.service.ExecuteSessionsListAction(traceID)
	respondServiceContractResult(w, traceID, result, err)
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
		result, err := t.service.ExecuteSessionGetAction(params, traceID)
		respondServiceContractResult(w, traceID, result, err)
	case http.MethodDelete:
		params := bridgeorchestration.SessionIDParams{ID: sessionID}
		result, err := t.service.ExecuteSessionDeleteAction(params, traceID)
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
		ID:    rawPath,
		Limit: session.DefaultDetailPageLimit,
	}
	query := r.URL.Query()
	if query.Has("limit") {
		limit, err := strconv.Atoi(strings.TrimSpace(query.Get("limit")))
		if err != nil || limit <= 0 || limit > session.MaxDetailPageLimit {
			return bridgeorchestration.SessionGetParams{}, http.StatusBadRequest, fmt.Errorf(
				"limit must be an integer between 1 and %d",
				session.MaxDetailPageLimit,
			)
		}
		params.Limit = limit
	}
	if query.Has("before") {
		before, err := strconv.Atoi(strings.TrimSpace(query.Get("before")))
		if err != nil || before < 0 {
			return bridgeorchestration.SessionGetParams{}, http.StatusBadRequest, errors.New("before must be a non-negative integer")
		}
		params.Before = &before
	}
	return params, http.StatusOK, nil
}

func (t *transport) handleSessionArtifactDownload(w http.ResponseWriter, r *http.Request, sessionID string, artifactID string) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}

	traceID := resolveTraceID("", r)
	request, code, err := buildArtifactDownloadRequest(sessionID, artifactID)
	if err != nil {
		writeError(w, code, err.Error(), traceID)
		return
	}

	store, err := artifacts.NewSessionArtifactStoreFromEnv()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error(), traceID)
		return
	}

	opened, code, err := openArtifactDownloadPayload(store, request)
	if err != nil {
		writeError(w, code, err.Error(), traceID)
		return
	}
	defer opened.file.Close()

	writeArtifactDownloadHeaders(w, r, opened)
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, opened.file)
}

func buildArtifactDownloadRequest(sessionID string, artifactID string) (artifactDownloadRequest, int, error) {
	normalizedSessionID, code, err := bridgeorchestration.RequireSessionID(sessionID)
	if err != nil {
		return artifactDownloadRequest{}, code, err
	}
	normalizedArtifactID, err := artifacts.NormalizeArtifactID(artifactID)
	if err != nil {
		return artifactDownloadRequest{}, http.StatusBadRequest, err
	}
	return artifactDownloadRequest{
		sessionID:  normalizedSessionID,
		artifactID: normalizedArtifactID,
	}, http.StatusOK, nil
}

func openArtifactDownloadPayload(
	store *artifacts.SessionArtifactStore,
	request artifactDownloadRequest,
) (artifactDownloadPayload, int, error) {
	file, info, artifact, err := store.OpenStoredFile(request.sessionID, request.artifactID)
	if err != nil {
		status, openErr := artifactOpenError(err)
		return artifactDownloadPayload{}, status, openErr
	}
	return artifactDownloadPayload{
		file:     file,
		info:     info,
		artifact: artifact,
	}, http.StatusOK, nil
}

func artifactOpenError(err error) (int, error) {
	if errors.Is(err, artifacts.ErrArtifactNotFound) {
		return http.StatusNotFound, err
	}
	if errors.Is(err, os.ErrNotExist) {
		return http.StatusNotFound, errors.New("artifact not found")
	}
	if errors.Is(err, artifacts.ErrInvalidStoredPath) {
		return http.StatusInternalServerError, errors.New("invalid artifact path")
	}
	return http.StatusInternalServerError, err
}

func writeArtifactDownloadHeaders(w http.ResponseWriter, r *http.Request, payload artifactDownloadPayload) {
	w.Header().Set("Content-Type", resolveArtifactMimeType(payload.artifact))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", payload.info.Size()))
	w.Header().Set(
		"Content-Disposition",
		fmt.Sprintf("%s; filename=%q", resolveArtifactDisposition(r), payload.artifact.Name),
	)
	if strings.TrimSpace(payload.artifact.SHA256) != "" {
		w.Header().Set("ETag", fmt.Sprintf(`"%s"`, payload.artifact.SHA256))
		w.Header().Set("X-Artifact-SHA256", payload.artifact.SHA256)
	}
}

func resolveArtifactMimeType(artifact *artifacts.SessionFileArtifact) string {
	if artifact == nil {
		return "application/octet-stream"
	}
	mimeType := strings.TrimSpace(artifact.MimeType)
	if mimeType != "" {
		return mimeType
	}
	byExtension := mime.TypeByExtension(filepath.Ext(artifact.Name))
	if byExtension != "" {
		return byExtension
	}
	return "application/octet-stream"
}

func resolveArtifactDisposition(r *http.Request) string {
	disposition := strings.TrimSpace(r.URL.Query().Get("disposition"))
	if disposition == "inline" {
		return "inline"
	}
	return "attachment"
}
