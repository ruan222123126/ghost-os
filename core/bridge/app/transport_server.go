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

// newServerOptionsFromEnv 收敛 server 相关配置，优先读配置文件并回退环境变量。
func newServerOptionsFromEnv(port int) (serverOptions, error) {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return serverOptions{}, err
	}

	return serverOptions{
		bindAddr:     resolveBindAddrFromConfig(fileCfg, port),
		maxBodyBytes: defaultMaxRequestBodyBytes,
		cors:         newCORSPolicyFromConfig(fileCfg),
		auth:         newAPITokenAuthFromConfig(fileCfg),
	}, nil
}

func resolveBindAddrFromConfig(fileCfg bridgeFileConfig, port int) string {
	if configured := strings.TrimSpace(valueOrEnv(fileCfg.BindAddr, "GHOST_BIND_ADDR", "")); configured != "" {
		return configured
	}
	if configured := strings.TrimSpace(getenvDefault("GHOST_BIND_ADDR", "")); configured != "" {
		return configured
	}
	return net.JoinHostPort("127.0.0.1", fmt.Sprintf("%d", port))
}

// resolveBindAddr 优先使用显式绑定地址，否则回退到本地回环端口。
// 注意：当配置文件存在但读取/解析失败时，返回错误以避免 fail-open。
func resolveBindAddr(port int) (string, error) {
	fileCfg, _, err := loadBridgeFileConfig()
	if err != nil {
		return "", err
	}
	return resolveBindAddrFromConfig(fileCfg, port), nil
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
	if err := service.StartBackgroundRuntimes(); err != nil {
		service.Close()
		return "", err
	}
	if err := service.BootstrapSystemTasks(); err != nil {
		service.Close()
		return "", err
	}
	defer service.Close()

	options, err := newServerOptionsFromEnv(port)
	if err != nil {
		return "", err
	}
	server := &http.Server{
		Addr:              options.bindAddr,
		Handler:           newHTTPHandler(service, options),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		<-ctx.Done()
		service.Close()
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
	mux.HandleFunc("/api/agent/stream", transport.handleAgentStream)
	mux.HandleFunc("/api/questions/answer", transport.handleQuestionAnswer)
	mux.HandleFunc("/api/questions/answer/stream", transport.handleQuestionAnswerStream)
	mux.HandleFunc("/api/config", transport.handleConfig)
	mux.HandleFunc("/api/config/providers", transport.handleConfigProviders)
	mux.HandleFunc("/api/config/providers/", transport.handleConfigProviderByName)
	mux.HandleFunc("/api/config/active-provider", transport.handleActiveProvider)
	mux.HandleFunc("/api/sessions", transport.handleSessionsList)
	mux.HandleFunc("/api/sessions/", transport.handleSessionByID)
	mux.HandleFunc("/api/rss/briefing", transport.handleRSSBriefing)
	mux.HandleFunc("/api/rss/inbox/groups", transport.handleRSSInboxGroups)
	mux.HandleFunc("/api/rss/inbox", transport.handleRSSInbox)
	mux.HandleFunc("/api/rss/inbox/", transport.handleRSSInboxByID)
	mux.HandleFunc("/api/system/tasks", transport.handleSystemTasks)
	mux.HandleFunc("/api/tasks", transport.handleTasks)
	mux.HandleFunc("/api/tasks/", transport.handleTaskByID)

	return withCORS(options.cors, withAuth(options.auth, mux))
}
