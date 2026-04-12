// bridgeconfig.Config and provider HTTP handlers.

package transport

import (
	bridgeorchestration "ghost-os/bridge/orchestration"
	"net/http"
	"net/url"
	"strings"
)

func (t *transport) handleConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		traceID := resolveTraceID("", r)
		t.dispatchActionObject(w, r, actionConfigGet, map[string]any{}, traceID)
	case http.MethodPost:
		var req bridgeorchestration.ConfigUpdateRequest
		if !decodeBodyOrWriteError(w, r, t.maxBodyBytes, &req) {
			return
		}

		traceID := resolveTraceID(req.TraceID, r)
		t.dispatchActionObject(w, r, actionConfigUpdate, req, traceID)
	default:
		writeMethodNotAllowed(w)
	}
}

func (t *transport) handleConfigProviders(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		traceID := resolveTraceID("", r)
		result, err := t.service.ExecuteProvidersGetAction(traceID)
		respondServiceContractResult(w, traceID, result, err)
	case http.MethodPost:
		var req bridgeorchestration.ProviderCreateRequest
		if !decodeBodyOrWriteError(w, r, t.maxBodyBytes, &req) {
			return
		}
		traceID := resolveTraceID(req.TraceID, r)
		result, err := t.service.ExecuteProviderCreateAction(req, traceID)
		respondServiceContractResult(w, traceID, result, err)
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
		var req bridgeorchestration.ProviderUpdateRequest
		if !decodeBodyOrWriteError(w, r, t.maxBodyBytes, &req) {
			return
		}
		traceID := resolveTraceID(req.TraceID, r)
		result, err := t.service.ExecuteProviderUpdateAction(name, req, traceID)
		respondServiceContractResult(w, traceID, result, err)
	case http.MethodDelete:
		traceID := resolveTraceID("", r)
		result, err := t.service.ExecuteProviderDeleteAction(name, traceID)
		respondServiceContractResult(w, traceID, result, err)
	default:
		writeMethodNotAllowed(w)
	}
}

func (t *transport) handleActiveProvider(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPut) {
		return
	}

	var req bridgeorchestration.SetActiveProviderRequest
	if !decodeBodyOrWriteError(w, r, t.maxBodyBytes, &req) {
		return
	}

	traceID := resolveTraceID(req.TraceID, r)
	result, err := t.service.ExecuteSetActiveProviderAction(req, traceID)
	respondServiceContractResult(w, traceID, result, err)
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
