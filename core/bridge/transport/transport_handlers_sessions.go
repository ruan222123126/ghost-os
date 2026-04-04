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
	"strings"

	"ghost-os/bridge/artifacts"
)

// handleSessionsList 列出会话概要，供 Web/CLI 构建侧边栏或历史视图。
func (t *transport) handleSessionsList(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}

	traceID := resolveTraceID("", r)
	payload, code, err := t.service.executeSessionsListAction(traceID)
	respondServiceResult(w, traceID, payload, code, err)
}

// handleSessionByID 处理单会话查询与删除，并在路径层面做 session id 基本校验。
func (t *transport) handleSessionByID(w http.ResponseWriter, r *http.Request) {
	rawPath := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, "/api/sessions/"))
	if rawPath == "" {
		writeError(w, http.StatusBadRequest, "session id is required", "")
		return
	}

	if strings.HasSuffix(rawPath, "/events") {
		id := strings.TrimSpace(strings.TrimSuffix(rawPath, "/events"))
		if id == "" || strings.Contains(id, "/") {
			writeError(w, http.StatusBadRequest, "session id is required", "")
			return
		}
		t.handleSessionEvents(w, r, id)
		return
	}

	if strings.Contains(rawPath, "/artifacts/") {
		sessionID, artifactID, ok := parseSessionArtifactPath(rawPath)
		if !ok {
			writeError(w, http.StatusBadRequest, "invalid artifact path", "")
			return
		}
		t.handleSessionArtifactDownload(w, r, sessionID, artifactID)
		return
	}

	if strings.Contains(rawPath, "/") {
		writeError(w, http.StatusBadRequest, "session id is required", "")
		return
	}

	params := sessionIDParams{ID: rawPath}
	traceID := resolveTraceID("", r)

	switch r.Method {
	case http.MethodGet:
		payload, code, err := t.service.executeSessionGetAction(params, traceID)
		respondServiceResult(w, traceID, payload, code, err)
	case http.MethodDelete:
		payload, code, err := t.service.executeSessionDeleteAction(params, traceID)
		respondServiceResult(w, traceID, payload, code, err)
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

func (t *transport) handleSessionArtifactDownload(w http.ResponseWriter, r *http.Request, sessionID string, artifactID string) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}

	traceID := resolveTraceID("", r)
	normalizedSessionID, code, err := requireSessionID(sessionID)
	if err != nil {
		writeError(w, code, err.Error(), traceID)
		return
	}
	normalizedArtifactID, err := artifacts.NormalizeArtifactID(artifactID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error(), traceID)
		return
	}

	store, err := artifacts.NewSessionArtifactStoreFromEnv()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error(), traceID)
		return
	}
	file, info, artifact, err := store.OpenStoredFile(normalizedSessionID, normalizedArtifactID)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, artifacts.ErrArtifactNotFound) {
			status = http.StatusNotFound
			writeError(w, status, err.Error(), traceID)
			return
		}
		if errors.Is(err, os.ErrNotExist) {
			status = http.StatusNotFound
			writeError(w, status, "artifact not found", traceID)
			return
		}
		if errors.Is(err, artifacts.ErrInvalidStoredPath) {
			writeError(w, status, "invalid artifact path", traceID)
			return
		}
		writeError(w, status, err.Error(), traceID)
		return
	}
	defer file.Close()

	mimeType := strings.TrimSpace(artifact.MimeType)
	if mimeType == "" {
		mimeType = mime.TypeByExtension(filepath.Ext(artifact.Name))
	}
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	disposition := strings.TrimSpace(r.URL.Query().Get("disposition"))
	if disposition != "inline" {
		disposition = "attachment"
	}

	w.Header().Set("Content-Type", mimeType)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", info.Size()))
	w.Header().Set("Content-Disposition", fmt.Sprintf("%s; filename=%q", disposition, artifact.Name))
	if strings.TrimSpace(artifact.SHA256) != "" {
		w.Header().Set("ETag", fmt.Sprintf(`"%s"`, artifact.SHA256))
		w.Header().Set("X-Artifact-SHA256", artifact.SHA256)
	}
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, file)
}
