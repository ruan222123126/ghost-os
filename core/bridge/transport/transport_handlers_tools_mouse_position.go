package transport

import (
	bridgeorchestration "ghost-os/bridge/orchestration"
	"net/http"
)

func (t *transport) handleMousePosition(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
	var req bridgeorchestration.MousePositionRequest
	if !decodeBodyOrWriteError(w, r, t.maxBodyBytes, &req) {
		return
	}
	traceID := resolveTraceID(req.TraceID, r)
	result, err := t.service.ExecuteMousePositionAction(r.Context(), req, traceID)
	respondServiceContractResult(w, traceID, result, err)
}
