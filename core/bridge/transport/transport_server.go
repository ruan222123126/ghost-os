package transport

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	mobilewebrtc "ghost-os/bridge/mobile/webrtc"
	bridgeorchestration "ghost-os/bridge/orchestration"
)

type serverOptions struct {
	bindAddr     string
	sessionsPath string
	maxBodyBytes int64
	cors         corsPolicy
	auth         apiTokenAuth
	mobileWebRTC bridgeorchestration.MobileWebRTCConfig
}

const (
	serverReadHeaderTimeout = 5 * time.Second
	serverShutdownTimeout   = 5 * time.Second

	startupStageConfig       = "config"
	startupStageOptions      = "options"
	startupStageSessionStore = "session_store"
	startupStageRuntimes     = "runtimes"
	startupStageMobileWebRTC = "mobile_webrtc"
	startupStageListen       = "listen"
)

type servePreflightState struct {
	options serverOptions
	service *bridgeorchestration.Service
}

type serveStartupError struct {
	stage string
	err   error
}

func (e serveStartupError) Error() string {
	return fmt.Sprintf("serve startup failed at stage=%s: %v", e.stage, e.err)
}

func (e serveStartupError) Unwrap() error {
	return e.err
}

func newServeStartupError(stage string, err error) error {
	if err == nil {
		return nil
	}
	return serveStartupError{stage: stage, err: err}
}

// newServerOptionsFromEnv 收敛 server 相关配置，优先读配置文件并回退环境变量。
func newServerOptionsFromEnv(port int) (serverOptions, error) {
	cfg, err := bridgeorchestration.LoadServerConfig()
	if err != nil {
		return serverOptions{}, err
	}

	return serverOptions{
		bindAddr:     resolveBindAddrFromConfig(cfg, port),
		sessionsPath: cfg.SessionsPath,
		maxBodyBytes: bridgeorchestration.DefaultMaxRequestBodyBytes,
		cors:         newCORSPolicy(cfg.CORSOrigins),
		auth:         newAPITokenAuth(cfg.APIToken),
		mobileWebRTC: cfg.MobileWebRTC,
	}, nil
}

func resolveBindAddrFromConfig(cfg bridgeorchestration.ServerConfig, port int) string {
	if configured := strings.TrimSpace(cfg.BindAddr); configured != "" {
		return configured
	}
	return net.JoinHostPort("127.0.0.1", fmt.Sprintf("%d", port))
}

// resolveBindAddr 优先使用显式绑定地址，否则回退到本地回环端口。
// 注意：当配置文件存在但读取/解析失败时，返回错误以避免 fail-open。
func resolveBindAddr(port int) (string, error) {
	cfg, err := bridgeorchestration.LoadServerConfig()
	if err != nil {
		return "", err
	}
	return resolveBindAddrFromConfig(cfg, port), nil
}

func newCORSPolicyFromEnv() (corsPolicy, error) {
	cfg, err := bridgeorchestration.LoadServerConfig()
	if err != nil {
		return corsPolicy{}, err
	}
	return newCORSPolicy(cfg.CORSOrigins), nil
}

func newAPITokenAuthFromEnv() (apiTokenAuth, error) {
	cfg, err := bridgeorchestration.LoadServerConfig()
	if err != nil {
		return apiTokenAuth{}, err
	}
	return newAPITokenAuth(cfg.APIToken), nil
}

// runServer 暴露 bridge HTTP API。
func runServer(ctx context.Context, port int) (string, error) {
	preflight, err := runServePreflight(port)
	if err != nil {
		return "", err
	}
	defer preflight.service.Close()

	return runServeListen(ctx, preflight)
}

func RunServer(ctx context.Context, port int) (string, error) {
	return runServer(ctx, port)
}

// runServePreflight 按固定顺序执行启动预检，确保失败阶段可观测。
func runServePreflight(port int) (servePreflightState, error) {
	logStartupCheckpoint(startupStageConfig, "begin", "")
	store, err := bridgeorchestration.NewConfigStoreFromEnv()
	if err != nil {
		return servePreflightState{}, newServeStartupError(startupStageConfig, err)
	}
	logStartupCheckpoint(startupStageConfig, "ready", "")

	logStartupCheckpoint(startupStageOptions, "begin", fmt.Sprintf("port=%d", port))
	options, err := newServerOptionsFromEnv(port)
	if err != nil {
		return servePreflightState{}, newServeStartupError(startupStageOptions, err)
	}
	logStartupCheckpoint(startupStageOptions, "ready", fmt.Sprintf("bind_addr=%s", options.bindAddr))

	logStartupCheckpoint(startupStageSessionStore, "begin", fmt.Sprintf("path=%s", options.sessionsPath))
	service, err := bridgeorchestration.NewServiceWithSessionStorePath(store, options.sessionsPath)
	if err != nil {
		return servePreflightState{}, newServeStartupError(startupStageSessionStore, err)
	}
	logStartupCheckpoint(startupStageSessionStore, "ready", "")

	logStartupCheckpoint(startupStageRuntimes, "begin", "")
	if err := startServeRuntimes(service); err != nil {
		service.Close()
		return servePreflightState{}, newServeStartupError(startupStageRuntimes, err)
	}
	logStartupCheckpoint(startupStageRuntimes, "ready", "")

	return servePreflightState{
		options: options,
		service: service,
	}, nil
}

func startServeRuntimes(service *bridgeorchestration.Service) error {
	if err := service.StartBackgroundRuntimes(); err != nil {
		return fmt.Errorf("start background runtimes: %w", err)
	}
	if err := service.BootstrapSystemTasks(); err != nil {
		return fmt.Errorf("bootstrap system tasks: %w", err)
	}
	return nil
}

func runServeListen(ctx context.Context, preflight servePreflightState) (string, error) {
	logStartupCheckpoint(startupStageMobileWebRTC, "begin", "")
	mobileTransport, err := mobilewebrtc.Start(ctx, preflight.options.mobileWebRTC, preflight.service)
	if err != nil {
		return "", newServeStartupError(startupStageMobileWebRTC, err)
	}
	if mobileTransport != nil {
		defer mobileTransport.Close()
		logStartupCheckpoint(startupStageMobileWebRTC, "ready", "enabled=true")
	} else {
		logStartupCheckpoint(startupStageMobileWebRTC, "ready", "enabled=false")
	}

	server := &http.Server{
		Addr:              preflight.options.bindAddr,
		Handler:           newHTTPHandlerWithContext(ctx, preflight.service, preflight.options),
		ReadHeaderTimeout: serverReadHeaderTimeout,
	}

	go func() {
		<-ctx.Done()
		preflight.service.Close()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), serverShutdownTimeout)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	listenAddr := normalizeListenAddr(preflight.options.bindAddr)
	logStartupCheckpoint(startupStageListen, "begin", fmt.Sprintf("addr=http://%s", listenAddr))

	err = server.ListenAndServe()
	if err == nil || errors.Is(err, http.ErrServerClosed) {
		logStartupCheckpoint(startupStageListen, "stopped", "")
		return "", nil
	}
	return "", newServeStartupError(startupStageListen, err)
}

func normalizeListenAddr(bindAddr string) string {
	if strings.HasPrefix(bindAddr, ":") {
		return "0.0.0.0" + bindAddr
	}
	return bindAddr
}

func logStartupCheckpoint(stage string, status string, detail string) {
	line := fmt.Sprintf("startup checkpoint stage=%s status=%s", stage, status)
	if strings.TrimSpace(detail) != "" {
		line = fmt.Sprintf("%s %s", line, detail)
	}
	log.Print(line)
}

type transport struct {
	usecases     transportUsecases
	maxBodyBytes int64
	runContext   context.Context
}

// newHTTPHandler 注册所有 HTTP 路由并挂载认证/CORS 中间件链。
func newHTTPHandler(service *bridgeorchestration.Service, options serverOptions) http.Handler {
	return newHTTPHandlerWithContext(context.Background(), service, options)
}

func newHTTPHandlerWithContext(ctx context.Context, service *bridgeorchestration.Service, options serverOptions) http.Handler {
	if ctx == nil {
		ctx = context.Background()
	}
	transport := &transport{
		usecases:     newTransportUsecases(service),
		maxBodyBytes: options.maxBodyBytes,
		runContext:   ctx,
	}

	mux := http.NewServeMux()
	mountTransportRoutes(mux, transport)

	return withCORS(options.cors, withAuth(options.auth, mux))
}

type corsPolicy struct {
	allowedOrigins map[string]struct{}
}

func newCORSPolicy(origins []string) corsPolicy {
	allowed := make(map[string]struct{})
	if len(origins) == 0 {
		return corsPolicy{allowedOrigins: allowed}
	}

	for _, origin := range origins {
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

func newAPITokenAuth(token string) apiTokenAuth {
	return apiTokenAuth{token: strings.TrimSpace(token)}
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
