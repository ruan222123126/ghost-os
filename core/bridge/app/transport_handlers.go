package app

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

	traceID := strings.TrimSpace(req.TraceID)
	action := strings.ToUpper(strings.TrimSpace(req.Action))
	t.dispatchAction(w, r, action, req.Params, traceID)
}

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
		SessionID: req.SessionID,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to encode agent request", traceID)
		return
	}

	t.dispatchAction(w, r, actionAgentSend, params, traceID)
}

func (t *transport) dispatchAction(w http.ResponseWriter, r *http.Request, action string, params json.RawMessage, traceID string) {
	payload, code, err := t.service.dispatchAction(r.Context(), action, params, traceID)
	respondActionResult(w, traceID, action, payload, code, err)
}

func (t *transport) handleConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		traceID := resolveTraceID("", r)
		payload, code, err := t.service.executeConfigGetAction(traceID)
		respondServiceResult(w, traceID, payload, code, err)
	case http.MethodPost:
		var req configUpdateRequest
		if !decodeBodyOrWriteError(w, r, t.maxBodyBytes, &req) {
			return
		}

		traceID := resolveTraceID(req.TraceID, r)
		payload, code, err := t.service.executeConfigUpdateAction(req, traceID)
		respondServiceResult(w, traceID, payload, code, err)
	default:
		writeMethodNotAllowed(w)
	}
}

func (t *transport) handleSessionsList(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}

	traceID := resolveTraceID("", r)
	payload, code, err := t.service.executeSessionsListAction(traceID)
	respondServiceResult(w, traceID, payload, code, err)
}

func (t *transport) handleSessionByID(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(strings.TrimPrefix(r.URL.Path, "/api/sessions/"))
	if id == "" || strings.Contains(id, "/") {
		writeError(w, http.StatusBadRequest, "session id is required", "")
		return
	}

	params := sessionIDParams{ID: id}
	traceID := resolveTraceID("", r)

	switch r.Method {
	case http.MethodGet:
		payload, code, err := t.service.executeSessionGetAction(params, traceID)
		respondServiceResult(w, traceID, payload, code, err)
	case http.MethodDelete:
		payload, code, err := t.service.executeSessionDeleteAction(params, traceID)
		respondServiceResult(w, traceID, payload, code, err)
	default:
		writeMethodNotAllowed(w)
	}
}
