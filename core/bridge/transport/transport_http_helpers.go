// Shared HTTP response/request helpers to keep handler behavior consistent.

package transport

import (
	"encoding/json"
	"errors"
	"fmt"
	bridgeorchestration "ghost-os/bridge/orchestration"
	"io"
	"log"
	"net/http"
	"strings"
	"sync/atomic"
	"time"
)

var (
	traceCounter         uint64
	errRequestBodyTooBig = errors.New("request body too large")
)

// decodeJSONBody 解码并校验单个 JSON 对象，拒绝未知字段和多段 JSON。
func decodeJSONBody(w http.ResponseWriter, r *http.Request, maxBytes int64, target any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		var maxErr *http.MaxBytesError
		if errors.As(err, &maxErr) {
			return errRequestBodyTooBig
		}
		if errors.Is(err, io.EOF) {
			return errors.New("invalid JSON body: empty request body")
		}
		return fmt.Errorf("invalid JSON body: %w", err)
	}

	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); err == nil {
		return errors.New("invalid JSON body: multiple JSON values are not allowed")
	} else if !errors.Is(err, io.EOF) {
		return fmt.Errorf("invalid JSON body: %w", err)
	}
	return nil
}

// decodeStatusCode 将输入解码错误映射为稳定 HTTP 状态码。
func decodeStatusCode(err error) int {
	if errors.Is(err, errRequestBodyTooBig) {
		return http.StatusRequestEntityTooLarge
	}
	return http.StatusBadRequest
}

// writeMethodNotAllowed 输出统一的 405 envelope。
func writeMethodNotAllowed(w http.ResponseWriter) {
	writeError(w, http.StatusMethodNotAllowed, "method not allowed", "")
}

// requireMethod 要求请求方法匹配，不匹配时直接写错误并短路。
func requireMethod(w http.ResponseWriter, r *http.Request, method string) bool {
	if r.Method != method {
		writeMethodNotAllowed(w)
		return false
	}
	return true
}

// decodeBodyOrWriteError 在 handler 入口执行解码并集中处理错误输出。
func decodeBodyOrWriteError(w http.ResponseWriter, r *http.Request, maxBytes int64, target any) bool {
	if err := decodeJSONBody(w, r, maxBytes, target); err != nil {
		writeError(w, decodeStatusCode(err), err.Error(), "")
		return false
	}
	return true
}

// resolveTraceID 依次使用 payload、header，再回退到本地生成值。
func resolveTraceID(candidate string, r *http.Request) string {
	if traceID := strings.TrimSpace(candidate); traceID != "" {
		return traceID
	}
	if traceID := strings.TrimSpace(r.Header.Get("X-Trace-ID")); traceID != "" {
		return traceID
	}
	return nextTraceID()
}

// nextTraceID 生成低冲突 trace id，便于本地日志串联单次调用链。
func nextTraceID() string {
	sequence := atomic.AddUint64(&traceCounter, 1)
	return fmt.Sprintf("bridge-%d-%d", time.Now().UnixMilli(), sequence)
}

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

// respondServiceResult 处理 service 返回值并写回统一 envelope。
func respondServiceResult(w http.ResponseWriter, traceID string, payload any, code int, err error) bool {
	if err != nil {
		writeError(w, code, err.Error(), traceID)
		return false
	}
	if code <= 0 {
		code = http.StatusOK
	}
	writeSuccess(w, code, payload, traceID)
	return true
}

// respondActionResult 附带 action 级日志记录，便于排查分发链路问题。
func respondActionResult(w http.ResponseWriter, traceID string, action string, payload any, code int, err error) bool {
	if err != nil {
		logAction(traceID, action, "error", err)
		writeError(w, code, err.Error(), traceID)
		return false
	}
	if code <= 0 {
		code = http.StatusOK
	}
	writeSuccess(w, code, payload, traceID)
	return true
}

func respondServiceContractResult(
	w http.ResponseWriter,
	traceID string,
	result bridgeorchestration.ServiceResult,
	err error,
) bool {
	if err != nil {
		writeError(w, bridgeorchestration.LegacyStatusFromServiceError(err), err.Error(), traceID)
		return false
	}
	writeSuccess(w, bridgeorchestration.LegacyStatusFromServiceOutcome(result.Outcome), result.Payload, traceID)
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
		writeError(w, bridgeorchestration.LegacyStatusFromServiceError(err), err.Error(), traceID)
		return false
	}
	writeSuccess(w, bridgeorchestration.LegacyStatusFromServiceOutcome(result.Outcome), result.Payload, traceID)
	return true
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
