package transport

import (
	"encoding/json"
	"log"
	"net/http"

	bridgeorchestration "ghost-os/bridge/orchestration"
)

// writeSuccess 输出 bus success envelope。
func writeSuccess(w http.ResponseWriter, code int, payload any, traceID string) {
	writeEnvelope(w, code, bridgeorchestration.APIResponse{
		Status:  bridgeorchestration.BusStatusSuccess,
		Payload: payload,
		Error:   "",
	}, traceID)
}

// writeError 输出 bus error envelope。
func writeError(w http.ResponseWriter, code int, message string, traceID string) {
	writeEnvelope(w, code, bridgeorchestration.APIResponse{
		Status:  bridgeorchestration.BusStatusError,
		Payload: map[string]any{},
		Error:   message,
	}, traceID)
}

func respondServiceContractResult(
	w http.ResponseWriter,
	traceID string,
	result bridgeorchestration.ServiceResult,
	err error,
) bool {
	if err != nil {
		writeError(w, httpStatusFromServiceError(err), err.Error(), traceID)
		return false
	}
	writeSuccess(w, httpStatusFromServiceOutcome(result.Outcome), result.Payload, traceID)
	return true
}

func respondServiceContractActionResult(
	w http.ResponseWriter,
	traceID string,
	action string,
	result bridgeorchestration.ServiceResult,
	err error,
) bool {
	if err != nil {
		logAction(traceID, action, "error", err)
		writeError(w, httpStatusFromServiceError(err), err.Error(), traceID)
		return false
	}
	writeSuccess(w, httpStatusFromServiceOutcome(result.Outcome), result.Payload, traceID)
	return true
}

func httpStatusFromServiceOutcome(outcome bridgeorchestration.ServiceOutcome) int {
	switch outcome {
	case bridgeorchestration.ServiceOutcomeCreated:
		return http.StatusCreated
	case bridgeorchestration.ServiceOutcomeAccepted:
		return http.StatusAccepted
	default:
		return http.StatusOK
	}
}

func httpStatusFromServiceError(err error) int {
	if err == nil {
		return http.StatusOK
	}
	return httpStatusFromServiceErrorKind(bridgeorchestration.ServiceErrorKindFromError(err))
}

func httpStatusFromServiceErrorKind(kind bridgeorchestration.ServiceErrorKind) int {
	switch kind {
	case bridgeorchestration.ServiceErrorInvalidInput:
		return http.StatusBadRequest
	case bridgeorchestration.ServiceErrorNotFound:
		return http.StatusNotFound
	case bridgeorchestration.ServiceErrorConflict:
		return http.StatusConflict
	case bridgeorchestration.ServiceErrorUnavailable:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}

// writeEnvelope 是所有响应的唯一出口，统一 header 与 payload 结构。
func writeEnvelope(w http.ResponseWriter, code int, response bridgeorchestration.APIResponse, traceID string) {
	w.Header().Set("Content-Type", "application/json")
	if traceID != "" {
		w.Header().Set("X-Trace-ID", traceID)
	}
	w.WriteHeader(code)
	if response.Payload == nil {
		response.Payload = map[string]any{}
	}
	_ = json.NewEncoder(w).Encode(response)
}

// logAction 记录 trace/action/status 三元组，错误时追加 error 字段。
func logAction(traceID string, action string, status string, err error) {
	if err != nil {
		log.Printf("trace_id=%s action=%s status=%s error=%v", traceID, action, status, err)
		return
	}
	log.Printf("trace_id=%s action=%s status=%s", traceID, action, status)
}
