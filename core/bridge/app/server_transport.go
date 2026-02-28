package app

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"ghost-os/bridge/session"
)

type corsPolicy struct {
	allowedOrigins map[string]struct{}
}

func newCORSPolicyFromEnv() corsPolicy {
	raw := strings.TrimSpace(os.Getenv("GHOST_CORS_ORIGINS"))
	allowed := make(map[string]struct{})
	if raw == "" {
		return corsPolicy{allowedOrigins: allowed}
	}

	for _, origin := range strings.Split(raw, ",") {
		trimmed := strings.TrimSpace(origin)
		if trimmed == "" {
			continue
		}
		allowed[trimmed] = struct{}{}
	}
	return corsPolicy{allowedOrigins: allowed}
}

func (p corsPolicy) allows(origin string) bool {
	if origin == "" {
		return true
	}
	_, ok := p.allowedOrigins[origin]
	return ok
}

type apiTokenAuth struct {
	token string
}

func newAPITokenAuthFromEnv() apiTokenAuth {
	return apiTokenAuth{token: strings.TrimSpace(os.Getenv("GHOST_API_TOKEN"))}
}

func (a apiTokenAuth) enabled() bool {
	return a.token != ""
}

func (a apiTokenAuth) authorized(r *http.Request) bool {
	if !a.enabled() {
		return true
	}

	provided := strings.TrimSpace(r.Header.Get("X-API-Token"))
	if provided == "" {
		provided = parseBearerToken(r.Header.Get("Authorization"))
	}
	if provided == "" {
		return false
	}

	return subtle.ConstantTimeCompare([]byte(provided), []byte(a.token)) == 1
}

func parseBearerToken(header string) string {
	parts := strings.Fields(strings.TrimSpace(header))
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

type serverOptions struct {
	bindAddr     string
	maxBodyBytes int64
	cors         corsPolicy
	auth         apiTokenAuth
}

func newServerOptionsFromEnv(port int) serverOptions {
	return serverOptions{
		bindAddr:     resolveBindAddr(port),
		maxBodyBytes: defaultMaxRequestBodyBytes,
		cors:         newCORSPolicyFromEnv(),
		auth:         newAPITokenAuthFromEnv(),
	}
}

func resolveBindAddr(port int) string {
	if configured := strings.TrimSpace(os.Getenv("GHOST_BIND_ADDR")); configured != "" {
		return configured
	}
	return net.JoinHostPort("127.0.0.1", fmt.Sprintf("%d", port))
}

// runServer 暴露 bridge HTTP API。
func runServer(ctx context.Context, port int) (string, error) {
	store, err := NewConfigStoreFromEnv()
	if err != nil {
		return "", err
	}

	sessionStore, err := session.NewStore(sessionsPathFromEnv())
	if err != nil {
		return "", err
	}

	service := newBridgeService(store, sessionStore, runAgentWithSession)
	options := newServerOptionsFromEnv(port)
	server := &http.Server{
		Addr:              options.bindAddr,
		Handler:           newHTTPHandler(service, options),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	logAddr := options.bindAddr
	if strings.HasPrefix(logAddr, ":") {
		logAddr = "0.0.0.0" + logAddr
	}
	log.Printf("bridge HTTP server listening on http://%s", logAddr)
	err = server.ListenAndServe()
	if err == nil || errors.Is(err, http.ErrServerClosed) {
		return "", nil
	}
	return "", err
}

type transport struct {
	service      *bridgeService
	maxBodyBytes int64
}

func newHTTPHandler(service *bridgeService, options serverOptions) http.Handler {
	transport := &transport{
		service:      service,
		maxBodyBytes: options.maxBodyBytes,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/bus", transport.handleBus)
	mux.HandleFunc("/api/agent", transport.handleAgent)
	mux.HandleFunc("/api/config", transport.handleConfig)
	mux.HandleFunc("/api/sessions", transport.handleSessionsList)
	mux.HandleFunc("/api/sessions/", transport.handleSessionByID)

	return withCORS(options.cors, withAuth(options.auth, mux))
}

func withAuth(auth apiTokenAuth, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}
		if auth.authorized(r) {
			next.ServeHTTP(w, r)
			return
		}
		writeError(w, http.StatusUnauthorized, "unauthorized", "")
	})
}

func withCORS(policy corsPolicy, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := strings.TrimSpace(r.Header.Get("Origin"))
		if origin != "" && !policy.allows(origin) {
			writeError(w, http.StatusForbidden, "origin is not allowed", "")
			return
		}

		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Trace-ID, X-API-Token, Authorization")
			w.Header().Set("Access-Control-Expose-Headers", "X-Trace-ID")
			w.Header().Set("Access-Control-Allow-Methods", "GET,POST,DELETE,OPTIONS")
			w.Header().Add("Vary", "Origin")
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (t *transport) handleBus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed", "")
		return
	}

	var req apiRequest
	if err := decodeJSONBody(w, r, t.maxBodyBytes, &req); err != nil {
		writeError(w, decodeStatusCode(err), err.Error(), "")
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
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed", "")
		return
	}

	var req agentRequest
	if err := decodeJSONBody(w, r, t.maxBodyBytes, &req); err != nil {
		writeError(w, decodeStatusCode(err), err.Error(), "")
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
	if err != nil {
		logAction(traceID, action, "error", err)
		writeError(w, code, err.Error(), traceID)
		return
	}

	writeSuccess(w, http.StatusOK, payload, traceID)
}

func (t *transport) handleConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		traceID := resolveTraceID("", r)
		payload, code, err := t.service.executeConfigGetAction(traceID)
		if err != nil {
			writeError(w, code, err.Error(), traceID)
			return
		}
		writeSuccess(w, http.StatusOK, payload, traceID)
	case http.MethodPost:
		var req configUpdateRequest
		if err := decodeJSONBody(w, r, t.maxBodyBytes, &req); err != nil {
			writeError(w, decodeStatusCode(err), err.Error(), "")
			return
		}

		traceID := resolveTraceID(req.TraceID, r)
		payload, code, err := t.service.executeConfigUpdateAction(req, traceID)
		if err != nil {
			writeError(w, code, err.Error(), traceID)
			return
		}
		writeSuccess(w, http.StatusOK, payload, traceID)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed", "")
	}
}

func (t *transport) handleSessionsList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed", "")
		return
	}

	traceID := resolveTraceID("", r)
	payload, code, err := t.service.executeSessionsListAction(traceID)
	if err != nil {
		writeError(w, code, err.Error(), traceID)
		return
	}
	writeSuccess(w, http.StatusOK, payload, traceID)
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
		if err != nil {
			writeError(w, code, err.Error(), traceID)
			return
		}
		writeSuccess(w, http.StatusOK, payload, traceID)
	case http.MethodDelete:
		payload, code, err := t.service.executeSessionDeleteAction(params, traceID)
		if err != nil {
			writeError(w, code, err.Error(), traceID)
			return
		}
		writeSuccess(w, http.StatusOK, payload, traceID)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method not allowed", "")
	}
}
