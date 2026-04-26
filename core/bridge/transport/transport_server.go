package transport

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	bridgeconfig "ghost-os/bridge/config"
	bridgeorchestration "ghost-os/bridge/orchestration"
	"ghost-os/bridge/session"
)

type serverOptions struct {
	bindAddr     string
	sessionsPath string
	maxBodyBytes int64
	cors         corsPolicy
	auth         apiTokenAuth
}

const (
	serverReadHeaderTimeout = 5 * time.Second
	serverShutdownTimeout   = 5 * time.Second

	startupStageConfig       = "config"
	startupStageOptions      = "options"
	startupStageSessionStore = "session_store"
	startupStageRuntimes     = "runtimes"
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
	cfg, err := bridgeconfig.LoadServerConfig()
	if err != nil {
		return serverOptions{}, err
	}

	return serverOptions{
		bindAddr:     resolveBindAddrFromConfig(cfg, port),
		sessionsPath: cfg.SessionsPath,
		maxBodyBytes: bridgeorchestration.DefaultMaxRequestBodyBytes,
		cors:         newCORSPolicyFromConfig(cfg),
		auth:         newAPITokenAuthFromConfig(cfg),
	}, nil
}

func resolveBindAddrFromConfig(cfg bridgeconfig.ServerConfig, port int) string {
	if configured := strings.TrimSpace(cfg.BindAddr); configured != "" {
		return configured
	}
	return net.JoinHostPort("127.0.0.1", fmt.Sprintf("%d", port))
}

// resolveBindAddr 优先使用显式绑定地址，否则回退到本地回环端口。
// 注意：当配置文件存在但读取/解析失败时，返回错误以避免 fail-open。
func resolveBindAddr(port int) (string, error) {
	cfg, err := bridgeconfig.LoadServerConfig()
	if err != nil {
		return "", err
	}
	return resolveBindAddrFromConfig(cfg, port), nil
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

// runServePreflight 按固定顺序执行启动预检，确保失败阶段可观测。
func runServePreflight(port int) (servePreflightState, error) {
	logStartupCheckpoint(startupStageConfig, "begin", "")
	store, err := bridgeconfig.NewStoreFromEnv()
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
	sessionStore, err := session.NewStore(options.sessionsPath, session.StoreOptions{
		HumanLogFullEnabled: func() bool {
			return store.Snapshot().SessionHumanLogFullEnabled
		},
	})
	if err != nil {
		return servePreflightState{}, newServeStartupError(startupStageSessionStore, err)
	}
	logStartupCheckpoint(startupStageSessionStore, "ready", "")

	service := bridgeorchestration.NewService(store, sessionStore, nil)
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
	server := &http.Server{
		Addr:              preflight.options.bindAddr,
		Handler:           newHTTPHandler(preflight.service, preflight.options),
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

	err := server.ListenAndServe()
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
	service      *bridgeorchestration.Service
	maxBodyBytes int64
}

// newHTTPHandler 注册所有 HTTP 路由并挂载认证/CORS 中间件链。
func newHTTPHandler(service *bridgeorchestration.Service, options serverOptions) http.Handler {
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
	mux.HandleFunc("/api/prompts/system", transport.handleSystemPrompts)
	mux.HandleFunc("/api/sessions", transport.handleSessionsList)
	mux.HandleFunc("/api/sessions/", transport.handleSessionByID)
	mux.HandleFunc("/api/rss/briefing", transport.handleRSSBriefing)
	mux.HandleFunc("/api/rss/inbox/groups", transport.handleRSSInboxGroups)
	mux.HandleFunc("/api/rss/inbox", transport.handleRSSInbox)
	mux.HandleFunc("/api/rss/inbox/", transport.handleRSSInboxByID)
	mux.HandleFunc("/api/system/tasks", transport.handleSystemTasks)
	mux.HandleFunc("/api/tasks", transport.handleTasks)
	mux.HandleFunc("/api/tasks/", transport.handleTaskByID)
	mux.HandleFunc("/api/skills", transport.handleSkills)
	mux.HandleFunc("/api/skills/", transport.handleSkillByID)
	mux.HandleFunc("/api/tools/screen/find-icon/template", transport.handleFindIconTemplateUpload)
	mux.HandleFunc("/api/tools/screen/find-icon/preview", transport.handleFindIconPreview)
	mux.HandleFunc("/api/tools/screen/mouse-position", transport.handleMousePosition)
	mux.HandleFunc("/api/tools", transport.handleTools)
	mux.HandleFunc("/api/tools/", transport.handleToolByName)

	return withCORS(options.cors, withAuth(options.auth, mux))
}
