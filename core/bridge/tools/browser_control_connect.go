package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"ghost-os/bridge/tools/internal/tooljson"
	"ghost-os/bridge/tools/internal/toolparams"
)

// Boundary contract: Central only sends structured launch params;
// execution owns browser process discovery/spawn details.
const browserLaunchAction = "BROWSER_LAUNCH"

const (
	browserLaunchDisplayModeBackground = "background"
	browserLaunchDisplayModeForeground = "foreground"
	defaultBrowserReuseProbeTimeout    = 500 * time.Millisecond
)

func (t *BrowserControlTool) executeConnect(ctx context.Context, params map[string]any) (string, error) {
	endpoint, err := toolparams.RequiredString(params, "endpoint")
	if err != nil {
		return "", err
	}
	scopeKey := t.sessionScope(ctx)
	sessionID := toolparams.OptionalString(params, "session_id", "")
	if sessionID == "" {
		sessionID = newBrowserSessionID()
	}

	discovery, err := discoverBrowserEndpoint(endpoint, browserParseTimeout(params))
	if err != nil {
		return "", err
	}
	session, err := newRemoteBrowserSession(sessionID, scopeKey, discovery.WSEndpoint)
	if err != nil {
		return "", err
	}
	t.storeSession(scopeKey, session)
	return tooljson.Encode(map[string]any{
		"action":            "connect",
		"session_id":        sessionID,
		"ws_endpoint":       discovery.WSEndpoint,
		"debug_port":        discovery.DebugPort,
		"connected":         true,
		"session_recovered": false,
	})
}

func (t *BrowserControlTool) executeLaunch(ctx context.Context, params map[string]any, traceID string) (string, error) {
	if t.execution == nil {
		return "", fmt.Errorf("execution client is not configured")
	}
	scopeKey := t.sessionScope(ctx)
	endpoint, err := launchEndpoint(params)
	if err != nil {
		return "", err
	}
	if discovery, ok := discoverReusableLaunchEndpoint(endpoint, params); ok {
		return t.buildLaunchResponse(scopeKey, params, discovery, true)
	}

	launchPayload, err := browserLaunchPayload(params, endpoint)
	if err != nil {
		return "", err
	}

	if _, err := t.execution.Call(ctx, browserLaunchAction, launchPayload, traceID); err != nil {
		return "", fmt.Errorf("execution %s failed: %w", browserLaunchAction, err)
	}

	discovery, err := waitForWSEndpoint(endpoint, browserParseWaitTimeout(params))
	if err != nil {
		return "", err
	}
	return t.buildLaunchResponse(scopeKey, params, discovery, false)
}

func discoverReusableLaunchEndpoint(
	endpoint string,
	params map[string]any,
) (browserEndpointDiscovery, bool) {
	timeout := browserParseTimeout(params)
	if timeout <= 0 {
		timeout = defaultBrowserReuseProbeTimeout
	}
	discovery, err := discoverBrowserEndpoint(endpoint, timeout)
	if err != nil {
		return browserEndpointDiscovery{}, false
	}
	return discovery, true
}

func (t *BrowserControlTool) buildLaunchResponse(
	scopeKey string,
	params map[string]any,
	discovery browserEndpointDiscovery,
	reused bool,
) (string, error) {
	sessionID := toolparams.OptionalString(params, "session_id", "")
	if sessionID == "" {
		sessionID = newBrowserSessionID()
	}
	if err := t.replaceSession(scopeKey, sessionID, discovery.WSEndpoint); err != nil {
		return "", err
	}
	return tooljson.Encode(map[string]any{
		"action":            "launch",
		"session_id":        sessionID,
		"ws_endpoint":       discovery.WSEndpoint,
		"debug_port":        discovery.DebugPort,
		"connected":         true,
		"reused_existing":   reused,
		"session_recovered": false,
	})
}

func launchEndpoint(params map[string]any) (string, error) {
	endpoint := toolparams.OptionalString(params, "endpoint", "")
	if endpoint != "" {
		return endpoint, nil
	}
	port, ok := toolparams.OptionalInt(params, "debug_port")
	if !ok {
		port = defaultBrowserDebugPort
	}
	if port <= 0 || port > maxBrowserDebugPort {
		return "", fmt.Errorf("debug_port must be between 1 and %d", maxBrowserDebugPort)
	}
	return fmt.Sprintf("http://127.0.0.1:%d", port), nil
}

func browserLaunchPayload(params map[string]any, endpoint string) (map[string]any, error) {
	displayMode, err := browserLaunchDisplayMode(params)
	if err != nil {
		return nil, err
	}
	if explicit := toolparams.OptionalString(params, "command", ""); explicit != "" {
		payload := map[string]any{
			"command":      explicit,
			"display_mode": displayMode,
		}
		port, hasPort, err := browserOptionalDebugPort(params)
		if err != nil {
			return nil, err
		}
		if hasPort {
			payload["debug_port"] = port
		}
		return payload, nil
	}
	port, err := browserLaunchPort(params, endpoint)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"debug_port":   port,
		"display_mode": displayMode,
	}, nil
}

func browserLaunchDisplayMode(params map[string]any) (string, error) {
	mode := strings.ToLower(strings.TrimSpace(
		toolparams.OptionalString(params, "display_mode", browserLaunchDisplayModeBackground),
	))
	switch mode {
	case browserLaunchDisplayModeBackground, browserLaunchDisplayModeForeground:
		return mode, nil
	default:
		return "", fmt.Errorf(
			"display_mode must be one of: %s, %s",
			browserLaunchDisplayModeBackground,
			browserLaunchDisplayModeForeground,
		)
	}
}

func browserOptionalDebugPort(params map[string]any) (int, bool, error) {
	port, ok := toolparams.OptionalInt(params, "debug_port")
	if !ok {
		return 0, false, nil
	}
	if err := validateBrowserDebugPort(port); err != nil {
		return 0, false, err
	}
	return port, true, nil
}

func validateBrowserDebugPort(port int) error {
	if port <= 0 || port > maxBrowserDebugPort {
		return fmt.Errorf("debug_port must be between 1 and %d", maxBrowserDebugPort)
	}
	return nil
}

func browserLaunchPort(params map[string]any, endpoint string) (int, error) {
	if port, ok, err := browserOptionalDebugPort(params); err != nil {
		return 0, err
	} else if ok {
		return port, nil
	}
	port, err := portFromEndpoint(endpoint)
	if err != nil {
		return 0, fmt.Errorf("debug_port is required when command is omitted: %w", err)
	}
	return port, nil
}

func portFromEndpoint(endpoint string) (int, error) {
	trimmed := strings.TrimSpace(endpoint)
	if trimmed == "" {
		return 0, fmt.Errorf("endpoint is required")
	}
	if !strings.Contains(trimmed, "://") {
		trimmed = "http://" + trimmed
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return 0, fmt.Errorf("invalid endpoint %q: %w", endpoint, err)
	}
	portText := strings.TrimSpace(parsed.Port())
	if portText == "" {
		return 0, fmt.Errorf("endpoint %q must include an explicit port", endpoint)
	}
	port, err := strconv.Atoi(portText)
	if err != nil {
		return 0, fmt.Errorf("invalid endpoint port %q", portText)
	}
	if port <= 0 || port > maxBrowserDebugPort {
		return 0, fmt.Errorf("endpoint port must be between 1 and %d", maxBrowserDebugPort)
	}
	return port, nil
}

func waitForWSEndpoint(endpoint string, timeout time.Duration) (browserEndpointDiscovery, error) {
	if timeout <= 0 {
		timeout = defaultBrowserEndpointTimeout
	}
	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		discovery, err := discoverBrowserEndpoint(endpoint, timeout)
		if err == nil {
			return discovery, nil
		}
		lastErr = err
		time.Sleep(defaultBrowserEndpointPoll)
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("timed out waiting for endpoint")
	}
	return browserEndpointDiscovery{}, lastErr
}

func fetchWebSocketURL(base string, timeout time.Duration) (string, error) {
	return fetchWebSocketURLWithClient(base, &http.Client{Timeout: httpClientTimeout(timeout)})
}

func fetchWebSocketURLWithClient(base string, client *http.Client) (string, error) {
	if client == nil {
		return "", fmt.Errorf("http client is required")
	}
	url := browserVersionURL(base)
	resp, err := client.Get(url)
	if err != nil {
		return "", fmt.Errorf("fetch %s failed: %w", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("fetch %s failed: status %d", url, resp.StatusCode)
	}

	var payload struct {
		WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", fmt.Errorf("decode %s failed: %w", url, err)
	}
	if strings.TrimSpace(payload.WebSocketDebuggerURL) == "" {
		return "", fmt.Errorf("missing webSocketDebuggerUrl in %s", url)
	}
	return payload.WebSocketDebuggerURL, nil
}

func httpClientTimeout(timeout time.Duration) time.Duration {
	if timeout > 0 {
		return timeout
	}
	return defaultBrowserFetchTimeout
}
