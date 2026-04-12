// Bus and agent-facing HTTP handlers.

package transport

import (
	"encoding/json"
	bridgeorchestration "ghost-os/bridge/orchestration"
	"net/http"
	"strings"
)

func (t *transport) handleBus(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}

	var req bridgeorchestration.APIRequest
	if !decodeBodyOrWriteError(w, r, t.maxBodyBytes, &req) {
		return
	}

	if err := bridgeorchestration.ValidateBusRequest(req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error(), "")
		return
	}

	traceID := resolveTraceID(req.TraceID, r)
	action := strings.ToUpper(strings.TrimSpace(req.Action))
	t.dispatchAction(w, r, action, req.Params, traceID)
}

// handleAgent 兼容简化 agent API，并转成 AGENT_SEND action 的标准调用。
func (t *transport) handleAgent(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}

	var req bridgeorchestration.AgentRequest
	if !decodeBodyOrWriteError(w, r, t.maxBodyBytes, &req) {
		return
	}

	traceID := resolveTraceID(req.TraceID, r)
	result, err := t.service.ExecuteAgentAction(r.Context(), bridgeorchestration.AgentParams{
		Mode:      req.Mode,
		Message:   req.Message,
		Images:    req.Images,
		SessionID: req.SessionID,
	}, traceID)
	respondServiceContractActionResult(w, traceID, bridgeorchestration.BusActionAgentSend, result, err)
}

// handleQuestionAnswer 以高层接口隐藏 HUMAN_RESPONSE + resume 的底层编排细节。
func (t *transport) handleQuestionAnswer(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}

	var req bridgeorchestration.HumanResponseParams
	if !decodeBodyOrWriteError(w, r, t.maxBodyBytes, &req) {
		return
	}

	traceID := resolveTraceID("", r)
	result, err := t.service.ExecuteHumanAnswerAndResumeAction(r.Context(), req, traceID)
	respondServiceContractResult(w, traceID, result, err)
}

func (t *transport) handleQuestionAnswerStream(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}

	var req bridgeorchestration.HumanResponseParams
	if !decodeBodyOrWriteError(w, r, t.maxBodyBytes, &req) {
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming not supported", "")
		return
	}

	traceID := resolveTraceID("", r)
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.Header().Set("X-Trace-ID", traceID)

	sink := newObservedSSEStreamSink(newSSEEventSink(w, flusher, traceID))
	_, sessionID, err := t.service.ExecuteHumanAnswerAndResumeStreamAction(r.Context(), req, traceID, sink)
	emitUnhandledStreamError(r.Context(), sink, traceID, firstNonEmpty(sessionID, req.SessionID), err)
}

func (t *transport) handleAgentStream(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}

	var req bridgeorchestration.AgentRequest
	if !decodeBodyOrWriteError(w, r, t.maxBodyBytes, &req) {
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming not supported", "")
		return
	}

	traceID := resolveTraceID(req.TraceID, r)
	if strings.TrimSpace(req.Message) != "" || len(req.Images) > 0 {
		if code, err := t.service.EnsureSessionNotInflight(req.SessionID); err != nil {
			respondServiceResult(w, traceID, nil, code, err)
			return
		}
		if code, err := t.service.EnsureSessionActive(req.SessionID); err != nil {
			respondServiceResult(w, traceID, nil, code, err)
			return
		}
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.Header().Set("X-Trace-ID", traceID)

	sink := newObservedSSEStreamSink(newSSEEventSink(w, flusher, traceID))
	_, sessionID, err := t.service.ExecuteAgentStreamAction(r.Context(), bridgeorchestration.AgentParams{
		Mode:      req.Mode,
		Message:   req.Message,
		Images:    req.Images,
		SessionID: req.SessionID,
	}, traceID, sink)
	emitUnhandledStreamError(r.Context(), sink, traceID, firstNonEmpty(sessionID, req.SessionID), err)
}

// dispatchAction 统一调用 service 并按 action 语义输出响应 envelope。
func (t *transport) dispatchAction(w http.ResponseWriter, r *http.Request, action string, params json.RawMessage, traceID string) {
	result, err := t.service.DispatchAction(r.Context(), action, params, traceID)
	respondServiceContractActionResult(w, traceID, action, result, err)
}

func (t *transport) dispatchActionObject(
	w http.ResponseWriter,
	r *http.Request,
	action string,
	params any,
	traceID string,
) bool {
	raw, err := json.Marshal(params)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to encode action params", traceID)
		return false
	}
	t.dispatchAction(w, r, action, raw, traceID)
	return true
}
