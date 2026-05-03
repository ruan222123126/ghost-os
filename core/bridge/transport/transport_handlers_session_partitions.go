package transport

import (
	"net/http"

	bridgeorchestration "ghost-os/bridge/orchestration"
)

func (t *transport) handleSessionPartitions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		traceID := resolveTraceID("", r)
		result, err := t.service.ExecuteSessionSidebarPartitionsGetAction(traceID)
		respondServiceContractResult(w, traceID, result, err)
	case http.MethodPut:
		var req bridgeorchestration.SessionSidebarPartitionPutRequest
		if !decodeBodyOrWriteError(w, r, t.maxBodyBytes, &req) {
			return
		}
		traceID := resolveTraceID(req.TraceID, r)
		result, err := t.service.ExecuteSessionSidebarPartitionsPutAction(req, traceID)
		respondServiceContractResult(w, traceID, result, err)
	default:
		writeMethodNotAllowed(w)
	}
}
