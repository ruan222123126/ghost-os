// Bus and agent-facing HTTP handlers.

package transport

import (
	"encoding/json"
	"net/http"
	"strings"
)

func (t *transport) handleBus(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}

	var req apiRequest
	if !decodeBodyOrWriteError(w, r, t.maxBodyBytes, &req) {
		return
	}

	if err := validateBusRequest(req); err != nil {
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

	var req agentRequest
	if !decodeBodyOrWriteError(w, r, t.maxBodyBytes, &req) {
		return
	}

	traceID := resolveTraceID(req.TraceID, r)
	params, err := json.Marshal(agentParams{
		Message:   req.Message,
		Images:    req.Images,
		SessionID: req.SessionID,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to encode agent request", traceID)
		return
	}

	t.dispatchAction(w, r, busActionAgentSend, params, traceID)
}

// handleQuestionAnswer 以高层接口隐藏 HUMAN_RESPONSE + resume 的底层编排细节。
func (t *transport) handleQuestionAnswer(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}

	var req humanResponseParams
	if !decodeBodyOrWriteError(w, r, t.maxBodyBytes, &req) {
		return
	}

	traceID := resolveTraceID("", r)
	payload, code, err := t.service.ExecuteHumanAnswerAndResumeAction(r.Context(), req, traceID)
	respondServiceResult(w, traceID, payload, code, err)
}

func (t *transport) handleQuestionAnswerStream(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}

	var req humanResponseParams
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

	sink := newSSEEventSink(w, flusher, traceID)
	_, _, _ = t.service.ExecuteHumanAnswerAndResumeStreamAction(r.Context(), req, traceID, sink)
}

func (t *transport) handleAgentStream(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}

	var req agentRequest
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

	sink := newSSEEventSink(w, flusher, traceID)
	_, _, _ = t.service.ExecuteAgentStreamAction(r.Context(), agentParams{
		Message:   req.Message,
		Images:    req.Images,
		SessionID: req.SessionID,
	}, traceID, sink)
}

// dispatchAction 统一调用 service 并按 action 语义输出响应 envelope。
func (t *transport) dispatchAction(w http.ResponseWriter, r *http.Request, action string, params json.RawMessage, traceID string) {
	payload, code, err := t.service.DispatchAction(r.Context(), action, params, traceID)
	respondActionResult(w, traceID, action, payload, code, err)
}
