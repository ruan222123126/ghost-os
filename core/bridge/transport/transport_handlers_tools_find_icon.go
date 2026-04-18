package transport

import (
	bridgeorchestration "ghost-os/bridge/orchestration"
	"net/http"
)

func (t *transport) handleFindIconTemplateUpload(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}
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
