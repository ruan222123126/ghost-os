// Bus transport handlers map actions to concrete service use cases.

package app

import (
	"encoding/json"
	"net/http"
	"strings"
)

// handleBus 处理统一 bus 入口：解码 envelope、校验 action，再分发到 service。
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
		SessionID: req.SessionID,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to encode agent request", traceID)
		return
	}

	t.dispatchAction(w, r, actionAgentSend, params, traceID)
}

// dispatchAction 统一调用 service 并按 action 语义输出响应 envelope。
func (t *transport) dispatchAction(w http.ResponseWriter, r *http.Request, action string, params json.RawMessage, traceID string) {
	payload, code, err := t.service.dispatchAction(r.Context(), action, params, traceID)
	respondActionResult(w, traceID, action, payload, code, err)
}

// handleConfig 提供配置读写路由：GET 读取快照，POST 更新并返回最新配置。
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

// handleSessionsList 列出会话概要，供 Web/CLI 构建侧边栏或历史视图。
func (t *transport) handleSessionsList(w http.ResponseWriter, r *http.Request) {
	if !requireMethod(w, r, http.MethodGet) {
		return
	}

	traceID := resolveTraceID("", r)
	payload, code, err := t.service.executeSessionsListAction(traceID)
	respondServiceResult(w, traceID, payload, code, err)
}

// handleSessionByID 处理单会话查询与删除，并在路径层面做 session id 基本校验。
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
