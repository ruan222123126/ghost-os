package app

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"ghost-os/bridge/session"
)

type serverOptions struct {
	bindAddr     string
	maxBodyBytes int64
	cors         corsPolicy
	auth         apiTokenAuth
}

// newServerOptionsFromEnv 收敛 server 相关环境配置，避免 runServer 中散落解析逻辑。
func newServerOptionsFromEnv(port int) serverOptions {
	return serverOptions{
		bindAddr:     resolveBindAddr(port),
		maxBodyBytes: defaultMaxRequestBodyBytes,
		cors:         newCORSPolicyFromEnv(),
		auth:         newAPITokenAuthFromEnv(),
	}
}

// resolveBindAddr 优先使用显式绑定地址，否则回退到本地回环端口。
func resolveBindAddr(port int) string {
	if configured := strings.TrimSpace(getenvDefault("GHOST_BIND_ADDR", "")); configured != "" {
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

	service := newBridgeService(store, sessionStore, nil)
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

// newHTTPHandler 注册所有 HTTP 路由并挂载认证/CORS 中间件链。
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
