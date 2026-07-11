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
	result, err := t.usecases.agent.Send(r.Context(), bridgeorchestration.AgentParams{
		Mode:             req.Mode,
		Message:          req.Message,
		Images:           req.Images,
		SessionID:        req.SessionID,
		ProjectRoot:      req.ProjectRoot,
		RuntimeOverrides: req.RuntimeOverrides,
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
	result, err := t.usecases.agent.Answer(r.Context(), req, traceID)
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
	_, sessionID, err := t.usecases.agent.AnswerStream(r.Context(), req, traceID, sink)
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
	params := bridgeorchestration.AgentParams{
		Mode:             req.Mode,
		Message:          req.Message,
		Images:           req.Images,
		SessionID:        req.SessionID,
		ProjectRoot:      req.ProjectRoot,
		RuntimeOverrides: req.RuntimeOverrides,
	}
	prepared, result, err := t.usecases.agent.PrepareStream(r.Context(), params, traceID)
	if err != nil {
		respondServiceContractActionResult(w, traceID, bridgeorchestration.BusActionAgentSend, result, err)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.Header().Set("X-Trace-ID", traceID)

	sink := newObservedSSEStreamSink(newSSEEventSink(w, flusher, traceID))
	_, sessionID, err := prepared.Run(r.Context(), sink)
	emitUnhandledStreamError(r.Context(), sink, traceID, firstNonEmpty(sessionID, req.SessionID), err)
}

func (t *transport) handleExternalAgentStream(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodPost) {
		return
	}

	var req bridgeorchestration.ExternalAgentRequest
	if !decodeBodyOrWriteError(w, r, t.maxBodyBytes, &req) {
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "streaming not supported", "")
		return
	}

	traceID := resolveTraceID("", r)
	prepared, result, err := t.usecases.agent.PrepareExternalStream(req, traceID, true)
	if err != nil {
		respondServiceContractActionResult(w, traceID, bridgeorchestration.BusActionExternalAgentStart, result, err)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.Header().Set("X-Trace-ID", traceID)

	sink := newObservedSSEStreamSink(newSSEEventSink(w, flusher, traceID))
	_, sessionID, err := prepared.Run(r.Context(), sink)
	emitUnhandledStreamError(r.Context(), sink, traceID, firstNonEmpty(sessionID, req.SessionID), err)
}

// dispatchAction 统一调用 service 并按 action 语义输出响应 envelope。
func (t *transport) dispatchAction(w http.ResponseWriter, r *http.Request, action string, params json.RawMessage, traceID string) {
	result, err := t.usecases.bus.Dispatch(r.Context(), action, params, traceID)
	respondServiceContractActionResult(w, traceID, action, result, err)
}
