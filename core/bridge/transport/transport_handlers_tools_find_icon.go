package transport

import (
	"fmt"
	bridgeorchestration "ghost-os/bridge/orchestration"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

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
	result, err := t.service.ExecuteFindIconTemplateUploadAction(req, traceID)
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
	result, err := t.service.ExecuteFindIconPreviewAction(r.Context(), req, traceID)
	respondServiceContractResult(w, traceID, result, err)
}

func (t *transport) handleFindIconTemplateDownload(w http.ResponseWriter, r *http.Request) {
	traceID := resolveTraceID("", r)
	templatePath, code, err := parseFindIconTemplateDownloadPath(r)
	if err != nil {
		writeError(w, code, err.Error(), traceID)
		return
	}
	file, info, code, err := openFindIconTemplateFile(templatePath)
	if err != nil {
		writeError(w, code, err.Error(), traceID)
		return
	}
	defer file.Close()

	if traceID != "" {
		w.Header().Set("X-Trace-ID", traceID)
	}
	writeFindIconTemplateDownloadHeaders(w, templatePath, info.Size())
	w.WriteHeader(http.StatusOK)
	_, _ = io.Copy(w, file)
}

func parseFindIconTemplateDownloadPath(r *http.Request) (string, int, error) {
	templatePath := strings.TrimSpace(r.URL.Query().Get("template_path"))
	if templatePath == "" {
		return "", http.StatusBadRequest, fmt.Errorf("template_path is required")
	}
	absolutePath, err := filepath.Abs(templatePath)
	if err != nil {
		return "", http.StatusBadRequest, fmt.Errorf("template_path is invalid")
	}
	rootPath, err := bridgeorchestration.ResolveFindIconTemplateRootPath()
	if err != nil {
		return "", http.StatusInternalServerError, fmt.Errorf("resolve find_icon template root: %w", err)
	}
	withinRoot, err := isPathWithinRoot(filepath.Clean(absolutePath), rootPath)
	if err != nil {
		return "", http.StatusBadRequest, fmt.Errorf("template_path is invalid")
	}
	if !withinRoot {
		return "", http.StatusBadRequest, fmt.Errorf("template_path must be inside workflow template root")
	}
	return filepath.Clean(absolutePath), http.StatusOK, nil
}

func openFindIconTemplateFile(path string) (*os.File, os.FileInfo, int, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, http.StatusNotFound, fmt.Errorf("template image not found")
		}
		return nil, nil, http.StatusInternalServerError, err
	}
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, nil, http.StatusInternalServerError, err
	}
	if info.IsDir() {
		_ = file.Close()
		return nil, nil, http.StatusBadRequest, fmt.Errorf("template_path must point to an image file")
	}
	return file, info, http.StatusOK, nil
}

func isPathWithinRoot(path string, root string) (bool, error) {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false, err
	}
	if rel == ".." {
		return false, nil
	}
	return !strings.HasPrefix(rel, ".."+string(filepath.Separator)), nil
}

func writeFindIconTemplateDownloadHeaders(w http.ResponseWriter, path string, size int64) {
	w.Header().Set("Content-Type", resolveFindIconTemplateMimeType(path))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", size))
	w.Header().Set(
		"Content-Disposition",
		fmt.Sprintf("inline; filename=%q", filepath.Base(path)),
	)
}

func resolveFindIconTemplateMimeType(path string) string {
	byExtension := mime.TypeByExtension(strings.ToLower(filepath.Ext(path)))
	if byExtension != "" {
		return byExtension
	}
	return "application/octet-stream"
}
