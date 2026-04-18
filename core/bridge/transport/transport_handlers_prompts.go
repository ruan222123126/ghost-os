package transport

import (
	bridgeconfig "ghost-os/bridge/config"
	"net/http"
)

func (t *transport) handleSystemPrompts(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		traceID := resolveTraceID("", r)
		result, err := t.service.ExecuteSystemPromptGetAction(traceID)
		respondServiceContractResult(w, traceID, result, err)
	case http.MethodPatch:
		var req bridgeconfig.SystemPromptUpdateRequest
		if !decodeBodyOrWriteError(w, r, t.maxBodyBytes, &req) {
			return
		}

		traceID := resolveTraceID(req.TraceID, r)
		result, err := t.service.ExecuteSystemPromptUpdateAction(req, traceID)
		respondServiceContractResult(w, traceID, result, err)
	default:
		writeMethodNotAllowed(w)
	}
}
