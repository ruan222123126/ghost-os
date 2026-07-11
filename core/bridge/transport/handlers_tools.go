package transport

import (
	"errors"
	bridgeorchestration "ghost-os/bridge/orchestration"
	"net/http"
	"net/url"
	"strings"
)

func (t *transport) handleTools(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}
	traceID := resolveTraceID("", r)
	result, err := t.usecases.tools.List(traceID)
	respondServiceContractResult(w, traceID, result, err)
}

func (t *transport) handleToolByName(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		writeMethodNotAllowed(w)
		return
	}
	name, err := parseToolPath(r.URL.Path)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error(), "")
		return
	}

	var req bridgeorchestration.ToolUpdateRequest
	if !decodeBodyOrWriteError(w, r, t.maxBodyBytes, &req) {
		return
	}
	traceID := resolveTraceID(req.TraceID, r)
	result, callErr := t.usecases.tools.Update(bridgeorchestration.ToolNameParams{Name: name}, req, traceID)
	respondServiceContractResult(w, traceID, result, callErr)
}

func (t *transport) handleFindIconTemplateUpload(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		t.handleFindIconTemplateCreate(w, r)
	case http.MethodGet:
		t.handleFindIconTemplateDownload(w, r)
	default:
		writeMethodNotAllowed(w)
	}
}

func (t *transport) handleFindIconTemplateCreate(w http.ResponseWriter, r *http.Request) {
	var req bridgeorchestration.FindIconTemplateUploadRequest
	if !decodeBodyOrWriteError(w, r, t.maxBodyBytes, &req) {
		return
	}
	traceID := resolveTraceID(req.TraceID, r)
	result, err := t.usecases.tools.UploadFindIconTemplate(req, traceID)
	respondServiceContractResult(w, traceID, result, err)
}

func (t *transport) handleFindIconPreview(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	var req bridgeorchestration.FindIconPreviewRequest
	if !decodeBodyOrWriteError(w, r, t.maxBodyBytes, &req) {
		return
	}
	traceID := resolveTraceID(req.TraceID, r)
	result, err := t.usecases.tools.PreviewFindIcon(r.Context(), req, traceID)
	respondServiceContractResult(w, traceID, result, err)
}

func (t *transport) handleFindIconTemplateDownload(w http.ResponseWriter, r *http.Request) {
	traceID := resolveTraceID("", r)
	templatePath := strings.TrimSpace(r.URL.Query().Get("template_path"))
	download, err := t.usecases.tools.DownloadFindIconTemplate(templatePath)
	if err != nil {
		writeError(w, httpStatusFromServiceError(err), err.Error(), traceID)
		return
	}
	defer download.Reader.Close()

	writeBinaryDownload(w, traceID, download, "inline", false)
}

func (t *transport) handleMousePosition(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	var req bridgeorchestration.MousePositionRequest
	if !decodeBodyOrWriteError(w, r, t.maxBodyBytes, &req) {
		return
	}
	traceID := resolveTraceID(req.TraceID, r)
	result, err := t.usecases.tools.MousePosition(r.Context(), req, traceID)
	respondServiceContractResult(w, traceID, result, err)
}

func parseToolPath(rawPath string) (string, error) {
	path := strings.TrimSpace(strings.TrimPrefix(rawPath, "/api/tools/"))
	if path == "" {
		return "", errors.New("tool name is required")
	}
	segments := strings.Split(path, "/")
	if len(segments) != 1 || strings.TrimSpace(segments[0]) == "" {
		return "", errors.New("invalid tool path")
	}

	decoded, err := url.PathUnescape(strings.TrimSpace(segments[0]))
	if err != nil {
		return "", errors.New("invalid tool path")
	}
	decoded = strings.TrimSpace(decoded)
	if decoded == "" || strings.Contains(decoded, "/") {
		return "", errors.New("invalid tool path")
	}
	return decoded, nil
}
