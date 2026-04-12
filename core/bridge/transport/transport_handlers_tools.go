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
	result, err := t.service.ExecuteToolListAction(traceID)
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
	result, callErr := t.service.ExecuteToolUpdateAction(bridgeorchestration.ToolNameParams{Name: name}, req, traceID)
	respondServiceContractResult(w, traceID, result, callErr)
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
