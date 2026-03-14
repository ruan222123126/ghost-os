// Config and provider HTTP handlers.

package transport

import (
	"net/http"
	"net/url"
	"strings"
)

func (t *transport) handleConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		traceID := resolveTraceID("", r)
		payload, code, err := t.service.executeConfigGetAction(traceID)
		respondServiceResult(w, traceID, payload, code, err)
	case http.MethodPost:
		var req configUpdateRequest
		if !decodeBodyOrWriteError(w, r, t.maxBodyBytes, &req) {
			return
		}

		traceID := resolveTraceID(req.TraceID, r)
		payload, code, err := t.service.executeConfigUpdateAction(req, traceID)
		respondServiceResult(w, traceID, payload, code, err)
	default:
		writeMethodNotAllowed(w)
	}
}

func (t *transport) handleConfigProviders(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		traceID := resolveTraceID("", r)
		payload, code, err := t.service.executeProvidersGetAction(traceID)
		respondServiceResult(w, traceID, payload, code, err)
	case http.MethodPost:
		var req providerCreateRequest
		if !decodeBodyOrWriteError(w, r, t.maxBodyBytes, &req) {
			return
		}
		traceID := resolveTraceID(req.TraceID, r)
		payload, code, err := t.service.executeProviderCreateAction(req, traceID)
		respondServiceResult(w, traceID, payload, code, err)
	default:
		writeMethodNotAllowed(w)
	}
}

func (t *transport) handleConfigProviderByName(w http.ResponseWriter, r *http.Request) {
	name, ok := providerNameFromPath(r.URL.Path)
	if !ok {
		writeError(w, http.StatusBadRequest, "provider name is required", "")
		return
	}

	switch r.Method {
	case http.MethodPut:
		var req providerUpdateRequest
		if !decodeBodyOrWriteError(w, r, t.maxBodyBytes, &req) {
			return
		}
		traceID := resolveTraceID(req.TraceID, r)
		payload, code, err := t.service.executeProviderUpdateAction(name, req, traceID)
		respondServiceResult(w, traceID, payload, code, err)
	case http.MethodDelete:
		traceID := resolveTraceID("", r)
		payload, code, err := t.service.executeProviderDeleteAction(name, traceID)
		respondServiceResult(w, traceID, payload, code, err)
	default:
		writeMethodNotAllowed(w)
	}
}

func (t *transport) handleActiveProvider(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPut) {
		return
	}

	var req setActiveProviderRequest
	if !decodeBodyOrWriteError(w, r, t.maxBodyBytes, &req) {
		return
	}

	traceID := resolveTraceID(req.TraceID, r)
	payload, code, err := t.service.executeSetActiveProviderAction(req, traceID)
	respondServiceResult(w, traceID, payload, code, err)
}

func providerNameFromPath(path string) (string, bool) {
	rawName := strings.TrimSpace(strings.TrimPrefix(path, "/api/config/providers/"))
	if rawName == "" || strings.Contains(rawName, "/") {
		return "", false
	}
	decoded, err := url.PathUnescape(rawName)
	if err != nil {
		return "", false
	}
	decoded = strings.TrimSpace(decoded)
	if decoded == "" || strings.Contains(decoded, "/") {
		return "", false
	}
	return decoded, true
}
