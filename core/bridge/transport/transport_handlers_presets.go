package transport

import (
	"errors"
	"net/http"
	"net/url"
	"strings"

	bridgeconfig "ghost-os/bridge/config"
)

func (t *transport) handlePresets(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		traceID := resolveTraceID("", r)
		result, err := t.service.ExecutePresetListAction(traceID)
		respondServiceContractResult(w, traceID, result, err)
	case http.MethodPost:
		var req bridgeconfig.PresetCreateRequest
		if !decodeBodyOrWriteError(w, r, t.maxBodyBytes, &req) {
			return
		}
		traceID := resolveTraceID(req.TraceID, r)
		result, err := t.service.ExecutePresetCreateAction(req, traceID)
		respondServiceContractResult(w, traceID, result, err)
	default:
		writeMethodNotAllowed(w)
	}
}

func (t *transport) handlePresetByID(w http.ResponseWriter, r *http.Request) {
	if presetID, ok := presetActivationIDFromPath(r.URL.Path); ok {
		handlePresetActivationRequest(t, w, r, presetID)
		return
	}

	presetID, err := presetIDFromPath(r.URL.Path)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error(), "")
		return
	}

	switch r.Method {
	case http.MethodPatch:
		var req bridgeconfig.PresetUpdateRequest
		if !decodeBodyOrWriteError(w, r, t.maxBodyBytes, &req) {
			return
		}
		traceID := resolveTraceID(req.TraceID, r)
		result, callErr := t.service.ExecutePresetUpdateAction(presetID, req, traceID)
		respondServiceContractResult(w, traceID, result, callErr)
	case http.MethodDelete:
		traceID := resolveTraceID("", r)
		result, callErr := t.service.ExecutePresetDeleteAction(presetID, traceID)
		respondServiceContractResult(w, traceID, result, callErr)
	default:
		writeMethodNotAllowed(w)
	}
}

func handlePresetActivationRequest(
	t *transport,
	w http.ResponseWriter,
	r *http.Request,
	presetID string,
) {
	if r.Method != http.MethodPut {
		writeMethodNotAllowed(w)
		return
	}

	traceID := resolveTraceID("", r)
	result, err := t.service.ExecutePresetApplyAction(presetID, traceID)
	respondServiceContractResult(w, traceID, result, err)
}

func presetIDFromPath(rawPath string) (string, error) {
	path := strings.TrimSpace(strings.TrimPrefix(rawPath, "/api/presets/"))
	if path == "" {
		return "", errors.New("preset id is required")
	}
	segments := strings.Split(path, "/")
	if len(segments) != 1 || strings.TrimSpace(segments[0]) == "" {
		return "", errors.New("invalid preset path")
	}
	decoded, err := url.PathUnescape(strings.TrimSpace(segments[0]))
	if err != nil {
		return "", errors.New("invalid preset path")
	}
	decoded = strings.TrimSpace(decoded)
	if decoded == "" || strings.Contains(decoded, "/") {
		return "", errors.New("invalid preset path")
	}
	return decoded, nil
}

func presetActivationIDFromPath(rawPath string) (string, bool) {
	path := strings.TrimSpace(strings.TrimPrefix(rawPath, "/api/presets/"))
	segments := strings.Split(path, "/")
	if len(segments) != 2 || strings.TrimSpace(segments[1]) != "activate" {
		return "", false
	}

	decoded, err := url.PathUnescape(strings.TrimSpace(segments[0]))
	if err != nil {
		return "", false
	}
	decoded = strings.TrimSpace(decoded)
	if decoded == "" || strings.Contains(decoded, "/") {
		return "", false
	}
	return decoded, true
}
